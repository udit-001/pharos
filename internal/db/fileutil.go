package db

import (
	"os"
	"path/filepath"
)

// writeToFile writes content to the given path, creating parent directories
// as needed. Used by the deep create/revise methods.
func writeToFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// readDirNames returns the names of files (not directories) in the given
// directory. Returns an empty slice if the directory doesn't exist.
func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
