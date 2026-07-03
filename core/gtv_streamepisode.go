package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const ApiBaseurlStreamEpisodeInfo   = "https://backend.gronkh.tv/v3/videos/episode/%s"

type ResponseStreamEpisode struct {
	StreamEpisode StreamEpisode `json:"data"`
}

type StreamEpCategory struct {
	Title  string `json:"title"`
}

type StreamEpChapter struct {
	Index       int              `json:"index"`
	StartOffset time.Duration    `json:"start_offset"`
	EndOffset   time.Duration    `json:"end_offset"`
	Duration    time.Duration    `json:"duration"`
	Category    StreamEpCategory `json:"category"`
}

type StreamEpVideoTag struct {
	Id    int    `json:"tag_id"`
	Title string `json:"title"`
}

type StreamEpMeta struct {
	Duration time.Duration `json:"duration"`
}

type StreamEpUrls struct {
	Playlist string `json:"playlist"`
}

type StreamEpisode struct {
	Id            string             `json:"id"`
	EpisodeNumber int                `json:"episode"`
	Title         string             `json:"title"`
	Views         int                `json:"views"`
	Meta          StreamEpMeta       `json:"meta"`
	Urls          StreamEpUrls       `json:"urls"`
	Chapters      []StreamEpChapter  `json:"chapters"`
	Tags          []StreamEpVideoTag `json:"tags"`
	//
	Formats       []VideoFormat      `json:"formats"`
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

func (ep *StreamEpisode) ChapterByNumber(number int) (*StreamEpChapter, error) {
	var chapter *StreamEpChapter
	idx := number-1
	if idx >= 0 && idx >= len(ep.Chapters) {
		return chapter, &ChapterNotFoundError{ChapterNum: number}
	}
	if len(ep.Chapters) > 0 && idx >= 0 {
		chapter = &ep.Chapters[idx]
	}
	return chapter, nil
}

func (ep *StreamEpisode) ProposeFilename(chapter *StreamEpChapter) string {
	if chapter != nil {
		return fmt.Sprintf("GTV%04d - %v. %s.ts", ep.EpisodeNumber, chapter.Index, sanitizeUnicodeFilename(ep.Chapters[chapter.Index].Category.Title))
	} else {
		return sanitizeUnicodeFilename(ep.Title) + ".ts"
	}
}

func StreamEpisodeFromUrl(url string) (StreamEpisode, error) {
	epNumber, err := ParseEpisodeNumberFromVideoUrl(url)
	if err != nil { return StreamEpisode{}, err }
	info_data, err := httpGet(
		fmt.Sprintf(ApiBaseurlStreamEpisodeInfo, epNumber),
		ApiHeadersMetaAdditional,
		time.Second*10,
	)
	if err != nil { return StreamEpisode{}, err }
	// Parse JSON Response
	epContainer := ResponseStreamEpisode{}
	json.Unmarshal(info_data, &epContainer)
	ep := epContainer.StreamEpisode
	// Title
	ep.Title = strings.ToValidUTF8(ep.Title, "")
	// Correct Duration
	ep.Meta.Duration *= time.Second
	// Sort Chapters, correct Offset and set index
	chaptersProcessed := []StreamEpChapter{}
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
		ApiHeadersMetaAdditional,
		time.Second*10,
	)
	formats := parseAvailFormatsFromM3u8(string(playlist_data))
	for _, f := range formats {
		if !strings.Contains(strings.ToLower(f.Name), "hevc") {
			ep.Formats = append(ep.Formats, f)
		}
	}
	return ep, err
}
