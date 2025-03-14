// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MaxRetries = 5
// The following two values are used to simulate buffering
const RatelimitDelay = 2.0      // in Seconds; How long to delay the next chunk download.
const RatelimitDelayAfter = 5.0 // in Seconds; Delay the next chunk download after this duration.

const ApiBaseurlStreamEpisodeInfo = "https://api.gronkh.tv/v1/video/info?episode=%s"
const ApiBaseurlStreamEpisodePlInfo = "https://api.gronkh.tv/v1/video/playlist?episode=%s"

type DownloadProgress struct {
	Aborted bool
	Error error
	Success bool
	Delaying bool
	Progress float32
	Rate float64
	Retries int
	Title string
	Waiting bool
}

var ApiHeadersBase = http.Header{
	"User-Agent":      {"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0"},
	"Accept-Language": {"de,en-US;q=0.7,en;q=0.3"},
	//"Accept-Encoding": {"gzip"},
	"Origin":         {"https://gronkh.tv"},
	"Referer":        {"https://gronkh.tv/"},
	"Connection":     {"keep-alive"},
	"Sec-Fetch-Dest": {"empty"},
	"Sec-Fetch-Mode": {"cors"},
	"Sec-Fetch-Site": {"same-site"},
	"Pragma":         {"no-cache"},
	"Cache-Control":  {"no-cache"},
	"TE":             {"trailers"},
}

var ApiHeadersMetaAdditional = http.Header{
	"Accept": {"application/json, text/plain, */*"},
}

var ApiHeadersVideoAdditional = http.Header{
	"Accept": {"*/*"},
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
	// Length
	ep.Length = ep.Length * time.Second
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

func DownloadEpisode(
	ep StreamEpisode,
	chapter Chapter,
	formatName string,
	outputFile string,
	overwrite bool,
	continueDl bool,
	startDuration time.Duration,
	stopDuration time.Duration,
	ratelimit float64,
	interruptChan chan os.Signal,
) iter.Seq[DownloadProgress] {
	return func (yield func(DownloadProgress) bool) {
		// Set automatic values
		if outputFile == "" {
			outputFile = ep.GetProposedFilename(chapter.Index)
		}
		if chapter.Index >= 0 {
			if startDuration < 0 {
				startDuration = time.Duration(ep.Chapters[chapter.Index].Offset)
			}
			if stopDuration < 0 && chapter.Index+1 < len(ep.Chapters) {
				// next chapter is stop
				stopDuration = time.Duration(ep.Chapters[chapter.Index+1].Offset)
			}
		}
		//
		var err error
		var nextChunk int = 0
		var videoFile *os.File
		var infoFile *os.File
		var infoFilename string
		if !overwrite && !continueDl {
			if _, err := os.Stat(outputFile); err == nil {
				yield(DownloadProgress{Error: &FileExistsError{Filename: outputFile}})
				return
			}
		}
		videoFile, err = os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE, 0660)
		if err != nil {
			yield(DownloadProgress{Error: err})
			return
		}
		defer videoFile.Close()
		if overwrite {
			videoFile.Truncate(0)
		}
		// always seek to the end
		videoFile.Seek(0, io.SeekEnd)
		// info file
		infoFilename = outputFile + ".dl-info"
		if continueDl {
			infoFileData, err := os.ReadFile(infoFilename)
			if err != nil {
				yield(DownloadProgress{Error: errors.New("could not access download info file, can't continue download")})
				return
			}
			i, err := strconv.ParseInt(string(infoFileData), 10, 32)
			nextChunk = int(i)
			if err != nil {
				yield(DownloadProgress{Error: err})
				return
			}
		}
		infoFile, err = os.OpenFile(infoFilename, os.O_RDWR|os.O_CREATE, 0660)
		if err != nil {
			yield(DownloadProgress{Error: err})
			return
		}
		infoFile.Truncate(0)
		infoFile.Seek(0, io.SeekStart)
		_, err = infoFile.Write([]byte(strconv.Itoa(nextChunk)))
		if err != nil {
			yield(DownloadProgress{Error: err})
			return
		}
		// download
		format, _ := ep.GetFormatByName(formatName) // we don't have to check the error, as it was already checked by CliRun()
		chunklist, err := GetStreamChunkList(format)
		chunklist = chunklist.Cut(startDuration, stopDuration)
		if err != nil {
			yield(DownloadProgress{Error: err})
			return
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
			yield(DownloadProgress{Aborted: true, Progress: progress, Rate: actualRate, Retries: 0, Title: ep.Title})
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
				if !yield(DownloadProgress{Progress: progress, Rate: actualRate, Delaying: false, Waiting: true, Retries: retries, Title: ep.Title}) { return }
				data, err = httpGet(chunklist.BaseUrl+"/"+chunk, []http.Header{ApiHeadersBase, ApiHeadersVideoAdditional}, time.Second*5)
				if err != nil {
					if retries == MaxRetries {
						yield(DownloadProgress{Error: err})
						return
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
			actualRate = rate - max(rate-ratelimit, 0)
			progress = float32(i+1) / float32(len(chunklist.Chunks))
			delayNow := bufferDt > RatelimitDelayAfter
			if !yield(DownloadProgress{Progress: progress, Rate: actualRate, Delaying: delayNow, Waiting: false, Retries: retries, Title: ep.Title}) { return }
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
			var dtIteration float64 = float64(time.Now().UnixNano()-time1) / 1000000000.0
			if !delayNow {
				bufferDt += dtIteration
			}
		}
		infoFile.Close()
		if !keyboardInterrupt {
			err := os.Remove(infoFilename)
			if err != nil {
				yield(DownloadProgress{Progress: progress, Rate: actualRate, Error: err})
				return
			}
		}
		yield(DownloadProgress{Progress: progress, Rate: actualRate, Success: true})
	}
}
