// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var videoUrlRegex = regexp.MustCompile(`gronkh\.tv\/([a-z]+)\/([0-9]+)`)

//

type VideoFormat struct {
	Name string `json:"format"`
	Url  string `json:"url"`
}

type Chapter struct {
	Index  int           `json:"index"`
	Title  string        `json:"title"`
	Offset time.Duration `json:"offset"`
}

type VideoTag struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

//

type GtvVideo struct {
	Category string `json:"category"`
	Id    string `json:"id"`
}

func ParseGtvVideoUrl(url string) (GtvVideo, error) {
	video := GtvVideo{}
	match := videoUrlRegex.FindStringSubmatch(url)
	if len(match) < 2 {
		return video, errors.New("Could not parse URL " + url)
	}
	video.Category = match[1]
	video.Id = match[2]
	return video, nil
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

type StreamEpisode struct {
	Episode     string        `json:"episode"`
	Title       string        `json:"title"`
	Formats     []VideoFormat `json:"formats"`
	Chapters    []Chapter     `json:"chapters"`
	PlaylistUrl string        `json:"playlist_url"`
	Length      time.Duration `json:"source_length"`
	Views       int           `json:"views"`
	Timestamp   string        `json:"created_at"`
	Tags        []VideoTag    `json:"tags"`
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
