//go:build !windows && !darwin && !linux

package autostart

import (
	"fmt"
	"runtime"
)

// entryPath fails on unsupported platforms. The shipped binaries target
// linux/darwin/windows (see .goreleaser.yaml); everything else gets a clear
// error from autostart.New rather than a panic.
func entryPath() (string, error) {
	return "", fmt.Errorf("autostart is not supported on %s", runtime.GOOS)
}
