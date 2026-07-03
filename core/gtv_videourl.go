package core

import "regexp"

var videoUrlRegex = regexp.MustCompile(`gronkh\.tv\/([a-z]+)\/([0-9]+)`)

func ParseEpisodeNumberFromVideoUrl(url string) (string, error) {
	match := videoUrlRegex.FindStringSubmatch(url)
	if len(match) < 2 {
		return "", &GtvVideoUrlParseError{Url: url}
	}
	cat := match[1]
	if cat != "stream" {
		return "", &VideoCategoryUnsupportedError{Category: cat}
	}
	return match[2], nil
}
