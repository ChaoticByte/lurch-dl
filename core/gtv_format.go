package core

import (
	"strings"
	"time"
)

type VideoFormat struct {
	Name string `json:"format"`
	Url  string `json:"url"`
}

func (vf *VideoFormat) StreamChunkList() (ChunkList, error) {
	baseUrl := vf.Url[:strings.LastIndex(vf.Url, "/")]
	data, err := httpGet(vf.Url, ApiHeadersMetaAdditional, time.Second*10)
	if err != nil {
		return ChunkList{}, err
	}
	chunklist, err := parseChunkListFromM3u8(string(data), baseUrl)
	return chunklist, err
}
