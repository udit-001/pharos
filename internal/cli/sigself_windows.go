//go:build windows

package cli

import "errors"

// signalSelfTerminate has no windows equivalent: Go's os.Process.Signal
// cannot deliver a console interrupt to the current process from a test.
// Tests using it skip on windows.
func signalSelfTerminate() error {
	return errors.New("signalSelfTerminate: not supported on windows")
}
