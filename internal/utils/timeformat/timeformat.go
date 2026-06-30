package timeformat

import (
	"strings"
	"time"
)

const UTC = "2006-01-02T15:04:05Z"

func FormatUTC(t time.Time) string {
	return t.UTC().Format(UTC)
}

func ParseRFC3339(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
