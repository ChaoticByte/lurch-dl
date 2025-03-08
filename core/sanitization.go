// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"strings"
	"unicode"
)

var FnInvalidRunes = []rune("/<>:\"\\|?*")

func sanitizeUnicodeFilename(filename string) string {
	filename = strings.Trim(strings.ToValidUTF8(filename, ""), " \033\007\u00A0\t\n\r.")
	var filenameBuilder strings.Builder
	for _, r := range filename {
		isInvalid := !unicode.IsPrint(r)
		if isInvalid {
			continue
		}
		for _, c := range FnInvalidRunes {
			if r == c {
				isInvalid = true
				break
			}
		}
		if !isInvalid {
			filenameBuilder.WriteRune(r)
		}
	}
	return filenameBuilder.String()
}
