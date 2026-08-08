package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage scratchpad tags",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Manage the scratchpad's first-class tags. A tag is a name plus a
description — the description is the semantic payload (it powers tag-based
search and lets the agent disambiguate one tag from another), so prefer
giving every tag a real description.

Examples:
  pharos tag list
  pharos tag create "ml" --description "machine learning career goal"
  pharos tag update "ml" --description "revised description"
  pharos tag delete "ml"`,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scratchpad tags",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		tags, err := s.ListTags()
		if err != nil {
			return formatError("failed to list tags", err)
		}
		if jsonEnabled(cmd) {
			return printTagsJSON(tags)
		}
		return printTagsTable(tags)
	},
}

var tagCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a tag",
	Long: `Create a new scratchpad tag with a semantic description. Fails if the tag
already exists — change an existing tag's description with 'tag update'.

Description is the tag's semantic payload: it is what powers tag search and
lets the agent tell one tag from another. Every tag should have one.

Examples:
  pharos tag create "ml" --description "machine learning career goal"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")

		if _, err := s.CreateTag(name, desc); err != nil {
			return formatError("failed to create tag", err)
		}
		fmt.Println()
		fmt.Printf("  ✓ Tag created: %s\n", name)
		fmt.Println()
		return nil
	},
}

var tagUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Set a tag's description",
	Long: `Update an existing tag's description (the semantic payload).

Example:
  pharos tag update "ml" --description "revised career goal"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		if err := s.UpdateTagDescription(name, desc); err != nil {
			return formatError("failed to update tag", err)
		}
		fmt.Println()
		fmt.Printf("  ✓ Tag updated: %s\n", name)
		fmt.Println()
		return nil
	},
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a tag",
	Long: `Permanently delete a tag and detach it from all scraps.

Example:
  pharos tag delete "ml"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		if err := s.DeleteTag(args[0]); err != nil {
			return formatError("failed to delete tag", err)
		}
		fmt.Println()
		fmt.Printf("  ✓ Tag deleted: %s\n", args[0])
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagCreateCmd)
	tagCmd.AddCommand(tagUpdateCmd)
	tagCmd.AddCommand(tagDeleteCmd)
	tagCreateCmd.Flags().String("description", "", "Semantic description of the tag")
	tagUpdateCmd.Flags().String("description", "", "Semantic description of the tag")
}

func printTagsJSON(tags []db.Tag) error {
	out := make([]map[string]any, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
		})
	}
	printJSON(out)
	return nil
}

func printTagsTable(tags []db.Tag) error {
	if len(tags) == 0 {
		fmt.Println()
		fmt.Println("  No tags yet. Create one with:")
		fmt.Println()
		fmt.Println("    pharos tag create \"<name>\" --description \"<reason>\"")
		fmt.Println()
		return nil
	}
	rows := make([][]string, 0, len(tags))
	for _, t := range tags {
		rows = append(rows, []string{t.Name, t.Description})
	}
	fmt.Println()
	fmt.Print(formatTable([]string{"NAME", "DESCRIPTION"}, rows))
	fmt.Println()
	return nil
}
