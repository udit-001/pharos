package db

import "fmt"

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
