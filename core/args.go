package core

import "time"

type Arguments struct {
	// "Raw"
	Help bool
	ListChapters bool
	ListFormats bool
	Url string
	ChapterNum int
	FormatName string
	OutputFile string
	TimestampStart string
	TimestampStop string
	Overwrite bool
	ContinueDl bool
	// Parsed
	Video GtvVideo
	StartDuration time.Duration
	StopDuration time.Duration
	ChapterIdx int
	Ratelimit float64
}
