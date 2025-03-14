package main

type GenericCliAgumentError struct {
	Msg string
}

func (err *GenericCliAgumentError) Error() string {
	return err.Msg
}

type GenericDownloadError struct {}

func (err *GenericDownloadError) Error() string {
	return "download failed"
}
