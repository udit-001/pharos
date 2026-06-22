package db

import (
	"fmt"
	"path/filepath"
	"time"

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
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM lessons WHERE id IN (SELECT rowid FROM lessons_fts WHERE lessons_fts MATCH ?) AND workspace_id = ? ORDER BY sequence_number ASC", lessonColumns),
		query, w.ws.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLessons(rows)
}

// AddLesson creates a new lesson in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
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
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM learning_records WHERE id IN (SELECT rowid FROM records_fts WHERE records_fts MATCH ?) AND workspace_id = ? ORDER BY sequence_number ASC", recordColumns),
		query, w.ws.ID,
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
	rows, err := w.db().Query(
		fmt.Sprintf("SELECT %s FROM references_t WHERE id IN (SELECT rowid FROM refs_fts WHERE refs_fts MATCH ?) AND workspace_id = ? ORDER BY title ASC", refColumns),
		query, w.ws.ID,
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

	result, err := w.db().Exec(
		`INSERT INTO lessons (workspace_id, title, sequence_number, filename, path, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ws.ID, title, seqNum, filename, w.Layout().LessonRelPath(filename), "", now, now,
	)
	if err != nil {
		return Lesson{}, fmt.Errorf("insert lesson: %w", err)
	}
	id, _ := result.LastInsertId()

	return Lesson{
		ID: id, WorkspaceID: w.ws.ID, Title: title, SequenceNumber: seqNum,
		Filename: filename, Path: w.Layout().LessonRelPath(filename),
		CreatedAt: now, UpdatedAt: now,
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
	_, err = w.db().Exec("UPDATE lessons SET title = ?, summary = ?, updated_at = ? WHERE id = ?", t, s, now, current.ID)
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

	var supersededBy interface{}
	result, err := w.db().Exec(
		`INSERT INTO learning_records (workspace_id, title, sequence_number, filename, path, status, superseded_by, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'active', ?, ?, ?, ?)`,
		w.ws.ID, title, seqNum, filename, w.Layout().RecordRelPath(filename), supersededBy, summary, now, now,
	)
	if err != nil {
		return LearningRecord{}, fmt.Errorf("insert record: %w", err)
	}
	id, _ := result.LastInsertId()

	return LearningRecord{
		ID: id, WorkspaceID: w.ws.ID, Title: title, SequenceNumber: seqNum,
		Filename: filename, Path: w.Layout().RecordRelPath(filename),
		Status: "active", Summary: summary, CreatedAt: now, UpdatedAt: now,
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

	result, err := w.db().Exec(
		`INSERT INTO references_t (workspace_id, title, slug, filename, path, summary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ws.ID, title, slug, filename, w.Layout().RefRelPath(filename), "", now, now,
	)
	if err != nil {
		return Reference{}, fmt.Errorf("insert reference: %w", err)
	}
	id, _ := result.LastInsertId()

	return Reference{
		ID: id, WorkspaceID: w.ws.ID, Title: title, Slug: slug,
		Filename: filename, Path: w.Layout().RefRelPath(filename),
		CreatedAt: now, UpdatedAt: now,
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
	_, err = w.db().Exec("UPDATE references_t SET title = ?, summary = ?, updated_at = ? WHERE id = ?", t, s, now, current.ID)
	return err
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
