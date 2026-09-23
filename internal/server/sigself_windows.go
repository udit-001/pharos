//go:build windows

package server

import "errors"

// signalSelfInterrupt has no windows equivalent: Go's os.Process.Signal
// cannot deliver a console interrupt to the current process from a test.
// The serve-lifetime test is skipped on windows (it is opt-in anyway).
func signalSelfInterrupt() error {
	return errors.New("signalSelfInterrupt: not supported on windows")
}
