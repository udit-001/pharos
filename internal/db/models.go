package db

// Workspace represents a learning workspace.
type Workspace struct {
	ID            int64  `db:"id" json:"id"`
	Name          string `db:"name" json:"name"`   // directory name
	Topic         string `db:"topic" json:"topic"` // user-friendly topic
	Path          string `db:"path" json:"path"`   // absolute path to workspace dir
	MissionWhy    string `db:"mission_why" json:"missionWhy"`
	LastLessonSeq *int   `db:"last_lesson_seq" json:"lastLessonSeq,omitempty"`
	LastRecordSeq *int   `db:"last_record_seq" json:"lastRecordSeq,omitempty"`
	LastRefSeq    *int   `db:"last_ref_seq" json:"lastRefSeq,omitempty"`
	LessonCount   int    `db:"-" json:"lessonCount"` // computed
	RecordCount   int    `db:"-" json:"recordCount"` // computed
	RefCount      int    `db:"-" json:"refCount"`    // computed
	CreatedAt     string `db:"created_at" json:"createdAt"`
	LastStudied   string `db:"last_studied" json:"lastStudied"`
}

// Lesson represents a single lesson file.
type Lesson struct {
	ID             int64  `db:"id" json:"id"`
	WorkspaceID    int64  `db:"workspace_id" json:"workspaceId"`
	Title          string `db:"title" json:"title"`
	SequenceNumber int    `db:"sequence_number" json:"sequenceNumber"`
	Filename       string `db:"filename" json:"filename"` // e.g. 0003-joins.html
	Path           string `db:"path" json:"path"`         // relative to workspace
	Summary        string `db:"summary" json:"summary"`
	BodyText       string `db:"body_text" json:"bodyText,omitempty"`
	CreatedAt      string `db:"created_at" json:"createdAt"`
	UpdatedAt      string `db:"updated_at" json:"updatedAt"`
}

// LearningRecord represents an ADR-style learning record.
type LearningRecord struct {
	ID             int64  `db:"id" json:"id"`
	WorkspaceID    int64  `db:"workspace_id" json:"workspaceId"`
	Title          string `db:"title" json:"title"`
	SequenceNumber int    `db:"sequence_number" json:"sequenceNumber"`
	Filename       string `db:"filename" json:"filename"` // e.g. 0003-joins.md
	Path           string `db:"path" json:"path"`         // relative to workspace
	Status         string `db:"status" json:"status"`     // active | superseded
	SupersededBy   int64  `db:"superseded_by" json:"supersededBy,omitempty"`
	Summary        string `db:"summary" json:"summary"`
	BodyText       string `db:"body_text" json:"bodyText,omitempty"`
	CreatedAt      string `db:"created_at" json:"createdAt"`
	UpdatedAt      string `db:"updated_at" json:"updatedAt"`
}

// Reference represents a reference document (cheat sheet).
type Reference struct {
	ID          int64  `db:"id" json:"id"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	Title       string `db:"title" json:"title"`
	Slug        string `db:"slug" json:"slug"`
	Filename    string `db:"filename" json:"filename"`
	Path        string `db:"path" json:"path"`
	Summary     string `db:"summary" json:"summary"`
	BodyText    string `db:"body_text" json:"bodyText,omitempty"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// DisplayName returns the user-friendly topic if set, else the directory name.
// Used everywhere a human reads the name; URLs and keys must still use Name.
func (w Workspace) DisplayName() string {
	if w.Topic != "" {
		return w.Topic
	}
	return w.Name
}

// SearchResult is one result from a cross-entity, cross-workspace search.
// URLs are a presentation concern and are constructed by the caller from the
// fields here.
type SearchResult struct {
	Type           string `json:"type"` // "lesson" | "record" | "ref"
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Snippet        string `json:"snippet,omitempty"` // body content preview when summary is empty
	WorkspaceName  string `json:"workspaceName"`
	WorkspaceID    int64  `json:"-"`
	SequenceNumber int    `json:"sequenceNumber,omitempty"` // lessons and records
	Slug           string `json:"slug,omitempty"`           // refs only
}

// Settings holds user preferences.
type Settings struct {
	ID                  int64  `db:"id" json:"id"`
	DefaultView         string `db:"default_view" json:"defaultView"`
	ItemsPerPage        int    `db:"items_per_page" json:"itemsPerPage"`
	LessonsDir          string `db:"lessons_dir" json:"lessonsDir"`
	RecordsDir          string `db:"records_dir" json:"recordsDir"`
	ReferenceDir        string `db:"reference_dir" json:"referenceDir"`
	AssetsDir           string `db:"assets_dir" json:"assetsDir"`
	LastActiveWorkspace string `db:"last_active_workspace" json:"lastActiveWorkspace"`
}
