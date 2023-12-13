// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

import (
	"encoding/json"
	"fmt"
	"os"
)


type JsonMessage interface {
	Marshal() ([]byte, error)
	OutputFile() *os.File
}


type JsonProgress struct {
	MsgType string `json:"type"`
	Progress float32 `json:"progress"`
	Rate float64 `json:"rate"`
	Delaying bool `json:"delaying"`
	Waiting bool `json:"waiting"`
	Retries int `json:"retries"`
}

func (m JsonProgress) Marshal() ([]byte, error) {
	m.MsgType = "progress"
	return json.Marshal(m)
}

func (m JsonProgress) OutputFile() *os.File { return os.Stdout }


type JsonVideoData struct {
	MsgType string `json:"type"`
	DataChunkIdx int `json:"idx"`
	// encoded to base64 per default, see
	// https://pkg.go.dev/encoding/json#Marshal
	Data []byte `json:"data"`
}

func (m JsonVideoData) Marshal() ([]byte, error) {
	m.MsgType = "video_data"
	return json.Marshal(m)
}

func (m JsonVideoData) OutputFile() *os.File { return os.Stdout }


type JsonVideoMeta struct {
	MsgType string `json:"type"`
	ProposedFilename string `json:"proposed_filename"`
	Title string `json:"title"`
	VideoClass string `json:"video_class"`
}

func (m JsonVideoMeta) Marshal() ([]byte, error) {
	m.MsgType = "video_meta"
	return json.Marshal(m)
}

func (m JsonVideoMeta) OutputFile() *os.File { return os.Stdout }


type JsonFormat struct {
	MsgType string `json:"type"`
	Format string `json:"format"`
}

func (m JsonFormat) Marshal() ([]byte, error) {
	m.MsgType = "format"
	return json.Marshal(m)
}

func (m JsonFormat) OutputFile() *os.File { return os.Stdout }


type JsonAvailableFormats struct {
	MsgType string `json:"type"`
	Formats []VideoFormat `json:"formats"`
}

func (m JsonAvailableFormats) Marshal() ([]byte, error) {
	m.MsgType = "available_formats"
	return json.Marshal(m)
}

func (m JsonAvailableFormats) OutputFile() *os.File { return os.Stdout }


type JsonAvailableChapters struct {
	MsgType string `json:"type"`
	Chapters []Chapter `json:"chapters"`
}

func (m JsonAvailableChapters) Marshal() ([]byte, error) {
	m.MsgType = "available_chapters"
	return json.Marshal(m)
}

func (m JsonAvailableChapters) OutputFile() *os.File { return os.Stdout }


type JsonInfo struct {
	MsgType string `json:"type"`
	Message string `json:"message"`
}

func (m JsonInfo) Marshal() ([]byte, error) {
	m.MsgType = "info"
	return json.Marshal(m)
}

func (m JsonInfo) OutputFile() *os.File { return os.Stdout }


type JsonError struct {
	MsgType string `json:"type"`
	Message string `json:"message"`
	Error error `json:"error"`
}

func (m JsonError) Marshal() ([]byte, error) {
	m.MsgType = "error"
	return json.Marshal(m)
}

func (m JsonError) OutputFile() *os.File { return os.Stderr }


func PrintJson(msg JsonMessage) {
	encoded, err := msg.Marshal()
	if err != nil {
		fmt.Fprintln(os.Stderr, "{\"type\":\"error\",\"message\":\"Couldn't convert output to json\",\"error\":{}}")
	} else {
		fmt.Fprintln(msg.OutputFile(), string(encoded))
	}
}
