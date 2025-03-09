
<details><summary>what could it be?</summary>

<br>

Definetly not a commandline downloader for https://gronkh.tv risen from the dead.

## Features

- Download [Stream-Episodes](https://gronkh.tv/streams/)
- Specify a start- and stop-timestamp to download only a portion of the video
- Download a specific chapter
- Continuable Downloads

## Known Issues

- You may get a "Windows Defender SmartScreen prevented an unrecognized app from starting" warning when running a new version for the first time
- Downloads are capped to 10 Mbyte/s and buffering is simulated to pre-empt IP blocking due to API ratelimiting
- Start- and stop-timestamps are not very accurate (± 8 seconds)
- Some videoplayers may have problems with the resulting file. To fix this, you can use ffmpeg to rewrite the video into a MKV-File: `ffmpeg -i video.ts -acodec copy -vcodec copy video.mkv`
- Emojis and other Unicode characters don't get displayed properly in a Powershell Console

## Supported Platforms

Tested on Linux and Windows (64bit).

## Download / Installation

New versions will appear under [Releases](https://github.com/ChaoticByte/lurch-dl/releases). Just download the application and run it via the terminal/cmd/powershell/...

On Linux, you may have to mark the file as executable before being able to run it.

## Cli Usage

Run `lurch-dl --help` to see available options.

### Examples

Download a video in its best available format (Windows):

```
.\lurch-dl.exe --url https://gronkh.tv/streams/777

Title: GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND SOLLTE VERBOTEN WERDEN!! ⭐ ️ 247 auf @GronkhTV ⭐ ️ !comic !archiv !a
Format: 1080p60
Downloaded 0.43% at 10.00 MB/s
...
```

Continue a download (Windows):

```
.\lurch-dl.exe --url https://gronkh.tv/streams/777 --continue

Title: GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND SOLLTE VERBOTEN WERDEN!! ⭐ ️ 247 auf @GronkhTV ⭐ ️ !comic !archiv !a
Format: 1080p60
Downloaded 0.68% at 10.00 MB/s
...
```

List all chapters (Windows):

```
.\lurch-dl.exe --url https://gronkh.tv/streams/777 --list-chapters

GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND SOLLTE VERBOTEN WERDEN!! ⭐ ️ 247 auf @GronkhTV ⭐ ️ !comic !archiv !a

Chapters:
  1         0s	Just Chatting
  2    2h53m7s	Alan Wake II
  3    9h35m0s	Just Chatting
```

Download a specific chapter (Windows):

```
.\lurch-dl.exe --url https://gronkh.tv/streams/777 --chapter 2

GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND SOLLTE VERBOTEN WERDEN!! ⭐ ️ 247 auf @GronkhTV ⭐ ️ !comic !archiv !a
Format: 1080p60
Chapter: 2. Alan Wake II

Downloaded 3.22% at 10.00 MB/s
...
```

Specify a start- and stop-timestamp (Linux):

```
./lurch-dl --url https://gronkh.tv/streams/777 --start 5h6m41s --stop 5h6m58s
...
```

List all available formats for a video (Linux):

```
./lurch-dl --url https://gronkh.tv/streams/777 --list-formats

Available formats:
 - 1080p60
 - 720p
 - 360p
```

Download the video in a specific format (Linux):

```
./lurch-dl --url https://gronkh.tv/streams/777 --format 720p

Title: GTV0777, 2023-11-09 - DIESER STREAM IST ILLEGAL UND SOLLTE VERBOTEN WERDEN!! ⭐ ️ 247 auf @GronkhTV ⭐ ️ !comic !archiv !a
Format: 720p
Downloaded 0.32% at 10.00 MB/s
...
```

Specify a filename (Windows):

```
.\lurch-dl.exe --url https://gronkh.tv/streams/777 --output Stream777.ts
...
```

</details>
