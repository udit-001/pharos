package cli

import (
	"fmt"
	"strconv"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
	migrateutil "github.com/udit-001/pharos/internal/migrate"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `Run, roll back, and inspect database migrations.

Migrations are stored in the project's migrations/ directory and
are embedded into the binary at build time.

Use 'pharos migrate up' to apply pending migrations,
'pharos migrate down' to roll back the last one, and
'pharos migrate status' to see the current state.

Examples:
  pharos migrate up
  pharos migrate down
  pharos migrate status
  pharos migrate up-to 1
  pharos migrate down-to 0`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rawDB, err := db.OpenRaw(storePath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer rawDB.Close()

		if err := migrateutil.Up(rawDB); err != nil {
			return fmt.Errorf("migrate up: %w", err)
		}

		fmt.Println()
		fmt.Println("  ✓ Migrations up to date")
		fmt.Println()
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back the most recent migration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rawDB, err := db.OpenRaw(storePath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer rawDB.Close()

		if err := migrateutil.Down(rawDB); err != nil {
			return fmt.Errorf("migrate down: %w", err)
		}

		fmt.Println()
		fmt.Println("  ✓ Rolled back last migration")
		fmt.Println()
		return nil
	},
}

var migrateUpToCmd = &cobra.Command{
	Use:   "up-to <version>",
	Short: "Run migrations up to a specific version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version: %s", args[0])
		}

		rawDB, err := db.OpenRaw(storePath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer rawDB.Close()

		if err := migrateutil.UpTo(rawDB, version); err != nil {
			return fmt.Errorf("migrate up-to %d: %w", version, err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Migrated up to version %d\n", version)
		fmt.Println()
		return nil
	},
}

var migrateDownToCmd = &cobra.Command{
	Use:   "down-to <version>",
	Short: "Roll back migrations to a specific version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version: %s", args[0])
		}

		rawDB, err := db.OpenRaw(storePath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer rawDB.Close()

		if err := migrateutil.DownTo(rawDB, version); err != nil {
			return fmt.Errorf("migrate down-to %d: %w", version, err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Rolled back to version %d\n", version)
		fmt.Println()
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rawDB, err := db.OpenRaw(storePath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer rawDB.Close()

		statuses, err := migrateutil.Status(rawDB)
		if err != nil {
			return fmt.Errorf("get migration status: %w", err)
		}

		fmt.Println()
		if len(statuses) == 0 {
			fmt.Println("  No migrations found.")
			fmt.Println()
			return nil
		}

		if jsonOut {
			printJSON(statuses)
			return nil
		}

		rows := make([][]string, 0, len(statuses))
		for _, s := range statuses {
			applied := "pending"
			if s.State == goose.StateApplied {
				applied = s.AppliedAt.Format("2006-01-02 15:04")
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", s.Source.Version),
				s.Source.Path,
				applied,
			})
		}

		fmt.Println(formatTable(
			[]string{"Version", "Migration", "Applied"},
			rows,
		))
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateUpToCmd)
	migrateCmd.AddCommand(migrateDownToCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
}
