package version

import (
	"runtime/debug"
	"strings"
)

// Version is the current version of the Pharos CLI.
// Overridden at build time via ldflags, or detected from Go module info.
var Version = "0.3.0"

// Commit is the git commit hash the binary was built from.
// Overridden at build time via ldflags, or detected from VCS build info.
var Commit = ""

// Date is the build date (RFC3339).
// Overridden at build time via ldflags, or detected from VCS build info.
var Date = ""

// DisplayVersion returns a user-friendly version string.
// For clean semver tags (e.g. 0.3.0) it returns "v0.3.0".
// For Go pseudo-versions (e.g. 0.8.2-0.20260704214233-5f2fb0f3ebdf+dirty)
// it collapses to "v0.8.2-dev".
func DisplayVersion() string {
	if !strings.Contains(Version, "-") {
		return "v" + Version
	}
	parts := strings.SplitN(Version, "-", 3)
	if len(parts) < 3 {
		return "v" + Version
	}
	return "v" + parts[0] + "-dev"
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = strings.TrimPrefix(info.Main.Version, "v")
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
