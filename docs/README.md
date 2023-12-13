# Documentation

## JSON Interface

When passing the commandline flag `--json`, all terminal output will be printed out in json format. When using `--json-data`, also the video data will be output in json to the terminal, instead of writing it to a file.
This can be used by developers as an API to this tool.

For example usage, have a look at [examples/download.py](./examples/download.py)

### Available Formats

```json
{
    "type": "available_formats",
    "formats": [
        {
            "format": "1080p60",
            "url": "https://01.cdn.vod.farm/transcode/..."
        },
        {
            "format": "720p",
            "url": "https://01.cdn.vod.farm/transcode/..."
        },
        {
            "format": "360p",
            "url": "https://01.cdn.vod.farm/transcode/..."
        }
    ]
}
```
This is only output if `--list-formats` is passed.

### Available Chapters

```json
{
    "type": "available_chapters",
    "chapters": [
        {
            // The index begins at 0, but when using
            // the cli parameter --chapter, you have
            // to add 1
            "index": 0,
            "title": "Just Chatting",
            "offset": 0
        },
        {
            "index": 1,
            "title": "DON'T SCREAM",
            // The offset is in μs (microseconds),
            // this is due to the implementation of
            // time.Duration in Go
            "offset": 1767000000000
        },
        {
            "index": 2,
            "title": "Just Chatting",
            "offset": 3501000000000
        },
        // ...
    ]
}
```
This is only output if `--list-chapters` is passed.

### Video Metadata

```json
{
    "type": "video_meta",
    "proposed_filename": "GTV0774, 2023-10-31 - (...).ts",
    "title": "GTV0774, 2023-10-31 - 🎃 HALLOWEEN HORROR ...",
    "video_class": "streams" // may be relevant in the future
}
```

### Chosen Video Format

```json
{
    "type": "format",
    "format": "1080p60"
}
```

### Progress

```json
{
    "type": "progress",
    "progress": 0.00017143837,
    "rate": 10000000,
    // Indicates that buffering is currently being simulated
    "delaying": false,
    // Set to true before a chunk is downloaded and to false
    // after a chunk is downloaded
    "waiting": false,
    // Indicates connection issues
    "retries": 0
}
```

### Video Data

```json
{
    "type": "video_data",
    "idx": 0,
    // The data is base64 encoded
    "data": "R0AREABC8CUAAcEAAP8B/wA..."
}
```
This is only output if `--json-data` is passed.

### Info Message

```json
{
    "type": "info",
    "message": "This is an info message"
}
```

### Error Message

```json
{
    "type": "error",
    "message": "This is an error message",
    // Additional error information, may be null
    "error": {"Op": "open", "Path": "example", "Err": 2}
}
```
