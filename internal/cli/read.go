package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/db"
)

type readSpec[T any] struct {
	fetch    func(ws *db.WorkspaceStore) ([]T, error)
	errLabel string
	findItem func(items []T, key string) (*T, error)
	keyName  string
	jsonOut  func(item T, ws db.Workspace, wsStore *db.WorkspaceStore) map[string]any
	plainOut func(item T, ws db.Workspace, wsStore *db.WorkspaceStore)
	bodyPath func(wsStore *db.WorkspaceStore, item T) string
}

func runRead[T any](cmd *cobra.Command, spec readSpec[T], key string) error {
	s := mustStore(cmd)
	wsName, _ := cmd.Flags().GetString("workspace")
	wsStore, err := resolveWorkspace(s, wsName)
	if err != nil {
		return err
	}
	ws := wsStore.Workspace()

	items, err := spec.fetch(wsStore)
	if err != nil {
		return formatError(spec.errLabel, err)
	}

	item, err := spec.findItem(items, key)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("%s %q not found", spec.keyName, key)
	}

	metaOnly, _ := cmd.Flags().GetBool("meta-only")

	if jsonOut {
		result := spec.jsonOut(*item, ws, wsStore)
		result["workspace"] = ws.Name
		if !metaOnly {
			data, err := os.ReadFile(spec.bodyPath(wsStore, *item))
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			result["body"] = string(data)
		}
		printJSON(result)
		return nil
	}

	fmt.Println()
	spec.plainOut(*item, ws, wsStore)
	fmt.Println()

	if !metaOnly {
		data, err := os.ReadFile(spec.bodyPath(wsStore, *item))
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		fmt.Println(string(data))
	}
	return nil
}
