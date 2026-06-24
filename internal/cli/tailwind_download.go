package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const tailwindReleaseURL = "https://github.com/tailwindlabs/tailwindcss/releases/latest/download"

var tailwindFlags struct {
	force   bool
	version string
}

var tailwindDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download the Tailwind CSS standalone CLI binary",
	Long: `Download the Tailwind CSS v4 standalone CLI binary for your platform.

The binary is placed at .bin/tailwindcss in the project root.

Supports linux (x64, arm64) and macOS (x64, arm64).
Use --version to download a specific version instead of latest.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTailwindDownload()
	},
}

func init() {
	tailwindCmd.AddCommand(tailwindDownloadCmd)
	tailwindDownloadCmd.Flags().BoolVarP(&tailwindFlags.force, "force", "f", false, "Re-download even if already present")
	tailwindDownloadCmd.Flags().StringVar(&tailwindFlags.version, "version", "", "Specific version (e.g. v4.3.1) instead of latest")
}

func runTailwindDownload() error {
	root := mustProjectRoot()
	binDir := filepath.Join(root, ".bin")
	binPath := filepath.Join(binDir, "tailwindcss")

	if _, err := os.Stat(binPath); err == nil && !tailwindFlags.force {
		fmt.Println()
		fmt.Printf("  tailwindcss already exists at %s\n", binPath)
		fmt.Println("  Use --force to re-download.")
		fmt.Println()
		return nil
	}

	assetName, err := assetForPlatform()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", binDir, err)
	}

	// Download to a temp path so a failed/corrupt download never overwrites an existing good binary.
	tmpPath := binPath + ".tmp"
	defer os.Remove(tmpPath)

	fmt.Println()
	fmt.Printf("  Downloading %s...\n", assetName)

	binURL := downloadURL(assetName, tailwindFlags.version)
	if err := downloadFile(tmpPath, binURL); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  Verifying checksum...\n")

	shaURL := downloadURL("sha256sums.txt", tailwindFlags.version)
	expected, err := fetchChecksum(shaURL, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	if err := verifyChecksum(tmpPath, expected); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, binPath); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if err := os.Chmod(binPath, 0o755); err != nil {
		return fmt.Errorf("set executable: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Tailwind CSS CLI ready")
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    make css     — Build CSS once")
	fmt.Println("    pharos dev   — Start dev server with auto-rebuild")
	fmt.Println()

	return nil
}

// downloadURL returns the GitHub release download URL for the given asset.
func downloadURL(asset, version string) string {
	if version != "" {
		v := strings.TrimPrefix(version, "v")
		return fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/v%s/%s", v, asset)
	}
	return fmt.Sprintf("%s/%s", tailwindReleaseURL, asset)
}

// downloadFile streams a URL to a file path.
func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s (HTTP %d)", url, resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	fmt.Printf("  Downloaded %.1f MB\n", float64(written)/(1<<20))
	return nil
}

// fetchChecksum downloads sha256sums.txt and returns the hex-encoded hash
// for the given asset name.
func fetchChecksum(url, assetName string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// sha256sums.txt has lines like: <hex>  ./<filename>
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := strings.TrimPrefix(strings.TrimSpace(parts[1]), "./")
			if name == assetName {
				return parts[0], nil
			}
		}
	}

	return "", fmt.Errorf("checksum for %s not found in response", assetName)
}

// verifyChecksum computes the SHA256 of a file and compares it to the expected
// hex-encoded hash.
func verifyChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open for verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("checksum read: %w", err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("checksum mismatch\n  expected: %s\n  got:      %s", expected, got)
	}

	fmt.Printf("  SHA256: %s\n", got)
	return nil
}

// assetForPlatform returns the tailwindcss asset name for the current OS/arch.
func assetForPlatform() (string, error) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "tailwindcss-linux-x64", nil
		case "arm64":
			return "tailwindcss-linux-arm64", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "tailwindcss-macos-x64", nil
		case "arm64":
			return "tailwindcss-macos-arm64", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "tailwindcss-windows-x64.exe", nil
		default:
			return "", fmt.Errorf("unsupported architecture: %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
