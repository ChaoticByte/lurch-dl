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

type CliOnlyArguments struct {
	Help         bool `json:"-"`
	ListChapters bool `json:"-"`
	ListFormats  bool `json:"-"`
	ChapterNum   int  `json:"chapter_num"`
}

func CliShowHelp() {
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

Version: ` + core.Version)
}

func CliParseArguments() (core.Arguments, CliOnlyArguments, error) {
	var err error
	var ratelimitMbs float64
	a := core.Arguments{}
	c := CliOnlyArguments{}
	flag.BoolVar(&c.Help, "h", false, "")
	flag.BoolVar(&c.Help, "help", false, "")
	flag.BoolVar(&c.ListChapters, "list-chapters", false, "")
	flag.BoolVar(&c.ListFormats, "list-formats", false, "")
	flag.StringVar(&a.Url, "url", "", "")
	flag.IntVar(&c.ChapterNum, "chapter", 0, "") // 0 -> chapter idx -1 -> complete stream
	flag.StringVar(&a.FormatName, "format", "auto", "")
	flag.StringVar(&a.OutputFile, "output", "", "")
	flag.StringVar(&a.TimestampStart, "start", "", "")
	flag.StringVar(&a.TimestampStop, "stop", "", "")
	flag.BoolVar(&a.Overwrite, "overwrite", false, "")
	flag.BoolVar(&a.ContinueDl, "continue", false, "")
	flag.Float64Var(&ratelimitMbs, "max-rate", 10.0, "")
	flag.Parse()
	a.Video, err = core.ParseGtvVideoUrl(a.Url)
	if err != nil {
		return a, c, err
	}
	if a.Video.Class != "streams" {
		return a, c, errors.New("video category '" + a.Video.Class + "' not supported")
	}
	if a.TimestampStart == "" {
		a.StartDuration = -1
	} else {
		a.StartDuration, err = time.ParseDuration(a.TimestampStart)
		if err != nil {
			return a, c, err
		}
	}
	if a.TimestampStop == "" {
		a.StopDuration = -1
	} else {
		a.StopDuration, err = time.ParseDuration(a.TimestampStop)
		if err != nil {
			return a, c, err
		}
	}
	a.ChapterIdx = c.ChapterNum - 1
	a.Ratelimit = ratelimitMbs * 1_000_000.0 // MB/s -> B/s
	if a.Ratelimit <= 0 {
		return a, c, errors.New("the value of --max-rate must be greater than 0")
	}
	return a, c, err
}

// Main

func CliRun() int {
	cli := Cli{}
	defer fmt.Print("\n")
	// cli arguments & help text
	flag.Usage = CliShowHelp
	args, cliArgs, err := CliParseArguments()
	if cliArgs.Help {
		CliShowHelp()
		return 0
	} else if args.Url == "" || err != nil {
		CliShowHelp()
		if err != nil {
			cli.ErrorMessage(err)
		}
		return 1
	}
	// detect terminal features
	XtermDetectFeatures()
	// Get video metadata
	if CliXtermTitle {
		XtermSetTitle("lurch-dl - Fetching video metadata ...")
	}
	streamEp, err := core.GetStreamEpisode(args.Video.Id)
	if err != nil {
		cli.ErrorMessage(err)
		return 1
	}
	fmt.Print("\n")
	fmt.Println(streamEp.Title)
	// Check and list chapters/formats and exit
	if args.ChapterIdx >= 0 {
		if args.ChapterIdx >= len(streamEp.Chapters) {
			cli.ErrorMessage(&core.ChapterNotFoundError{ChapterNum: cliArgs.ChapterNum})
			CliAvailableChapters(streamEp.Chapters)
			return 1
		}
	}
	if cliArgs.ListChapters || cliArgs.ListFormats {
		if cliArgs.ListChapters {
			fmt.Print("\n")
			CliAvailableChapters(streamEp.Chapters)
		}
		if cliArgs.ListFormats {
			fmt.Print("\n")
			CliAvailableFormats(streamEp.Formats)
		}
		return 0
	}
	format, err := streamEp.GetFormatByName(args.FormatName)
	if err != nil {
		cli.ErrorMessage(err)
		CliAvailableFormats(streamEp.Formats)
		return 1
	}
	cli.InfoMessage(fmt.Sprintf("Format:  %v", format.Name))
	if args.ChapterIdx >= 0 {
		cli.InfoMessage(fmt.Sprintf("Chapter: %v. %v", cliArgs.ChapterNum, streamEp.Chapters[args.ChapterIdx].Title))
	}
	// We already set the output file correctly so we can output it
	if args.OutputFile == "" {
		args.OutputFile = streamEp.GetProposedFilename(args.ChapterIdx)
	}
	// Start Download
	cli.InfoMessage(fmt.Sprintf("Output:  %v", args.OutputFile))
	fmt.Print("\n")
	if err = streamEp.Download(args, &cli, make(chan os.Signal, 1)); err != nil {
		cli.ErrorMessage(err)
		return 1
	}
	fmt.Print("\n")
	return 0
}

func CliAvailableChapters(chapters []core.Chapter) {
	fmt.Println("Chapters:")
	for _, f := range chapters {
		fmt.Printf("%3d %10s\t%s\n", f.Index+1, f.Offset, f.Title)
	}
}

func CliAvailableFormats(formats []core.VideoFormat) {
	fmt.Println("Available formats:")
	for _, f := range formats {
		fmt.Println(" - " + f.Name)
	}
}

type Cli struct{}

func (cli *Cli) DownloadProgress(progress float32, rate float64, delaying bool, waiting bool, retries int, title string) {
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

func (cli *Cli) Aborted() {
	fmt.Print("\nAborted.                                                ")
}

func (cli *Cli) InfoMessage(msg string) {
	fmt.Println(msg)
}

func (cli *Cli) ErrorMessage(err error) {
	fmt.Print("\n")
	fmt.Println("An error occured:", err)
}
