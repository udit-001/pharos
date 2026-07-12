package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var navCmd = &cobra.Command{
	Use:   "nav <url>",
	Short: "Navigate the dashboard to a URL",
	Long: `Navigate the open dashboard browser tab to the given URL.

Broadcasts a "navigate" event to all dashboard subscribers. The agent
constructs the URL from the workspace's URL scheme (e.g.
/workspace/my-ws/lesson/1). If no dashboard tab is open, the event is
delivered to zero subscribers and the command exits with code 2.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		port, ok := runningServerPort()
		if !ok {
			fmt.Fprintln(os.Stderr, "No dashboard server running. Start it with: pharos start")
			os.Exit(1)
		}

		body, _ := json.Marshal(map[string]string{
			"type": "navigate",
			"url":  url,
		})
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Post(
			"http://127.0.0.1:"+strconv.Itoa(port)+"/api/notify",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to reach dashboard: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result struct {
			Delivered int `json:"delivered"`
		}
		_ = json.Unmarshal(respBody, &result)

		if result.Delivered == 0 {
			fmt.Fprintf(os.Stderr, "No dashboard tab open. Open http://127.0.0.1:%d in a browser.\n", port)
			os.Exit(2)
		}
		fmt.Printf("Navigated to %s\n", url)
	},
}

func init() {
	rootCmd.AddCommand(navCmd)
}
