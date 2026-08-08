package cli

import (
	"fmt"
	"strings"

	"github.com/udit-001/pharos/internal/db"
)

// scrap stored prints the flat / JSON views for the scratchpad commands.

// tagNames flattens a []db.Tag into its names (used by every scrap/tag print
// path — single source for the tag->names mapping).
func tagNames(tags []db.Tag) []string {
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names
}

func printScrapsJSON(s *db.Store, scraps []db.Scrap) error {
	out := make([]map[string]any, 0, len(scraps))
	for _, sc := range scraps {
		tags, err := s.TagsForScrap(sc.Slug)
		if err != nil {
			return formatError("failed to read scrap tags", err)
		}
		out = append(out, map[string]any{
			"slug":      sc.Slug,
			"title":     sc.Title,
			"status":    sc.Status,
			"tags":      tagNames(tags),
			"body":      sc.Body,
			"updatedAt": sc.UpdatedAt,
		})
	}
	printJSON(out)
	return nil
}

func printScrapsTable(s *db.Store, scraps []db.Scrap) error {
	if len(scraps) == 0 {
		fmt.Println()
		fmt.Println("  No scraps found. Create one with:")
		fmt.Println()
		fmt.Println("    pharos scrap add \"<title>\" --body-file <path>")
		fmt.Println()
		return nil
	}
	rows := make([][]string, 0, len(scraps))
	for _, sc := range scraps {
		tags, err := s.TagsForScrap(sc.Slug)
		if err != nil {
			return formatError("failed to read scrap tags", err)
		}
		rows = append(rows, []string{
			sc.Slug,
			strings.ReplaceAll(sc.Title, "\n", " "),
			sc.Status,
			"[" + strings.Join(tagNames(tags), ", ") + "]",
			formatDateShort(sc.UpdatedAt),
		})
	}
	fmt.Println()
	fmt.Print(formatTable([]string{"SLUG", "TITLE", "STATUS", "TAGS", "UPDATED"}, rows))
	fmt.Println()
	return nil
}

func printScrapJSON(sc db.Scrap, tags []db.Tag) error {
	printJSON(map[string]any{
		"slug":      sc.Slug,
		"title":     sc.Title,
		"status":    sc.Status,
		"tags":      tagNames(tags),
		"body":      sc.Body,
		"createdAt": sc.CreatedAt,
		"updatedAt": sc.UpdatedAt,
	})
	return nil
}

func printScrapDetail(sc db.Scrap, tags []db.Tag) {
	names := tagNames(tags)
	fmt.Println()
	fmt.Printf("  %s  [%s]\n", sc.Title, sc.Status)
	if len(names) > 0 {
		fmt.Printf("  tags: %s\n", strings.Join(names, ", "))
	}
	fmt.Printf("  slug: %s\n", sc.Slug)
	fmt.Println()
	fmt.Println(sc.Body)
	fmt.Println()
}
