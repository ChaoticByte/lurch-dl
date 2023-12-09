// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"
)

// The following two values are used to simulate buffering
const RatelimitDelay = 2.0 // in Seconds; How long to delay the next chunk download.
const RatelimitDelayAfter = 5.0 // in Seconds; Delay the next chunk download after this duration.

type GtvVideo struct {
	Category string
	Id string
}

var videoUrlRegex = regexp.MustCompile(`gronkh\.tv\/([a-z]+)\/([0-9]+)`)

func ParseGtvVideoUrl(url string) (GtvVideo, error) {
	video := GtvVideo{}
	match := videoUrlRegex.FindStringSubmatch(url)
	if match == nil || len(match) < 2 {
		return video, errors.New("Could not parse URL " + url)
	}
	video.Category = match[1]
	video.Id = match[2]
	return video, nil
}

type FileExistsError struct {
	Filename string
}

func (err *FileExistsError) Error() string {
	return "File '" + err.Filename + "' already exists."
}

const MaxRetries = 5

func DownloadStreamEpisode(episodeMeta StreamEpisodeMeta, format VideoFormat, chapterIdx int, start time.Duration, stop time.Duration, filename string, overwrite bool, continueDl bool, ratelimit float64, cli *Cli) error {
	var err error
	var nextChunk int = 0
	// video file
	if filename == "" {
		filename = episodeMeta.ProposedFilename
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
	chunklist, err := GetVideoChunkList(format)
	if chapterIdx >= 0 {
		if start < 0 {
			start = time.Duration(episodeMeta.Chapters[chapterIdx].Offset)
		}
		if stop < 0 && chapterIdx+1 < len(episodeMeta.Chapters) {
			// next chapter is stop
			stop = time.Duration(episodeMeta.Chapters[chapterIdx+1].Offset)
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
		cli.Progress(progress, actualRate, false, false, 0);
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
			cli.Progress(progress, actualRate, false, true, retries)
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
		cli.Progress(progress, actualRate, delayNow, false, retries)
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
