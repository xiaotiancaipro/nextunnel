package configs

import (
	"fmt"
	"strconv"
	"strings"
)

type Logs struct {
	File       string `toml:"file"`
	Level      string `toml:"level"`
	MaxSize    string `toml:"maxSize"`
	MaxBackups int    `toml:"maxBackups"`
	MaxAge     int    `toml:"maxAge"`
}

func (l *Logs) MaxSizeBytes() (int64, error) {
	return l.parseMaxSize(l.MaxSize)
}

// parseMaxSize parses sizes like "100MB", "1GB", "512KB".
// A bare number (e.g. "100") is treated as megabytes for backward compatibility.
func (l *Logs) parseMaxSize(s string) (int64, error) {

	s = strings.TrimSpace(s)
	if s == "" {
		return 100 * 1024 * 1024, nil
	}

	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid maxSize %q: missing numeric value", s)
	}

	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid maxSize %q: %w", s, err)
	}

	unit := strings.ToUpper(strings.TrimSpace(s[i:]))
	if unit == "" {
		unit = "MB"
	}

	var multiplier int64
	switch unit {
	case "B":
		multiplier = 1
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid maxSize %q: unknown unit %q", s, unit)
	}

	bytes := int64(n * float64(multiplier))
	if bytes <= 0 {
		return 0, fmt.Errorf("invalid maxSize %q: size must be positive", s)
	}
	return bytes, nil

}
