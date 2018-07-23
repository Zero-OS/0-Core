// +build production

package utils

var invalidKeys = []string{
	"development", "debug",
}

func isValidParam(key string) bool {
	for _, invalid := range invalidKeys {
		if key == invalid {
			return false
		}
	}

	return true
}
