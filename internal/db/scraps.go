package db

import (
	"fmt"
	"strings"
)

// Scratchpad (global, sealed from workspaces). Scraps and tags are top-level
// objects on the Store — NOT workspace-scoped, mirroring the global nature of
// the scratchpad itself. All the SQL for scraps/tags lives here so the Store
// interface stays the single seam (LEARN-12).

const scrapColumns = `id, slug, title, body, status, created_at, updated_at`

const tagColumns = `id, name, description, created_at, updated_at`

// Scrap status values — exactly two. A scratchpad scrap is either still on the
// (default) agent read, or finished. There is no "archived" limbo state.
const (
	ScrapStatusActive = "active"
	ScrapStatusDone   = "done"
)

func scanScrap(row interface{ Scan(...any) error }) (Scrap, error) {
	var s Scrap
	err := row.Scan(&s.ID, &s.Slug, &s.Title, &s.Body, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func scanScraps(rows RowScanner) ([]Scrap, error) {
	return scanRows(rows, "scrap", scanScrap)
}

func scanTag(row interface{ Scan(...any) error }) (Tag, error) {
	var t Tag
	err := row.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func scanTags(rows RowScanner) ([]Tag, error) {
	return scanRows(rows, "tag", scanTag)
}

// CreateScrap creates a new active scrap. Title is required and drives the
// stable slug (derived once via Slugify). The listed tag names are attached;
// any that don't already exist are auto-created with an empty description.
func (s *Store) CreateScrap(title, body string, tagNames []string) (Scrap, error) {
	slug := Slugify(title)
	if slug == "" {
		return Scrap{}, fmt.Errorf("scrap title must produce a slug")
	}

	res, err := s.db.Exec(
		"INSERT INTO scraps (slug, title, body, status) VALUES (?, ?, ?, 'active')",
		slug, title, body,
	)
	if err != nil {
		return Scrap{}, fmt.Errorf("insert scrap: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Scrap{}, err
	}

	if len(tagNames) > 0 {
		if err := s.setScrapTags(id, tagNames); err != nil {
			return Scrap{}, err
		}
	}

	return s.ScrapBySlug(slug)
}

// ScrapBySlug returns a single scrap by its stable slug.
func (s *Store) ScrapBySlug(slug string) (Scrap, error) {
	row := s.db.QueryRow("SELECT "+scrapColumns+" FROM scraps WHERE slug = ?", slug)
	sc, err := scanScrap(row)
	if err != nil {
		return Scrap{}, fmt.Errorf("scrap %q not found: %w", slug, err)
	}
	return sc, nil
}

// ListScraps lists scraps, optionally filtered by status ("active" or "done",
// or empty for all). Default callers pass "active" for the agent context read.
func (s *Store) ListScraps(status string) ([]Scrap, error) {
	q := "SELECT " + scrapColumns + " FROM scraps"
	args := []any{}
	if status != "" {
		q += " WHERE status = ?"
		args = append(args, status)
	}
	q += " ORDER BY updated_at DESC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScraps(rows)
}

// SearchScraps performs full-text search across scrap title + body plus the
// descriptions of attached tags, optionally filtered by status. A scrap
// surfaces if it matches its own text OR any attached tag's name/description.
func (s *Store) SearchScraps(query, status string) ([]Scrap, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Scrap{}, nil
	}

	// Title/body via the FTS index. scraps_fts is external-content over scraps
	// (content_rowid=id), so its rowid equals the scrap's id, not a named column.
	titleBody := "SELECT rowid FROM scraps_fts WHERE scraps_fts MATCH ?"

	// Tag name/description via the join (scraps whose tags match the terms).
	tagLike, likeArgs := ftsTermsLike(query)
	tagIdsSQL := fmt.Sprintf(
		"SELECT DISTINCT st.scrap_id FROM scrap_tags st JOIN tags t ON t.id = st.tag_id WHERE %s",
		tagLike,
	)

	args := []any{q}
	args = append(args, likeArgs...)

	where := "(id IN (%s) OR id IN (%s))"
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	combined := fmt.Sprintf(
		"SELECT %s FROM scraps WHERE %s ORDER BY updated_at DESC",
		scrapColumns, fmt.Sprintf(where, titleBody, tagIdsSQL),
	)
	rows, err := s.db.Query(combined, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScraps(rows)
}

// UpdateScrap mutates a scrap in place. Pointers are optional: nil means
// "unchanged". tags (when non-nil) replaces the full tag set — it is not
// append-only. The slug never changes (stable find-then-update handle); a
// changed title does not regenerate it.
func (s *Store) UpdateScrap(slug string, title, body *string, status *string, tags *[]string) (Scrap, error) {
	current, err := s.ScrapBySlug(slug)
	if err != nil {
		return Scrap{}, err
	}

	newTitle := current.Title
	if title != nil {
		newTitle = *title
	}
	newBody := current.Body
	if body != nil {
		newBody = *body
	}
	newStatus := current.Status
	if status != nil {
		newStatus = *status
	}

	_, err = s.db.Exec(
		"UPDATE scraps SET title = ?, body = ?, status = ?, updated_at = ? WHERE id = ?",
		newTitle, newBody, newStatus, nowTimestamp(), current.ID,
	)
	if err != nil {
		return Scrap{}, fmt.Errorf("update scrap %q: %w", slug, err)
	}

	if tags != nil {
		if err := s.setScrapTags(current.ID, *tags); err != nil {
			return Scrap{}, err
		}
	}

	return s.ScrapBySlug(slug)
}

// DeleteScrap permanently removes a scrap (join rows cascade).
func (s *Store) DeleteScrap(slug string) error {
	res, err := s.db.Exec("DELETE FROM scraps WHERE slug = ?", slug)
	if err != nil {
		return fmt.Errorf("delete scrap %q: %w", slug, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("scrap %q not found", slug)
	}
	return nil
}

// setScrapTags replaces the tag association set for a scrap. Each name must
// already exist as a Tag — the agent creates tags deliberately with a
// description via tag create (a bare auto-created string adds no information,
// LEARN-179). Detaching a tag does NOT delete the Tag object — it only severs
// the association.
func (s *Store) setScrapTags(scrapID int64, tagNames []string) error {
	if _, err := s.db.Exec("DELETE FROM scrap_tags WHERE scrap_id = ?", scrapID); err != nil {
		return fmt.Errorf("clear scrap tags: %w", err)
	}
	for _, name := range tagNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		tag, err := s.TagByName(name)
		if err != nil {
			return fmt.Errorf("tag %q does not exist — create it first with 'pharos tag create %q --description ...'",
				name, name)
		}
		if _, err := s.db.Exec(
			"INSERT OR IGNORE INTO scrap_tags (scrap_id, tag_id) VALUES (?, ?)",
			scrapID, tag.ID,
		); err != nil {
			return fmt.Errorf("attach tag %q: %w", name, err)
		}
	}
	return nil
}

// TagByName returns a single tag by its unique name.
func (s *Store) TagByName(name string) (Tag, error) {
	row := s.db.QueryRow("SELECT "+tagColumns+" FROM tags WHERE name = ?", name)
	tag, err := scanTag(row)
	if err != nil {
		return Tag{}, err
	}
	return tag, nil
}

// CreateTag adds a NEW tag with the given description. Fails if a tag with
// that name already exists — description mutation belongs to UpdateTag, and
// add-or-update would silently clobber an existing tag's semantic payload.
func (s *Store) CreateTag(name, description string) (Tag, error) {
	if _, err := s.TagByName(name); err == nil {
		return Tag{}, fmt.Errorf("tag %q already exists — use 'tag update' to change its description", name)
	}
	if _, err := s.db.Exec(
		"INSERT INTO tags (name, description) VALUES (?, ?)",
		name, description,
	); err != nil {
		return Tag{}, fmt.Errorf("create tag %q: %w", name, err)
	}
	return s.TagByName(name)
}

// ListTags lists all tags ordered by name.
func (s *Store) ListTags() ([]Tag, error) {
	rows, err := s.db.Query("SELECT " + tagColumns + " FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTags(rows)
}

// TagsForScrap returns the tags attached to a scrap by slug, ordered by name.
func (s *Store) TagsForScrap(slug string) ([]Tag, error) {
	rows, err := s.db.Query(
		"SELECT t.id, t.name, t.description, t.created_at, t.updated_at FROM tags t "+
			"JOIN scrap_tags st ON st.tag_id = t.id "+
			"JOIN scraps sc ON sc.id = st.scrap_id "+
			"WHERE sc.slug = ? ORDER BY t.name ASC",
		slug,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTags(rows)
}

// UpdateTagDescription sets a tag's description (the semantic payload that
// powers tag-description search). Returns an error if the tag is missing.
func (s *Store) UpdateTagDescription(name, description string) error {
	res, err := s.db.Exec(
		"UPDATE tags SET description = ?, updated_at = ? WHERE name = ?",
		description, nowTimestamp(), name,
	)
	if err != nil {
		return fmt.Errorf("update tag %q: %w", name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tag %q not found", name)
	}
	return nil
}

// DeleteTag permanently removes a tag and its associations (cascade).
func (s *Store) DeleteTag(name string) error {
	res, err := s.db.Exec("DELETE FROM tags WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("delete tag %q: %w", name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tag %q not found", name)
	}
	return nil
}

// ftsTermsLike builds a LIKE-based matcher over tag name + description for the
// raw query terms (used for tag-description search). Returns a SQL fragment
// and the args to bind, in placeholder order.
func ftsTermsLike(query string) (string, []any) {
	var parts []string
	var args []any
	for _, tok := range strings.Fields(query) {
		tok = strings.TrimRight(tok, "*")
		if tok == "" {
			continue
		}
		parts = append(parts,
			"(t.name LIKE ? OR t.description LIKE ?)")
		args = append(args, "%"+tok+"%", "%"+tok+"%")
	}
	if len(parts) == 0 {
		return "0", args
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}
