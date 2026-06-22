package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// parseSeq converts a string argument to a sequence number.
func parseSeq(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid sequence number %q", s)
	}
	if n < 1 {
		return 0, fmt.Errorf("sequence number must be positive, got %d", n)
	}
	return n, nil
}

// urlPathEscapeCLI replaces spaces for URL path segments (CLI-side).
func urlPathEscapeCLI(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}
