package domain

import "strings"

const shortIDLength = 8

func ShortID(id string) string {
	compact := strings.ReplaceAll(id, "-", "")
	if len(compact) <= shortIDLength {
		return compact
	}
	return compact[len(compact)-shortIDLength:]
}
