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

func times(count int) string {
	if count == 1 {
		return "1 time"
	}
	return fmt.Sprintf("%d times", count)
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
