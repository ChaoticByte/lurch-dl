// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"net/http"
)

const ApiBaseurlStreamEpisodeInfo = "https://api.gronkh.tv/v1/video/info?episode=%s"
const ApiBaseurlStreamEpisodePlInfo = "https://api.gronkh.tv/v1/video/playlist?episode=%s"

var ApiHeadersBase = http.Header{
	"User-Agent":      {"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0"},
	"Accept-Language": {"de,en-US;q=0.7,en;q=0.3"},
	//"Accept-Encoding": {"gzip"},
	"Origin":         {"https://gronkh.tv"},
	"Referer":        {"https://gronkh.tv/"},
	"Connection":     {"keep-alive"},
	"Sec-Fetch-Dest": {"empty"},
	"Sec-Fetch-Mode": {"cors"},
	"Sec-Fetch-Site": {"same-site"},
	"Pragma":         {"no-cache"},
	"Cache-Control":  {"no-cache"},
	"TE":             {"trailers"},
}

var ApiHeadersMetaAdditional = http.Header{
	"Accept": {"application/json, text/plain, */*"},
}

var ApiHeadersVideoAdditional = http.Header{
	"Accept": {"*/*"},
}
