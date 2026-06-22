package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize pharos: create the database",
	Long: `Set up pharos on this machine.

Creates the SQLite database at ~/.pharos/pharos.db (override with
--db) and runs migrations.

This command is idempotent: if the database already exists it is left
untouched (use --force to recreate it). Workspaces are created
separately with 'pharos workspace create <name>'.

Examples:
  pharos init                # Set up pharos
  pharos init --force        # Recreate the database from scratch`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		// DB setup. Idempotent: an existing database is a no-op unless --force,
		// in which case it (and its WAL/SHM sidecars) is removed and recreated.
		if _, err := os.Stat(storePath); err == nil {
			if force {
				for _, suffix := range []string{"", "-wal", "-shm"} {
					if err := os.Remove(storePath + suffix); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("remove existing database: %w", err)
					}
				}
				s, err := db.Open(storePath)
				if err != nil {
					return fmt.Errorf("create database: %w", err)
				}
				_ = s.Close()
				fmt.Printf("  ✓ Recreated database: %s\n", storePath)
			} else {
				fmt.Printf("  • Database already initialized: %s\n", storePath)
			}
		} else {
			s, err := db.Open(storePath)
			if err != nil {
				return fmt.Errorf("create database: %w", err)
			}
			_ = s.Close()
			fmt.Printf("  ✓ Initialized database: %s\n", storePath)
		}

		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("    pharos workspace create \"Your topic\"")
		fmt.Println()
		fmt.Println("  Install the teaching skill for your AI agent:")
		fmt.Println("    pharos skills install --agent <name>")
		fmt.Println()
		fmt.Println("  Supported agents: opencode, claude-code, codex, pi.dev")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "Recreate the database if it exists")
}
