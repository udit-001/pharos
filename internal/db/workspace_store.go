package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
)

// WorkspaceStore is a scoped view over the database bound to a single workspace.
// Created via Store.Workspace(name) — the workspace is resolved once at the
// seam, so callers never thread workspaceID through every call.
//
// WorkspaceStore owns the SQL for all workspace-scoped item operations
// (lessons, records, references). The flat Store no longer exposes item
// methods — callers must go through this seam.
type WorkspaceStore struct {
	store *Store
	ws    Workspace
}

// Workspace returns the resolved workspace.
func (w *WorkspaceStore) Workspace() Workspace { return w.ws }

// Layout returns the on-disk layout for this workspace.
func (w *WorkspaceStore) Layout() Layout { return NewLayout(w.ws.Path) }

// db returns the underlying *sqlx.DB for direct query access.
func (w *WorkspaceStore) db() *sqlx.DB { return w.store.db }

// ── Lessons ──

// GetLessons returns all lessons in this workspace, ordered by sequence number.
func (w *WorkspaceStore) GetLessons() ([]Lesson, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? ORDER BY sequence_number ASC", lessonColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}

// GetLessonBySeq returns a single lesson by its sequence number, or an error if not found.
func (w *WorkspaceStore) GetLessonBySeq(seq int) (*Lesson, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? AND sequence_number = ?", lessonColumns),
		w.ws.ID, seq,
	)
	lesson, err := scanLesson(row)
	if err != nil {
		return nil, fmt.Errorf("lesson %d not found: %w", seq, err)
	}
	return &lesson, nil
}

// SearchLessons performs full-text search within this workspace.
func (w *WorkspaceStore) SearchLessons(query string) ([]Lesson, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Lesson{}, nil
	}
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons_fts JOIN lessons ON lessons.id = lessons_fts.rowid WHERE lessons_fts MATCH ? AND lessons.workspace_id = ? ORDER BY bm25(lessons_fts, 10.0, 5.0, 1.0), lessons.sequence_number ASC", lessonColumnsQualified),
		q, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}

// Search returns results from all entity types in this workspace.
func (w *WorkspaceStore) Search(query string) ([]SearchResult, error) {
	var results []SearchResult

	lessons, err := w.SearchLessons(query)
	if err != nil {
		return nil, fmt.Errorf("search lessons: %w", err)
	}
	for _, l := range lessons {
		sr := SearchResult{
			Type: "lesson", Title: l.Title, Summary: l.Summary,
			WorkspaceName: w.ws.Name, WorkspaceID: w.ws.ID,
			SequenceNumber: l.SequenceNumber,
		}
		if sr.Summary == "" && l.BodyText != "" {
			sr.Snippet = truncateSnippet(stripLeadingTitle(l.BodyText, l.Title), 200)
		}
		results = append(results, sr)
	}

	recs, err := w.SearchRecords(query)
	if err != nil {
		return nil, fmt.Errorf("search records: %w", err)
	}
	for _, rec := range recs {
		sr := SearchResult{
			Type: "record", Title: rec.Title, Summary: rec.Summary,
			WorkspaceName: w.ws.Name, WorkspaceID: w.ws.ID,
			SequenceNumber: rec.SequenceNumber,
		}
		if sr.Summary == "" && rec.BodyText != "" {
			sr.Snippet = truncateSnippet(rec.BodyText, 200)
		}
		results = append(results, sr)
	}

	refs, err := w.SearchRefs(query)
	if err != nil {
		return nil, fmt.Errorf("search refs: %w", err)
	}
	for _, ref := range refs {
		slug := ref.Slug
		if slug == "" {
			slug = Slugify(ref.Title)
		}
		sr := SearchResult{
			Type: "ref", Title: ref.Title, Summary: ref.Summary,
			WorkspaceName: w.ws.Name, WorkspaceID: w.ws.ID,
			Slug: slug,
		}
		if sr.Summary == "" && ref.BodyText != "" {
			sr.Snippet = truncateSnippet(stripLeadingTitle(ref.BodyText, ref.Title), 200)
		}
		results = append(results, sr)
	}

	if results == nil {
		return []SearchResult{}, nil
	}
	return results, nil
}

// AddLesson creates a new lesson in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
//
// AddLesson is a low-level insert used by tests and internal wiring. Callers
// that need body_text indexing should use CreateLesson instead.
func (w *WorkspaceStore) AddLesson(l Lesson) (Lesson, error) {
	l.WorkspaceID = w.ws.ID
	now := time.Now().UTC().Format(time.RFC3339)

	// Determine next sequence number
	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM lessons WHERE workspace_id = ?", l.WorkspaceID)
	l.SequenceNumber = maxSeq + 1

	result, err := w.db().Exec(
		`INSERT INTO lessons (workspace_id, title, sequence_number, filename, path, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		l.WorkspaceID, l.Title, l.SequenceNumber, l.Filename, l.Path, l.Summary, now, now,
	)
	if err != nil {
		return Lesson{}, fmt.Errorf("add lesson: %w", err)
	}
	id, _ := result.LastInsertId()
	l.ID = id
	l.CreatedAt = now
	l.UpdatedAt = now
	return l, nil
}

// ── Learning records ──

// GetRecords returns all learning records in this workspace.
func (w *WorkspaceStore) GetRecords() ([]LearningRecord, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? ORDER BY sequence_number ASC", recordColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// GetRecordBySeq returns a single learning record by its sequence number, or an error if not found.
func (w *WorkspaceStore) GetRecordBySeq(seq int) (*LearningRecord, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? AND sequence_number = ?", recordColumns),
		w.ws.ID, seq,
	)
	record, err := scanRecord(row)
	if err != nil {
		return nil, fmt.Errorf("record %d not found: %w", seq, err)
	}
	return &record, nil
}

// SearchRecords performs full-text search within this workspace.
func (w *WorkspaceStore) SearchRecords(query string) ([]LearningRecord, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []LearningRecord{}, nil
	}
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM records_fts JOIN learning_records ON learning_records.id = records_fts.rowid WHERE records_fts MATCH ? AND learning_records.workspace_id = ? ORDER BY bm25(records_fts, 10.0, 5.0, 1.0), learning_records.sequence_number ASC", recordColumnsQualified),
		q, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// AddRecord creates a new learning record in this workspace. WorkspaceID is
// set automatically from the scoped workspace.
func (w *WorkspaceStore) AddRecord(r LearningRecord) (LearningRecord, error) {
	r.WorkspaceID = w.ws.ID
	now := time.Now().UTC().Format(time.RFC3339)

	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM learning_records WHERE workspace_id = ?", r.WorkspaceID)
	r.SequenceNumber = maxSeq + 1

	if r.Status == "" {
		r.Status = "active"
	}

	var supersededBy interface{}
	if r.SupersededBy > 0 {
		supersededBy = r.SupersededBy
	}

	result, err := w.db().Exec(
		`INSERT INTO learning_records (workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.WorkspaceID, r.Title, r.SequenceNumber, r.Filename, r.Path, r.Status, supersededBy, r.Summary, now, now,
	)
	if err != nil {
		return LearningRecord{}, fmt.Errorf("add learning record: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt = now
	r.UpdatedAt = now
	return r, nil
}

// ── References ──

// GetRefs returns all references in this workspace.
func (w *WorkspaceStore) GetRefs() ([]Reference, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? ORDER BY title ASC", refColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

// GetRefBySlug returns a single reference by its slug, or an error if not found.
func (w *WorkspaceStore) GetRefBySlug(slug string) (*Reference, error) {
	row := w.db().QueryRow(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? AND slug = ?", refColumns),
		w.ws.ID, slug,
	)
	ref, err := scanRef(row)
	if err != nil {
		return nil, fmt.Errorf("reference %q not found: %w", slug, err)
	}
	return &ref, nil
}

// SearchRefs performs full-text search within this workspace.
func (w *WorkspaceStore) SearchRefs(query string) ([]Reference, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Reference{}, nil
	}
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM refs_fts JOIN references_t ON references_t.id = refs_fts.rowid WHERE refs_fts MATCH ? AND references_t.workspace_id = ? ORDER BY bm25(refs_fts, 10.0, 5.0, 1.0), references_t.title ASC", refColumnsQualified),
		q, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

// AddRef creates a new reference in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
func (w *WorkspaceStore) AddRef(r Reference) (Reference, error) {
	r.WorkspaceID = w.ws.ID
	now := time.Now().UTC().Format(time.RFC3339)

	if r.Slug == "" {
		r.Slug = Slugify(r.Title)
	}

	result, err := w.db().Exec(
		`INSERT INTO references_t (workspace_id, title, slug, filename, path, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.WorkspaceID, r.Title, r.Slug, r.Filename, r.Path, r.Summary, now, now,
	)
	if err != nil {
		return Reference{}, fmt.Errorf("add reference: %w", err)
	}
	id, _ := result.LastInsertId()
	r.ID = id
	r.CreatedAt = now
	r.UpdatedAt = now
	return r, nil
}

// ── Workspace-scoped mutations ──

// SetLastViewed records which item was last viewed in this workspace.
func (w *WorkspaceStore) SetLastViewed(itemType string, seq int) error {
	return w.store.SetLastViewed(w.ws.ID, itemType, seq)
}

// Touch updates the last_studied timestamp for this workspace.
func (w *WorkspaceStore) Touch() error {
	return w.store.TouchWorkspace(w.ws.ID)
}

// UpdateMission updates the mission_why field for this workspace.
func (w *WorkspaceStore) UpdateMission(missionWhy string) error {
	return w.store.UpdateWorkspaceMission(w.ws.ID, missionWhy)
}

// UpdateTopic updates the topic field for this workspace.
func (w *WorkspaceStore) UpdateTopic(topic string) error {
	return w.store.UpdateWorkspaceTopic(w.ws.ID, topic)
}

// ── Deep create/revise/supersede methods ──

// CreateLesson creates a new lesson: sequencing, slugify, filename, write file,
// DB row — all in one method. The CLI shrinks to parse-and-call.
func (w *WorkspaceStore) CreateLesson(title, bodyHTML string) (Lesson, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM lessons WHERE workspace_id = ?", w.ws.ID)
	seqNum := maxSeq + 1

	slug := Slugify(title)
	filename := fmt.Sprintf("%04d-%s.html", seqNum, slug)

	if err := writeToFile(w.Layout().LessonPath(filename), bodyHTML); err != nil {
		return Lesson{}, fmt.Errorf("write lesson file: %w", err)
	}

	bodyText := extractText(bodyHTML)

	result, err := w.db().Exec(
		`INSERT INTO lessons (workspace_id, title, sequence_number, filename, path, summary, body_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ws.ID, title, seqNum, filename, w.Layout().LessonRelPath(filename), "", bodyText, now, now,
	)
	if err != nil {
		return Lesson{}, fmt.Errorf("insert lesson: %w", err)
	}
	id, _ := result.LastInsertId()

	return Lesson{
		ID: id, WorkspaceID: w.ws.ID, Title: title, SequenceNumber: seqNum,
		Filename: filename, Path: w.Layout().LessonRelPath(filename),
		BodyText: bodyText, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ReviseLesson overwrites a lesson's content in place. Sequence and filename
// are unchanged.
func (w *WorkspaceStore) ReviseLesson(seq int, bodyHTML string, title *string, summary *string) error {
	lessons, err := w.GetLessons()
	if err != nil {
		return fmt.Errorf("find lesson: %w", err)
	}
	var current *Lesson
	for i := range lessons {
		if lessons[i].SequenceNumber == seq {
			current = &lessons[i]
			break
		}
	}
	if current == nil {
		return fmt.Errorf("lesson %d not found", seq)
	}

	if err := writeToFile(w.Layout().LessonPath(current.Filename), bodyHTML); err != nil {
		return fmt.Errorf("write lesson file: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	t := current.Title
	if title != nil {
		t = *title
	}
	s := current.Summary
	if summary != nil {
		s = *summary
	}
	bodyText := extractText(bodyHTML)
	_, err = w.db().Exec("UPDATE lessons SET title = ?, summary = ?, body_text = ?, updated_at = ? WHERE id = ?", t, s, bodyText, now, current.ID)
	return err
}

// CreateRecord creates a new learning record: sequencing, slugify, filename,
// write file, DB row — all in one method.
func (w *WorkspaceStore) CreateRecord(title, bodyMD, summary string) (LearningRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var maxSeq int
	w.db().Get(&maxSeq, "SELECT COALESCE(MAX(sequence_number), 0) FROM learning_records WHERE workspace_id = ?", w.ws.ID)
	seqNum := maxSeq + 1

	slug := Slugify(title)
	filename := fmt.Sprintf("%04d-%s.md", seqNum, slug)

	if err := writeToFile(w.Layout().RecordPath(filename), bodyMD); err != nil {
		return LearningRecord{}, fmt.Errorf("write record file: %w", err)
	}

	bodyText := extractTextFromMarkdown(bodyMD)

	var supersededBy interface{}
	result, err := w.db().Exec(
		`INSERT INTO learning_records (workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, body_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, ?)`,
		w.ws.ID, title, seqNum, filename, w.Layout().RecordRelPath(filename), supersededBy, summary, bodyText, now, now,
	)
	if err != nil {
		return LearningRecord{}, fmt.Errorf("insert record: %w", err)
	}
	id, _ := result.LastInsertId()

	return LearningRecord{
		ID: id, WorkspaceID: w.ws.ID, Title: title, SequenceNumber: seqNum,
		Filename: filename, Path: w.Layout().RecordRelPath(filename),
		Status: "active", Summary: summary, BodyText: bodyText, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// SupersedeRecord atomically creates a new record and marks the old one as
// superseded. Returns the new record.
func (w *WorkspaceStore) SupersedeRecord(seq int, title, bodyMD, summary string) (LearningRecord, LearningRecord, error) {
	records, err := w.GetRecords()
	if err != nil {
		return LearningRecord{}, LearningRecord{}, fmt.Errorf("find old record: %w", err)
	}
	var old *LearningRecord
	for i := range records {
		if records[i].SequenceNumber == seq {
			old = &records[i]
			break
		}
	}
	if old == nil {
		return LearningRecord{}, LearningRecord{}, fmt.Errorf("record %d not found", seq)
	}

	created, err := w.CreateRecord(title, bodyMD, summary)
	if err != nil {
		return LearningRecord{}, LearningRecord{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = w.db().Exec("UPDATE learning_records SET status = 'superseded', superseded_by = ?, updated_at = ? WHERE id = ?", created.ID, now, old.ID)
	if err != nil {
		return created, LearningRecord{}, fmt.Errorf("supersede old record: %w", err)
	}

	old.Status = "superseded"
	old.SupersededBy = created.ID
	old.UpdatedAt = now

	return created, *old, nil
}

// CreateRef creates a new reference: slug-based filename, write file, DB row.
func (w *WorkspaceStore) CreateRef(title, bodyHTML string) (Reference, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	slug := Slugify(title)
	filename := slug + ".html"

	// Check for duplicate slug
	existing, _ := w.GetRefs()
	for _, r := range existing {
		if r.Slug == slug {
			return Reference{}, fmt.Errorf("reference with slug %q already exists", slug)
		}
	}

	if err := writeToFile(w.Layout().RefPath(filename), bodyHTML); err != nil {
		return Reference{}, fmt.Errorf("write reference file: %w", err)
	}

	bodyText := extractText(bodyHTML)

	result, err := w.db().Exec(
		`INSERT INTO references_t (workspace_id, title, slug, filename, path, summary, body_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ws.ID, title, slug, filename, w.Layout().RefRelPath(filename), "", bodyText, now, now,
	)
	if err != nil {
		return Reference{}, fmt.Errorf("insert reference: %w", err)
	}
	id, _ := result.LastInsertId()

	return Reference{
		ID: id, WorkspaceID: w.ws.ID, Title: title, Slug: slug,
		Filename: filename, Path: w.Layout().RefRelPath(filename),
		BodyText: bodyText, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ReviseRef overwrites a reference's content in place. Slug is unchanged.
func (w *WorkspaceStore) ReviseRef(slug, bodyHTML string, title *string, summary *string) error {
	refs, err := w.GetRefs()
	if err != nil {
		return fmt.Errorf("find reference: %w", err)
	}
	var current *Reference
	for i := range refs {
		if refs[i].Slug == slug {
			current = &refs[i]
			break
		}
	}
	if current == nil {
		return fmt.Errorf("reference %q not found", slug)
	}

	if err := writeToFile(w.Layout().RefPath(current.Filename), bodyHTML); err != nil {
		return fmt.Errorf("write reference file: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	t := current.Title
	if title != nil {
		t = *title
	}
	s := current.Summary
	if summary != nil {
		s = *summary
	}
	bodyText := extractText(bodyHTML)
	_, err = w.db().Exec("UPDATE references_t SET title = ?, summary = ?, body_text = ?, updated_at = ? WHERE id = ?", t, s, bodyText, now, current.ID)
	return err
}

// ── Glossary Terms ──

// GetGlossaryTerms returns all glossary terms in this workspace, ordered by
// category (uncategorised last) then term alphabetically.
func (w *WorkspaceStore) GetGlossaryTerms() ([]GlossaryTerm, error) {
	rows, err := w.db().Query(
		fmt.Sprintf(`SELECT %s FROM glossary_terms WHERE workspace_id = ?
			ORDER BY CASE WHEN category IS NULL OR category = '' THEN 1 ELSE 0 END,
			category COLLATE NOCASE ASC, term COLLATE NOCASE ASC`, glossaryTermColumns),
		w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGlossaryTerms(rows)
}

// AddGlossaryTerm inserts or updates a single glossary term for the workspace.
// If a term with the same name (case-insensitive) already exists, it is updated.
// Empty category/avoid strings are treated as "leave unchanged" on update —
// to clear a category or avoid, delete the term and re-add it.
func (w *WorkspaceStore) AddGlossaryTerm(term, definition, category, avoid string) error {
	term = strings.TrimSpace(term)
	definition = strings.TrimSpace(definition)
	category = strings.TrimSpace(category)
	avoid = strings.TrimSpace(avoid)
	if term == "" {
		return fmt.Errorf("term must not be empty")
	}
	if definition == "" {
		return fmt.Errorf("definition must not be empty")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := w.db().Exec(
		`INSERT INTO glossary_terms (workspace_id, term, definition, category, avoid, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_id, term) DO UPDATE SET
			definition = excluded.definition,
			category = COALESCE(NULLIF(excluded.category, ''), glossary_terms.category),
			avoid = COALESCE(NULLIF(excluded.avoid, ''), glossary_terms.avoid),
			updated_at = excluded.updated_at`,
		w.ws.ID, term, definition, category, avoid, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert term %q: %w", term, err)
	}
	return nil
}

// DeleteGlossaryTerm removes a glossary term by name (case-insensitive).
// Returns nil if the term doesn't exist — delete is idempotent.
func (w *WorkspaceStore) DeleteGlossaryTerm(term string) error {
	term = strings.TrimSpace(term)
	if term == "" {
		return fmt.Errorf("term must not be empty")
	}
	_, err := w.db().Exec(
		"DELETE FROM glossary_terms WHERE workspace_id = ? AND term = ? COLLATE NOCASE",
		w.ws.ID, term,
	)
	if err != nil {
		return fmt.Errorf("delete term %q: %w", term, err)
	}
	return nil
}

// CreateAsset writes a file to the workspace's assets directory.
func (w *WorkspaceStore) CreateAsset(filename, content string) error {
	return writeToFile(w.Layout().AssetPath(filename), content)
}

// ListAssets returns the filenames in the workspace's assets directory.
func (w *WorkspaceStore) ListAssets() ([]string, error) {
	entries, err := readDirNames(filepath.Join(w.ws.Path, "assets"))
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ── Construction ──

// Workspace returns a WorkspaceStore scoped to the named workspace. The
// workspace is resolved (name→ID) once here; subsequent calls need not
// pass the ID. Returns an error if the workspace does not exist.
func (s *Store) Workspace(name string) (*WorkspaceStore, error) {
	ws, err := s.GetWorkspaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("workspace %q not found: %w", name, err)
	}
	return &WorkspaceStore{store: s, ws: ws}, nil
}

// WorkspaceByID returns a WorkspaceStore scoped to the workspace with the
// given ID. Used when the ID is already known.
func (s *Store) WorkspaceByID(id int64) (*WorkspaceStore, error) {
	ws, err := s.GetWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("workspace %d not found: %w", id, err)
	}
	return &WorkspaceStore{store: s, ws: ws}, nil
}

// extractText parses HTML and returns the plain text content, stripping all
// markup. Used to index lesson body content for full-text search.
func extractText(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	doc.Find("head, script, style, noscript").Remove()
	return strings.TrimSpace(doc.Text())
}

// extractTextFromMarkdown strips markdown formatting and returns plain text.
func extractTextFromMarkdown(md string) string {
	lines := strings.Split(md, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Strip indented code blocks (4 spaces or tab)
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			if trimmed == "" || strings.HasPrefix(trimmed, "```") {
				continue
			}
			// Not strictly a code block, keep the text but trimmed
		}

		// Strip blockquote and heading markers
		processed := strings.TrimLeft(line, " >")
		processed = strings.TrimLeft(processed, "#")
		processed = strings.TrimLeft(processed, " ")

		// Strip list markers: -, *, +, 1.
		if len(processed) > 0 {
			c := processed[0]
			if c == '-' || c == '+' || c == '*' {
				if len(processed) == 1 || processed[1] == ' ' {
					processed = strings.TrimLeft(processed[1:], " ")
				}
			} else if c >= '0' && c <= '9' {
				if idx := strings.Index(processed, ". "); idx > 0 && idx < 4 {
					processed = processed[idx+2:]
				} else if idx := strings.Index(processed, ") "); idx > 0 && idx < 4 {
					processed = processed[idx+2:]
				}
			}
		}

		// Skip horizontal rules
		if isHorizontalRule(processed) {
			continue
		}

		// Strip remaining inline formatting: keep text from links, strip markers
		processed = stripInlineMarkdown(processed)

		processed = strings.TrimSpace(processed)
		if processed != "" {
			result = append(result, processed)
		}
	}

	return strings.TrimSpace(strings.Join(result, " "))
}

func isHorizontalRule(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return false
	}
	for _, r := range s {
		if r != '-' && r != '*' && r != '_' {
			return false
		}
	}
	return true
}

func stripInlineMarkdown(s string) string {
	var b strings.Builder
	i := 0
	runes := []rune(s)

	for i < len(runes) {
		c := runes[i]

		// Skip image ![alt](url) — keep alt text
		if c == '!' && i+1 < len(runes) && runes[i+1] == '[' {
			end := findMatchingBracket(runes, i+1, '[', ']')
			if end < 0 {
				b.WriteRune(c)
				i++
				continue
			}
			altText := runes[i+2 : end]
			// Skip the URL part
			if end+1 < len(runes) && runes[end+1] == '(' {
				parenEnd := findMatchingParen(runes, end+1)
				if parenEnd >= 0 {
					// Recurse on alt text (may contain formatting)
					b.WriteString(stripInlineMarkdown(string(altText)))
					b.WriteRune(' ')
					i = parenEnd + 1
					continue
				}
			}
			b.WriteString(stripInlineMarkdown(string(altText)))
			b.WriteRune(' ')
			i = end + 1
			continue
		}

		// Link [text](url) — keep text
		if c == '[' {
			end := findMatchingBracket(runes, i, '[', ']')
			if end < 0 {
				b.WriteRune(c)
				i++
				continue
			}
			text := runes[i+1 : end]
			if end+1 < len(runes) && runes[end+1] == '(' {
				parenEnd := findMatchingParen(runes, end+1)
				if parenEnd >= 0 {
					b.WriteString(stripInlineMarkdown(string(text)))
					b.WriteRune(' ')
					i = parenEnd + 1
					continue
				}
			}
			b.WriteString(stripInlineMarkdown(string(text)))
			i = end + 1
			continue
		}

		// Skip HTML tags
		if c == '<' {
			gt := strings.IndexRune(string(runes[i:]), '>')
			if gt >= 0 {
				// Check if it's a closing or opening tag
				tag := string(runes[i : i+gt+1])
				if !strings.Contains(tag, " ") && !strings.HasPrefix(tag, "</") && !strings.HasPrefix(tag, "<!") && !strings.HasPrefix(tag, "<?") {
					// Simple tag like <br> or <hr>
				}
				i += gt + 1
				continue
			}
		}

		// Skip backtick code spans — keep the text inside
		if c == '`' {
			end := i + 1
			for end < len(runes) && runes[end] == '`' {
				end++
			}
			backtickLen := end - i
			closing := findBacktickSequence(runes, end, backtickLen)
			if closing >= 0 {
				b.WriteString(string(runes[end:closing]))
				b.WriteRune(' ')
				i = closing + backtickLen
				continue
			}
		}

		b.WriteRune(c)
		i++
	}

	return b.String()
}

func findMatchingBracket(runes []rune, start int, open, close rune) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		if runes[i] == open {
			depth++
		} else if runes[i] == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findMatchingParen(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		if runes[i] == '(' {
			depth++
		} else if runes[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findBacktickSequence(runes []rune, start int, n int) int {
	for i := start; i <= len(runes)-n; i++ {
		match := true
		for j := 0; j < n; j++ {
			if runes[i+j] != '`' {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// truncateSnippet returns a short preview of text, breaking at a word boundary.
// Used to show body content previews in search results when Summary is empty.
// stripLeadingTitle removes the first line of bodyText if it matches the
// entity's title, avoiding redundant text in search snippets.
func stripLeadingTitle(bodyText, title string) string {
	bodyText = strings.TrimSpace(bodyText)
	if title == "" {
		return bodyText
	}
	if strings.HasPrefix(bodyText, title) {
		rest := strings.TrimSpace(bodyText[len(title):])
		return rest
	}
	return bodyText
}

func truncateSnippet(s string, maxLen int) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	cut := strings.LastIndex(trimmed[:maxLen], " ")
	if cut < 1 {
		cut = maxLen
	}
	return strings.TrimSpace(trimmed[:cut]) + "..."
}

// IndexLessons reads all lessons with empty body_text, extracts plain text
// from their HTML files on disk, and updates the DB so the FTS index captures
// lesson body content. Returns the number of lessons updated and any errors.
// If one file fails, processing continues with the remaining lessons.
func (w *WorkspaceStore) IndexLessons() (int, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE workspace_id = ? AND body_text = ''", lessonColumns),
		w.ws.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("query lessons: %w", err)
	}
	defer rows.Close()

	lessons, err := scanLessons(rows)
	if err != nil {
		return 0, fmt.Errorf("scan lessons: %w", err)
	}

	if len(lessons) == 0 {
		return 0, nil
	}

	var updated int
	var errs []string
	for _, l := range lessons {
		path := w.Layout().LessonPath(l.Filename)
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("lesson %d (%s): read file: %v", l.SequenceNumber, l.Filename, err))
			continue
		}

		bodyText := extractText(string(data))
		if _, err := w.db().Exec("UPDATE lessons SET body_text = ? WHERE id = ?", bodyText, l.ID); err != nil {
			errs = append(errs, fmt.Sprintf("lesson %d (%s): update: %v", l.SequenceNumber, l.Filename, err))
			continue
		}
		updated++
	}

	if len(errs) > 0 {
		return updated, fmt.Errorf("index: %d error(s): %s", len(errs), strings.Join(errs, "; "))
	}

	return updated, nil
}

// IndexRefs reads all references with empty body_text, extracts plain text
// from their HTML files on disk, and updates the DB so the FTS index captures
// reference body content. Returns the number of references updated and any
// errors. If one file fails, processing continues with the remaining refs.
func (w *WorkspaceStore) IndexRefs() (int, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE workspace_id = ? AND body_text = ''", refColumns),
		w.ws.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("query references: %w", err)
	}
	defer rows.Close()

	refs, err := scanRefs(rows)
	if err != nil {
		return 0, fmt.Errorf("scan references: %w", err)
	}

	if len(refs) == 0 {
		return 0, nil
	}

	var updated int
	var errs []string
	for _, r := range refs {
		path := w.Layout().RefPath(r.Filename)
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("ref %s (%s): read file: %v", r.Slug, r.Filename, err))
			continue
		}

		bodyText := extractText(string(data))
		if _, err := w.db().Exec("UPDATE references_t SET body_text = ? WHERE id = ?", bodyText, r.ID); err != nil {
			errs = append(errs, fmt.Sprintf("ref %s (%s): update: %v", r.Slug, r.Filename, err))
			continue
		}
		updated++
	}

	if len(errs) > 0 {
		return updated, fmt.Errorf("index refs: %d error(s): %s", len(errs), strings.Join(errs, "; "))
	}

	return updated, nil
}

// IndexRecords reads all learning records with empty body_text, extracts plain
// text from their markdown files on disk, and updates the DB so the FTS index
// captures record body content. Returns the number of records updated and any
// errors. If one file fails, processing continues with the remaining records.
func (w *WorkspaceStore) IndexRecords() (int, error) {
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE workspace_id = ? AND body_text = ''", recordColumns),
		w.ws.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("query records: %w", err)
	}
	defer rows.Close()

	recs, err := scanRecords(rows)
	if err != nil {
		return 0, fmt.Errorf("scan records: %w", err)
	}

	if len(recs) == 0 {
		return 0, nil
	}

	var updated int
	var errs []string
	for _, r := range recs {
		path := w.Layout().RecordPath(r.Filename)
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("record %d (%s): read file: %v", r.SequenceNumber, r.Filename, err))
			continue
		}

		bodyText := extractTextFromMarkdown(string(data))
		if _, err := w.db().Exec("UPDATE learning_records SET body_text = ? WHERE id = ?", bodyText, r.ID); err != nil {
			errs = append(errs, fmt.Sprintf("record %d (%s): update: %v", r.SequenceNumber, r.Filename, err))
			continue
		}
		updated++
	}

	if len(errs) > 0 {
		return updated, fmt.Errorf("index records: %d error(s): %s", len(errs), strings.Join(errs, "; "))
	}

	return updated, nil
}
