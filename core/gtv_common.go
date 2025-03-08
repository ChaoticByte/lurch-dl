// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"errors"
	"regexp"
	"time"
)

// The following two values are used to simulate buffering
const RatelimitDelay = 2.0      // in Seconds; How long to delay the next chunk download.
const RatelimitDelayAfter = 5.0 // in Seconds; Delay the next chunk download after this duration.

var videoUrlRegex = regexp.MustCompile(`gronkh\.tv\/([a-z]+)\/([0-9]+)`)

type GtvVideo struct {
	Class string `json:"class"`
	Id    string `json:"id"`
}

func ParseGtvVideoUrl(url string) (GtvVideo, error) {
	video := GtvVideo{}
	match := videoUrlRegex.FindStringSubmatch(url)
	if match == nil || len(match) < 2 {
		return video, errors.New("Could not parse URL " + url)
	}
	video.Class = match[1]
	video.Id = match[2]
	return video, nil
}

type VideoFormat struct {
	Name string `json:"format"`
	Url  string `json:"url"`
}

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
