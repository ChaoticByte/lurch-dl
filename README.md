
Definitely not an unofficial commandline downloader for https://gronkh.tv

## Compatibility

This tool is only compatible with recent Linux-based operating systems.  
To run it on Windows, make use of WSL.

## Features

- Download [Stream-Episodes](https://gronkh.tv/streams/)
- Specify a start- and stop-timestamp to download only a portion of the video
- Download a specific chapter
- Continuable Downloads
- Show infos about that Episode

## Known Issues / Limitations

- Downloads are **capped to 10 Mbyte/s by default** and buffering is simulated to pre-empt IP blocking due to API rate-limiting
- Because of the length of video chunks, **start- and stop-timestamps are inaccurate** (± 8 seconds)
- **Some videoplayers may have problems with the downloaded video file**. To fix this, you can use ffmpeg to rewrite the video into a MKV-File:  
  `ffmpeg -i video.ts -acodec copy -vcodec copy video.mkv`

## Download / Installation

New versions will appear under [Releases (remotebranch.eu)](https://remotebranch.eu/ChaoticByte/lurch-dl/releases).  
Just download the application and run it via your favourite terminal emulator.

> Note: **You may have to mark the file as executable before being able to run it.**

## Usage


Run `lurch-dl --help` to see available options.

> Note: This tool runs entirely on the command line.

### Examples

Download a video in its best available format:

```
./lurch-dl --url https://gronkh.tv/stream/777
```

Continue a download:

```
./lurch-dl --url https://gronkh.tv/stream/777 --continue
```

Download a specific chapter:

```
./lurch-dl --url https://gronkh.tv/stream/777 --chapter 2
```

Specify a start- and stop-timestamp:

```
./lurch-dl --url https://gronkh.tv/stream/777 --start 5h6m41s --stop 5h6m58s
```

List all available formats, chapters, and more info for a video:

```
./lurch-dl --url https://gronkh.tv/stream/777 --info
```

Download the video in a specific format:

```
./lurch-dl --url https://gronkh.tv/stream/777 --format 720p
```

Specify a filename:

```
./lurch-dl --url https://gronkh.tv/stream/777 --output Stream777.ts
```
