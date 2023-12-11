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

func (ep *StreamEpisode) GetFormatIdx(formatName string) (int, error) {
	idx := 0
	var err error = nil
	if formatName == "auto" {
		// at the moment, the best format is always the first
		return idx, nil
	} else {
		formatFound := false
		for i, f := range ep.Formats {
			if f.Name == formatName {
				idx = i
				formatFound = true
			}
		}
		if !formatFound { err = &FormatNotFoundError{FormatName: formatName} }
		return idx, err
	}
}

func (ep *StreamEpisode) Download(formatIdx int, chapterIdx int, start time.Duration, stop time.Duration, filename string, overwrite bool, continueDl bool, ratelimit float64, cli *Cli) error {
	var err error
	var nextChunk int = 0
	// video file
	if filename == "" {
		filename = ep.ProposedFilename
	}
	if !overwrite && !continueDl {
		if _, err := os.Stat(filename); err == nil {
			return &FileExistsError{Filename: filename}
		}
	}
	videoFile, err := os.OpenFile(filename, os.O_RDWR | os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer videoFile.Close()
	if overwrite {
		videoFile.Truncate(0)
	}
	// always seek to the end
	videoFile.Seek(0, io.SeekEnd)
	// info file
	infoFilename := filename + ".dl-info"
	if continueDl {
		infoFileData, err := os.ReadFile(infoFilename)
		if err != nil {
			cli.ErrorMessage(fmt.Sprint(err), err)
			return errors.New("could not access download info file, can't continue download")
		}
		i, err := strconv.ParseInt(string(infoFileData), 10, 32)
		nextChunk = int(i)
		if err != nil {
			return err
		}
	}
	infoFile, err := os.OpenFile(infoFilename, os.O_RDWR | os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	infoFile.Truncate(0)
	infoFile.Seek(0, io.SeekStart)
	infoFile.Write([]byte(strconv.Itoa(nextChunk)))
	if err != nil {
		return err
	}
	// download
	chunklist, err := GetStreamChunkList(ep.Formats[formatIdx])
	if chapterIdx >= 0 {
		if start < 0 {
			start = time.Duration(ep.Chapters[chapterIdx].Offset)
		}
		if stop < 0 && chapterIdx+1 < len(ep.Chapters) {
			// next chapter is stop
			stop = time.Duration(ep.Chapters[chapterIdx+1].Offset)
		}
	}
	chunklist = chunklist.Cut(start, stop)
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
		cli.Progress(progress, actualRate, false, false, 0, ep.Title);
		cli.Aborted()
	}()
	for i, chunk := range chunklist.Chunks {
		if i < nextChunk { continue }
		var time1 int64
		var data []byte
		retries := 0
		for {
			if keyboardInterrupt { break }
			time1 = time.Now().UnixNano()
			cli.Progress(progress, actualRate, false, true, retries, ep.Title)
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
		actualRate = rate - max(rate - ratelimit, 0)
		progress = float32(i+1) / float32(len(chunklist.Chunks))
		delayNow := bufferDt > RatelimitDelayAfter
		cli.Progress(progress, actualRate, delayNow, false, retries, ep.Title)
		if delayNow {
			bufferDt = 0
			// this simulates that the buffering is finished and the player is playing
			time.Sleep(time.Duration(RatelimitDelay * float64(time.Second)))
		} else if rate > ratelimit {
			// slow down, we are too fast.
			deferTime := (rate - ratelimit) / ratelimit * dtDownload
			time.Sleep(time.Duration(deferTime * float64(time.Second)))
		}
		videoFile.Write(data)
		nextChunk++
		infoFile.Truncate(0)
		infoFile.Seek(0, io.SeekStart)
		infoFile.Write([]byte(strconv.Itoa(nextChunk)))
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
