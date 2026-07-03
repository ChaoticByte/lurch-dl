package core

import "time"

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
