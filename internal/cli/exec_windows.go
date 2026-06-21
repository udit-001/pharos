//go:build windows

package cli

import "os/exec"

func execCommand(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	// On Windows, ensure proper process group handling
	return c
}
