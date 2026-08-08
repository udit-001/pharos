package db

import (
	"strings"
	"testing"
)

func TestScrapCreateAndGet(t *testing.T) {
	store := newTestStore(t)
	mkTags(t, store, "ml", "career")

	sc, err := store.CreateScrap("Want to be an ML engineer", "roadmap: learn linear algebra first", []string{"ml", "career"})
	if err != nil {
		t.Fatalf("create scrap: %v", err)
	}
	if sc.Slug != "want-to-be-an-ml-engineer" {
		t.Errorf("slug = %q, want want-to-be-an-ml-engineer", sc.Slug)
	}
	if sc.Status != "active" {
		t.Errorf("status = %q, want active (default)", sc.Status)
	}
	if sc.ID == 0 {
		t.Errorf("id = 0, want non-zero")
	}

	got, err := store.ScrapBySlug("want-to-be-an-ml-engineer")
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if got.Body != "roadmap: learn linear algebra first" {
		t.Errorf("body = %q", got.Body)
	}

	// Non-existent slug returns an error.
	if _, err := store.ScrapBySlug("nope"); err == nil {
		t.Errorf("ScrapBySlug(nope) expected error, got nil")
	}
}

func TestScrapTagsQueryableAndDescriptions(t *testing.T) {
	store := newTestStore(t)

	// Attaching a tag REQUIRES it to already exist (strict — no auto-create),
	// so create the tag first, then give it its semantic description.
	if _, err := store.CreateTag("ml", ""); err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err := store.CreateScrap("scrap one", "some body", []string{"ml"}); err != nil {
		t.Fatalf("create scrap one: %v", err)
	}
	if _, err := store.CreateScrap("unrelated", "just a note", nil); err != nil {
		t.Fatalf("create unrelated: %v", err)
	}

	// Give the tag its semantic payload.
	if err := store.UpdateTagDescription("ml", "machine learning career goal"); err != nil {
		t.Fatalf("update tag description: %v", err)
	}
	tag, err := store.TagByName("ml")
	if err != nil {
		t.Fatalf("tag missing: %v", err)
	}
	if tag.Description != "machine learning career goal" {
		t.Errorf("description = %q", tag.Description)
	}
}

// mkTags creates tags that scrap tests attach via CreateScrap/UpdateScrap
// (currently strict — a tag must already exist to be attached).
func mkTags(t *testing.T, store *Store, names ...string) {
	t.Helper()
	for _, n := range names {
		if _, err := store.CreateTag(n, ""); err != nil {
			t.Fatalf("create tag %q: %v", n, err)
		}
	}
}

func TestScrapListAndStatusFilter(t *testing.T) {
	store := newTestStore(t)
	mk := func(title, status string) {
		sc, err := store.CreateScrap(title, "body", nil)
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		if status != "active" {
			if _, err := store.UpdateScrap(sc.Slug, nil, nil, &status, nil); err != nil {
				t.Fatalf("set status %s on %s: %v", status, title, err)
			}
		}
	}
	mk("one", "active")
	mk("two", "done")
	mk("three", "active")

	active, err := store.ListScraps("active")
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("active = %d, want 2", len(active))
	}

	done, err := store.ListScraps("done")
	if err != nil {
		t.Fatalf("list done: %v", err)
	}
	if len(done) != 1 {
		t.Errorf("done = %d, want 1", len(done))
	}
}

func TestScrapSearchTitleBodyAndTagDescription(t *testing.T) {
	store := newTestStore(t)
	mkTags(t, store, "ml", "traveling")
	if _, err := store.CreateScrap("linear algebra first", "transpose and dot products", []string{"ml"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := store.CreateScrap("paris travel", "eiffel tower tips", []string{"traveling"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Semantic payload lives on the tag description; a body that never mentions
	// "career" should still surface via the tag description search.
	if err := store.UpdateTagDescription("ml", "machine learning career goal"); err != nil {
		t.Fatalf("update tag description: %v", err)
	}

	// Title match.
	byTitle, err := store.SearchScraps("paris", "active")
	if err != nil {
		t.Fatalf("search by title: %v", err)
	}
	if len(byTitle) != 1 || byTitle[0].Title != "paris travel" {
		t.Errorf("title search = %+v, want paris travel", byTitle)
	}

	// Body match.
	byBody, err := store.SearchScraps("transpose", "active")
	if err != nil {
		t.Fatalf("search by body: %v", err)
	}
	if len(byBody) != 1 {
		t.Errorf("body search = %d hits, want 1", len(byBody))
	}

	// Tag-description match: query term that appears ONLY in the tag description.
	byTag, err := store.SearchScraps("career", "active")
	if err != nil {
		t.Fatalf("search by tag description: %v", err)
	}
	if len(byTag) != 1 || byTag[0].Title != "linear algebra first" {
		t.Errorf("tag-description search = %+v, want linear algebra first", byTag)
	}
}

func TestScrapUpdateAndDelete(t *testing.T) {
	store := newTestStore(t)
	mkTags(t, store, "a", "b")
	sc, err := store.CreateScrap("original", "old body", []string{"a"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Update body (status unchanged).
	body := "new body"
	upd, err := store.UpdateScrap(sc.Slug, nil, &body, nil, nil)
	if err != nil {
		t.Fatalf("update body: %v", err)
	}
	if upd.Body != "new body" || upd.Status != "active" {
		t.Errorf("after body update: body=%q status=%q", upd.Body, upd.Status)
	}

	// Change tag set: drop "a", add "b".
	if _, err := store.UpdateScrap(sc.Slug, nil, nil, nil, &[]string{"b"}); err != nil {
		t.Fatalf("update tags: %v", err)
	}
	// The Tag object itself is first-class — detaching it from a scrap does NOT
	// delete the tag. It just severs the association (scrap_tags row).
	if _, err := store.TagByName("a"); err != nil {
		t.Errorf("tag 'a' should persist after detaching from scrap, got error: %v", err)
	}

	// Delete cascades join rows; scrap is gone.
	if err := store.DeleteScrap(sc.Slug); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.ScrapBySlug(sc.Slug); err == nil {
		t.Errorf("scrap still exists after delete")
	}
}

func TestScrapSlugStableAcrossUpdate(t *testing.T) {
	store := newTestStore(t)
	sc, err := store.CreateScrap("stable title", "body", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	body := "updated body"
	upd, err := store.UpdateScrap(sc.Slug, nil, &body, nil, nil)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	// The slug must NOT change across updates — it is the stable handle for
	// find-then-update (the `reference` convention).
	if upd.Slug != sc.Slug {
		t.Errorf("slug changed across update: %q -> %q", sc.Slug, upd.Slug)
	}
	if upd.Body != "updated body" {
		t.Errorf("body after update = %q", upd.Body)
	}
}

func TestScrapTagsForScrap(t *testing.T) {
	store := newTestStore(t)
	mkTags(t, store, "gamma", "alpha", "beta")
	sc, err := store.CreateScrap("one", "body", []string{"gamma", "alpha", "beta"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tags, err := store.TagsForScrap(sc.Slug)
	if err != nil {
		t.Fatalf("tags for scrap: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("tags = %d, want 3", len(tags))
	}
	if tags[0].Name != "alpha" || tags[1].Name != "beta" || tags[2].Name != "gamma" {
		t.Errorf("tags not ordered by name: %+v", tags)
	}
}

func TestScrapAttachUnknownTagRejected(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateScrap("must fail", "body", []string{"does-not-exist"}); err == nil {
		t.Errorf("expected error for unknown tag, got nil (no silent auto-create)")
	} else if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("error should name the missing tag, got: %v", err)
	}
}

func TestScrapFPSCleanTitleProducesSlug(t *testing.T) {
	store := newTestStore(t)
	sc, err := store.CreateScrap(" Parities && Slashes / _ gone ", "body", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if strings.ContainsAny(sc.Slug, "/ &_ ") {
		t.Errorf("slug has unsafe chars: %q", sc.Slug)
	}
	if sc.Slug == "" {
		t.Errorf("slug empty")
	}
}
