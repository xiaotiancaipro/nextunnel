package utils

import "strings"

func Normalize(stringArr []string) []string {
	final := make([]string, 0, len(stringArr))
	for _, s := range stringArr {
		s = strings.TrimSpace(s)
		if s != "" {
			final = append(final, s)
		}
	}
	return final
}

func NullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func PickName(names map[string]string, stringArr []string) string {
	for _, locale := range stringArr {
		if name := strings.TrimSpace(names[locale]); name != "" {
			return name
		}
	}
	return ""
}
