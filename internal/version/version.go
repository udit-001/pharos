package version

import "runtime/debug"

// Version is the current version of the Pharos CLI.
// Overridden at build time via ldflags, or detected from Go module info.
var Version = "0.2.0"

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}
