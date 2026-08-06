package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/extract"
)

var documentExtractCmd = &cobra.Command{
	Use:   "extract <path>",
	Short: "Ingest a document into the workspace as a source document",
	Long: `Ingest a local file (PDF, DOCX, EPUB, ...) as a source document: the raw
file is kept in the workspace's sources/ directory and its extracted text is
indexed for agent retrieval.

By default this prints the source-document handle (metadata) only — the agent
pulls content on demand. Select what to receive:
  --text        print the full extracted text
  --chapter N   print chapter N of the text
  --lines a,b   print lines a..b of the text (1-based)
  --out <path>  write the selected text to a file (works with --text/--chapter/--lines)

Cost pre-flight: read 'estimatedTokens' in the --json handle before generating;
do not feed the whole document to the model repeatedly.

Examples:
  pharos document extract local-guide.pdf --workspace "sql" --json
  pharos document extract local-guide.pdf --chapter 4 --out /tmp/ch4.txt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		srcPath := args[0]
		if _, err := os.Stat(srcPath); err != nil {
			return fmt.Errorf("source file %q: %w", srcPath, err)
		}
		wsName, _ := cmd.Flags().GetString("workspace")
		wsStore, err := resolveWorkspace(s, wsName)
		if err != nil {
			return err
		}
		ws := wsStore.Workspace()

		title, _ := cmd.Flags().GetString("title")
		if strings.TrimSpace(title) == "" {
			title = filepath.Base(srcPath)
		}

		doc, err := wsStore.CreateSourceDoc(srcPath, title)
		if err != nil {
			return formatError("failed to ingest source document", err)
		}

		// Select the text to hand back, on demand.
		text := ""
		chapter, _ := cmd.Flags().GetInt("chapter")
		switch {
		case chapter > 0:
			ch, err := wsStore.GetSourceChapter(doc.ID, chapter)
			if err != nil {
				return err
			}
			text = ch.Segment
		case cmd.Flags().Changed("lines"):
			full, err := wsStore.GetSourceText(doc.ID)
			if err != nil {
				return err
			}
			lines, _ := cmd.Flags().GetString("lines")
			text = selectLines(full, lines)
		case cmd.Flags().Changed("text"):
			t, err := wsStore.GetSourceText(doc.ID)
			if err != nil {
				return err
			}
			text = t
		}

		out, _ := cmd.Flags().GetString("out")
		if out != "" && text != "" {
			if err := os.WriteFile(out, []byte(text), 0o644); err != nil {
				return fmt.Errorf("write --out %s: %w", out, err)
			}
		}

		chapters, err := wsStore.GetSourceChapters(doc.ID)
		if err != nil {
			return err
		}
		if chapters == nil {
			chapters = []extract.Chapter{}
		}

		type handle struct {
			ID              int64             `json:"id"`
			Title           string            `json:"title"`
			Path            string            `json:"path"`
			SourceFile      string            `json:"sourceFile"`
			Format          string            `json:"format"`
			Method          string            `json:"method"`
			Pages           int               `json:"pages"`
			EstimatedTokens int               `json:"estimatedTokens"`
			ChapterCount    int               `json:"chapterCount"`
			Chapters        []extract.Chapter `json:"chapters"`
			Workspace       string            `json:"workspace"`
		}
		h := handle{
			ID: doc.ID, Title: doc.Title, Path: doc.Path, SourceFile: doc.Filename,
			Format: doc.Format, Method: doc.Method, Pages: doc.Pages,
			EstimatedTokens: doc.EstimatedTokens, ChapterCount: len(chapters),
			Chapters: chapters, Workspace: ws.Name,
		}

		if jsonOut {
			type result struct {
				handle
				Text     string `json:"text,omitempty"`
				TextFile string `json:"textFile,omitempty"`
			}
			r := result{handle: h}
			if text != "" {
				if out != "" {
					r.TextFile = out
				} else {
					r.Text = text
				}
			}
			printJSON(r)
			return nil
		}

		fmt.Println()
		fmt.Printf("  ✓ Source ingested: %s (%s)\n", doc.Title, doc.Format)
		fmt.Printf("    Path: %s/%s\n", ws.Path, doc.Path)
		if text != "" {
			if out != "" {
				fmt.Printf("    Text written to: %s\n", out)
			} else {
				fmt.Println("    Text:")
				fmt.Println(text)
			}
		} else {
			fmt.Println("    Add --text / --chapter N / --lines a,b to pull content, or use 'pharos document query'.")
		}
		fmt.Println()
		return nil
	},
}

// selectLines returns lines a..b (1-based) of text, joined with newlines.
func selectLines(text, spec string) string {
	a, b := 1, -1
	parts := strings.SplitN(spec, ",", 2)
	if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
		a = v
	}
	if len(parts) == 2 {
		if v, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			b = v
		}
	}
	lines := strings.Split(text, "\n")
	var out []string
	for i, ln := range lines {
		num := i + 1
		if num < a || (b > 0 && num > b) {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

func init() {
	documentCmd.AddCommand(documentExtractCmd)
	documentExtractCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	documentExtractCmd.Flags().String("title", "", "Source document title (defaults to the file name)")
	documentExtractCmd.Flags().Bool("text", false, "Print the full extracted text")
	documentExtractCmd.Flags().Int("chapter", 0, "Print text for this chapter (1-based)")
	documentExtractCmd.Flags().String("lines", "", "Print lines a,b of the text (1-based)")
	documentExtractCmd.Flags().String("out", "", "Write the selected text to this file")
}
