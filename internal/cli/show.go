package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type showSpec struct {
	urlPath func(wsName string) string
	label   string
}

func runShow(cmd *cobra.Command, spec showSpec) error {
	s := mustStore(cmd)
	wsName, _ := cmd.Flags().GetString("workspace")

	wsStore, err := resolveWorkspace(s, wsName)
	if err != nil {
		return err
	}
	ws := wsStore.Workspace()

	url := "http://127.0.0.1:9090" + spec.urlPath(ws.Name)

	if jsonOut {
		printJSON(map[string]string{"url": url})
		return nil
	}

	fmt.Println()
	fmt.Printf("  View %s at: %s\n", spec.label, url)
	fmt.Printf("  Dashboard must be running (use 'pharos start').\n")
	fmt.Println()
	return nil
}
