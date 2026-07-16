package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

const (
	dailyLogDateFormat = "20060102"
	defaultMaxLogSize  = 100 * 1024 * 1024 // 100MB
)

type dailyLogWriter struct {
	dir        string
	prefix     string
	ext        string
	maxSize    int64
	maxBackups int
	maxAge     int
	mu         sync.Mutex
	file       *os.File
	date       string
	segment    int
	size       int64
}

type segmentInfo struct {
	segment int
	size    int64
	path    string
}

type datedLogFile struct {
	date string
	path string
}

func newDailyLogWriter(dir, prefix, ext string, maxSize int64, maxBackups, maxAge int) *dailyLogWriter {
	if maxSize <= 0 {
		maxSize = defaultMaxLogSize
	}
	return &dailyLogWriter{
		dir:        dir,
		prefix:     prefix,
		ext:        ext,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
	}
}

func parseLogFilePath(path string) (dir, prefix, ext string) {
	dir = filepath.Dir(path)
	base := filepath.Base(path)
	ext = filepath.Ext(base)
	prefix = strings.TrimSuffix(base, ext)
	return dir, prefix, ext
}

func parseLogSegmentName(name, prefix, ext string) (segment int, date string, ok bool) {
	expectPrefix := prefix + "-"
	if !strings.HasPrefix(name, expectPrefix) || !strings.HasSuffix(name, ext) {
		return 0, "", false
	}
	middle := strings.TrimSuffix(strings.TrimPrefix(name, expectPrefix), ext)
	if len(middle) < len(dailyLogDateFormat) {
		return 0, "", false
	}

	date = middle[:len(dailyLogDateFormat)]
	if _, err := time.Parse(dailyLogDateFormat, date); err != nil {
		return 0, "", false
	}

	rest := middle[len(dailyLogDateFormat):]
	if rest == "" {
		return 1, date, true
	}
	if !strings.HasPrefix(rest, ".") {
		return 0, "", false
	}
	segment, err := strconv.Atoi(rest[1:])
	if err != nil || segment < 1 {
		return 0, "", false
	}
	return segment, date, true
}

func (d *dailyLogWriter) pathForSegment(date string, segment int) string {
	return filepath.Join(d.dir, fmt.Sprintf("%s-%s.%d%s", d.prefix, date, segment, d.ext))
}

func (d *dailyLogWriter) today() string {
	return sharedtimezone.Today()
}

func (d *dailyLogWriter) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureOpenLocked(); err != nil {
		return 0, err
	}
	if d.size+int64(len(p)) > d.maxSize {
		if err := d.rotateSegmentLocked(); err != nil {
			return 0, err
		}
	}
	n, err := d.file.Write(p)
	if err != nil {
		return n, err
	}
	d.size += int64(n)
	return n, nil
}

func (d *dailyLogWriter) Sync() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.file == nil {
		return nil
	}
	return d.file.Sync()
}

func (d *dailyLogWriter) Rotate() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.file != nil {
		_ = d.file.Close()
		d.file = nil
	}
	d.date = ""
	d.segment = 0
	d.size = 0
	if err := d.ensureOpenLocked(); err != nil {
		return err
	}
	d.cleanupLocked()
	return nil
}

func (d *dailyLogWriter) ensureOpenLocked() error {
	today := d.today()
	if d.file != nil && d.date == today {
		return nil
	}
	if d.file != nil {
		_ = d.file.Close()
		d.file = nil
	}
	d.date = today
	d.segment = 0
	d.size = 0
	return d.openSegmentLocked(today, 0)
}

func (d *dailyLogWriter) rotateSegmentLocked() error {
	if d.file != nil {
		_ = d.file.Close()
		d.file = nil
	}
	return d.openSegmentLocked(d.date, d.segment)
}

func (d *dailyLogWriter) openSegmentLocked(date string, afterSegment int) error {
	if err := os.MkdirAll(d.dir, 0o744); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	segment, size, err := d.resolveSegment(date, afterSegment)
	if err != nil {
		return err
	}

	path := d.pathForSegment(date, segment)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	d.file = f
	d.segment = segment
	d.size = size
	return nil
}

func (d *dailyLogWriter) resolveSegment(date string, afterSegment int) (segment int, size int64, err error) {
	if afterSegment > 0 {
		return afterSegment + 1, 0, nil
	}
	segments, err := d.listSegmentsForDate(date)
	if err != nil {
		return 0, 0, err
	}
	if len(segments) == 0 {
		return 1, 0, nil
	}
	latest := segments[len(segments)-1]
	if latest.size >= d.maxSize {
		return latest.segment + 1, 0, nil
	}
	return latest.segment, latest.size, nil
}

func (d *dailyLogWriter) listSegmentsForDate(date string) ([]segmentInfo, error) {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}

	var segments []segmentInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		segment, fileDate, ok := parseLogSegmentName(entry.Name(), d.prefix, d.ext)
		if !ok || fileDate != date {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		segments = append(segments, segmentInfo{
			segment: segment,
			size:    info.Size(),
			path:    filepath.Join(d.dir, entry.Name()),
		})
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].segment < segments[j].segment
	})
	return segments, nil
}

func (d *dailyLogWriter) cleanupLocked() {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return
	}

	byDate := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		_, date, ok := parseLogSegmentName(entry.Name(), d.prefix, d.ext)
		if !ok {
			continue
		}
		byDate[date] = append(byDate[date], filepath.Join(d.dir, entry.Name()))
	}
	if len(byDate) == 0 {
		return
	}

	var dates []datedLogFile
	for date, paths := range byDate {
		for _, path := range paths {
			dates = append(dates, datedLogFile{date: date, path: path})
		}
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].date > dates[j].date
	})

	uniqueDates := make([]string, 0, len(byDate))
	seen := make(map[string]struct{})
	for _, f := range dates {
		if _, ok := seen[f.date]; ok {
			continue
		}
		seen[f.date] = struct{}{}
		uniqueDates = append(uniqueDates, f.date)
	}

	if d.maxAge > 0 {
		cutoff := sharedtimezone.DaysAgo(d.maxAge)
		for date, paths := range byDate {
			if date < cutoff {
				for _, path := range paths {
					_ = os.Remove(path)
				}
			}
		}
	}
	if d.maxBackups > 0 && len(uniqueDates) > d.maxBackups {
		for _, date := range uniqueDates[d.maxBackups:] {
			for _, path := range byDate[date] {
				_ = os.Remove(path)
			}
		}
	}
}
