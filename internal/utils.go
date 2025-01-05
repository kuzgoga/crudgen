package internal

import (
	"strings"
	"unicode"
)

func CapitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func ToPlural(entityName string) string {
	return strings.ToLower(entityName) + "s"
}
