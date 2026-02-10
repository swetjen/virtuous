package httpapi

import (
	"regexp"

	"github.com/swetjen/virtuous/internal/textutil"
)

var pathParamRegexp = regexp.MustCompile(`\{([^/}]+)\}`)

func parsePathParams(path string) []string {
	matches := pathParamRegexp.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		params = append(params, match[1])
	}
	return params
}

// camelizeDown converts a name into lower camel case.
func camelizeDown(word string) string {
	return textutil.CamelizeDown(word)
}

func lowerFirst(s string) string {
	return textutil.LowerFirst(s)
}
