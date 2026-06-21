package db

import "fmt"

// WorkspaceStore is a view over *Store bound to a single workspace. It is
// created via Store.Workspace(name) — the workspace is resolved once at the
// seam, so callers no longer thread workspaceID through every call.
//
// This is the scoped adapter introduced in LEARN-11: large interface surface
// of the flat Store collapses to workspace-local methods, concentrating the
// name→ID resolution in one place.
type WorkspaceStore struct {
	store *Store
	ws     Workspace
}

// Workspace returns the resolved workspace.
func (w *WorkspaceStore) Workspace() Workspace { return w.ws }

// ── Lessons ──

// GetLessons returns all lessons in this workspace, ordered by sequence.
func (w *WorkspaceStore) GetLessons() ([]Lesson, error) {
	return w.store.GetLessons(w.ws.ID)
}

// SearchLessons performs full-text search within this workspace.
func (w *WorkspaceStore) SearchLessons(query string) ([]Lesson, error) {
	return w.store.SearchLessons(query, w.ws.ID)
}

// AddLesson creates a new lesson in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
func (w *WorkspaceStore) AddLesson(l Lesson) (Lesson, error) {
	l.WorkspaceID = w.ws.ID
	return w.store.AddLesson(l)
}

// ── Learning records ──

// GetRecords returns all learning records in this workspace.
func (w *WorkspaceStore) GetRecords() ([]LearningRecord, error) {
	return w.store.GetLearningRecords(w.ws.ID)
}

// SearchRecords performs full-text search within this workspace.
func (w *WorkspaceStore) SearchRecords(query string) ([]LearningRecord, error) {
	return w.store.SearchLearningRecords(query, w.ws.ID)
}

// AddRecord creates a new learning record in this workspace. WorkspaceID is
// set automatically from the scoped workspace.
func (w *WorkspaceStore) AddRecord(r LearningRecord) (LearningRecord, error) {
	r.WorkspaceID = w.ws.ID
	return w.store.AddLearningRecord(r)
}

// ── References ──

// GetRefs returns all references in this workspace.
func (w *WorkspaceStore) GetRefs() ([]Reference, error) {
	return w.store.GetReferences(w.ws.ID)
}

// SearchRefs performs full-text search within this workspace.
func (w *WorkspaceStore) SearchRefs(query string) ([]Reference, error) {
	return w.store.SearchReferences(query, w.ws.ID)
}

// AddRef creates a new reference in this workspace. WorkspaceID is set
// automatically from the scoped workspace.
func (w *WorkspaceStore) AddRef(r Reference) (Reference, error) {
	r.WorkspaceID = w.ws.ID
	return w.store.AddReference(r)
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
