// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MaxRetries = 5
// The following two values are used to simulate buffering
const RatelimitDelay = 2.0      // in Seconds; How long to delay the next chunk download.
const RatelimitDelayAfter = 5.0 // in Seconds; Delay the next chunk download after this duration.

const ApiBaseurlStreamEpisodeInfo   = "https://backend.gronkh.tv/v3/videos/episode/%s"

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
	"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.7727.56 Safari/537.36"},
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

//

type Category struct {
	Title  string `json:"title"`
}

type Chapter struct {
	Index       int           `json:"index"`
	StartOffset time.Duration `json:"start_offset"`
	EndOffset   time.Duration `json:"end_offset"`
	Duration    time.Duration `json:"duration"`
	Category    Category      `json:"category"`
}

type VideoTag struct {
	Id    int    `json:"tag_id"`
	Title string `json:"title"`
}

//

type ChunkList struct {
	BaseUrl       string
	Chunks        []string
	ChunkDuration float64
}

func (cl *ChunkList) Cut(from time.Duration, to time.Duration) ChunkList {
	var newChunks []string
	var firstChunk = 0
	if from != -1 {
		firstChunk = int(from.Seconds() / cl.ChunkDuration)
	}
	if to != -1 {
		lastChunk := min(int(to.Seconds()/cl.ChunkDuration)+1, len(cl.Chunks))
		newChunks = cl.Chunks[firstChunk:lastChunk]
	} else {
		newChunks = cl.Chunks[firstChunk:]
	}
	return ChunkList{
		BaseUrl:       cl.BaseUrl,
		Chunks:        newChunks,
		ChunkDuration: cl.ChunkDuration,
	}
}

//

type VideoFormat struct {
	Name string `json:"format"`
	Url  string `json:"url"`
}

func (vf *VideoFormat) StreamChunkList() (ChunkList, error) {
	baseUrl := vf.Url[:strings.LastIndex(vf.Url, "/")]
	data, err := httpGet(vf.Url, []http.Header{ApiHeadersBase, ApiHeadersMetaAdditional}, time.Second*10)
	if err != nil {
		return ChunkList{}, err
	}
	chunklist, err := parseChunkListFromM3u8(string(data), baseUrl)
	return chunklist, err
}

//

var videoUrlRegex = regexp.MustCompile(`gronkh\.tv\/([a-z]+)\/([0-9]+)`)

func ParseGtvVideoUrl(url string) (string, error) {
	match := videoUrlRegex.FindStringSubmatch(url)
	if len(match) < 2 {
		return "", &GtvVideoUrlParseError{Url: url}
	}
	cat := match[1]
	if cat != "stream" {
		return "", &VideoCategoryUnsupportedError{Category: cat}
	}
	return match[2], nil
}

//

type StreamEpisodeResponse struct {
	StreamEpisode StreamEpisode `json:"data"`
}

type StreamEpisodeMeta struct {
	Duration time.Duration `json:"duration"`
}

type StreamEpisodeUrls struct {
	Playlist string `json:"playlist"`
}

type StreamEpisode struct {
	// 
	Id            string            `json:"id"`
	EpisodeNumber int               `json:"episode"`
	Title         string            `json:"title"`
	Views         int               `json:"views"`
	Meta          StreamEpisodeMeta `json:"meta"`
	Urls          StreamEpisodeUrls `json:"urls"`
	Chapters      []Chapter         `json:"chapters"`
	Tags          []VideoTag        `json:"tags"`
	//
	Formats       []VideoFormat     `json:"formats"`
}

func (ep *StreamEpisode) FormatByName(formatName string) (VideoFormat, error) {
	var idx int
	var err error = nil
	if formatName == "auto" {
		// since gronkh.tv 0.2.2, the last format is the best
		return ep.Formats[len(ep.Formats)-1], nil
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

func (ep *StreamEpisode) ChapterByNumber(number int) (*Chapter, error) {
	var chapter *Chapter
	idx := number-1
	if idx >= 0 && idx >= len(ep.Chapters) {
		return chapter, &ChapterNotFoundError{ChapterNum: number}
	}
	if len(ep.Chapters) > 0 && idx >= 0 {
		chapter = &ep.Chapters[idx]
	}
	return chapter, nil
}

func (ep *StreamEpisode) ProposeFilename(chapter *Chapter) string {
	if chapter != nil {
		return fmt.Sprintf("GTV%04d - %v. %s.ts", ep.EpisodeNumber, chapter.Index, sanitizeUnicodeFilename(ep.Chapters[chapter.Index].Category.Title))
	} else {
		return sanitizeUnicodeFilename(ep.Title) + ".ts"
	}
}

func (ep *StreamEpisode) DownloadStreamEpisode(
	chapter *Chapter,
	formatName string,
	outputFile string,
	overwrite bool,
	continueDl bool,
	startOffset time.Duration,
	stopOffset time.Duration,
	ratelimit float64,
	interruptChan chan os.Signal,
) iter.Seq[DownloadProgress] {
	return func (yield func(DownloadProgress) bool) {
		// Set automatic values
		if outputFile == "" {
			outputFile = ep.ProposeFilename(chapter)
		}
		if chapter != nil {
			if startOffset < 0 {
				startOffset = time.Duration(ep.Chapters[chapter.Index].StartOffset)
			}
			if stopOffset < 0 {
				// next chapter is stop
				stopOffset = time.Duration(ep.Chapters[chapter.Index].EndOffset)
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
				yield(DownloadProgress{Error: &DownloadInfoFileReadError{}})
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
		format, _ := ep.FormatByName(formatName) // we don't have to check the error, as it was already checked by CliRun()
		chunklist, err := format.StreamChunkList()
		chunklist = chunklist.Cut(startOffset, stopOffset)
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
			if i < nextChunk { continue }
			var time1 int64
			var data []byte
			retries := 0
			for {
				if keyboardInterrupt { break }
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
			if keyboardInterrupt { break }
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

func StreamEpisodeFromUrl(url string) (StreamEpisode, error) {
	epNumber, err := ParseGtvVideoUrl(url)
	if err != nil { return StreamEpisode{}, err }
	info_data, err := httpGet(
		fmt.Sprintf(ApiBaseurlStreamEpisodeInfo, epNumber),
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second*10,
	)
	if err != nil { return StreamEpisode{}, err }
	// Parse JSON Response
	epContainer := StreamEpisodeResponse{}
	json.Unmarshal(info_data, &epContainer)
	ep := epContainer.StreamEpisode
	// Title
	ep.Title = strings.ToValidUTF8(ep.Title, "")
	// Correct Duration
	ep.Meta.Duration *= time.Second
	// Sort Chapters, correct Offset and set index
	chaptersProcessed := []Chapter{}
	for _, chap := range ep.Chapters {
		// filter out invalid data
		if chap.EndOffset == 1 || chap.Duration == 1 || chap.Category.Title == "" {
			continue
		}
		chaptersProcessed = append(chaptersProcessed, chap)
	}
	sort.Slice(chaptersProcessed, func(i int, j int) bool {
		return chaptersProcessed[i].StartOffset < chaptersProcessed[j].StartOffset
	})
	for i := range chaptersProcessed {
		chaptersProcessed[i].StartOffset *= time.Second
		chaptersProcessed[i].EndOffset *= time.Second
		chaptersProcessed[i].Duration *= time.Second
		chaptersProcessed[i].Index = i
	}
	ep.Chapters = chaptersProcessed
	// Formats
	playlist_data, err := httpGet(
		ep.Urls.Playlist,
		[]http.Header{ApiHeadersBase, ApiHeadersMetaAdditional},
		time.Second*10,
	)
	ep.Formats = parseAvailFormatsFromM3u8(string(playlist_data))
	return ep, err
}
