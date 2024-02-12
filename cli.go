// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Global Variables
var CliJson bool
var CliJsonData bool
var CliXtermTitle bool

//

func XtermDetectFeatures() {
	for _, entry := range os.Environ() {
		kv := strings.Split(entry, "=")
		if len(kv) > 1 && kv[0] == "TERM" {
			if strings.Contains(kv[1], "xterm") ||
			   strings.Contains(kv[1], "rxvt")  ||
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

func SafeNewline() {
	// only for non-json output
	if !CliJson {
		fmt.Print("\n")
	}
}

// Commandline Interface Arguments

type CliArguments struct {
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

func CliShowHelp() {
	if CliJson {
		PrintJson(JsonError{Message: "Not printing help text in json output mode"})
	} else {
		fmt.Println(`
lurch-dl --url string       The url to the video
         [-h --help]        Show this help and exit
         [--list-chapters]  List chapters and exit
         [--list-formats]   List available formats and exit
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
         [--json]           Print all terminal output in json format
         [--json-data]      Print video data to stdout in json format
                            implies --json, supersedes --output
                            disarms --continue and --overwrite

Version: ` + Version)
	}
}

func CliParseArguments() (CliArguments, error) {
	var err error
	var ratelimitMbs float64
	a := CliArguments{}
	flag.BoolVar(&a.Help, "h", false, "")
	flag.BoolVar(&a.Help, "help", false, "")
	flag.BoolVar(&a.ListChapters, "list-chapters", false, "")
	flag.BoolVar(&a.ListFormats, "list-formats", false, "")
	flag.StringVar(&a.Url, "url", "", "")
	flag.IntVar(&a.ChapterNum, "chapter", 0, "") // 0 -> chapter idx -1 -> complete stream
	flag.StringVar(&a.FormatName, "format", "auto", "")
	flag.StringVar(&a.OutputFile, "output", "", "")
	flag.StringVar(&a.TimestampStart, "start", "", "")
	flag.StringVar(&a.TimestampStop, "stop", "", "")
	flag.BoolVar(&a.Overwrite, "overwrite", false, "")
	flag.BoolVar(&a.ContinueDl, "continue", false, "")
	flag.Float64Var(&ratelimitMbs, "max-rate", 10.0, "")
	flag.BoolVar(&CliJson, "json", false, "")
	flag.BoolVar(&CliJsonData, "json-data", false, "")
	flag.Parse()
	CliJson = CliJson || CliJsonData
	a.Video, err = ParseGtvVideoUrl(a.Url)
	if err != nil {
		return a, err
	}
	if a.Video.Class != "streams" {
		return a, errors.New("video category '" + a.Video.Class + "' not supported")
	}
	if a.TimestampStart == "" {
		a.StartDuration = -1
	} else {
		a.StartDuration, err = time.ParseDuration(a.TimestampStart)
		if err != nil {
			return a, err
		}
	}
	if a.TimestampStop == "" {
		a.StopDuration = -1
	} else {
		a.StopDuration, err = time.ParseDuration(a.TimestampStop)
		if err != nil {
			return a, err
		}
	}
	a.ChapterIdx = a.ChapterNum - 1
	a.Ratelimit = ratelimitMbs * 1_000_000.0 // MB/s -> B/s
	if a.Ratelimit <= 0 {
		return a, errors.New("the value of --max-rate must be greater than 0")
	}
	return a, err
}

// Main

func CliRun() int {
	defer SafeNewline()
	// cli arguments & help text
	flag.Usage = CliShowHelp
	args, err := CliParseArguments()
	if args.Help {
		CliShowHelp()
		return 0
	} else if args.Url == "" || err != nil  {
		CliShowHelp()
		if err != nil {
			CliErrorMessage(err)
		}
		return 1
	}
	// detect terminal features
	if !CliJson { XtermDetectFeatures() }
	// Get video metadata
	if CliXtermTitle { XtermSetTitle("lurch-dl - Fetching video metadata ...") }
	streamEp, err := GetStreamEpisode(args.Video.Id, args.ChapterIdx)
	if err != nil {
		CliErrorMessage(err)
		return 1
	}
	if CliJson {
		PrintJson(JsonVideoMeta{ProposedFilename: streamEp.ProposedFilename, Title: streamEp.Title, VideoClass: args.Video.Class})
	} else {
		SafeNewline()
		fmt.Println(streamEp.Title)
	}
	// Check and list chapters/formats and exit
	if args.ChapterIdx >= 0 {
		if args.ChapterIdx >= len(streamEp.Chapters) {
			CliErrorMessage(&ChapterNotFoundError{ChapterNum: args.ChapterNum})
			if !CliJson {
				CliAvailableChapters(streamEp.Chapters)
			}
			return 1
		}
	}
	if args.ListChapters || args.ListFormats {
		if args.ListChapters {
			SafeNewline()
			CliAvailableChapters(streamEp.Chapters)
		}
		if args.ListFormats {
			SafeNewline()
			CliAvailableFormats(streamEp.Formats)
		}
		return 0
	}
	format, err := streamEp.GetFormatByName(args.FormatName)
	if err != nil {
		CliErrorMessage(err)
		if !CliJson {
			CliAvailableFormats(streamEp.Formats)
		}
		return 1
	}
	CliShowFormat(format)
	if args.ChapterIdx >= 0 {
		CliInfoMessage(fmt.Sprintf("Chapter: %v. %v", args.ChapterNum, streamEp.Chapters[args.ChapterIdx].Title))
	}
	// Set auto values
	if args.OutputFile == "" {
		args.OutputFile = streamEp.ProposedFilename
	}
	if args.ChapterIdx >= 0 {
		if args.StartDuration < 0 {
			args.StartDuration = time.Duration(streamEp.Chapters[args.ChapterIdx].Offset)
		}
		if args.StopDuration < 0 && args.ChapterIdx + 1 < len(streamEp.Chapters) {
			// next chapter is stop
			args.StopDuration = time.Duration(streamEp.Chapters[args.ChapterIdx + 1].Offset)
		}
	}
	// Start Download
	SafeNewline()
	if err = streamEp.Download(args); err != nil {
		CliErrorMessage(err)
		return 1
	}
	SafeNewline()
	return 0
}

func CliAvailableChapters(chapters []Chapter) {
	if CliJson {
		PrintJson(JsonAvailableChapters{Chapters: chapters})
	} else {
		fmt.Println("Chapters:")
		for _, f := range chapters {
			fmt.Printf("%3d %10s\t%s\n", f.Index+1, f.Offset, f.Title)
		}
	}
}

func CliAvailableFormats(formats []VideoFormat) {
	if CliJson {
		PrintJson(JsonAvailableFormats{Formats: formats})
	} else {
		fmt.Println("Available formats:")
		for _, f := range formats {
			fmt.Println(" - " + f.Name)
		}
	}
}

func CliShowFormat(format VideoFormat) {
	if CliJson {
		PrintJson(JsonFormat{Format: format.Name})
	} else {
		fmt.Printf("Format: %v\n", format.Name)
	}
}

func CliDownloadProgress(progress float32, rate float64, delaying bool, waiting bool, retries int, title string) {
	if CliJson {
		PrintJson(
			JsonProgress{
				Progress: progress,
				Rate: rate,
				Delaying: delaying,
				Waiting: waiting,
				Retries: retries,
			})
	} else {
		if retries > 0 {
			if retries == 1 { fmt.Print("\n") }
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s (retry %v) ...      ", progress * 100.0, rate / 1000000.0, retries)
			fmt.Print("\n")
		} else if waiting {
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s ...                 \r", progress * 100.0, rate / 1000000.0)
		} else if delaying {
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s (delaying) ...      \r", progress * 100.0, rate / 1000000.0)
		} else {
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s                     \r", progress * 100.0, rate / 1000000.0)
		}
		if CliXtermTitle {
			XtermSetTitle(fmt.Sprintf("lurch-dl - Downloaded %.2f%% at %.2f MB/s - %v", progress * 100.0, rate / 1000000.0, title))
		}
	}
}

func CliInfoMessage(msg string) {
	if CliJson {
		PrintJson(JsonInfo{Message: msg})
	} else {
		fmt.Println(msg)
	}
}

func CliErrorMessage(err error) {
	if CliJson {
		PrintJson(JsonError{Message: err.Error(), Error: err})
	} else {
		SafeNewline()
		fmt.Println("An error occured:", err)
	}
}

func CliAborted() {
	if CliJson {
		PrintJson(JsonError{Message: "aborted"})
	} else {
		fmt.Print("\nAborted.                                                ")
	}
}
