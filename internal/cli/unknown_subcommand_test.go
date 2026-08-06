package cli

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

// executeForTest runs the root command with the given args and returns the
// resulting error (no os.Exit, no t.Fatal on error). DB opening is neutralised
// so we isolate cobra's command-routing behaviour from storage.
func executeForTest(t *testing.T, args []string) error {
	t.Helper()
	root := newRootForTest()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return nil }
	root.PersistentPostRunE = nil
	return root.Execute()
}

var parentCommands = []string{
	"asset", "config", "document", "glossary", "lesson", "migrate", "mission", "notes",
	"question", "quiz", "record", "reference", "resources", "skills",
	"tailwind", "workspace",
}

// TestParentCommandsRejectUnknownSubcommands guards that parent commands
// (subcommand groups with no own action) reject unknown subcommands with a
// non-nil error, rather than silently printing help and exiting 0.
func TestParentCommandsRejectUnknownSubcommands(t *testing.T) {
	for _, p := range parentCommands {
		t.Run(p, func(t *testing.T) {
			err := executeForTest(t, []string{p, "definitely-not-a-subcommand"})
			if err == nil {
				t.Errorf("pharos %s <unknown>: expected error, got nil", p)
			}
		})
	}
}

// TestParentCommandsWithoutArgsShowHelp ensures bare parent invocations still
// succeed (print help) — i.e. the fix must not make `pharos skills` error.
func TestParentCommandsWithoutArgsShowHelp(t *testing.T) {
	for _, p := range parentCommands {
		t.Run(p, func(t *testing.T) {
			err := executeForTest(t, []string{p})
			if err != nil {
				t.Errorf("pharos %s: expected no error (help), got %v", p, err)
			}
		})
	}
}
