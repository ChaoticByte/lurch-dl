// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"io"
	"net/http"
	"time"
)

var ApiHeadersBase = http.Header{
	"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.7727.56 Safari/537.36"},
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

func httpGet(url string, additionalHeaders http.Header, timeout time.Duration) ([]byte, error) {
	data := []byte{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data, err
	}
	for k, v := range ApiHeadersBase { req.Header.Set(k, v[0]) }
	for k, v := range additionalHeaders { req.Header.Set(k, v[0]) }
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return data, err
	}
	data, err = io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return data, &HttpStatusCodeError{Url: url, StatusCode: resp.StatusCode}
	}
	return data, err
}
