package timezone

import (
	"fmt"
	"time"
)

const defaultLocation = "Asia/Shanghai"

var displayLoc *time.Location

func Init(name string) error {
	if name == "" {
		name = defaultLocation
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return fmt.Errorf("load location %q: %w", name, err)
	}
	displayLoc = loc
	return nil
}

func Location() *time.Location {
	if displayLoc == nil {
		return time.UTC
	}
	return displayLoc
}

func Format(t time.Time) string {
	return t.In(Location()).Format("2006-01-02 15:04:05")
}

func Today() string {
	return time.Now().In(Location()).Format("20060102")
}

func DaysAgo(days int) string {
	return time.Now().In(Location()).AddDate(0, 0, -days).Format("20060102")
}
