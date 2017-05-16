package screen

import "regexp"

var (
	cleanregex = regexp.MustCompile(`\033\[[\d;]*[\w]`)
)

func StringWidth(s string) int {
	return len(cleanregex.ReplaceAllLiteralString(s, ""))
}
