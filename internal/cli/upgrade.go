package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/udit-001/pharos/internal/version"
)

var upgradeForce bool

func init() {
	upgradeCmd.Flags().BoolVarP(&upgradeForce, "force", "f", false, "Reinstall even if already up to date")
	upgradeCmd.Flags().Bool("no-skills", false, "Skip the skill upgrade prompt")
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade pharos to the latest version via 'go install'",
	Long: `Upgrade pharos to the latest release by running:
  go install github.com/udit-001/pharos/cmd/pharos@latest

This compiles from source — no binary download.

If the server is running, it will be stopped before the upgrade and
restarted afterwards.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		goPath, err := exec.LookPath("go")
		if err != nil {
			fmt.Println()
			fmt.Println("  Go is not installed on your PATH.")
			fmt.Println("  Install manually with:")
			fmt.Println("    go install github.com/udit-001/pharos/cmd/pharos@latest")
			fmt.Println()
			return nil
		}

		fmt.Println()
		fmt.Printf("  Checking for upgrades...\n")

		latest, err := latestVersionFromProxy(goPath)
		if err != nil {
			return err
		}
		fmt.Printf("  Latest version: %s\n", latest)

		current := strings.TrimPrefix(version.Version, "v")
		tag := strings.TrimPrefix(latest, "v")
		if !upgradeForce && current != "" && current != "dev" && semverCompare(current, tag) >= 0 {
			fmt.Printf("  Already up to date (v%s)\n", current)
			fmt.Println()
			return nil
		}

		var serverInfo *pidInfo
		if info, err := readPidFile(); err == nil && isServerRunning(info.Port) {
			serverInfo = info
			fmt.Printf("  Stopping server (PID %d)...\n", info.PID)
			proc, err := os.FindProcess(info.PID)
			if err != nil {
				fmt.Printf("  Warning: could not find server process: %v\n", err)
				fmt.Printf("  Please stop it manually and re-run upgrade.\n")
				fmt.Println()
				return nil
			}
			if err := proc.Signal(syscall.SIGINT); err != nil {
				fmt.Printf("  Warning: could not stop server: %v\n", err)
				fmt.Printf("  Please stop it manually and re-run upgrade.\n")
				fmt.Println()
				return nil
			}
			cleanupPidFile()
			fmt.Printf("  Server stopped.\n")
		}

		module := fmt.Sprintf("github.com/udit-001/pharos/cmd/pharos@%s", latest)
		fmt.Printf("  Running: go install %s\n", module)

		c := exec.Command(goPath, "install", module)
		output, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go install failed: %w\n%s", err, string(output))
		}

		fmt.Printf("  Upgraded to %s\n", latest)

		if serverInfo != nil {
			fmt.Printf("  Restarting server...\n")
			rc, err := startDaemon(serverInfo.Port)
			if err != nil {
				fmt.Printf("  Warning: could not restart server: %v\n", err)
				fmt.Printf("  Run 'pharos start' manually.\n")
			} else {
				fmt.Printf("  Server restarted in background (PID: %d)\n", rc.Process.Pid)
				fmt.Printf("  http://127.0.0.1:%d\n", serverInfo.Port)
			}
		} else {
			fmt.Printf("  Run 'pharos start' to launch the server.\n")
		}

		noSkills, _ := cmd.Flags().GetBool("no-skills")
		if !noSkills {
			offerSkillUpgrade()
		}

		fmt.Println()
		return nil
	},
}

func latestVersionFromProxy(goPath string) (string, error) {
	cmd := exec.Command(goPath, "list", "-m", "-versions", "github.com/udit-001/pharos")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("check for versions failed: %w", err)
	}
	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return "", fmt.Errorf("no versions found — push a git tag first")
	}
	return parts[len(parts)-1], nil
}

func semverCompare(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	min := len(pa)
	if len(pb) < min {
		min = len(pb)
	}
	for i := 0; i < min; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	if len(pa) < len(pb) {
		return -1
	}
	if len(pa) > len(pb) {
		return 1
	}
	return 0
}

func parseSemver(v string) []int {
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nums
		}
		nums = append(nums, n)
	}
	return nums
}
