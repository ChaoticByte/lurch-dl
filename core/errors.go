// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import "fmt"

type HttpStatusCodeError struct {
	Url        string
	StatusCode int
}

func (err *HttpStatusCodeError) Error() string {
	var e string
	switch err.StatusCode {
	case 400:
		e = "Bad Request"
	case 401:
		e = "Unauthorized"
	case 403:
		e = "Forbidden"
	case 404:
		e = "Not Found"
	case 500, 502, 504:
		e = "Server Error"
	case 503:
		e = "Service Unavailable"
	default:
		e = "Request failed"
	}
	return fmt.Sprintf("%v - got status code %v while fetching %v", e, err.StatusCode, err.Url)
}

type FileExistsError struct {
	Filename string
}

func (err *FileExistsError) Error() string {
	return "file '" + err.Filename + "' already exists - see the available options on how to proceed"
}

type FormatNotFoundError struct {
	FormatName string
}

func (err *FormatNotFoundError) Error() string {
	return "format " + err.FormatName + " is not available"
}

type ChapterNotFoundError struct {
	ChapterNum int
}

func (err *ChapterNotFoundError) Error() string {
	return fmt.Sprintf("chapter %v not found", err.ChapterNum)
}

type VideoCategoryUnsupportedError struct {
	Category string
}

func (err *VideoCategoryUnsupportedError) Error() string {
	return fmt.Sprintf("video category '%v' not supported", err.Category)
}

type GtvVideoUrlParseError struct {
	Url string
}

func (err *GtvVideoUrlParseError) Error() string {
	return fmt.Sprintf("Could not parse URL %v", err.Url)
}

type DownloadInfoFileReadError struct {}

func (err *DownloadInfoFileReadError) Error() string {
	return "could not read download info file, can't continue download"
}
