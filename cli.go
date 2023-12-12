// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

//

func XtermSetTitle(title string) {
	fmt.Printf("\033]2;%s\007", title)
}

func DrawLine() {
	terminalWidth, _, err := term.GetSize(0)
	if err != nil { return }
	r := ""
	for i:=0; i<terminalWidth-1; i++ {
		r += "─"
	}
	fmt.Println(r)
}

//

type Cli struct {
	jsonCli bool
	jsonData bool
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
	flag.BoolVar(&cli.jsonCli, "json", false, "")
	flag.BoolVar(&cli.jsonData, "json-data", false, "")
	flag.Usage = cli.Help
	flag.Parse()
	cli.jsonCli = cli.jsonCli || cli.jsonData // --json-data implies --json
	// detect terminal type and set variables accordingly
	if !cli.jsonCli {
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
	if video.Class != "streams" {
		if cli.jsonCli {
			PrintJson(JsonError{Message: "Video category '" + video.Class + "' not supported"})
		} else {
			fmt.Println("Video category '" + video.Class + "' not supported.")
		}
		os.Exit(1)
	}
	if !cli.jsonCli && cli.xtermTitle { XtermSetTitle("lurch-dl - Fetching video metadata ...") }
	streamEp, err := GetStreamEpisode(video.Id, chapterIdx)
	if err != nil {
		cli.ErrorMessage(fmt.Sprint(err), err)
		os.Exit(1)
	}
	if cli.jsonCli {
		PrintJson(JsonVideoMeta{ProposedFilename: streamEp.ProposedFilename, Title: streamEp.Title, VideoClass: video.Class})
	} else {
		DrawLine()
		fmt.Println(streamEp.Title)
	}
	if listChapters || listFormats {
		if listChapters {
			if !cli.jsonCli { DrawLine() }
			cli.AvailableChapters(streamEp.Chapters)
		}
		if listFormats {
			if !cli.jsonCli { DrawLine() }
			cli.AvailableFormats(streamEp.Formats)
		}
		if !cli.jsonCli { DrawLine() }
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
		if !cli.jsonCli {
			cli.AvailableFormats(streamEp.Formats)
		}
		os.Exit(1)
	}
	if !cli.jsonCli { DrawLine() }
	cli.Format(streamEp.Formats[formatIdx])
	if chapterIdx >= 0 {
		cli.InfoMessage(fmt.Sprintf("Chapter: %v. %v", chapterNum, streamEp.Chapters[chapterIdx].Title))
	}
	if !cli.jsonCli {
		DrawLine()
		defer fmt.Print("\n")
	}
	if err = streamEp.Download(formatIdx, chapterIdx, startDuration, stopDuration, outputFile, overwrite, continueDl, ratelimit, cli); err != nil {
		cli.ErrorMessage(fmt.Sprint(err), err)
		os.Exit(1)
	}
}

func (cli *Cli) AvailableChapters(chapters []Chapter) {
	if cli.jsonCli {
		PrintJson(JsonAvailableChapters{Chapters: chapters})
	} else {
		fmt.Println("Chapters:")
		for _, f := range chapters {
			fmt.Printf("%3d %10s\t%s\n", f.Index+1, f.Offset, f.Title)
		}
	}
}

func (cli *Cli) AvailableFormats(formats []VideoFormat) {
	if cli.jsonCli {
		PrintJson(JsonAvailableFormats{Formats: formats})
	} else {
		fmt.Println("Available formats:")
		for _, f := range formats {
			fmt.Println(" - " + f.Name)
		}
	}
}

func (cli *Cli) Format(format VideoFormat) {
	if cli.jsonCli {
		PrintJson(JsonFormat{Format: format.Name})
	} else {
		fmt.Printf("Format: %v\n", format.Name)
	}
}

func (cli *Cli) Progress(progress float32, rate float64, delaying bool, waiting bool, retries int, title string) {
	if cli.jsonCli {
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
		if cli.xtermTitle {
			XtermSetTitle(fmt.Sprintf("lurch-dl - Downloaded %.2f%% at %.2f MB/s - %v", progress * 100.0, rate / 1000000.0, title))
		}
	}
}

func (cli *Cli) InfoMessage(msg string) {
	if cli.jsonCli {
		PrintJson(JsonInfo{Message: msg})
	} else {
		fmt.Println(msg)
	}
}

func (cli *Cli) ErrorMessage(msg string, err error) {
	if cli.jsonCli {
		PrintJson(JsonError{Message: msg, Error: err})
	} else {
		if msg != "" {
			fmt.Println(msg)
		}
	}
}

func (cli *Cli) Aborted() {
	if cli.jsonCli {
		PrintJson(JsonError{Message: "aborted"})
	} else {
		fmt.Print("\nAborted.                                                ")
	}
}

func (cli *Cli) Help() {
	if cli.jsonCli {
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
         [--json]           Print all terminal output in json format
         [--json-data]      Print video data to stdout in json format
                            implies --json, supersedes --output
                            disarms --continue and --overwrite

Version: ` + Version)
	}
}
