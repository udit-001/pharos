package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// runShowHelp prints the command's help and returns nil. It is the RunE for
// parent commands (those that only group subcommands) so cobra treats them as
// runnable — which in turn lets Args validation (e.g. cobra.NoArgs) run and
// reject unknown subcommands instead of silently printing help and exiting 0.
func runShowHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}

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

// parseLessonFlag reads the optional --lesson flag (a lesson slug linking a
// quiz to the lesson whose skill it practices). Returns (slug, hasFlag, err);
// when the flag is empty, hasFlag is false so callers can leave the link unset.
// Shared by `quiz create` and `quiz revise`.
func parseLessonFlag(cmd *cobra.Command) (string, bool, error) {
	raw, _ := cmd.Flags().GetString("lesson")
	if strings.TrimSpace(raw) == "" {
		return "", false, nil
	}
	return strings.TrimSpace(raw), true, nil
}

// lessonRef formats a quiz's lesson link for display: the slug when linked,
// "—" when not. Shared by quiz show/attempts.
func lessonRef(slug *string) string {
	if slug == nil {
		return "—"
	}
	return *slug
}
