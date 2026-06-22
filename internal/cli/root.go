package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
	"github.com/udit-001/pharos/internal/version"
)

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "pharos.db"
	}
	return filepath.Join(home, ".pharos", "pharos.db")
}

func defaultWorkspacesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./workspaces"
	}
	return filepath.Join(home, ".pharos", "workspaces")
}

var (
	storePath string
	jsonOut   bool
)

// ctxStore is the context key for the injected *db.Store.
type ctxStore struct{}

// mustStore retrieves the injected *db.Store from the command context.
// It panics if the store is not present (which means PersistentPreRunE
// was skipped — i.e. a help/version/init/migrate command).
func mustStore(cmd *cobra.Command) *db.Store {
	s, ok := cmd.Context().Value(ctxStore{}).(*db.Store)
	if !ok || s == nil {
		panic("store not available in context — command missing from PersistentPreRunE skip list?")
	}
	return s
}

var rootCmd = &cobra.Command{
	Use:     "pharos",
	Short:   "Manage learning lessons and workspaces",
	Version: version.Version,
	Long: `A CLI tool to create and manage learning workspaces.

Data is stored in a local SQLite database. Each workspace is a
directory containing MISSION.md, lessons/, learning-records/,
reference/, assets/, RESOURCES.md, GLOSSARY.md, and NOTES.md.

Use 'pharos init' to set up pharos, then 'pharos workspace create'
to start a workspace.

Most commands support --json for machine-readable output.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version" || cmd.Name() == "init" || cmd.Name() == "migrate" || cmd.Name() == "dev" {
			return nil
		}
		// Migrate subcommands also handle their own DB
		if cmd.Parent() != nil && cmd.Parent().Name() == "migrate" {
			return nil
		}
		s, err := db.Open(storePath)
		if err != nil {
			return fmt.Errorf("open database: %w\n\n  Run 'pharos init' to set up pharos", err)
		}
		ctx := context.WithValue(cmd.Context(), ctxStore{}, s)
		cmd.SetContext(ctx)
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		s, ok := cmd.Context().Value(ctxStore{}).(*db.Store)
		if ok && s != nil {
			return s.Close()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&storePath, "db", defaultDBPath(), "Path to SQLite database")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output as JSON")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func formatError(msg string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", msg, err)
	}
	return fmt.Errorf("%s", msg)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

// writeWorkspaceFile writes content from a body file to a workspace path,
// touches last_studied, and prints a success message. Used by mission,
// resources, and glossary commands for non-interactive updates.
func writeWorkspaceFile(wsStore *db.WorkspaceStore, targetPath, bodyFile, label string) error {
	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return fmt.Errorf("read body file: %w", err)
	}
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", label, err)
	}
	_ = wsStore.Touch()
	fmt.Println()
	fmt.Printf("  ✓ %s updated\n", label)
	fmt.Println()
	return nil
}

func formatDateShort(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

func formatTable(header []string, rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	colWidths := make([]int, len(header))
	for i, h := range header {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	for i := range colWidths {
		if colWidths[i] > 40 {
			colWidths[i] = 40
		}
	}

	var b strings.Builder
	for i, h := range header {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%-*s", colWidths[i], h)
	}
	b.WriteString("\n")

	sepCount := 0
	for _, w := range colWidths {
		sepCount += w
	}
	b.WriteString(strings.Repeat("─", sepCount+2*(len(header)-1)))
	b.WriteString("\n")

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			display := cell
			if len(display) > colWidths[i] {
				display = display[:colWidths[i]-3] + "..."
			}
			fmt.Fprintf(&b, "%-*s", colWidths[i], display)
		}
		b.WriteString("\n")
	}

	return b.String()
}
