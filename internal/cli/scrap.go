package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var scrapCmd = &cobra.Command{
	Use:   "scrap",
	Short: "Manage global scratchpad scraps",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Manage the global scratchpad — loose, unstructured captures that live
outside any workspace (park a resource, a half-formed intent, or an
aspiration like "I want to be an ML engineer").

Use --search to find an existing scrap before adding (find-then-update),
then update it by slug rather than duplicating.

Examples:
  pharos scrap list
  pharos scrap list --search "ml engineer"
  pharos scrap list --status done
  pharos scrap read ml-engineer
  pharos scrap add "ML engineer roadmap" --body-file /tmp/roadmap.md --tag ml --tag career
  pharos scrap update ml-engineer --body-file /tmp/roadmap.md --status active`,
}

var scrapListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scratchpad scraps",
	Long: `List scraps. Without --status, only ACTIVE scraps are shown (the agent's
default context read). Pass --status done to see what was already covered,
or --search to filter by full-text match across title, body, and tag
descriptions.

Examples:
  pharos scrap list
  pharos scrap list --status done
  pharos scrap list --search "ml engineer"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		status, _ := cmd.Flags().GetString("status")
		search, _ := cmd.Flags().GetString("search")

		var scraps []db.Scrap
		var err error
		if search != "" {
			scraps, err = s.SearchScraps(search, status)
		} else {
			scraps, err = s.ListScraps(status)
		}
		if err != nil {
			return formatError("failed to list scraps", err)
		}

		if jsonEnabled(cmd) {
			return printScrapsJSON(s, scraps)
		}
		return printScrapsTable(s, scraps)
	},
}

var scrapReadCmd = &cobra.Command{
	Use:   "read <slug>",
	Short: "Read a scrap's content",
	Long: `Read one scrap's body, status, and attached tags by its slug.

Examples:
  pharos scrap read ml-engineer
  pharos scrap read ml-engineer --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		scrap, err := s.ScrapBySlug(slug)
		if err != nil {
			return formatError("scrap not found", err)
		}
		tags, err := s.TagsForScrap(slug)
		if err != nil {
			return formatError("failed to read scrap tags", err)
		}

		if jsonEnabled(cmd) {
			return printScrapJSON(scrap, tags)
		}
		printScrapDetail(scrap, tags)
		return nil
	},
}

var scrapAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new scrap",
	Long: `Create a new ACTIVE scrap with the given title (title is required and
drives the stable slug). Pass --body-file for the free-form body, and repeat
--tag to attach tags.

Before adding, search first: if an active scrap already covers the idea,
update it by slug instead of duplicating.

Examples:
  pharos scrap add "ML engineer roadmap" --body-file /tmp/roadmap.md --tag ml --tag career`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		title := args[0]
		bodyFile, _ := cmd.Flags().GetString("body-file")
		var tags []string
		if cmd.Flags().Changed("tag") {
			tags, _ = cmd.Flags().GetStringArray("tag")
		}

		if bodyFile == "" {
			return fmt.Errorf("--body-file is required\n  pharos scrap add %q --body-file <path>", title)
		}
		body, err := readBodyFile(bodyFile)
		if err != nil {
			return err
		}

		scrap, err := s.CreateScrap(title, string(body), tags)
		if err != nil {
			return formatError("failed to create scrap", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Scrap created: %s\n", scrap.Slug)
		fmt.Println()
		return nil
	},
}

var scrapUpdateCmd = &cobra.Command{
	Use:   "update <slug>",
	Short: "Update a scrap in place",
	Long: `Update an existing scrap by stable slug (find-then-update). The slug
never changes. Use --title to rename, --body-file to replace the body,
--status active|done to change status, and repeat --tag to replace the full
tag set.

Examples:
  pharos scrap update ml-engineer --body-file /tmp/roadmap.md
  pharos scrap update ml-engineer --title "ML engineer" --status done
  pharos scrap update ml-engineer --tag ml --tag career`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]

		var status *string
		statusFlag, _ := cmd.Flags().GetString("status")
		if statusFlag != "" {
			if statusFlag != "active" && statusFlag != "done" {
				return fmt.Errorf("--status must be 'active' or 'done', got %q", statusFlag)
			}
			status = &statusFlag
		}

		var title *string
		titleFlag, _ := cmd.Flags().GetString("title")
		if titleFlag != "" {
			title = &titleFlag
		}

		var body *string
		bodyFile, _ := cmd.Flags().GetString("body-file")
		if bodyFile != "" {
			data, err := readBodyFile(bodyFile)
			if err != nil {
				return err
			}
			b := string(data)
			body = &b
		}
		var tags *[]string
		tagFlag, _ := cmd.Flags().GetStringArray("tag")
		if cmd.Flags().Changed("tag") {
			ts := tagFlag
			tags = &ts
		}

		if title == nil && body == nil && status == nil && tags == nil {
			return fmt.Errorf("nothing to update — pass --title, --body-file, --status, and/or --tag\n  pharos scrap update %q --body-file <path>", slug)
		}

		updated, err := s.UpdateScrap(slug, title, body, status, tags)
		if err != nil {
			return formatError("failed to update scrap", err)
		}

		fmt.Println()
		fmt.Printf("  ✓ Scrap updated: %s\n", updated.Slug)
		fmt.Println()
		return nil
	},
}

var scrapDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a scrap permanently",
	Long: `Permanently delete a scrap and its tag associations.

Examples:
  pharos scrap delete ml-engineer`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		if err := s.DeleteScrap(args[0]); err != nil {
			return formatError("failed to delete scrap", err)
		}
		fmt.Println()
		fmt.Printf("  ✓ Scrap deleted: %s\n", args[0])
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scrapCmd)
	scrapCmd.AddCommand(scrapListCmd)
	scrapCmd.AddCommand(scrapReadCmd)
	scrapCmd.AddCommand(scrapAddCmd)
	scrapCmd.AddCommand(scrapUpdateCmd)
	scrapCmd.AddCommand(scrapDeleteCmd)
	scrapListCmd.Flags().String("status", "active", "Filter by status: active (default) or done")
	scrapListCmd.Flags().String("search", "", "Full-text search across title, body, and tag descriptions")
	scrapAddCmd.Flags().String("body-file", "", "Read scrap body from a file (required)")
	scrapAddCmd.Flags().StringArray("tag", nil, "Attach a tag (repeatable)")
	scrapUpdateCmd.Flags().String("title", "", "Replace the scrap title (slug stays stable)")
	scrapUpdateCmd.Flags().String("body-file", "", "Replace scrap body from a file")
	scrapUpdateCmd.Flags().String("status", "", "Set status: active or done")
	scrapUpdateCmd.Flags().StringArray("tag", nil, "Replace the full tag set (repeatable)")
}
