package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// goos is overridable in tests so the Windows-specific branch can be
// exercised on non-Windows hosts (where cygpath does not exist).
var goos = runtime.GOOS

// cygpathToWindows converts an MSYS/Git Bash path to a native Windows path by
// shelling out to `cygpath -w`, which understands /tmp, /etc, symlinks, and
// custom mount points — everything MSYS's own (unreliable) argument mangling
// can miss. Overridable in tests.
var cygpathToWindows = defaultCygpathToWindows

func defaultCygpathToWindows(path string) (string, error) {
	out, err := exec.Command("cygpath", "-w", path).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// looksLikeMSYSPath reports whether p is a Unix-style absolute path that a
// native Windows binary cannot resolve directly (e.g. /c/..., /tmp/...).
// UNC paths (//server/share) are excluded.
func looksLikeMSYSPath(p string) bool {
	return strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//")
}

// readBodyFile reads the --body-file path and rejects empty content.
//
// On Windows, a Unix-style path (e.g. /tmp/x.html from Git Bash) is first
// converted to a native Windows path via `cygpath -w`. MSYS's own argument
// conversion is unreliable: when it skips, Go resolves the path to an empty
// or unintended file and the command silently reported "✓ created" while
// writing nothing. An empty (or whitespace-only) result is rejected loudly
// regardless, so a conversion that still misses the file fails instead of
// writing nothing.
func readBodyFile(path string) ([]byte, error) {
	readPath := path
	converted := false
	if goos == "windows" && looksLikeMSYSPath(path) {
		if c, err := cygpathToWindows(path); err == nil {
			readPath = c
			converted = true
		}
	}

	data, err := os.ReadFile(readPath)
	if err != nil {
		if goos == "windows" && looksLikeMSYSPath(path) && !converted {
			return nil, fmt.Errorf("read body file %q: %w\n  This looks like a Git Bash/MSYS path; pass a Windows path or run under Git Bash so `cygpath` can convert it", path, err)
		}
		return nil, fmt.Errorf("read body file %q: %w", path, err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		msg := fmt.Sprintf("--body-file %q is empty (0 bytes of content)", path)
		if goos == "windows" && looksLikeMSYSPath(path) && !converted {
			msg += "\n  Git Bash /tmp/... paths may resolve to an unintended file; pass a Windows path"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return data, nil
}
