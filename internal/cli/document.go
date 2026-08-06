package cli

import "github.com/spf13/cobra"

var documentCmd = &cobra.Command{
	Use:   "document",
	Short: "Ingest and query source documents",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Ingest local documents (PDF, DOCX, EPUB, ...) into a workspace as source
documents and query their text.

This is a separate retrieval surface from 'pharos search': search indexes only
lessons, records, references, and quizzes, so source-document text is never
returned by 'pharos search'. The teach skill drives this feature to ground
lessons on passages retrieved here.

Examples:
  pharos document extract ~/book.pdf --title "My Book"
  pharos document extract ~/book.pdf --chapter 3 --out /tmp/ch3.txt
  pharos document query "mitochondria photosynthesis"`,
}

func init() {
	rootCmd.AddCommand(documentCmd)
}
