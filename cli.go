// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

//

func XtermSetTitle(title string) {
	fmt.Printf("\033]2;%s\007", title)
}

//

type Cli struct {
	jsonOutput bool
	xtermTitle bool
}

func (cli *Cli) Run() {
	// cli arguments
	var help bool
	var listChapters bool
	var listFormats bool
	var url string
	var chapterNum int
	var formatName string
	var outputFile string
	var timestampStart string
	var timestampStop string
	var overwrite bool
	var continueDl bool
	var ratelimit float64
	// var outputFile string
	flag.BoolVar(&help, "h", false, "")
	flag.BoolVar(&help, "help", false, "")
	flag.BoolVar(&listChapters, "list-chapters", false, "")
	flag.BoolVar(&listFormats, "list-formats", false, "")
	flag.StringVar(&url, "url", "", "")
	flag.IntVar(&chapterNum, "chapter", 0, "") // 0 is out of range bc. chapters start at 1 -> 0 means not defined
	flag.StringVar(&formatName, "format", "auto", "")
	flag.StringVar(&outputFile, "output", "", "")
	flag.StringVar(&timestampStart, "start", "", "")
	flag.StringVar(&timestampStop, "stop", "", "")
	flag.BoolVar(&overwrite, "overwrite", false, "")
	flag.BoolVar(&continueDl, "continue", false, "")
	flag.Float64Var(&ratelimit, "max-rate", 10.0, "")
	flag.BoolVar(&cli.jsonOutput, "json", false, "")
	flag.Usage = cli.Help
	flag.Parse()
	// detect terminal type and set variables accordingly
	if !cli.jsonOutput {
		for _, entry := range os.Environ() {
			kv := strings.Split(entry, "=")
			if len(kv) > 1 && kv[0] == "TERM" {
				if strings.Contains(kv[1], "xterm") ||
				   strings.Contains(kv[1], "rxvt")  ||
				   strings.Contains(kv[1], "alacritty") {
					cli.xtermTitle = true
					break
				}
			}
		}
	}
	//
	var startDuration time.Duration
	var stopDuration time.Duration
	var err error
	if ratelimit <= 0 {
		cli.ErrorMessage("The value of --max-rate must be greater than 0", nil)
		os.Exit(1)
	}
	ratelimit *= 1_000_000.0 // MB
	if timestampStart == "" {
		startDuration = -1
	} else {
		startDuration, err = time.ParseDuration(timestampStart)
		if err != nil {
			cli.ErrorMessage(fmt.Sprintf("Couldn't parse start timestamp '%v'", timestampStart), err)
			os.Exit(1)
		}
	}
	if timestampStop == "" {
		stopDuration = -1
	} else {
		stopDuration, err = time.ParseDuration(timestampStop)
		if err != nil {
			cli.ErrorMessage(fmt.Sprintf("Couldn't parse stop timestamp '%v'", timestampStop), err)
			os.Exit(1)
		}
	}
	chapterIdx := chapterNum-1
	// run actions
	if help {
		cli.Help()
		os.Exit(0)
	} else if url == "" {
		cli.Help()
		os.Exit(1)
	}
	video, err := ParseGtvVideoUrl(url)
	if err != nil {
		cli.ErrorMessage(fmt.Sprint(err), err)
		os.Exit(1)
	}
	if video.Category != "streams" {
		if cli.jsonOutput {
			PrintJson(JsonError{Message: "Video category '" + video.Category + "' not supported"})
		} else {
			fmt.Println("Video category '" + video.Category + "' not supported.")
		}
		os.Exit(1)
	}
	if !cli.jsonOutput && cli.xtermTitle { XtermSetTitle("lurch-dl - Fetching video metadata ...") }
	streamEp, err := GetStreamEpisode(video.Id, chapterIdx)
	if err != nil {
		cli.ErrorMessage(fmt.Sprint(err), err)
		os.Exit(1)
	}
	if cli.jsonOutput {
		PrintJson(JsonTitle{Title: streamEp.Title})
	} else {
		fmt.Println(streamEp.Title)
	}
	if listChapters || listFormats {
		if listChapters {
			if !cli.jsonOutput { fmt.Print("\n") }
			cli.AvailableChapters(streamEp.Chapters)
		}
		if listFormats {
			if !cli.jsonOutput { fmt.Print("\n") }
			cli.AvailableFormats(streamEp.Formats)
		}
		os.Exit(0)
	}
	if chapterIdx >= 0 {
		if chapterIdx >= len(streamEp.Chapters) {
			cli.ErrorMessage(fmt.Sprintf("Chapter %v not found", chapterNum), nil)
			os.Exit(1)
		}
	}
	formatIdx, err := streamEp.GetFormatIdx(formatName)
	if err != nil {
		cli.ErrorMessage(fmt.Sprint(err), err)
		if !cli.jsonOutput {
			cli.AvailableFormats(streamEp.Formats)
		}
		os.Exit(1)
	}
	if !cli.jsonOutput { fmt.Print("\n") }
	cli.Format(streamEp.Formats[formatIdx])
	if chapterIdx >= 0 {
		cli.InfoMessage(fmt.Sprintf("Chapter: %v. %v", chapterNum, streamEp.Chapters[chapterIdx].Title))
	}
	if !cli.jsonOutput {
		fmt.Print("\n")
		defer fmt.Print("\n")
	}
	if err = streamEp.Download(formatIdx, chapterIdx, startDuration, stopDuration, outputFile, overwrite, continueDl, ratelimit, cli); err != nil {
		if !cli.jsonOutput { fmt.Print("\n") }
		cli.ErrorMessage(fmt.Sprint(err), err)
		os.Exit(1)
	}
}

func (cli *Cli) AvailableChapters(chapters []Chapter) {
	if cli.jsonOutput {
		PrintJson(JsonAvailableChapters{Chapters: chapters})
	} else {
		fmt.Println("Chapters:")
		for _, f := range chapters {
			fmt.Printf("%3d %10s\t%s\n", f.Index+1, f.Offset, f.Title)
		}
	}
}

func (cli *Cli) AvailableFormats(formats []VideoFormat) {
	if cli.jsonOutput {
		PrintJson(JsonAvailableFormats{Formats: formats})
	} else {
		fmt.Println("Available formats:")
		for _, f := range formats {
			fmt.Println(" - " + f.Name)
		}
	}
}

func (cli *Cli) Format(format VideoFormat) {
	if cli.jsonOutput {
		PrintJson(JsonFormat{Format: format.Name})
	} else {
		fmt.Printf("Format: %v\n", format.Name)
	}
}

func (cli *Cli) Progress(progress float32, rate float64, delaying bool, waiting bool, retries int, title string) {
	if cli.jsonOutput {
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
			fmt.Printf("\nDownloaded %.2f%% at %.2f MB/s (retry %v) ...      \r", progress * 100.0, rate / 1000000.0, retries)
		} else if waiting {
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s ...                 \r", progress * 100.0, rate / 1000000.0)
		} else if delaying {
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s (delaying) ...      \r", progress * 100.0, rate / 1000000.0)
		} else {
			fmt.Printf("Downloaded %.2f%% at %.2f MB/s                     \r", progress * 100.0, rate / 1000000.0)
		}
		if cli.xtermTitle {
			XtermSetTitle(fmt.Sprintf("lurch-dl - Downloaded %.2f%% at %.2f MB/s - %v", progress * 100.0, rate / 1000000.0, title))
		}
	}
}

func (cli *Cli) InfoMessage(msg string) {
	if cli.jsonOutput {
		PrintJson(JsonInfo{Message: msg})
	} else {
		fmt.Println(msg)
	}
}

func (cli *Cli) ErrorMessage(msg string, err error) {
	if cli.jsonOutput {
		PrintJson(JsonError{Message: msg, Error: err})
	} else {
		if msg != "" {
			fmt.Println(msg)
		}
	}
}

func (cli *Cli) Aborted() {
	if cli.jsonOutput {
		PrintJson(JsonError{Message: "aborted"})
	} else {
		fmt.Print("\nAborted.                                                ")
	}
}

func (cli *Cli) Help() {
	if cli.jsonOutput {
		PrintJson(JsonError{Message: "Not printing help text in json output mode"})
	} else {
		fmt.Println(`lurch-dl --url string       The url to the video
         [-h --help]        Show this help and exit
         [--list-chapters]  List chapters and exit
         [--list-formats]   List available formats and exit
         [--chapter int]    The chapter you want to download
                            The calculated start and stop timestamps can be
                            overwritten by --start and --stop
                            default: -1 (disabled)
         [--format string]  The desired video format
                            default: auto
         [--output string]  The output file. Will be determined automatically
                            if omitted.
         [--start string]   Define a video timestamp to start at, e.g. 12m34s
         [--stop string]    Define a video timestamp to stop at, e.g. 1h23m45s
         [--continue]       Continue the download if possible
         [--overwrite]      Overwrite the output file if it already exists
         [--max-rate]       The maximum download rate in MB/s - don't set this
                            too high, you may run into a ratelimit and your
                            IP address might get banned from the servers.
                            default: 10.0
         [--json]           Provide all terminal output in json format

Version: ` + Version)
	}
}

type JsonProgress struct {
	MsgType string `json:"type"`
	Progress float32 `json:"progress"`
	Rate float64 `json:"rate"`
	Delaying bool `json:"delaying"`
	Waiting bool `json:"waiting"`
	Retries int `json:"retries"`
}

type JsonTitle struct {
	MsgType string `json:"type"`
	Title string `json:"title"`
}

type JsonFormat struct {
	MsgType string `json:"type"`
	Format string `json:"format"`
}

type JsonAvailableFormats struct {
	MsgType string `json:"type"`
	Formats []VideoFormat `json:"formats"`
}

type JsonAvailableChapters struct {
	MsgType string `json:"type"`
	Chapters []Chapter `json:"chapters"`
}

type JsonInfo struct {
	MsgType string `json:"type"`
	Message string `json:"message"`
}

type JsonError struct {
	MsgType string `json:"type"`
	Message string `json:"message"`
	Error error `json:"error"`
}

type JsonUnknown struct {
	MsgType string `json:"type"`
	Message any `json:"message"`
}

func PrintJson(msg any) {
	outputFile := os.Stdout
	var m any = JsonUnknown{MsgType: "unknown", Message: msg} // default
	switch v := msg.(type) {
	case JsonProgress:
		v.MsgType = "progress"
		m = v
	case JsonTitle:
		v.MsgType = "title"
		m = v
	case JsonFormat:
		v.MsgType = "format"
		m = v
	case JsonAvailableFormats:
		v.MsgType = "available_formats"
		m = v
	case JsonAvailableChapters:
		v.MsgType = "available_chapters"
		m = v
	case JsonInfo:
		v.MsgType = "info"
		m = v
	case JsonError:
		v.MsgType = "error"
		m = v
		outputFile = os.Stderr
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "{\"type\":\"error\",\"message\":\"Couldn't convert output to json\",\"error\":{}}")
	} else {
		fmt.Fprintln(outputFile, string(encoded))
	}
}
