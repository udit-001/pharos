package db

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// safeAssetPath resolves filename to an absolute path inside the workspace's
// assets/ directory, rejecting traversal (.., absolute paths) and the
// directory itself. Subdirectories (e.g. fonts/inter-latin.woff2) are allowed.
// Unexported — tested through WriteAsset/DeleteAsset, not exposed as part
// of the interface.
func (w *WorkspaceStore) safeAssetPath(filename string) (string, error) {
	return w.Layout().SafeJoin("assets", filename)
}

// AssetPath returns the absolute, traversal-safe path for an asset filename,
// or an error if it escapes the assets/ directory. Exported so the HTTP layer
// can serve asset files — including subdirectories (fonts/, contrib/) —
// without re-implementing the traversal check that safeAssetPath enforces.
func (w *WorkspaceStore) AssetPath(filename string) (string, error) {
	return w.safeAssetPath(filename)
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

// fileExists reports whether path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
