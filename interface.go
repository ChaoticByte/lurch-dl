// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

type UserInterface interface {
	Run()
	AvailableFormats(formats []VideoFormat)
	Chapters(chapters []Chapter)
	Progress(percentage float32, rate float64, delaying bool, waiting bool, retries int)
	InfoMessage(msg string)
	Aborted()
	Help()
}
