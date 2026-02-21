// Copyright (c) 2025, Julian Müller (ChaoticByte)

package core

import (
	"slices"
	"strings"
	"unicode"
)

var FnInvalidRunes = []rune("/<>:\"\\|?*")

func sanitizeUnicodeFilename(filename string) string {
	filename = strings.Trim(strings.ToValidUTF8(filename, ""), " \033\007\u00A0\t\n\r.")
	var filenameBuilder strings.Builder
	for _, r := range filename {
		if unicode.IsPrint(r) && !slices.Contains(FnInvalidRunes, r) {
			filenameBuilder.WriteRune(r)
		}
	}
	return filenameBuilder.String()
}
