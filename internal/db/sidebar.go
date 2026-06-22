package db

// SidebarData is the lightweight workspace tree for sidebar and workspace
// page rendering. It carries only the fields render needs — not full model
// structs — so callers behind the seam get exactly what they need.
type SidebarData struct {
	Workspace Workspace
	Lessons   []SidebarLesson
	Records   []SidebarRecord
	Refs      []SidebarRef
}

// SidebarLesson is the sidebar projection of a lesson.
type SidebarLesson struct {
	Seq   int
	Title string
}

// SidebarRecord is the sidebar projection of a learning record.
type SidebarRecord struct {
	Seq     int
	Title   string
	Status  string
	Summary string
}

// SidebarRef is the sidebar projection of a reference.
type SidebarRef struct {
	Slug  string
	Title string
}

// GetSidebarData loads the workspace tree (lessons, records, refs) in their
// lightweight sidebar projections. This is the deep interface: one call
// replaces three separate Get* calls, and the caller gets only the fields
// it needs. The workspace itself is already available on the WorkspaceStore.
func (w *WorkspaceStore) GetSidebarData() (SidebarData, error) {
	lessons, err := w.GetLessons()
	if err != nil {
		return SidebarData{}, err
	}
	records, err := w.GetRecords()
	if err != nil {
		return SidebarData{}, err
	}
	refs, err := w.GetRefs()
	if err != nil {
		return SidebarData{}, err
	}

	sl := make([]SidebarLesson, len(lessons))
	for i, l := range lessons {
		sl[i] = SidebarLesson{Seq: l.SequenceNumber, Title: l.Title}
	}
	sr := make([]SidebarRecord, len(records))
	for i, r := range records {
		sr[i] = SidebarRecord{Seq: r.SequenceNumber, Title: r.Title, Status: r.Status, Summary: r.Summary}
	}
	sf := make([]SidebarRef, len(refs))
	for i, ref := range refs {
		sf[i] = SidebarRef{Slug: ref.Slug, Title: ref.Title}
	}

	return SidebarData{Workspace: w.ws, Lessons: sl, Records: sr, Refs: sf}, nil
}
