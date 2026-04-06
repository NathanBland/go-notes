package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
	cacheclient "github.com/nathanbland/go-notes/internal/platform/cache"
	"github.com/nathanbland/go-notes/internal/platform/db"
)

func TestStackPostgresAndValkey(t *testing.T) {
	stack := newIntegrationStack(t)

	var timezone string
	if err := stack.pool.QueryRow(stack.ctx, "SHOW TIME ZONE").Scan(&timezone); err != nil {
		t.Fatalf("failed to read timezone: %v", err)
	}
	if timezone != "UTC" {
		t.Fatalf("expected UTC timezone, got %q", timezone)
	}

	if err := stack.cache.Set(stack.ctx, "integration:key", "value", time.Minute); err != nil {
		t.Fatalf("failed to set cache key: %v", err)
	}
	value, ok := stack.cache.Get(stack.ctx, "integration:key")
	if !ok || value != "value" {
		t.Fatalf("unexpected cache value ok=%v value=%q", ok, value)
	}
	if err := stack.cache.Delete(stack.ctx, "integration:key"); err != nil {
		t.Fatalf("failed to delete cache key: %v", err)
	}
}

func TestNoteCRUDAndCacheInvalidation(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "crud-user")

	title := "  Shared Title  "
	created, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &title,
		Content:     "  Shared content  ",
		Tags:        []string{"work", "team"},
		Shared:      true,
	})
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}
	if created.ShareSlug == nil || *created.ShareSlug == "" {
		t.Fatalf("expected share slug for shared note, got %+v", created)
	}

	noteKey := noteCacheKey(user.ID, created.ID)
	sharedKey := sharedNoteCacheKey(*created.ShareSlug)

	if payload, ok := stack.cache.Get(stack.ctx, noteKey); !ok || payload == "" {
		t.Fatalf("expected note cache entry after create")
	}
	if payload, ok := stack.cache.Get(stack.ctx, sharedKey); !ok || payload == "" {
		t.Fatalf("expected shared cache entry after create")
	}

	loaded, err := stack.notesService.GetByIDForOwner(stack.ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("failed to load note by id: %v", err)
	}
	if loaded.Content != "Shared content" {
		t.Fatalf("expected trimmed content, got %q", loaded.Content)
	}

	sharedLoaded, err := stack.notesService.GetByShareSlug(stack.ctx, *created.ShareSlug)
	if err != nil {
		t.Fatalf("failed to load shared note: %v", err)
	}
	if sharedLoaded.ID != created.ID {
		t.Fatalf("expected shared note %s, got %s", created.ID, sharedLoaded.ID)
	}

	updatedTitle := "Updated Title"
	updatedContent := "Updated body"
	updatedTags := []string{"team", "updated"}
	sharedFalse := false
	updated, err := stack.notesService.Patch(stack.ctx, notes.PatchInput{
		ID:          created.ID,
		OwnerUserID: user.ID,
		TitleSet:    true,
		Title:       &updatedTitle,
		ContentSet:  true,
		Content:     &updatedContent,
		TagsSet:     true,
		Tags:        &updatedTags,
		SharedSet:   true,
		Shared:      &sharedFalse,
	})
	if err != nil {
		t.Fatalf("failed to patch note: %v", err)
	}
	if updated.ShareSlug != nil {
		t.Fatalf("expected share slug to be cleared, got %+v", updated)
	}

	if _, ok := stack.cache.Get(stack.ctx, sharedKey); ok {
		t.Fatalf("expected old shared cache key to be invalidated")
	}
	if payload, ok := stack.cache.Get(stack.ctx, noteKey); !ok || payload == "" {
		t.Fatalf("expected note cache entry to be refreshed after patch")
	}

	if _, err := stack.notesService.GetByShareSlug(stack.ctx, *created.ShareSlug); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected old shared slug to be gone, got %v", err)
	}

	reloaded, err := stack.notesService.GetByIDForOwner(stack.ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("failed to reload patched note: %v", err)
	}
	if reloaded.Title == nil || *reloaded.Title != updatedTitle {
		t.Fatalf("expected updated title, got %+v", reloaded)
	}
	if reloaded.Content != updatedContent {
		t.Fatalf("expected updated content, got %q", reloaded.Content)
	}

	if err := stack.notesService.Delete(stack.ctx, user.ID, created.ID); err != nil {
		t.Fatalf("failed to delete note: %v", err)
	}
	if _, ok := stack.cache.Get(stack.ctx, noteKey); ok {
		t.Fatalf("expected note cache to be invalidated after delete")
	}
	if _, err := stack.notesService.GetByIDForOwner(stack.ctx, user.ID, created.ID); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected deleted note lookup to fail, got %v", err)
	}
}

func TestOwnerScopedNotesAreNotAccessibleByDifferentUser(t *testing.T) {
	stack := newIntegrationStack(t)
	owner := stack.createUser(t, "owner-user")
	other := stack.createUser(t, "other-user")

	title := "Private note"
	created, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: owner.ID,
		Title:       &title,
		Content:     "private content",
		Tags:        []string{"private"},
		Shared:      false,
	})
	if err != nil {
		t.Fatalf("failed to create owner note: %v", err)
	}

	if _, err := stack.notesService.GetByIDForOwner(stack.ctx, other.ID, created.ID); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected other user get-by-id to fail with not found, got %v", err)
	}

	revised := "should not work"
	if _, err := stack.notesService.Patch(stack.ctx, notes.PatchInput{
		ID:          created.ID,
		OwnerUserID: other.ID,
		ContentSet:  true,
		Content:     &revised,
	}); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected other user patch to fail with not found, got %v", err)
	}

	if err := stack.notesService.Delete(stack.ctx, other.ID, created.ID); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected other user delete to fail with not found, got %v", err)
	}

	ownerLoaded, err := stack.notesService.GetByIDForOwner(stack.ctx, owner.ID, created.ID)
	if err != nil {
		t.Fatalf("expected owner note to remain readable, got %v", err)
	}
	if ownerLoaded.Content != "private content" {
		t.Fatalf("expected owner note to remain unchanged, got %+v", ownerLoaded)
	}
}

func TestSavedQueryPersistenceAndReuse(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "saved-query-user")

	titleOne := "Work item"
	if _, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &titleOne,
		Content:     "golang planning note",
		Tags:        []string{"work", "planning"},
	}); err != nil {
		t.Fatalf("failed to create work note: %v", err)
	}

	titleTwo := "Personal item"
	if _, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &titleTwo,
		Content:     "weekend errands",
		Tags:        []string{"personal"},
	}); err != nil {
		t.Fatalf("failed to create personal note: %v", err)
	}

	saved, err := stack.notesService.CreateSavedQuery(stack.ctx, notes.CreateSavedQueryInput{
		OwnerUserID: user.ID,
		Name:        "Work search",
		Query: notes.EncodeListFilters(notes.ListFilters{
			OwnerUserID:   user.ID,
			Page:          1,
			PageSize:      20,
			SearchMode:    notes.SearchModeFTS,
			Status:        notes.StatusActive,
			TagMatchMode:  notes.TagMatchAny,
			SortField:     "relevance",
			SortDirection: "desc",
			Tags:          []string{"work"},
			Search:        strPtr("golang"),
		}),
	})
	if err != nil {
		t.Fatalf("failed to create saved query: %v", err)
	}
	if saved.ID == uuid.Nil || saved.Query == "" {
		t.Fatalf("expected saved query to persist with query payload, got %+v", saved)
	}

	items, err := stack.notesService.ListSavedQueries(stack.ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to list saved queries: %v", err)
	}
	if len(items) != 1 || items[0].ID != saved.ID {
		t.Fatalf("unexpected saved query list: %+v", items)
	}

	loaded, err := stack.notesService.GetSavedQuery(stack.ctx, user.ID, saved.ID)
	if err != nil {
		t.Fatalf("failed to get saved query: %v", err)
	}
	if loaded.Name != "Work search" {
		t.Fatalf("unexpected saved query: %+v", loaded)
	}

	if err := stack.notesService.DeleteSavedQuery(stack.ctx, user.ID, saved.ID); err != nil {
		t.Fatalf("failed to delete saved query: %v", err)
	}
	if _, err := stack.notesService.GetSavedQuery(stack.ctx, user.ID, saved.ID); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected deleted saved query to be gone, got %v", err)
	}
}

func TestListFiltersPaginationAndCaching(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "list-user")
	otherUser := stack.createUser(t, "other-user")

	firstTitle := "Alpha plan"
	first, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &firstTitle,
		Content:     "first integration note",
		Tags:        []string{"work", "alpha"},
		Shared:      true,
		ShareSlug:   strPtr("alpha-share"),
	})
	if err != nil {
		t.Fatalf("failed to seed first note: %v", err)
	}

	// Use the persisted database timestamp as the date-filter boundary instead
	// of local wall-clock time. That avoids flaky tests when PostgreSQL `now()`
	// and the test process clock do not line up to the same instant.
	checkpoint := first.CreatedAt.Add(time.Millisecond)
	time.Sleep(25 * time.Millisecond)

	secondTitle := "Bravo followup"
	second, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &secondTitle,
		Content:     "golang pagination example",
		Tags:        []string{"work", "beta"},
	})
	if err != nil {
		t.Fatalf("failed to seed second note: %v", err)
	}

	thirdTitle := "Zulu archive"
	third, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &thirdTitle,
		Content:     "archived note content",
		Tags:        []string{"personal"},
		Archived:    true,
	})
	if err != nil {
		t.Fatalf("failed to seed archived note: %v", err)
	}

	otherTitle := "Other owner"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: otherUser.ID,
		Title:       &otherTitle,
		Content:     "should never appear",
		Tags:        []string{"work"},
	}); err != nil {
		t.Fatalf("failed to seed other owner note: %v", err)
	}

	defaultFilters := notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusActive,
		SortField:     "updated_at",
		SortDirection: "desc",
	}
	defaultResult, err := stack.notesService.List(stack.ctx, defaultFilters)
	if err != nil {
		t.Fatalf("failed to list active notes: %v", err)
	}
	if defaultResult.Total != 2 {
		t.Fatalf("expected 2 active notes, got %d", defaultResult.Total)
	}
	if len(defaultResult.Notes) != 2 {
		t.Fatalf("expected 2 notes in first page, got %d", len(defaultResult.Notes))
	}
	if defaultResult.Notes[0].ID != second.ID {
		t.Fatalf("expected newest active note first, got %s", defaultResult.Notes[0].ID)
	}

	listKey := listCacheKey(defaultFilters)
	if payload, ok := stack.cache.Get(stack.ctx, listKey); !ok || payload == "" {
		t.Fatalf("expected list cache entry for default filter set")
	}

	search := "golang"
	searchResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
		Search:        &search,
	})
	if err != nil {
		t.Fatalf("failed to search notes: %v", err)
	}
	if searchResult.Total != 1 || len(searchResult.Notes) != 1 || searchResult.Notes[0].ID != second.ID {
		t.Fatalf("unexpected search result: %+v", searchResult)
	}

	sharedTrue := true
	tagSharedResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAll,
		SortField:     "updated_at",
		SortDirection: "desc",
		Tags:          []string{"alpha"},
		Shared:        &sharedTrue,
	})
	if err != nil {
		t.Fatalf("failed to filter by tag/shared: %v", err)
	}
	if tagSharedResult.Total != 1 || len(tagSharedResult.Notes) != 1 || tagSharedResult.Notes[0].ID != first.ID {
		t.Fatalf("unexpected tag/shared result: %+v", tagSharedResult)
	}

	anyTagResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
		Tags:          []string{"alpha", "beta"},
	})
	if err != nil {
		t.Fatalf("failed to filter by any-tag: %v", err)
	}
	if anyTagResult.Total != 2 {
		t.Fatalf("expected 2 notes for any-tag filter, got %+v", anyTagResult)
	}

	allTagResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAll,
		SortField:     "updated_at",
		SortDirection: "desc",
		Tags:          []string{"work", "alpha"},
	})
	if err != nil {
		t.Fatalf("failed to filter by all-tags: %v", err)
	}
	if allTagResult.Total != 1 || len(allTagResult.Notes) != 1 || allTagResult.Notes[0].ID != first.ID {
		t.Fatalf("unexpected all-tag result: %+v", allTagResult)
	}

	titleWithinTagResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "title",
		SortDirection: "asc",
		Tags:          []string{"work"},
	})
	if err != nil {
		t.Fatalf("failed to sort tag-filtered notes by title: %v", err)
	}
	if titleWithinTagResult.Total != 2 || len(titleWithinTagResult.Notes) != 2 {
		t.Fatalf("expected two work-tagged notes sorted by title, got %+v", titleWithinTagResult)
	}
	if titleWithinTagResult.Notes[0].Title == nil || *titleWithinTagResult.Notes[0].Title != "Alpha plan" {
		t.Fatalf("expected Alpha plan first in tag-filtered title sort, got %+v", titleWithinTagResult.Notes)
	}
	if titleWithinTagResult.Notes[1].Title == nil || *titleWithinTagResult.Notes[1].Title != "Bravo followup" {
		t.Fatalf("expected Bravo followup second in tag-filtered title sort, got %+v", titleWithinTagResult.Notes)
	}

	createdAfter := checkpoint
	dateResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "created_at",
		SortDirection: "asc",
		CreatedAfter:  &createdAfter,
	})
	if err != nil {
		t.Fatalf("failed to filter by date: %v", err)
	}
	if dateResult.Total != 2 || len(dateResult.Notes) != 2 {
		t.Fatalf("expected two notes after checkpoint, got %+v", dateResult)
	}

	paged, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          2,
		PageSize:      1,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "title",
		SortDirection: "asc",
	})
	if err != nil {
		t.Fatalf("failed to page notes: %v", err)
	}
	if paged.Total != 3 || len(paged.Notes) != 1 {
		t.Fatalf("expected second page with one row out of three, got %+v", paged)
	}
	if paged.Notes[0].Title == nil || *paged.Notes[0].Title != "Bravo followup" {
		t.Fatalf("expected title-sorted second page to be Bravo, got %+v", paged.Notes[0])
	}

	tagCountSorted, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "tag_count",
		SortDirection: "desc",
	})
	if err != nil {
		t.Fatalf("failed to sort by tag count: %v", err)
	}
	if len(tagCountSorted.Notes) < 2 || len(tagCountSorted.Notes[0].Tags) < len(tagCountSorted.Notes[1].Tags) {
		t.Fatalf("expected tag_count desc ordering, got %+v", tagCountSorted.Notes)
	}

	primaryTagSorted, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "primary_tag",
		SortDirection: "asc",
	})
	if err != nil {
		t.Fatalf("failed to sort by primary tag: %v", err)
	}
	if len(primaryTagSorted.Notes) != 3 {
		t.Fatalf("expected three notes in primary tag sort, got %+v", primaryTagSorted.Notes)
	}
	if primaryTagSorted.Notes[0].ID != third.ID {
		t.Fatalf("expected primary_tag asc ordering to follow first stored tag, got %+v", primaryTagSorted.Notes)
	}
	if len(primaryTagSorted.Notes[1].Tags) == 0 || primaryTagSorted.Notes[1].Tags[0] != "work" {
		t.Fatalf("expected second note to have work as the primary tag, got %+v", primaryTagSorted.Notes[1])
	}
	if len(primaryTagSorted.Notes[2].Tags) == 0 || primaryTagSorted.Notes[2].Tags[0] != "work" {
		t.Fatalf("expected third note to have work as the primary tag, got %+v", primaryTagSorted.Notes[2])
	}
}

func TestListTagsReturnsOwnerScopedCounts(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "tag-owner")
	other := stack.createUser(t, "tag-other")

	firstTitle := "First"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &firstTitle,
		Content:     "first",
		Tags:        []string{"go", "mcp"},
	}); err != nil {
		t.Fatalf("failed to seed first tagged note: %v", err)
	}

	secondTitle := "Second"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &secondTitle,
		Content:     "second",
		Tags:        []string{"go", "sql"},
	}); err != nil {
		t.Fatalf("failed to seed second tagged note: %v", err)
	}

	otherTitle := "Other"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: other.ID,
		Title:       &otherTitle,
		Content:     "other",
		Tags:        []string{"private"},
	}); err != nil {
		t.Fatalf("failed to seed other tagged note: %v", err)
	}

	summary, err := stack.notesService.ListTags(stack.ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	if len(summary) != 3 {
		t.Fatalf("expected 3 owner-scoped tags, got %+v", summary)
	}
	if summary[0].Tag != "go" || summary[0].Count != 2 {
		t.Fatalf("expected go to lead with count 2, got %+v", summary)
	}
	for _, item := range summary {
		if item.Tag == "private" {
			t.Fatalf("expected other-user tags to be excluded, got %+v", summary)
		}
	}
}

func TestRenameTagIsOwnerScopedAndRefreshesCaches(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "rename-owner")
	other := stack.createUser(t, "rename-other")

	titleOne := "Planning one"
	first, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &titleOne,
		Content:     "first",
		Tags:        []string{"planning", "go", "planning"},
	})
	if err != nil {
		t.Fatalf("failed to create first note: %v", err)
	}

	titleTwo := "Planning two"
	second, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &titleTwo,
		Content:     "second",
		Tags:        []string{"mcp", "planning"},
		Shared:      true,
	})
	if err != nil {
		t.Fatalf("failed to create second note: %v", err)
	}

	otherTitle := "Other planning"
	otherNote, err := stack.notesService.Create(stack.ctx, notes.CreateInput{
		OwnerUserID: other.ID,
		Title:       &otherTitle,
		Content:     "other",
		Tags:        []string{"planning"},
	})
	if err != nil {
		t.Fatalf("failed to create other note: %v", err)
	}

	result, err := stack.notesService.RenameTag(stack.ctx, user.ID, "planning", "roadmap")
	if err != nil {
		t.Fatalf("failed to rename tag: %v", err)
	}
	if result.AffectedNotes != 2 {
		t.Fatalf("expected 2 affected notes, got %+v", result)
	}

	reloadedFirst, err := stack.notesService.GetByIDForOwner(stack.ctx, user.ID, first.ID)
	if err != nil {
		t.Fatalf("failed to reload first note: %v", err)
	}
	if len(reloadedFirst.Tags) != 2 || reloadedFirst.Tags[0] != "roadmap" || reloadedFirst.Tags[1] != "go" {
		t.Fatalf("expected ordered deduped tags on first note, got %+v", reloadedFirst.Tags)
	}

	reloadedSecond, err := stack.notesService.GetByIDForOwner(stack.ctx, user.ID, second.ID)
	if err != nil {
		t.Fatalf("failed to reload second note: %v", err)
	}
	if len(reloadedSecond.Tags) != 2 || reloadedSecond.Tags[1] != "roadmap" {
		t.Fatalf("expected renamed tag on second note, got %+v", reloadedSecond.Tags)
	}

	otherReloaded, err := stack.notesService.GetByIDForOwner(stack.ctx, other.ID, otherNote.ID)
	if err != nil {
		t.Fatalf("failed to reload other note: %v", err)
	}
	if len(otherReloaded.Tags) != 1 || otherReloaded.Tags[0] != "planning" {
		t.Fatalf("expected other owner's tags to remain unchanged, got %+v", otherReloaded.Tags)
	}

	if payload, ok := stack.cache.Get(stack.ctx, noteCacheKey(user.ID, first.ID)); !ok || !strings.Contains(payload, "roadmap") {
		t.Fatalf("expected renamed first note to refresh cache, ok=%v payload=%q", ok, payload)
	}
	if second.ShareSlug != nil {
		if payload, ok := stack.cache.Get(stack.ctx, sharedNoteCacheKey(*second.ShareSlug)); !ok || !strings.Contains(payload, "roadmap") {
			t.Fatalf("expected renamed shared note to refresh shared cache, ok=%v payload=%q", ok, payload)
		}
	}
}

func TestFindRelatedNotes(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "related-user")
	other := stack.createUser(t, "related-other")

	sourceTitle := "Source note"
	source, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &sourceTitle,
		Content:     "source",
		Tags:        []string{"go", "mcp", "sql"},
	})
	if err != nil {
		t.Fatalf("failed to seed source note: %v", err)
	}

	firstTitle := "Alpha related"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &firstTitle,
		Content:     "first",
		Tags:        []string{"go", "mcp"},
	}); err != nil {
		t.Fatalf("failed to seed first related note: %v", err)
	}

	secondTitle := "Beta related"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &secondTitle,
		Content:     "second",
		Tags:        []string{"go", "notes"},
	}); err != nil {
		t.Fatalf("failed to seed second related note: %v", err)
	}

	otherTitle := "Other user overlap"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: other.ID,
		Title:       &otherTitle,
		Content:     "other",
		Tags:        []string{"go", "mcp"},
	}); err != nil {
		t.Fatalf("failed to seed other-user related note: %v", err)
	}

	unrelatedTitle := "Unrelated"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &unrelatedTitle,
		Content:     "unrelated",
		Tags:        []string{"personal"},
	}); err != nil {
		t.Fatalf("failed to seed unrelated note: %v", err)
	}

	items, err := stack.notesService.FindRelatedNotes(stack.ctx, user.ID, source.ID, 5)
	if err != nil {
		t.Fatalf("failed to find related notes: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 related notes, got %+v", items)
	}
	if items[0].Note.Title == nil || *items[0].Note.Title != firstTitle || items[0].SharedTagCount != 2 {
		t.Fatalf("unexpected first related note: %+v", items[0])
	}
	if len(items[0].SharedTags) != 2 || items[0].SharedTags[0] != "go" || items[0].SharedTags[1] != "mcp" {
		t.Fatalf("unexpected first shared tags: %+v", items[0].SharedTags)
	}
	if items[1].Note.Title == nil || *items[1].Note.Title != secondTitle || items[1].SharedTagCount != 1 {
		t.Fatalf("unexpected second related note: %+v", items[1])
	}
}

func TestAdvancedSearchAndFilters(t *testing.T) {
	stack := newIntegrationStack(t)
	user := stack.createUser(t, "search-user")

	alphaTitle := "API design"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &alphaTitle,
		Content:     "postgres full text search with ranking",
		Tags:        []string{"postgres", "search", "teaching"},
	}); err != nil {
		t.Fatalf("failed to seed ranked note: %v", err)
	}

	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Content:     "search notes without a title",
		Tags:        []string{"search"},
	}); err != nil {
		t.Fatalf("failed to seed untitled note: %v", err)
	}

	betaTitle := "Search basics"
	if _, err := stack.store.CreateNote(stack.ctx, notes.CreateInput{
		OwnerUserID: user.ID,
		Title:       &betaTitle,
		Content:     "plain substring matching for note search",
		Tags:        []string{"search", "basics"},
	}); err != nil {
		t.Fatalf("failed to seed plain-search note: %v", err)
	}

	plain := "substring"
	plainResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Search:        &plain,
		SearchMode:    notes.SearchModePlain,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
	})
	if err != nil {
		t.Fatalf("failed plain search: %v", err)
	}
	if plainResult.Total != 1 || len(plainResult.Notes) != 1 || plainResult.Notes[0].Title == nil || *plainResult.Notes[0].Title != betaTitle {
		t.Fatalf("unexpected plain search result: %+v", plainResult)
	}

	fts := "\"full text search\" ranking"
	ftsResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Search:        &fts,
		SearchMode:    notes.SearchModeFTS,
		Status:        notes.StatusAll,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "relevance",
		SortDirection: "desc",
	})
	if err != nil {
		t.Fatalf("failed fts search: %v", err)
	}
	if ftsResult.Total != 1 || len(ftsResult.Notes) != 1 || ftsResult.Notes[0].Title == nil || *ftsResult.Notes[0].Title != alphaTitle {
		t.Fatalf("unexpected fts search result: %+v", ftsResult)
	}

	hasTitle := false
	untitledResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		HasTitle:      &hasTitle,
		SearchMode:    notes.SearchModePlain,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
	})
	if err != nil {
		t.Fatalf("failed has_title filter: %v", err)
	}
	if untitledResult.Total != 1 || len(untitledResult.Notes) != 1 || untitledResult.Notes[0].Title != nil {
		t.Fatalf("unexpected untitled filter result: %+v", untitledResult)
	}

	minTags := int32(2)
	maxTags := int32(2)
	tagCountResult, err := stack.notesService.List(stack.ctx, notes.ListFilters{
		OwnerUserID:   user.ID,
		Page:          1,
		PageSize:      20,
		Status:        notes.StatusAll,
		SearchMode:    notes.SearchModePlain,
		TagMatchMode:  notes.TagMatchAny,
		TagCountMin:   &minTags,
		TagCountMax:   &maxTags,
		SortField:     "title",
		SortDirection: "asc",
	})
	if err != nil {
		t.Fatalf("failed tag count filter: %v", err)
	}
	if tagCountResult.Total != 1 || len(tagCountResult.Notes) != 1 || tagCountResult.Notes[0].Title == nil || *tagCountResult.Notes[0].Title != betaTitle {
		t.Fatalf("unexpected tag count filter result: %+v", tagCountResult)
	}
}

type integrationStack struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	store        *db.Store
	cache        *cacheclient.Client
	notesService *notes.Service
}

func newIntegrationStack(t *testing.T) *integrationStack {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	databaseURL := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable")
	valkeyAddr := getenv("VALKEY_ADDR", "127.0.0.1:6379")

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE notes, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("failed to reset tables: %v", err)
	}

	cache, err := cacheclient.New(valkeyAddr, "")
	if err != nil {
		t.Fatalf("failed to connect to valkey: %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Close()
	})

	store := db.NewStore(pool)
	return &integrationStack{
		ctx:          ctx,
		pool:         pool,
		store:        store,
		cache:        cache,
		notesService: notes.NewService(store, cache, time.Minute, time.Minute),
	}
}

func (s *integrationStack) createUser(t *testing.T, subject string) auth.User {
	t.Helper()
	user, err := s.store.UpsertUserFromOIDC(s.ctx, auth.Identity{
		Issuer:      "https://issuer.example",
		Subject:     subject,
		DisplayName: strPtr("Integration User " + subject),
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func noteCacheKey(ownerID, noteID uuid.UUID) string {
	return fmt.Sprintf("notes:id:%s:%s", ownerID, noteID)
}

func sharedNoteCacheKey(slug string) string {
	return "notes:shared:" + slug
}

func listCacheKey(filters notes.ListFilters) string {
	if filters.SearchMode == "" {
		filters.SearchMode = notes.SearchModePlain
	}
	payload, _ := json.Marshal(filters)
	hash := sha256.Sum256(payload)
	return fmt.Sprintf("notes:list:%s:%s", filters.OwnerUserID, hex.EncodeToString(hash[:]))
}

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func strPtr(value string) *string {
	return &value
}
