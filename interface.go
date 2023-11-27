// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

type UserInterface interface {
	Run()
	AvailableFormats(formats []VideoFormat)
	AvailableChapters(chapters []Chapter)
	Format(format VideoFormat)
	Progress(percentage float32, rate float64, delaying bool, waiting bool, retries int)
	InfoMessage(msg string)
	ErrorMessage(msg string, err error)
	Aborted()
	Help()
}
