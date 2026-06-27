package db

import (
	"fmt"
	"strings"
)

// RowScanner is the minimal rows interface needed by scanRows. It matches
// both *sql.Rows and *sqlx.Rows.
type RowScanner interface {
	Next() bool
	Scan(...any) error
	Close() error
	Err() error
}

// scanRows iterates rows, calling scanOne for each, and returns the collected
// slice. This is the single loop that the four entity-specific scanXs helpers
// previously duplicated (LEARN-15). Each entity keeps its own scanX(row) —
// the per-entity column layout genuinely differs — but delegates the loop
// here.
func scanRows[T any](rows RowScanner, label string, scanOne func(row interface{ Scan(...any) error }) (T, error)) ([]T, error) {
	var items []T
	for rows.Next() {
		item, err := scanOne(rows)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", label, err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// indexItems processes a batch of items: for each, calls process (which reads
// the file, extracts text, and updates the DB row). Errors are collected
// per-item (processing continues with the rest); the aggregate error is
// returned. Shared by IndexLessons/IndexRefs/IndexRecords — the per-entity
// differences (path, extract fn, table, identifier) live in the process and
// ident closures.
func indexItems[T any](items []T, process func(T) error, ident func(T) string, label string) (int, error) {
	var updated int
	var errs []string
	for _, item := range items {
		if err := process(item); err != nil {
			errs = append(errs, fmt.Sprintf("%s %s: %v", label, ident(item), err))
			continue
		}
		updated++
	}
	if len(errs) > 0 {
		return updated, fmt.Errorf("index %s: %d error(s): %s", label, len(errs), strings.Join(errs, "; "))
	}
	return updated, nil
}
