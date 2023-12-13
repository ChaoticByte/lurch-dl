#!/usr/bin/env python3

# Copyright (c) 2023 Julian Müller (ChaoticByte)

import base64
import json
import subprocess

URL = "https://gronkh.tv/streams/774"
START = "1h5m"
STOP = "1h10m"
OUT = "test.ts"

if __name__ == "__main__":
    proc = subprocess.Popen(
        ["./lurch-dl",
         "--url", URL,
         "--start", START,
         "--stop", STOP,
         "--json-data"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE)
    with open(OUT, "wb") as f:
        f.truncate(0)
        f.seek(0)
        while True:
            l = proc.stdout.readline()
            if not l:
                # stdout may be empty, but
                # what about stderr?
                l = proc.stderr.readline()
                if not l:
                    break
            # parse the json output
            msg = json.loads(l)
            if msg["type"] == "video_data":
                # decode & write the video data
                # to the output file
                data = base64.b64decode(msg["data"])
                f.write(data)
                f.flush()
            elif msg["type"] == "progress":
                # status info about download
                print(f"{(msg['progress']*100.0):06.2f}%\t{msg['rate']/1_000_000} MB/s")
            elif msg["type"] == "video_meta":
                # print out video title
                print("Downloading '" + msg["title"] + "' ...")
    print(proc.returncode)
