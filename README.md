
<details><summary>what could it be?</summary>

<br>

Definetly not a commandline downloader for https://gronkh.tv risen from the dead.

## Features

- Download [Stream-Episodes](https://gronkh.tv/streams/)
- Specify a start- and stop-timestamp to download only a portion of the video
- Download a specific chapter
- Continuable Downloads
- Show infos about that Episode

## Known Issues

- Downloads are capped to 10 Mbyte/s and buffering is simulated to pre-empt IP blocking due to API ratelimiting
- Start- and stop-timestamps are not very accurate (± 8 seconds)
- Some videoplayers may have problems with the resulting file. To fix this, you can use ffmpeg to rewrite the video into a MKV-File: `ffmpeg -i video.ts -acodec copy -vcodec copy video.mkv`

## Supported Platforms

Only compatible with Linux.

## Download / Installation

New versions will appear under [Releases](https://github.com/ChaoticByte/lurch-dl/releases). Just download the application and run it via the terminal/cmd/powershell/...

On Linux, you may have to mark the file as executable before being able to run it.

## Cli Usage

Run `lurch-dl --help` to see available options.

### Examples

Download a video in its best available format:

```
./lurch-dl --url https://gronkh.tv/streams/777

Title:     GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND ...
Format:    1080p60
Output:    GTV0777, 2023-11-09 - DIESER STREAM IST [...].ts

Downloaded 0.32% at 10.00 MB/s ...
```

Continue a download:

```
./lurch-dl --url https://gronkh.tv/streams/777 --continue
```

Download a specific chapter:

```
./lurch-dl --url https://gronkh.tv/streams/777 --chapter 2

Title:     GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND ...
Format:    1080p60
Chapter:   2. Alan Wake II
Output:    GTV0777 - 2. Alan Wake II.ts

Downloaded 0.33% at 4.28 MB/s ...
```

Specify a start- and stop-timestamp:

```
./lurch-dl --url https://gronkh.tv/streams/777 --start 5h6m41s --stop 5h6m58s
```

List all available formats for a video:

```
./lurch-dl --url https://gronkh.tv/streams/777 --info

Title:     GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND ...
Episode:   777
Length:    9h48m55s
Views:     45424
Timestamp: 2023-11-09T18:23:01Z
Tags:      -
Formats:   1080p60, 720p, 360p
Chapters:
           1         0s Just Chatting
           2    2h53m7s Alan Wake II
           3    9h35m0s Just Chatting
```

Download the video in a specific format:

```
./lurch-dl --url https://gronkh.tv/streams/777 --format 720p

[...]
Format:    720p
[...]
```

Specify a filename:

```
./lurch-dl --url https://gronkh.tv/streams/777 --output Stream777.ts
```

</details>
