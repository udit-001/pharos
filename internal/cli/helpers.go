package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
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

// parseLessonFlag reads the optional --lesson flag (a lesson sequence number
// linking a quiz to the lesson whose skill it practices). Returns (seq, hasFlag,
// err); when the flag is empty, hasFlag is false so callers can leave the link
// unset. Shared by `quiz create` and `quiz revise`.
func parseLessonFlag(cmd *cobra.Command) (int, bool, error) {
	raw, _ := cmd.Flags().GetString("lesson")
	if strings.TrimSpace(raw) == "" {
		return 0, false, nil
	}
	seq, err := parseSeq(raw)
	if err != nil {
		return 0, false, fmt.Errorf("--lesson %w", err)
	}
	return seq, true, nil
}

// lessonRef formats a quiz's lesson link for display: "#N" when linked, "—"
// when not. Shared by quiz read/list/attempts/show.
func lessonRef(seq *int) string {
	if seq == nil {
		return "—"
	}
	return fmt.Sprintf("#%d", *seq)
}
