// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

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
	Index int `json:"index"`
	Title string `json:"title"`
	Offset time.Duration `json:"offset"`
}

type StreamEpisode struct {
	Episode string
	Formats []VideoFormat
	Title string `json:"title"`
	ProposedFilename string
	PlaylistUrl string `json:"playlist_url"`
	Chapters []Chapter `json:"chapters"`
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
		if !formatFound { err = &FormatNotFoundError{FormatName: formatName} }
		return ep.Formats[idx], err
	}
}

func (ep *StreamEpisode) Download(args CliArguments) error {
	var err error
	var nextChunk int = 0
	var videoFile *os.File
	var infoFile *os.File
	var infoFilename string
	if !CliJsonData {
		if !args.Overwrite && !args.ContinueDl {
			if _, err := os.Stat(args.OutputFile); err == nil {
				return &FileExistsError{Filename: args.OutputFile}
			}
		}
		videoFile, err = os.OpenFile(args.OutputFile, os.O_RDWR | os.O_CREATE, 0660)
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
				CliErrorMessage(err)
				return errors.New("could not access download info file, can't continue download")
			}
			i, err := strconv.ParseInt(string(infoFileData), 10, 32)
			nextChunk = int(i)
			if err != nil {
				return err
			}
		}
		infoFile, err = os.OpenFile(infoFilename, os.O_RDWR | os.O_CREATE, 0660)
		if err != nil {
			return err
		}
		infoFile.Truncate(0)
		infoFile.Seek(0, io.SeekStart)
		infoFile.Write([]byte(strconv.Itoa(nextChunk)))
		if err != nil {
			return err
		}
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
	keyboardInterruptChan := make(chan os.Signal, 1)
	signal.Notify(keyboardInterruptChan, os.Interrupt)
	go func() {
		// Handle Keyboard Interrupts
		<-keyboardInterruptChan
		keyboardInterrupt = true
		CliDownloadProgress(progress, actualRate, false, false, 0, ep.Title);
		CliAborted()
	}()
	for i, chunk := range chunklist.Chunks {
		if i < nextChunk { continue }
		var time1 int64
		var data []byte
		retries := 0
		for {
			if keyboardInterrupt { break }
			time1 = time.Now().UnixNano()
			CliDownloadProgress(progress, actualRate, false, true, retries, ep.Title)
			data, err = httpGet(chunklist.BaseUrl + "/" + chunk, []http.Header{ApiHeadersBase, ApiHeadersVideoAdditional}, time.Second * 5)
			if err != nil {
				if retries == MaxRetries {
					return err
				}
				retries++
				continue
			}
			break
		}
		if keyboardInterrupt { break }
		var dtDownload float64 = float64(time.Now().UnixNano() - time1) / 1000000000.0
		rate := float64(len(data)) / dtDownload
		actualRate = rate - max(rate - args.Ratelimit, 0)
		progress = float32(i+1) / float32(len(chunklist.Chunks))
		delayNow := bufferDt > RatelimitDelayAfter
		CliDownloadProgress(progress, actualRate, delayNow, false, retries, ep.Title)
		if delayNow {
			bufferDt = 0
			// this simulates that the buffering is finished and the player is playing
			time.Sleep(time.Duration(RatelimitDelay * float64(time.Second)))
		} else if rate > args.Ratelimit {
			// slow down, we are too fast.
			deferTime := (rate - args.Ratelimit) / args.Ratelimit * dtDownload
			time.Sleep(time.Duration(deferTime * float64(time.Second)))
		}
		if CliJsonData {
			PrintJson(JsonVideoData{DataChunkIdx: i, Data: data})
		} else {
			videoFile.Write(data)
			nextChunk++
			infoFile.Truncate(0)
			infoFile.Seek(0, io.SeekStart)
			infoFile.Write([]byte(strconv.Itoa(nextChunk)))
		}
		var dtIteration float64 = float64(time.Now().UnixNano() - time1) / 1000000000.0
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

func GetStreamEpisode(episode string, chapterIdx int) (StreamEpisode, error) {
	meta := StreamEpisode{}
	meta.Episode = episode
	info_data, err := httpGet(
		fmt.Sprintf(ApiBaseurlStreamEpisodeInfo, episode),
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second * 10,
	)
	if err != nil { return meta, err }
	// Title
	json.Unmarshal(info_data, &meta)
	meta.Title = strings.ToValidUTF8(meta.Title, "")
	// sanitized proposedFilename
	if chapterIdx >= 0 && chapterIdx < len(meta.Chapters) {
		meta.ProposedFilename = fmt.Sprintf("GTV%04s - %v. %s.ts", episode, chapterIdx+1, meta.Chapters[chapterIdx].Title)
	} else {
		meta.ProposedFilename = sanitizeUnicodeFilename(meta.Title) + ".ts"
	}
	// Sort Chapters, correct offset and set index
	sort.Slice(meta.Chapters, func(i int, j int) bool {
		return meta.Chapters[i].Offset < meta.Chapters[j].Offset
	})
	for i := range meta.Chapters {
		meta.Chapters[i].Offset = meta.Chapters[i].Offset * time.Second
		meta.Chapters[i].Index = i
	}
	// Formats
	playlist_url_data, err := httpGet(
		fmt.Sprintf(ApiBaseurlStreamEpisodePlInfo, episode),
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second * 10,
	)
	if err != nil {
		return meta, err
	}
	json.Unmarshal(playlist_url_data, &meta)
	playlist_data, err := httpGet(
		meta.PlaylistUrl,
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second * 10,
	)
	meta.Formats = parseAvailFormatsFromM3u8(string(playlist_data))
	return meta, err
}

func GetStreamChunkList(video VideoFormat) (ChunkList, error) {
	baseUrl := video.Url[:strings.LastIndex(video.Url, "/")]
	data, err := httpGet(video.Url, []http.Header{ApiHeadersBase, ApiHeadersMetaAdditional}, time.Second * 10)
	if err != nil {
		return ChunkList{}, err
	}
	chunklist, err := parseChunkListFromM3u8(string(data), baseUrl)
	return chunklist, err
}
