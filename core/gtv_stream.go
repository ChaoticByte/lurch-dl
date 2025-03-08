// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MaxRetries = 5

type Chapter struct {
	Index  int           `json:"index"`
	Title  string        `json:"title"`
	Offset time.Duration `json:"offset"`
}

type StreamEpisode struct {
	Episode string        `json:"episode"`
	Formats []VideoFormat `json:"formats"`
	Title   string        `json:"title"`
	// ProposedFilename string `json:"proposed_filename"`
	PlaylistUrl string    `json:"playlist_url"`
	Chapters    []Chapter `json:"chapters"`
}

func (ep *StreamEpisode) GetFormatByName(formatName string) (VideoFormat, error) {
	var idx int
	var err error = nil
	if formatName == "auto" {
		// at the moment, the best format is always the first -> 0
		return ep.Formats[idx], nil
	} else {
		formatFound := false
		for i, f := range ep.Formats {
			if f.Name == formatName {
				idx = i
				formatFound = true
			}
		}
		if !formatFound {
			err = &FormatNotFoundError{FormatName: formatName}
		}
		return ep.Formats[idx], err
	}
}

func (ep *StreamEpisode) GetProposedFilename(chapterIdx int) string {
	if chapterIdx >= 0 && chapterIdx < len(ep.Chapters) {
		return fmt.Sprintf("GTV%04s - %v. %s.ts", ep.Episode, chapterIdx+1, sanitizeUnicodeFilename(ep.Chapters[chapterIdx].Title))
	} else {
		return sanitizeUnicodeFilename(ep.Title) + ".ts"
	}
}

func (ep *StreamEpisode) Download(args Arguments, ui UserInterface, interruptChan chan os.Signal) error {
	// Set automatic values
	if args.OutputFile == "" {
		args.OutputFile = ep.GetProposedFilename(args.ChapterIdx)
	}
	if args.ChapterIdx >= 0 {
		if args.StartDuration < 0 {
			args.StartDuration = time.Duration(ep.Chapters[args.ChapterIdx].Offset)
		}
		if args.StopDuration < 0 && args.ChapterIdx+1 < len(ep.Chapters) {
			// next chapter is stop
			args.StopDuration = time.Duration(ep.Chapters[args.ChapterIdx+1].Offset)
		}
	}
	//
	var err error
	var nextChunk int = 0
	var videoFile *os.File
	var infoFile *os.File
	var infoFilename string
	if !args.Overwrite && !args.ContinueDl {
		if _, err := os.Stat(args.OutputFile); err == nil {
			return &FileExistsError{Filename: args.OutputFile}
		}
	}
	videoFile, err = os.OpenFile(args.OutputFile, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer videoFile.Close()
	if args.Overwrite {
		videoFile.Truncate(0)
	}
	// always seek to the end
	videoFile.Seek(0, io.SeekEnd)
	// info file
	infoFilename = args.OutputFile + ".dl-info"
	if args.ContinueDl {
		infoFileData, err := os.ReadFile(infoFilename)
		if err != nil {
			return errors.New("could not access download info file, can't continue download")
		}
		i, err := strconv.ParseInt(string(infoFileData), 10, 32)
		nextChunk = int(i)
		if err != nil {
			return err
		}
	}
	infoFile, err = os.OpenFile(infoFilename, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	infoFile.Truncate(0)
	infoFile.Seek(0, io.SeekStart)
	_, err = infoFile.Write([]byte(strconv.Itoa(nextChunk)))
	if err != nil {
		return err
	}
	// download
	format, _ := ep.GetFormatByName(args.FormatName) // we don't have to check the error, as it was already checked by CliRun()
	chunklist, err := GetStreamChunkList(format)
	chunklist = chunklist.Cut(args.StartDuration, args.StopDuration)
	if err != nil {
		return err
	}
	var bufferDt float64
	var progress float32
	var actualRate float64
	keyboardInterrupt := false
	signal.Notify(interruptChan, os.Interrupt)
	go func() {
		// Handle Interrupts
		<-interruptChan
		keyboardInterrupt = true
		ui.DownloadProgress(progress, actualRate, false, false, 0, ep.Title)
		ui.Aborted()
	}()
	for i, chunk := range chunklist.Chunks {
		if i < nextChunk {
			continue
		}
		var time1 int64
		var data []byte
		retries := 0
		for {
			if keyboardInterrupt {
				break
			}
			time1 = time.Now().UnixNano()
			ui.DownloadProgress(progress, actualRate, false, true, retries, ep.Title)
			data, err = httpGet(chunklist.BaseUrl+"/"+chunk, []http.Header{ApiHeadersBase, ApiHeadersVideoAdditional}, time.Second*5)
			if err != nil {
				if retries == MaxRetries {
					return err
				}
				retries++
				continue
			}
			break
		}
		if keyboardInterrupt {
			break
		}
		var dtDownload float64 = float64(time.Now().UnixNano()-time1) / 1000000000.0
		rate := float64(len(data)) / dtDownload
		actualRate = rate - max(rate-args.Ratelimit, 0)
		progress = float32(i+1) / float32(len(chunklist.Chunks))
		delayNow := bufferDt > RatelimitDelayAfter
		ui.DownloadProgress(progress, actualRate, delayNow, false, retries, ep.Title)
		if delayNow {
			bufferDt = 0
			// this simulates that the buffering is finished and the player is playing
			time.Sleep(time.Duration(RatelimitDelay * float64(time.Second)))
		} else if rate > args.Ratelimit {
			// slow down, we are too fast.
			deferTime := (rate - args.Ratelimit) / args.Ratelimit * dtDownload
			time.Sleep(time.Duration(deferTime * float64(time.Second)))
		}
		videoFile.Write(data)
		nextChunk++
		infoFile.Truncate(0)
		infoFile.Seek(0, io.SeekStart)
		infoFile.Write([]byte(strconv.Itoa(nextChunk)))
		var dtIteration float64 = float64(time.Now().UnixNano()-time1) / 1000000000.0
		if !delayNow {
			bufferDt += dtIteration
		}
	}
	infoFile.Close()
	if !keyboardInterrupt {
		err := os.Remove(infoFilename)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetStreamEpisode(episode string) (StreamEpisode, error) {
	ep := StreamEpisode{}
	ep.Episode = episode
	info_data, err := httpGet(
		fmt.Sprintf(ApiBaseurlStreamEpisodeInfo, episode),
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second*10,
	)
	if err != nil {
		return ep, err
	}
	// Title
	json.Unmarshal(info_data, &ep)
	ep.Title = strings.ToValidUTF8(ep.Title, "")
	// Sort Chapters, correct offset and set index
	sort.Slice(ep.Chapters, func(i int, j int) bool {
		return ep.Chapters[i].Offset < ep.Chapters[j].Offset
	})
	for i := range ep.Chapters {
		ep.Chapters[i].Offset = ep.Chapters[i].Offset * time.Second
		ep.Chapters[i].Index = i
	}
	// Formats
	playlist_url_data, err := httpGet(
		fmt.Sprintf(ApiBaseurlStreamEpisodePlInfo, episode),
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second*10,
	)
	if err != nil {
		return ep, err
	}
	json.Unmarshal(playlist_url_data, &ep)
	playlist_data, err := httpGet(
		ep.PlaylistUrl,
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second*10,
	)
	ep.Formats = parseAvailFormatsFromM3u8(string(playlist_data))
	return ep, err
}

func GetStreamChunkList(video VideoFormat) (ChunkList, error) {
	baseUrl := video.Url[:strings.LastIndex(video.Url, "/")]
	data, err := httpGet(video.Url, []http.Header{ApiHeadersBase, ApiHeadersMetaAdditional}, time.Second*10)
	if err != nil {
		return ChunkList{}, err
	}
	chunklist, err := parseChunkListFromM3u8(string(data), baseUrl)
	return chunklist, err
}
