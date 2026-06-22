package db

import (
	"strings"
)

// Slugify creates a URL-safe slug from a string: lowercase, hyphens for
// spaces/slashes/underscores, non-alphanumeric removed, collapsed hyphens.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var result strings.Builder
	prevHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			prevHyphen = false
		} else if r == '-' {
			if !prevHyphen && result.Len() > 0 {
				result.WriteRune(r)
				prevHyphen = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}
