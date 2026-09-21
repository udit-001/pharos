// Package vendor owns the pinned global vendor cache: a manifest of
// third-party libraries (name, version, CDN path, sha256) baked into the
// binary, a per-OS cache directory they are fetched into, and the serving
// seam the server's frame injection reads through.
//
// The pin is the compat contract: companion scripts and theme glue in the
// binary are authored for exactly these versions. vega's co-lib majors are
// hardcoded (see vegaDownloads in the old asset registry for the history).
package vendor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry pins one downloaded file: the CDN-relative path (prefixed with the
// CDN base at fetch time, so tests can swap in an httptest server) and the
// sha256 the cached bytes must match.
type Entry struct {
	Lib     string // "mermaid", "katex", … — also the /vendor/ URL segment
	Version string // full pinned version, e.g. "11.17.2"
	File    string // path within the lib's cache dir, e.g. "contrib/auto-render.min.js"
	Path    string // CDN-relative path, e.g. "npm/mermaid@11.17.2/dist/mermaid.min.js"
	SHA256  string // hex sha256 of the pinned bytes
}

// Manifest pins every downloaded file. Filled in below.
var Manifest []Entry

// SyncOptions carries the sync knobs. Dir empty → DefaultDir(); BaseURL
// empty → the real CDN; Budget zero → 10s; Out nil → silent (log only).
type SyncOptions struct {
	Dir     string
	BaseURL string
	Budget  time.Duration
	Out     io.Writer
}

// SyncResult reports what a Sync pass did, for logs and the CLI.
type SyncResult struct {
	Fetched  int // files downloaded and hash-verified this pass
	Verified int // files already in cache with a matching hash
	Skipped  int // files given up on (offline, budget, pin mismatch)
}

// DefaultDir resolves the per-OS vendor cache directory:
// %LocalAppData%\pharos\vendor on Windows, XDG cache dir on Linux,
// ~/Library/Caches on macOS (all via os.UserCacheDir).
func DefaultDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "pharos", "vendor")
}

// Sync fills and verifies the cache. For every manifest entry: a cache file
// with a matching hash is left alone (idempotent, near-instant); a missing
// file is fetched once and verified; a hash mismatch is refetched once and
// given up on (skip + log) if the pin still disagrees. Fetch errors (offline)
// and an exhausted budget degrade to skip + log — start must succeed
// regardless. Progress lines go to opts.Out ("[vendor] fetching x@v … ok").
func Sync(opts SyncOptions) (SyncResult, error) {
	dir := opts.Dir
	if dir == "" {
		dir = DefaultDir()
	}
	budget := opts.Budget
	if budget == 0 {
		budget = 10 * time.Second
	}
	deadline := time.Now().Add(budget)
	base := opts.BaseURL
	if base == "" {
		base = "https://cdn.jsdelivr.net"
	}
	client := &http.Client{Timeout: 30 * time.Second}

	var res SyncResult
	if len(Manifest) == 0 {
		return res, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return res, fmt.Errorf("vendor cache dir: %w", err)
	}

	byLib := groupByLib(Manifest)
	for _, lib := range byLib {
		libRes := syncLib(lib, dir, base, client, deadline)
		if libRes.line != "" && opts.Out != nil {
			fmt.Fprintln(opts.Out, libRes.line)
		}
		res.Fetched += libRes.fetched
		res.Verified += libRes.verified
		res.Skipped += libRes.skipped
	}
	return res, nil
}

// libGroup is one lib's manifest entries plus the sync outcome to report.
type libGroup struct {
	lib, version string
	entries      []Entry
}

func groupByLib(entries []Entry) []libGroup {
	var groups []libGroup
	index := map[string]int{}
	for _, e := range entries {
		i, ok := index[e.Lib]
		if !ok {
			groups = append(groups, libGroup{lib: e.Lib, version: e.Version})
			i = len(groups) - 1
			index[e.Lib] = i
		}
		groups[i].entries = append(groups[i].entries, e)
	}
	return groups
}

// libOutcome carries per-lib results and the progress line to print.
type libOutcome struct {
	fetched, verified, skipped int
	line                       string
}

func syncLib(g libGroup, dir, base string, client *http.Client, deadline time.Time) libOutcome {
	var out libOutcome
	label := g.lib + "@" + g.version
	failures := 0
	for _, e := range g.entries {
		target := filepath.Join(dir, e.Lib+"@"+e.Version, filepath.FromSlash(e.File))
		if ok, _ := hashMatches(target, e.SHA256); ok {
			out.verified++
			continue
		}
		if !time.Now().Before(deadline) {
			out.skipped++
			failures++
			log.Printf("[vendor] %s: budget exhausted before fetching %s", label, e.File)
			continue
		}
		if err := fetchAndStore(base, e, target, client); err != nil {
			out.skipped++
			failures++
			log.Printf("[vendor] %s: %v", label, err)
			continue
		}
		out.fetched++
	}
	switch {
	case failures == 0:
		out.line = fmt.Sprintf("[vendor] fetching %s … ok", label)
	case out.fetched > 0:
		out.line = fmt.Sprintf("[vendor] fetching %s … ok (%d/%d)", label, out.fetched+out.verified, len(g.entries))
	default:
		out.line = fmt.Sprintf("[vendor] fetching %s … skipped (offline or pin mismatch; see log)", label)
	}
	return out
}

// hashMatches reports whether the file at path exists and its sha256 equals
// want (hex).
func hashMatches(path, want string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) == want, nil
}

// fetchAndStore downloads one entry, verifies it against the pin, and writes
// it to target (temp file + rename). A pin mismatch aborts the write — bad
// bytes must never enter the cache.
func fetchAndStore(base string, e Entry, target string, client *http.Client) error {
	url := base + "/" + e.Path
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", e.File, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", e.File, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: server returned %s", e.File, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s: %w", e.File, err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != e.SHA256 {
		return fmt.Errorf("fetch %s: sha256 mismatch (pin drift or tampered CDN response); keeping cache as-is", e.File)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("store %s: %w", e.File, err)
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("store %s: %w", e.File, err)
	}
	return os.Rename(tmp, target)
}

// ── Serving seam ────────────────────────────────────────────────────────
//
// The server's frame injection and /vendor route read vendored bytes
// exclusively through Open/URL/Available — one small seam, two sources
// (the pinned cache on disk, companions embedded in the binary).

// dirOverride is set by tests to point the serving seam at a temp cache.
var dirOverride string

// SetDir overrides the cache directory used by the serving seam (tests).
// Pass "" to restore the default.
func SetDir(dir string) { dirOverride = dir }

// Dir returns the cache directory the serving seam reads from.
func Dir() string {
	if dirOverride != "" {
		return dirOverride
	}
	return DefaultDir()
}

// entry lookup: (lib, file) → manifest entry, built once.
var entryOnce sync.Once
var entryIndex map[string]Entry

func lookup(lib, file string) (Entry, bool) {
	entryOnce.Do(func() {
		entryIndex = make(map[string]Entry, len(Manifest))
		for _, e := range Manifest {
			entryIndex[e.Lib+"/"+e.File] = e
		}
	})
	e, ok := entryIndex[lib+"/"+file]
	return e, ok
}

// Available reports whether the (lib, file) pair can be served right now:
// an embedded companion, or a pinned file present in the cache.
func Available(lib, file string) bool {
	if _, ok := Companions[lib][file]; ok {
		return true
	}
	e, ok := lookup(lib, file)
	if !ok {
		return false
	}
	_, err := os.Stat(cachePath(e))
	return err == nil
}

// Open returns the vendored bytes for (lib, file) — cache first, then
// embedded companions. ok=false when the pair is unknown or not cached.
func Open(lib, file string) ([]byte, bool) {
	if data, ok := Companions[lib][file]; ok {
		return data, true
	}
	e, ok := lookup(lib, file)
	if !ok {
		return nil, false
	}
	data, err := os.ReadFile(cachePath(e))
	if err != nil {
		return nil, false
	}
	return data, true
}

// URL returns the versioned /vendor URL for (lib, file), or "" when the
// pair is unknown or its cache file is absent (callers drop empty URLs —
// the missing-lib degrade). The version query is the pinned sha256 (cache
// files) or the embedded bytes' sha256 (companions), so the URL changes
// exactly when the served bytes change.
func URL(lib, file string) string {
	if data, ok := Companions[lib][file]; ok {
		return "/vendor/" + lib + "/" + file + "?v=" + assetVersion(data)
	}
	e, ok := lookup(lib, file)
	if !ok {
		return ""
	}
	if _, err := os.Stat(cachePath(e)); err != nil {
		return ""
	}
	return "/vendor/" + lib + "/" + file + "?v=" + e.SHA256[:12]
}

// cachePath resolves an entry to its cache file path.
func cachePath(e Entry) string {
	return filepath.Join(Dir(), e.Lib+"@"+e.Version, filepath.FromSlash(e.File))
}

// assetVersion is the 12-hex-char content version used in URLs.
func assetVersion(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}
