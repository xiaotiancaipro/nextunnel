package utils

import "strings"

func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func NullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}
