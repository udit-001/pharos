package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/udit/learn-tool/internal/db"
)

// listSpec describes how to list one entity type in a workspace. It captures
// everything that varies between `lesson list`, `record list`, and
// `reference list`; the shared skeleton (resolve workspace, fetch, JSON/empty
// handling, formatTable, print) lives in runList.
//
// This is the seam extracted in LEARN-14: the three commands were structurally
// identical ~75-line files. Now each is a thin spec + runList call.
type listSpec[T any] struct {
	fetch      func(ws *db.WorkspaceStore, search string) ([]T, error)
	errLabel   string // "failed to list lessons" — wrapped around the fetch error
	emptyMsg   string // "No lessons yet."
	createHint string // "learn lesson create \"Title\" --workspace %q" — %q filled with ws.Name
	headers    []string
	buildRow   func(item T) []string
}

// runList is the shared skeleton for the three list commands: resolve the
// workspace, fetch (with optional --search), then either emit JSON or build a
// table. Entity-specific behaviour comes from the spec.
func runList[T any](cmd *cobra.Command, spec listSpec[T]) error {
	s := mustStore(cmd)
	wsName, _ := cmd.Flags().GetString("workspace")
	wsStore, err := resolveWorkspace(s, wsName)
	if err != nil {
		return err
	}
	ws := wsStore.Workspace()

	search, _ := cmd.Flags().GetString("search")
	items, err := spec.fetch(wsStore, search)
	if err != nil {
		return formatError(spec.errLabel, err)
	}

	if jsonOut {
		if items == nil {
			items = []T{}
		}
		printJSON(items)
		return nil
	}

	fmt.Println()
	fmt.Printf("  Workspace: %s\n\n", ws.Name)

	if len(items) == 0 {
		fmt.Printf("  %s\n", spec.emptyMsg)
		fmt.Printf("  Use '%s' to add one.\n", fmt.Sprintf(spec.createHint, ws.Name))
		fmt.Println()
		return nil
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, spec.buildRow(item))
	}
	fmt.Println(formatTable(spec.headers, rows))
	fmt.Println()
	return nil
}
