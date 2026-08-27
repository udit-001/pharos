package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/udit-001/pharos/internal/extract"
)

// backfillSlugs assigns slugs to lessons and learning_records, rewrites
// seq-based in-lesson links, and adds uniqueness constraints.
// Idempotent — safe to call multiple times.
func backfillSlugs(db *sql.DB) error {
	// Check if slug column exists (pre-migration databases).
	var hasCol int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('lessons') WHERE name = 'slug'`).Scan(&hasCol); err != nil {
		return fmt.Errorf("check slug column: %w", err)
	}
	if hasCol == 0 {
		return nil // migration hasn't run yet
	}

	// Build workspace ID → path mapping for file operations.
	wsPathMap, err := buildWorkspacePathMap(db)
	if err != nil {
		return fmt.Errorf("build workspace path map: %w", err)
	}

	// Backfill lessons.
	if err := backfillLessonSlugs(db, wsPathMap); err != nil {
		return fmt.Errorf("backfill lesson slugs: %w", err)
	}

	// Backfill learning records.
	if err := backfillRecordSlugs(db, wsPathMap); err != nil {
		return fmt.Errorf("backfill record slugs: %w", err)
	}

	// Rewrite seq-based links in lesson HTML files (always run — not gated on empty slugs).
	if err := rewriteLessonLinks(db, wsPathMap); err != nil {
		return fmt.Errorf("rewrite lesson links: %w", err)
	}

	// Re-extract body_text for FTS index.
	if err := reindexLessonBodyText(db, wsPathMap); err != nil {
		return fmt.Errorf("reindex lesson body_text: %w", err)
	}

	// Add uniqueness constraints (IF NOT EXISTS — idempotent).
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_lessons_ws_slug ON lessons(workspace_id, slug)`); err != nil {
		return fmt.Errorf("create lessons slug index: %w", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_records_ws_slug ON learning_records(workspace_id, slug)`); err != nil {
		return fmt.Errorf("create records slug index: %w", err)
	}

	// Backfill last-viewed slugs from seq columns (pre-migration databases).
	if err := backfillLastViewedSlugs(db); err != nil {
		return fmt.Errorf("backfill last viewed slugs: %w", err)
	}

	return nil
}

// workspacePath is a workspace ID → filesystem path mapping.
type workspacePath struct {
	ID   int64
	Path string
}

func buildWorkspacePathMap(db *sql.DB) (map[int64]string, error) {
	rows, err := db.Query(`SELECT id, path FROM workspaces`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[int64]string{}
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		m[id] = path
	}
	return m, rows.Err()
}

func backfillLessonSlugs(db *sql.DB, wsPaths map[int64]string) error {
	rows, err := db.Query(`SELECT id, workspace_id, title, filename, sequence_number FROM lessons WHERE slug = '' OR slug IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type lesson struct {
		ID, WSID int64
		Title    string
		Filename string
		Seq      int
	}
	var lessons []lesson
	for rows.Next() {
		var l lesson
		if err := rows.Scan(&l.ID, &l.WSID, &l.Title, &l.Filename, &l.Seq); err != nil {
			return err
		}
		lessons = append(lessons, l)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Assign slugs, tracking per-workspace collisions.
	slugSeen := map[int64]map[string]int64{} // wsID → slug → lessonID
	for _, l := range lessons {
		slug := Slugify(l.Title)
		if slug == "" {
			slug = "untitled"
		}

		if slugSeen[l.WSID] == nil {
			slugSeen[l.WSID] = map[string]int64{}
		}

		finalSlug := slug
		for i := 2; ; i++ {
			if existingID, collision := slugSeen[l.WSID][finalSlug]; !collision || existingID == l.ID {
				break
			}
			finalSlug = fmt.Sprintf("%s-%d", slug, i)
		}
		slugSeen[l.WSID][finalSlug] = l.ID

		// Update slug in DB. Filename stays as-is — the DB maps slug → filename,
		// so old names like 0001-joins.html work fine via the slug lookup.
		if _, err := db.Exec(`UPDATE lessons SET slug = ? WHERE id = ?`, finalSlug, l.ID); err != nil {
			return fmt.Errorf("update lesson slug %d: %w", l.ID, err)
		}
	}

	return nil
}

func backfillRecordSlugs(db *sql.DB, wsPaths map[int64]string) error {
	rows, err := db.Query(`SELECT id, workspace_id, title, filename FROM learning_records WHERE slug = '' OR slug IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type record struct {
		ID, WSID int64
		Title    string
		Filename string
	}
	var records []record
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.ID, &r.WSID, &r.Title, &r.Filename); err != nil {
			return err
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	slugSeen := map[int64]map[string]int64{}
	for _, r := range records {
		slug := Slugify(r.Title)
		if slug == "" {
			slug = "untitled"
		}

		if slugSeen[r.WSID] == nil {
			slugSeen[r.WSID] = map[string]int64{}
		}

		finalSlug := slug
		for i := 2; ; i++ {
			if existingID, collision := slugSeen[r.WSID][finalSlug]; !collision || existingID == r.ID {
				break
			}
			finalSlug = fmt.Sprintf("%s-%d", slug, i)
		}
		slugSeen[r.WSID][finalSlug] = r.ID

		if _, err := db.Exec(`UPDATE learning_records SET slug = ? WHERE id = ?`, finalSlug, r.ID); err != nil {
			return fmt.Errorf("update record slug %d: %w", r.ID, err)
		}
		// Filename stays as-is — DB maps slug → filename.
	}

	// Re-extract body_text for FTS index.
	if err := reindexRecordBodyText(db, wsPaths); err != nil {
		return fmt.Errorf("reindex record body_text: %w", err)
	}

	return nil
}

// lessonLinkRe matches href="/workspace/{ws}/lesson/{n}" in lesson HTML.
var lessonLinkRe = regexp.MustCompile(`href="/workspace/([^/]+)/lesson/(\d+)"`)

// rewriteLessonLinks replaces seq-based lesson links with slug-based links.
func rewriteLessonLinks(db *sql.DB, wsPaths map[int64]string) error {
	// Build slug lookup: (workspace_name, seq) → slug
	type wsSeqSlug struct {
		WSName string
		Seq    int
		Slug   string
	}
	slugLookup := map[wsSeqSlug]string{}

	rows, err := db.Query(`
		SELECT w.name, l.sequence_number, l.slug
		FROM lessons l JOIN workspaces w ON l.workspace_id = w.id
		WHERE l.slug != ''`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s wsSeqSlug
		if err := rows.Scan(&s.WSName, &s.Seq, &s.Slug); err != nil {
			return err
		}
		slugLookup[s] = s.Slug
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// For each lesson file, find and replace seq-based links.
	for wsID, wsPath := range wsPaths {
		lessonsDir := filepath.Join(wsPath, "lessons")
		entries, err := os.ReadDir(lessonsDir)
		if err != nil {
			continue // workspace might not have lessons dir
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}
			filePath := filepath.Join(lessonsDir, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			changed := false
			result := lessonLinkRe.ReplaceAllStringFunc(string(content), func(match string) string {
				parts := lessonLinkRe.FindStringSubmatch(match)
				if len(parts) < 3 {
					return match
				}
				wsName := parts[1]
				seq, err := strconv.Atoi(parts[2])
				if err != nil {
					return match
				}
				slug, ok := slugLookup[wsSeqSlug{WSName: wsName, Seq: seq}]
				if !ok {
					return match // seq not found, leave as-is
				}
				changed = true
				return fmt.Sprintf(`href="/workspace/%s/lesson/%s"`, wsName, slug)
			})

			if changed {
				if err := os.WriteFile(filePath, []byte(result), 0644); err != nil {
					return fmt.Errorf("write %s: %w", filePath, err)
				}
				// Update body_text for this lesson.
				var lessonID int64
				err := db.QueryRow(`SELECT id FROM lessons WHERE workspace_id = ? AND filename = ?`, wsID, entry.Name()).Scan(&lessonID)
				if err == nil {
					bodyText := extract.FromHTML(result)
					db.Exec(`UPDATE lessons SET body_text = ? WHERE id = ?`, bodyText, lessonID)
				}
			}
		}
	}

	return nil
}

// reindexLessonBodyText re-extracts body_text from lesson HTML files for FTS.
func reindexLessonBodyText(db *sql.DB, wsPaths map[int64]string) error {
	rows, err := db.Query(`
		SELECT l.id, w.path, l.filename
		FROM lessons l JOIN workspaces w ON l.workspace_id = w.id
		WHERE l.body_text = '' OR l.body_text IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var wsPath, filename string
		if err := rows.Scan(&id, &wsPath, &filename); err != nil {
			continue
		}
		if wsPath == "" || filename == "" {
			continue
		}
		filePath := filepath.Join(wsPath, "lessons", filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		bodyText := extract.FromHTML(string(content))
		db.Exec(`UPDATE lessons SET body_text = ? WHERE id = ?`, bodyText, id)
	}
	return rows.Err()
}

// reindexRecordBodyText re-extracts body_text from record markdown files.
func reindexRecordBodyText(db *sql.DB, wsPaths map[int64]string) error {
	rows, err := db.Query(`
		SELECT r.id, w.path, r.filename
		FROM learning_records r JOIN workspaces w ON r.workspace_id = w.id
		WHERE r.body_text = '' OR r.body_text IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var wsPath, filename string
		if err := rows.Scan(&id, &wsPath, &filename); err != nil {
			continue
		}
		if wsPath == "" || filename == "" {
			continue
		}
		filePath := filepath.Join(wsPath, "learning-records", filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		bodyText := extract.FromMarkdown(string(content))
		db.Exec(`UPDATE learning_records SET body_text = ? WHERE id = ?`, bodyText, id)
	}
	return rows.Err()
}

// backfillLastViewedSlugs migrates last_*_seq columns to last_*_slug on
// the workspaces table. Idempotent — skips if old seq columns are already gone.
func backfillLastViewedSlugs(db *sql.DB) error {
	// Check if old seq columns still exist.
	var hasLessonSeq int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('workspaces') WHERE name = 'last_lesson_seq'`).Scan(&hasLessonSeq); err != nil {
		return fmt.Errorf("check last_lesson_seq column: %w", err)
	}
	if hasLessonSeq == 0 {
		return nil // migration already ran
	}

	// Backfill lesson slug from seq.
	db.Exec(`
		UPDATE workspaces SET last_lesson_slug = (
			SELECT l.slug FROM lessons l
			WHERE l.workspace_id = workspaces.id AND l.sequence_number = workspaces.last_lesson_seq
		) WHERE last_lesson_seq IS NOT NULL AND last_lesson_slug IS NULL`)

	// Backfill record slug from seq.
	db.Exec(`
		UPDATE workspaces SET last_record_slug = (
			SELECT r.slug FROM learning_records r
			WHERE r.workspace_id = workspaces.id AND r.sequence_number = workspaces.last_record_seq
		) WHERE last_record_seq IS NOT NULL AND last_record_slug IS NULL`)

	// Backfill ref slug from row ID (refs are already slug-based).
	db.Exec(`
		UPDATE workspaces SET last_ref_slug = (
			SELECT r.slug FROM references_t r
			WHERE r.id = workspaces.last_ref_seq
		) WHERE last_ref_seq IS NOT NULL AND last_ref_slug IS NULL`)

	// Drop old columns.
	db.Exec(`ALTER TABLE workspaces DROP COLUMN last_lesson_seq`)
	db.Exec(`ALTER TABLE workspaces DROP COLUMN last_record_seq`)
	db.Exec(`ALTER TABLE workspaces DROP COLUMN last_ref_seq`)

	return nil
}
