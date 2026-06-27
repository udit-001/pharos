package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all workspaces",
	Long: `Full-text search across lessons, learning records, and references.

Searches lesson body content, and titles/summaries of all entity types.
Searches across all workspaces unless --workspace is specified.

Examples:
  pharos search "SQL joins"
  pharos search "joins" --workspace "sql-for-research"
  pharos search index
  pharos search index --all`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		q := args[0]

		wsName, _ := cmd.Flags().GetString("workspace")

		var results []db.SearchResult
		var err error

		if wsName != "" {
			var wsStore *db.WorkspaceStore
			wsStore, err = s.Workspace(wsName)
			if err != nil {
				return fmt.Errorf("workspace %q not found", wsName)
			}
			results, err = wsStore.Search(q)
		} else {
			results, err = s.Search(q)
		}
		if err != nil {
			return formatError("search failed", err)
		}

		if jsonOut {
			printJSON(results)
			return nil
		}

		if len(results) == 0 {
			fmt.Println()
			fmt.Printf("  No results for %q.\n", q)
			fmt.Println()
			return nil
		}

		fmt.Println()
		fmt.Printf("  Results for %q:\n\n", q)

		rows := make([][]string, 0, len(results))
		for _, r := range results {
			rows = append(rows, []string{r.Type, r.WorkspaceName, truncate(r.Title, 40), truncate(r.Summary, 40)})
		}
		fmt.Println(formatTable([]string{"Type", "Workspace", "Title", "Summary"}, rows))
		fmt.Println()
		return nil
	},
}

var searchIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Build the search index from files on disk",
	Long: `Read all lesson, record, and reference files from disk and index their body content for full-text search.

Run this once after upgrading to a version with body_text indexing to pick
up existing content. Idempotent — already-indexed items are skipped.

Examples:
  pharos search index
  pharos search index --all
  pharos search index --workspace "sql-for-research"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		all, _ := cmd.Flags().GetBool("all")

		var total int

		if all {
			n, err := s.IndexSearch()
			if err != nil {
				return formatError("index failed", err)
			}
			total = n
		} else {
			wsName, _ := cmd.Flags().GetString("workspace")
			wsStore, err := resolveWorkspace(s, wsName)
			if err != nil {
				return err
			}
			ws := wsStore.Workspace()

			var errs []error
			n, err := wsStore.IndexLessons()
			total = n
			if err != nil {
				errs = append(errs, err)
			}
			n, err = wsStore.IndexRefs()
			total += n
			if err != nil {
				errs = append(errs, err)
			}
			n, err = wsStore.IndexRecords()
			total += n
			if err != nil {
				errs = append(errs, err)
			}
			if err := errors.Join(errs...); err != nil {
				return formatError("index failed", err)
			}

			if jsonOut {
				printJSON(map[string]any{
					"workspace": ws.Name,
					"indexed":   total,
				})
				return nil
			}

			if total == 0 {
				fmt.Printf("  ✓ All items in %s already indexed.\n", ws.DisplayName())
			} else {
				fmt.Printf("  ✓ %s: %d items indexed\n", ws.DisplayName(), total)
			}
			fmt.Println()
			return nil
		}

		if jsonOut {
			printJSON(map[string]any{"indexed": total})
			return nil
		}

		if total == 0 {
			fmt.Println()
			fmt.Println("  All items already indexed.")
			fmt.Println()
		} else {
			fmt.Printf("  Total: %d items indexed\n", total)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.AddCommand(searchIndexCmd)
	searchCmd.Flags().StringP("workspace", "w", "", "Scope search to a single workspace")
	searchIndexCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	searchIndexCmd.Flags().Bool("all", false, "Index all workspaces")
}
