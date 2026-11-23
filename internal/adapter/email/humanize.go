package email

import (
	"fmt"
	"strings"
	"time"
)

const digestLength = 12

func humanize(d time.Duration) string {
	switch {
	case d < time.Second:
		return "1s"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d/time.Minute))
	}
	hours := int(d / time.Hour)
	minutes := int(d % time.Hour / time.Minute)
	if minutes == 0 {
		return fmt.Sprintf("%d h", hours)
	}
	return fmt.Sprintf("%d h %d min", hours, minutes)
}

func spell(d time.Duration) string {
	switch {
	case d < time.Minute:
		return quantity(max(int(d/time.Second), 1), "second", "seconds")
	case d < time.Hour:
		return quantity(int(d/time.Minute), "minute", "minutes")
	}
	hours := quantity(int(d/time.Hour), "hour", "hours")
	minutes := int(d % time.Hour / time.Minute)
	if minutes == 0 {
		return hours
	}
	return hours + " " + quantity(minutes, "minute", "minutes")
}

func times(count int) string {
	return quantity(count, "time", "times")
}

func quantity(count int, one, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", count, many)
}

func shortDigest(digest string) string {
	hex := digest[strings.LastIndex(digest, ":")+1:]
	if len(hex) > digestLength {
		return hex[:digestLength]
	}
	return hex
}

func truncate(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
