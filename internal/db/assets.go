package db

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AssetSpec describes a managed asset — vendored (a downloaded lib plus
// embedded companions) or seeded (embedded-only). The CLI builds specs from
// the knownAssets registry; the store installs them without knowing the
// source, so the install policy is tested behind one interface.
type AssetSpec struct {
	Source         string            // "seeded" | "vendored"
	Filename       string            // downloaded lib filename; "" for embedded-only
	DefaultVersion string            // pinned; "" for embedded-only
	URLTemplate    string            // {{VERSION}} placeholder; "" for embedded-only
	Files          map[string][]byte // embedded content (companions + seeded files)
}

// AssetResult reports what InstallAsset did — observable through the
// interface, so tests assert on outcomes, not internal state.
type AssetResult struct {
	LibWritten   bool
	FilesWritten []string
	Skipped      []string
}

// safeAssetPath resolves filename to an absolute path inside the workspace's
// assets/ directory, rejecting traversal (.., absolute paths) and the
// directory itself. Subdirectories (e.g. fonts/inter-latin.woff2) are allowed.
// Unexported — tested through WriteAsset/DeleteAsset/InstallAsset, not exposed
// as part of the interface.
func (w *WorkspaceStore) safeAssetPath(filename string) (string, error) {
	dir := filepath.Join(w.ws.Path, "assets")
	target := filepath.Join(dir, filename)
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(filename) {
		return "", fmt.Errorf("invalid asset path %q", filename)
	}
	return target, nil
}

// WriteAsset writes data to the workspace's assets/ directory, creating
// parent directories as needed. Sanitized via safeAssetPath.
func (w *WorkspaceStore) WriteAsset(filename string, data []byte) error {
	path, err := w.safeAssetPath(filename)
	if err != nil {
		return err
	}
	return writeBytesToFile(path, data)
}

// DeleteAsset removes a file from the workspace's assets/ directory. Returns
// a clear "not found" error if the file is absent.
func (w *WorkspaceStore) DeleteAsset(filename string) error {
	path, err := w.safeAssetPath(filename)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("asset %q not found", filename)
	}
	return os.Remove(path)
}

// ListAssets returns the files in the workspace's assets/ directory,
// including subdirectories (e.g. fonts/inter-latin.woff2), as slash-relative
// paths. Sorted for stable output.
func (w *WorkspaceStore) ListAssets() ([]string, error) {
	dir := filepath.Join(w.ws.Path, "assets")
	var names []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// InstallAsset writes a vendored or seeded asset. libData is the fetched lib
// bytes — nil for embedded-only assets, or when the CLI chose not to fetch
// (e.g. the lib is already present on an idempotent add). force=false skips
// files already present (idempotent install); force=true overwrites
// everything (sync to binary). The lib is written only when libData is
// non-nil — the CLI owns the fetch decision; the store owns the per-file
// skip/overwrite policy.
//
// Unlike the old asset_add short-circuit, Files are iterated per-file, so a
// missing companion is restored even when the lib is already present.
func (w *WorkspaceStore) InstallAsset(spec AssetSpec, libData []byte, force bool) (AssetResult, error) {
	var res AssetResult

	if spec.Filename != "" && libData != nil {
		path, err := w.safeAssetPath(spec.Filename)
		if err != nil {
			return res, err
		}
		if force || !fileExists(path) {
			if err := writeBytesToFile(path, libData); err != nil {
				return res, fmt.Errorf("write %s: %w", spec.Filename, err)
			}
			res.LibWritten = true
		} else {
			res.Skipped = append(res.Skipped, spec.Filename)
		}
	}

	for fname, content := range spec.Files {
		path, err := w.safeAssetPath(fname)
		if err != nil {
			return res, err
		}
		if force || !fileExists(path) {
			if err := writeBytesToFile(path, content); err != nil {
				return res, fmt.Errorf("write %s: %w", fname, err)
			}
			res.FilesWritten = append(res.FilesWritten, fname)
		} else {
			res.Skipped = append(res.Skipped, fname)
		}
	}

	sort.Strings(res.FilesWritten)
	sort.Strings(res.Skipped)
	return res, nil
}

// fileExists reports whether path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
