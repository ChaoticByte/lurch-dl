package core

import "time"

type Arguments struct {
	Url string `json:"url"`
	FormatName string `json:"format_name"`
	OutputFile string `json:"output_file"`
	TimestampStart string `json:"timestamp_start"`
	TimestampStop string `json:"timestamp_stop"`
	Overwrite bool `json:"overwrite"`
	ContinueDl bool `json:"continue"`
	// Parsed
	Video GtvVideo `json:"-"`
	StartDuration time.Duration `json:"-"`
	StopDuration time.Duration `json:"-"`
	ChapterIdx int `json:"-"`
	Ratelimit float64 `json:"-"`
}
