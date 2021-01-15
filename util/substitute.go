package util

import (
	"regexp"
	"strings"
)

func Substitute(pattern string, properties func(key string) (string, error)) (string, error) {
	re := regexp.MustCompile(`\(([^()]+)\)`)
	ind := re.FindAllStringSubmatchIndex(pattern, -1)
	var pieces []string
	start := 0
	for _, match := range ind {
		pieces = append(pieces, pattern[start:match[0]])
		start = match[1]
		key := pattern[match[2]:match[3]]
		value, err := properties(key)
		if err != nil {
			return "", err
		}
		pieces = append(pieces, value)
	}
	pieces = append(pieces, pattern[start:])
	return strings.Join(pieces, ""), nil
}
