// Copyright (c) 2025, Julian Müller (ChaoticByte)

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ChaoticByte/lurch-dl/core"
)

// Global Variables
var CliXtermTitle bool

//

func XtermDetectFeatures() {
	for _, entry := range os.Environ() {
		kv := strings.Split(entry, "=")
		if len(kv) > 1 && kv[0] == "TERM" {
			if strings.Contains(kv[1], "xterm") ||
				strings.Contains(kv[1], "rxvt") ||
				strings.Contains(kv[1], "alacritty") {
				CliXtermTitle = true
				break
			}
		}
	}
}

func XtermSetTitle(title string) {
	fmt.Printf("\033]2;%s\007", title)
}

// Commandline

var Arguments struct {
	Url string `json:"url"`
	FormatName string `json:"format_name"`
	OutputFile string `json:"output_file"`
	TimestampStart string `json:"timestamp_start"`
	TimestampStop string `json:"timestamp_stop"`
	Overwrite bool `json:"overwrite"`
	ContinueDl bool `json:"continue"`
	//
	Help bool `json:"-"`
	VideoInfo bool `json:"-"`
	ListFormats bool `json:"-"`
	UnparsedChapterNum   int  `json:"chapter_num"`
	// Parsed
	Video core.GtvVideo `json:"-"`
	StartDuration time.Duration `json:"-"`
	StopDuration time.Duration `json:"-"`
	ChapterIdx int `json:"-"`
	Ratelimit float64 `json:"-"`
}

func CliShowHelp() {
	fmt.Println(`
lurch-dl --url string       The url to the video
         [-h --help]        Show this help and exit
         [--info]           Show video info (chapters, formats, length, ...)
         [--chapter int]    The chapter you want to download
                            The calculated start and stop timestamps can be
                            overwritten by --start and --stop
                            default: 0 (complete stream)
         [--format string]  The desired video format
                            default: auto
         [--output string]  The output file. Will be determined automatically
                            if omitted.
         [--start string]   Define a video timestamp to start at, e.g. 12m34s
         [--stop string]    Define a video timestamp to stop at, e.g. 1h23m45s
         [--continue]       Continue the download if possible
         [--overwrite]      Overwrite the output file if it already exists
         [--max-rate float] The maximum download rate in MB/s - don't set this
                            too high, you may run into a ratelimit and your
                            IP address might get banned from the servers.
                            default: 10.0

Version: ` + core.Version)
}

func CliParseArguments() error {
	var err error
	var ratelimitMbs float64
	flag.BoolVar(&Arguments.Help, "h", false, "")
	flag.BoolVar(&Arguments.Help, "help", false, "")
	flag.BoolVar(&Arguments.VideoInfo, "info", false, "")
	flag.StringVar(&Arguments.Url, "url", "", "")
	flag.IntVar(&Arguments.UnparsedChapterNum, "chapter", 0, "") // 0 -> chapter idx -1 -> complete stream
	flag.StringVar(&Arguments.FormatName, "format", "auto", "")
	flag.StringVar(&Arguments.OutputFile, "output", "", "")
	flag.StringVar(&Arguments.TimestampStart, "start", "", "")
	flag.StringVar(&Arguments.TimestampStop, "stop", "", "")
	flag.BoolVar(&Arguments.Overwrite, "overwrite", false, "")
	flag.BoolVar(&Arguments.ContinueDl, "continue", false, "")
	flag.Float64Var(&ratelimitMbs, "max-rate", 10.0, "")
	flag.Parse()
	Arguments.Video, err = core.ParseGtvVideoUrl(Arguments.Url)
	if err != nil {
		return err
	}
	if Arguments.Video.Category != "streams" {
		return errors.New("video category '" + Arguments.Video.Category + "' not supported")
	}
	if Arguments.TimestampStart == "" {
		Arguments.StartDuration = -1
	} else {
		Arguments.StartDuration, err = time.ParseDuration(Arguments.TimestampStart)
		if err != nil {
			return err
		}
	}
	if Arguments.TimestampStop == "" {
		Arguments.StopDuration = -1
	} else {
		Arguments.StopDuration, err = time.ParseDuration(Arguments.TimestampStop)
		if err != nil {
			return err
		}
	}
	Arguments.ChapterIdx = Arguments.UnparsedChapterNum - 1
	Arguments.Ratelimit = ratelimitMbs * 1_000_000.0 // MB/s -> B/s
	if Arguments.Ratelimit <= 0 {
		return errors.New("the value of --max-rate must be greater than 0")
	}
	return err
}

// Main

func CliRun() int {
	defer fmt.Print("\n")
	// cli arguments & help text
	flag.Usage = CliShowHelp
	err := CliParseArguments()
	if Arguments.Help {
		CliShowHelp()
		return 0
	} else if Arguments.Url == "" || err != nil {
		CliShowHelp()
		if err != nil {
			CliErrorMessage(err)
		}
		return 1
	}
	// detect terminal features
	XtermDetectFeatures()
	// Get video metadata
	if CliXtermTitle {
		XtermSetTitle("lurch-dl - Fetching video metadata ...")
	}
	streamEp, err := core.GetStreamEpisode(Arguments.Video.Id)
	if err != nil {
		CliErrorMessage(err)
		return 1
	}
	fmt.Print("\n")
	fmt.Printf("Title:     %s\n", streamEp.Title)
	// Check and list chapters/formats and exit
	if Arguments.ChapterIdx >= 0 {
		if Arguments.ChapterIdx >= len(streamEp.Chapters) {
			CliErrorMessage(&core.ChapterNotFoundError{ChapterNum: Arguments.UnparsedChapterNum})
			CliAvailableChapters(streamEp.Chapters)
			return 1
		}
	}
	if Arguments.VideoInfo {
		fmt.Printf("Episode:   %s\n", streamEp.Episode)
		fmt.Printf("Length:    %s\n", streamEp.Length)
		fmt.Printf("Views:     %d\n", streamEp.Views)
		fmt.Printf("Timestamp: %s\n", streamEp.Timestamp)
		if len(streamEp.Tags) > 0 {
			fmt.Print("Tags:      ")
			for i, t := range streamEp.Tags {
				if i == 0 {
					fmt.Print(t.Title)
				} else {
					fmt.Print(", ", t.Title)
				}
			}
			fmt.Print("\n")
		} else {
			fmt.Println("Tags:      -")
		}
		CliAvailableFormats(streamEp.Formats)
		CliAvailableChapters(streamEp.Chapters)
		return 0
	}
	format, err := streamEp.GetFormatByName(Arguments.FormatName)
	if err != nil {
		CliErrorMessage(err)
		CliAvailableFormats(streamEp.Formats)
		return 1
	}
	fmt.Printf("Format:    %v\n", format.Name)
	// chapter
	targetChapter := core.Chapter{Index: -1} // set Index to -1 for noop
	if len(streamEp.Chapters) > 0 && Arguments.ChapterIdx >= 0 {
		targetChapter = streamEp.Chapters[Arguments.ChapterIdx]
		fmt.Printf("Chapter:   %v. %v\n", Arguments.UnparsedChapterNum, targetChapter.Title)
	}
	// We already set the output file correctly so we can output it
	if Arguments.OutputFile == "" {
		Arguments.OutputFile = streamEp.GetProposedFilename(Arguments.ChapterIdx)
	}
	// Start Download
	fmt.Printf("Output:    %v\n", Arguments.OutputFile)
	fmt.Print("\n")
	successful := false
	aborted := false
	for p := range core.DownloadEpisode(
		streamEp,
		targetChapter,
		Arguments.FormatName,
		Arguments.OutputFile,
		Arguments.Overwrite,
		Arguments.ContinueDl,
		Arguments.StartDuration,
		Arguments.StopDuration,
		Arguments.Ratelimit,
		make(chan os.Signal, 1),
	) { // Iterate over download progress
		if p.Error != nil {
			CliErrorMessage(p.Error)
			return 1
		}
		if p.Success {
			successful = true
		} else if p.Aborted {
			aborted = true
		} else {
			CliDownloadProgress(p.Progress, p.Rate, p.Delaying, p.Waiting, p.Retries, p.Title)
		}
	}
	fmt.Print("\n")
	if aborted {
		fmt.Print("\nAborted.                                                ")
		return 130
	} else if !successful {
		CliErrorMessage(errors.New("download failed"))
		return 1
	} else { return 0 }
}

func CliAvailableChapters(chapters []core.Chapter) {
	fmt.Println("Chapters:")
	for _, f := range chapters {
		fmt.Printf("         %3d %10s\t%s\n", f.Index+1, f.Offset, f.Title)
	}
}

func CliAvailableFormats(formats []core.VideoFormat) {
	fmt.Print("Formats:   ")
	for i, f := range formats {
		if i == 0 {
			fmt.Print(f.Name)
		} else {
			fmt.Print(", ", f.Name)
		}
	}
	fmt.Print("\n")
}

func CliDownloadProgress(progress float32, rate float64, delaying bool, waiting bool, retries int, title string) {
	if retries > 0 {
		if retries == 1 {
			fmt.Print("\n")
		}
		fmt.Printf("Downloaded %.2f%% at %.2f MB/s (retry %v) ...      ", progress*100.0, rate/1000000.0, retries)
		fmt.Print("\n")
	} else if waiting {
		fmt.Printf("Downloaded %.2f%% at %.2f MB/s ...                 \r", progress*100.0, rate/1000000.0)
	} else if delaying {
		fmt.Printf("Downloaded %.2f%% at %.2f MB/s (delaying) ...      \r", progress*100.0, rate/1000000.0)
	} else {
		fmt.Printf("Downloaded %.2f%% at %.2f MB/s                     \r", progress*100.0, rate/1000000.0)
	}
	if CliXtermTitle {
		XtermSetTitle(fmt.Sprintf("lurch-dl - Downloaded %.2f%% at %.2f MB/s - %v", progress*100.0, rate/1000000.0, title))
	}
}

func CliErrorMessage(err error) {
	fmt.Print("\n")
	fmt.Println("An error occured:", err)
}
