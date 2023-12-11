// Copyright (c) 2023 Julian Müller (ChaoticByte)

package main

type FileExistsError struct {
	Filename string
}

func (err *FileExistsError) Error() string {
	return "File '" + err.Filename + "' already exists."
}

type FormatNotFoundError struct {
	FormatName string
}

func (err *FormatNotFoundError) Error() string {
	return "Format " + err.FormatName + " is not available."
}
