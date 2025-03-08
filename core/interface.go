package core

type UserInterface interface {
	DownloadProgress(progress float32, rate float64, delaying bool, waiting bool, retries int, title string)
	InfoMessage(msg string)
	ErrorMessage(err error)
	Aborted()
}
