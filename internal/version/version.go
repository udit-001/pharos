package version

import "runtime/debug"

// Version is the current version of the Pharos CLI.
// Overridden at build time via ldflags, or detected from Go module info.
var Version = "0.3.0"

// Commit is the git commit hash the binary was built from.
// Overridden at build time via ldflags, or detected from VCS build info.
var Commit = ""

// Date is the build date (RFC3339).
// Overridden at build time via ldflags, or detected from VCS build info.
var Date = ""

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if Commit == "" {
				Commit = setting.Value
			}
		case "vcs.time":
			if Date == "" {
				Date = setting.Value
			}
		}
	}
}
