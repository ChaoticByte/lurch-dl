package core

type DownloadProgress struct {
	Aborted bool
	Error error
	Success bool
	Delaying bool
	Progress float32
	Rate float64
	Retries int
	Title string
	Waiting bool
}
