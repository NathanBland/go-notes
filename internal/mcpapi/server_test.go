package mcpapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/nathanbland/go-notes/internal/notes"
)

type fakeNotesService struct {
	listResult  notes.ListResult
	listErr     error
	tagResult   []notes.TagSummary
	tagErr      error
	related     []notes.RelatedNote
	relatedErr  error
	gotNote     notes.Note
	getErr      error
	created     notes.Note
	createErr   error
	savedQuery  notes.SavedQuery
	savedList   []notes.SavedQuery
	savedErr    error
	lastSaved   notes.CreateSavedQueryInput
	lastSavedID uuid.UUID
	updated     notes.Note
	patchErr    error
	deleteErr   error
	lastCreate  notes.CreateInput
	lastList    notes.ListFilters
	lastGetID   uuid.UUID
	lastOwner   uuid.UUID
	lastPatch   notes.PatchInput
	lastDelete  uuid.UUID
}

func (f *fakeNotesService) Create(ctx context.Context, input notes.CreateInput) (notes.Note, error) {
	f.lastCreate = input
	if f.createErr != nil {
		return notes.Note{}, f.createErr
	}
	return f.created, nil
}

func (f *fakeNotesService) CreateSavedQuery(ctx context.Context, input notes.CreateSavedQueryInput) (notes.SavedQuery, error) {
	f.lastSaved = input
	if f.savedErr != nil {
		return notes.SavedQuery{}, f.savedErr
	}
	return f.savedQuery, nil
}

func (f *fakeNotesService) Delete(ctx context.Context, ownerUserID, noteID uuid.UUID) error {
	f.lastOwner = ownerUserID
	f.lastDelete = noteID
	return f.deleteErr
}

func (f *fakeNotesService) DeleteSavedQuery(ctx context.Context, ownerUserID, savedQueryID uuid.UUID) error {
	f.lastOwner = ownerUserID
	f.lastSavedID = savedQueryID
	return f.savedErr
}

func (f *fakeNotesService) FindRelatedNotes(ctx context.Context, ownerUserID, noteID uuid.UUID, limit int32) ([]notes.RelatedNote, error) {
	f.lastOwner = ownerUserID
	f.lastGetID = noteID
	if f.relatedErr != nil {
		return nil, f.relatedErr
	}
	return f.related, nil
}

func (f *fakeNotesService) GetByIDForOwner(ctx context.Context, ownerUserID, noteID uuid.UUID) (notes.Note, error) {
	f.lastOwner = ownerUserID
	f.lastGetID = noteID
	if f.getErr != nil {
		return notes.Note{}, f.getErr
	}
	return f.gotNote, nil
}

func (f *fakeNotesService) GetSavedQuery(ctx context.Context, ownerUserID, savedQueryID uuid.UUID) (notes.SavedQuery, error) {
	f.lastOwner = ownerUserID
	f.lastSavedID = savedQueryID
	if f.savedErr != nil {
		return notes.SavedQuery{}, f.savedErr
	}
	return f.savedQuery, nil
}

func (f *fakeNotesService) List(ctx context.Context, filters notes.ListFilters) (notes.ListResult, error) {
	f.lastList = filters
	if f.listErr != nil {
		return notes.ListResult{}, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeNotesService) ListSavedQueries(ctx context.Context, ownerUserID uuid.UUID) ([]notes.SavedQuery, error) {
	f.lastOwner = ownerUserID
	if f.savedErr != nil {
		return nil, f.savedErr
	}
	return f.savedList, nil
}

func (f *fakeNotesService) ListTags(ctx context.Context, ownerUserID uuid.UUID) ([]notes.TagSummary, error) {
	f.lastOwner = ownerUserID
	if f.tagErr != nil {
		return nil, f.tagErr
	}
	return f.tagResult, nil
}

func (f *fakeNotesService) Patch(ctx context.Context, input notes.PatchInput) (notes.Note, error) {
	f.lastPatch = input
	if f.patchErr != nil {
		return notes.Note{}, f.patchErr
	}
	return f.updated, nil
}

func TestNormalizeListArgs(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	shared := true
	filters, err := normalizeListArgs(ownerID, listNotesArgs{
		Page:          intPtr(2),
		PageSize:      intPtr(10),
		Search:        stringPtr("golang"),
		SearchMode:    stringPtr(notes.SearchModeFTS),
		Status:        stringPtr(notes.StatusAll),
		Shared:        &shared,
		HasTitle:      &shared,
		Tag:           stringPtr("work"),
		Tags:          []string{"alpha", "work"},
		TagCountMin:   intPtr(1),
		TagCountMax:   intPtr(4),
		TagMode:       stringPtr(notes.TagMatchAll),
		Sort:          stringPtr("relevance"),
		Order:         stringPtr("asc"),
		CreatedAfter:  stringPtr("2026-04-01T00:00:00-07:00"),
		UpdatedBefore: stringPtr("2026-04-02T00:00:00-07:00"),
	})
	if err != nil {
		t.Fatalf("unexpected normalize error: %v", err)
	}
	if filters.OwnerUserID != ownerID || filters.Page != 2 || filters.PageSize != 10 {
		t.Fatalf("unexpected pagination filters: %+v", filters)
	}
	if filters.Search == nil || *filters.Search != "golang" || len(filters.Tags) != 2 || filters.Tags[0] != "work" || filters.Tags[1] != "alpha" {
		t.Fatalf("unexpected optional filters: %+v", filters)
	}
	if filters.Shared == nil || !*filters.Shared || filters.SearchMode != notes.SearchModeFTS || filters.HasTitle == nil || !*filters.HasTitle || filters.TagCountMin == nil || *filters.TagCountMin != 1 || filters.TagCountMax == nil || *filters.TagCountMax != 4 || filters.TagMatchMode != notes.TagMatchAll || filters.SortField != "relevance" || filters.SortDirection != "asc" {
		t.Fatalf("unexpected shared/sort filters: %+v", filters)
	}
	if filters.CreatedAfter == nil || filters.CreatedAfter.Location() != time.UTC {
		t.Fatalf("expected UTC created_after filter, got %+v", filters.CreatedAfter)
	}
}

func TestNormalizeListArgsValidation(t *testing.T) {
	ownerID := uuid.New()
	for _, args := range []listNotesArgs{
		{Page: intPtr(-1)},
		{PageSize: intPtr(101)},
		{Status: stringPtr("bad")},
		{SearchMode: stringPtr("bad")},
		{TagMode: stringPtr("bad")},
		{TagCountMin: intPtr(-1)},
		{Sort: stringPtr("bad")},
		{Sort: stringPtr("relevance"), Search: stringPtr("go")},
		{Order: stringPtr("bad")},
		{CreatedAfter: stringPtr("nope")},
	} {
		if _, err := normalizeListArgs(ownerID, args); err == nil {
			t.Fatalf("expected validation error for %+v", args)
		}
	}
}

func TestHandleListNotes(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	service := &fakeNotesService{
		listResult: notes.ListResult{
			Notes: []notes.Note{{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), OwnerUserID: ownerID, Content: "hello"}},
			Total: 1,
			Filters: notes.ListFilters{
				OwnerUserID:   ownerID,
				Page:          1,
				PageSize:      20,
				Status:        notes.StatusActive,
				SortField:     "updated_at",
				SortDirection: "desc",
			},
		},
	}
	api := &Server{notes: service, ownerID: ownerID}

	result, err := api.handleListNotes(context.Background(), mcp.CallToolRequest{}, listNotesArgs{})
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful tool result, got %+v", result)
	}
	if service.lastList.OwnerUserID != ownerID {
		t.Fatalf("expected owner-scoped list, got %+v", service.lastList)
	}
}

func TestHandleGetNote(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	service := &fakeNotesService{gotNote: notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "hello"}}
	api := &Server{notes: service, ownerID: ownerID}

	result, err := api.handleGetNote(context.Background(), mcp.CallToolRequest{}, getNoteArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if result.IsError || service.lastGetID != noteID || service.lastOwner != ownerID {
		t.Fatalf("unexpected result=%+v service=%+v", result, service)
	}

	result, err = api.handleGetNote(context.Background(), mcp.CallToolRequest{}, getNoteArgs{ID: "not-a-uuid"})
	if err != nil {
		t.Fatalf("unexpected invalid-id error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected invalid id to return tool error")
	}

	service.getErr = notes.ErrNotFound
	result, err = api.handleGetNote(context.Background(), mcp.CallToolRequest{}, getNoteArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected not-found handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected not found to return tool error")
	}
}

func TestHandleFindRelatedNotes(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	service := &fakeNotesService{
		related: []notes.RelatedNote{{
			Note:           notes.Note{ID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), OwnerUserID: ownerID, Content: "related"},
			SharedTags:     []string{"go", "mcp"},
			SharedTagCount: 2,
		}},
	}
	api := &Server{notes: service, ownerID: ownerID}

	result, err := api.handleFindRelatedNotes(context.Background(), mcp.CallToolRequest{}, findRelatedNotesArgs{ID: noteID.String(), Limit: intPtr(3)})
	if err != nil {
		t.Fatalf("unexpected related handler error: %v", err)
	}
	if result.IsError || service.lastOwner != ownerID || service.lastGetID != noteID {
		t.Fatalf("unexpected related result=%+v service=%+v", result, service)
	}

	result, err = api.handleFindRelatedNotes(context.Background(), mcp.CallToolRequest{}, findRelatedNotesArgs{ID: "not-a-uuid"})
	if err != nil {
		t.Fatalf("unexpected invalid-id error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected invalid id to return tool error")
	}

	result, err = api.handleFindRelatedNotes(context.Background(), mcp.CallToolRequest{}, findRelatedNotesArgs{ID: noteID.String(), Limit: intPtr(21)})
	if err != nil {
		t.Fatalf("unexpected limit validation error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected invalid limit to return tool error")
	}

	service.relatedErr = notes.ErrNotFound
	result, err = api.handleFindRelatedNotes(context.Background(), mcp.CallToolRequest{}, findRelatedNotesArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected not-found related handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected missing source note to return tool error")
	}
}

func TestHandleCreateNote(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	createdID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	service := &fakeNotesService{created: notes.Note{ID: createdID, OwnerUserID: ownerID, Content: "body"}}
	api := &Server{notes: service, ownerID: ownerID}

	title := "  hello  "
	result, err := api.handleCreateNote(context.Background(), mcp.CallToolRequest{}, createNoteArgs{
		Title:   &title,
		Content: " body ",
		Tags:    []string{"go"},
		Shared:  true,
	})
	if err != nil {
		t.Fatalf("unexpected create handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected create success, got %+v", result)
	}
	if service.lastCreate.OwnerUserID != ownerID || service.lastCreate.Title == nil || *service.lastCreate.Title != "hello" {
		t.Fatalf("unexpected create input: %+v", service.lastCreate)
	}
	if len(service.lastCreate.Tags) != 1 || service.lastCreate.Tags[0] != "go" {
		t.Fatalf("expected normalized create tags, got %+v", service.lastCreate.Tags)
	}

	result, err = api.handleCreateNote(context.Background(), mcp.CallToolRequest{}, createNoteArgs{})
	if err != nil {
		t.Fatalf("unexpected blank-content handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected blank content to return tool error")
	}

	service.createErr = errors.New("boom")
	result, err = api.handleCreateNote(context.Background(), mcp.CallToolRequest{}, createNoteArgs{Content: "body"})
	if err != nil {
		t.Fatalf("unexpected create failure handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected create failure to return tool error")
	}
}

func TestHandleUpdateNote(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	updated := notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "after", Tags: []string{"go", "mcp"}}
	service := &fakeNotesService{updated: updated}
	api := &Server{notes: service, ownerID: ownerID}

	title := "  Revised title  "
	content := "  after  "
	shared := true
	result, err := api.handleUpdateNote(context.Background(), mcp.CallToolRequest{}, updateNoteArgs{
		ID:          noteID.String(),
		Title:       &title,
		Content:     &content,
		Tags:        []string{"go", "mcp", "go"},
		ReplaceTags: true,
		Shared:      &shared,
	})
	if err != nil {
		t.Fatalf("unexpected update handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful update result, got %+v", result)
	}
	if !service.lastPatch.TitleSet || service.lastPatch.Title == nil || *service.lastPatch.Title != "Revised title" {
		t.Fatalf("unexpected title patch input: %+v", service.lastPatch)
	}
	if !service.lastPatch.ContentSet || service.lastPatch.Content == nil || *service.lastPatch.Content != "after" {
		t.Fatalf("unexpected content patch input: %+v", service.lastPatch)
	}
	if !service.lastPatch.TagsSet || service.lastPatch.Tags == nil || len(*service.lastPatch.Tags) != 2 {
		t.Fatalf("unexpected tag patch input: %+v", service.lastPatch)
	}
	if !service.lastPatch.SharedSet || service.lastPatch.Shared == nil || !*service.lastPatch.Shared {
		t.Fatalf("unexpected shared patch input: %+v", service.lastPatch)
	}

	result, err = api.handleUpdateNote(context.Background(), mcp.CallToolRequest{}, updateNoteArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected empty update handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected update without changes to fail")
	}

	blank := "   "
	result, err = api.handleUpdateNote(context.Background(), mcp.CallToolRequest{}, updateNoteArgs{ID: noteID.String(), Content: &blank})
	if err != nil {
		t.Fatalf("unexpected blank content handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected blank content update to fail")
	}

	service.patchErr = notes.ErrNotFound
	result, err = api.handleUpdateNote(context.Background(), mcp.CallToolRequest{}, updateNoteArgs{ID: noteID.String(), ClearTitle: true})
	if err != nil {
		t.Fatalf("unexpected not-found update handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected not found update to return tool error")
	}
}

func TestHandleModifyNoteTags(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	current := notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "body", Tags: []string{"go", "notes"}}
	service := &fakeNotesService{
		gotNote: current,
		updated: notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "body", Tags: []string{"go", "notes", "mcp"}},
	}
	api := &Server{notes: service, ownerID: ownerID}

	result, err := api.handleAddNoteTags(context.Background(), mcp.CallToolRequest{}, modifyNoteTagsArgs{
		ID:   noteID.String(),
		Tags: []string{"mcp", "go"},
	})
	if err != nil {
		t.Fatalf("unexpected add tags error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected add tags success, got %+v", result)
	}
	if service.lastPatch.Tags == nil || len(*service.lastPatch.Tags) != 3 || (*service.lastPatch.Tags)[2] != "mcp" {
		t.Fatalf("unexpected merged tags: %+v", service.lastPatch.Tags)
	}

	service.gotNote = notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "body", Tags: []string{"go", "notes", "mcp"}}
	service.updated = notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "body", Tags: []string{"go"}}
	result, err = api.handleRemoveNoteTags(context.Background(), mcp.CallToolRequest{}, modifyNoteTagsArgs{
		ID:   noteID.String(),
		Tags: []string{"notes", "mcp"},
	})
	if err != nil {
		t.Fatalf("unexpected remove tags error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected remove tags success, got %+v", result)
	}
	if service.lastPatch.Tags == nil || len(*service.lastPatch.Tags) != 1 || (*service.lastPatch.Tags)[0] != "go" {
		t.Fatalf("unexpected trimmed tags: %+v", service.lastPatch.Tags)
	}

	result, err = api.handleAddNoteTags(context.Background(), mcp.CallToolRequest{}, modifyNoteTagsArgs{
		ID:   noteID.String(),
		Tags: []string{"   "},
	})
	if err != nil {
		t.Fatalf("unexpected empty tag handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected empty tag add to fail")
	}

	service.getErr = notes.ErrNotFound
	result, err = api.handleRemoveNoteTags(context.Background(), mcp.CallToolRequest{}, modifyNoteTagsArgs{
		ID:   noteID.String(),
		Tags: []string{"go"},
	})
	if err != nil {
		t.Fatalf("unexpected not-found remove handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected missing note to return tool error")
	}
}

func TestHandleDeleteAndStateTools(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	service := &fakeNotesService{
		updated: notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "body", Shared: true, Archived: true},
	}
	api := &Server{notes: service, ownerID: ownerID}

	deleteResult, err := api.handleDeleteNote(context.Background(), mcp.CallToolRequest{}, noteIDArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected delete handler error: %v", err)
	}
	if deleteResult.IsError || service.lastDelete != noteID || service.lastOwner != ownerID {
		t.Fatalf("unexpected delete result=%+v service=%+v", deleteResult, service)
	}

	shareResult, err := api.handleShareNote(context.Background(), mcp.CallToolRequest{}, noteIDArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected share handler error: %v", err)
	}
	if shareResult.IsError || !service.lastPatch.SharedSet || service.lastPatch.Shared == nil || !*service.lastPatch.Shared {
		t.Fatalf("unexpected share patch input: %+v result=%+v", service.lastPatch, shareResult)
	}

	unshareResult, err := api.handleUnshareNote(context.Background(), mcp.CallToolRequest{}, noteIDArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected unshare handler error: %v", err)
	}
	if unshareResult.IsError || service.lastPatch.Shared == nil || *service.lastPatch.Shared {
		t.Fatalf("unexpected unshare patch input: %+v result=%+v", service.lastPatch, unshareResult)
	}

	archiveResult, err := api.handleArchiveNote(context.Background(), mcp.CallToolRequest{}, noteIDArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected archive handler error: %v", err)
	}
	if archiveResult.IsError || !service.lastPatch.ArchivedSet || service.lastPatch.Archived == nil || !*service.lastPatch.Archived {
		t.Fatalf("unexpected archive patch input: %+v result=%+v", service.lastPatch, archiveResult)
	}

	unarchiveResult, err := api.handleUnarchiveNote(context.Background(), mcp.CallToolRequest{}, noteIDArgs{ID: noteID.String()})
	if err != nil {
		t.Fatalf("unexpected unarchive handler error: %v", err)
	}
	if unarchiveResult.IsError || service.lastPatch.Archived == nil || *service.lastPatch.Archived {
		t.Fatalf("unexpected unarchive patch input: %+v result=%+v", service.lastPatch, unarchiveResult)
	}
}

func TestHandleSetNoteTagsAndListTags(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	service := &fakeNotesService{
		updated: notes.Note{ID: noteID, OwnerUserID: ownerID, Content: "body", Tags: []string{"go", "mcp"}},
		tagResult: []notes.TagSummary{
			{Tag: "go", Count: 3},
			{Tag: "mcp", Count: 1},
		},
	}
	api := &Server{notes: service, ownerID: ownerID}

	setResult, err := api.handleSetNoteTags(context.Background(), mcp.CallToolRequest{}, modifyNoteTagsArgs{
		ID:   noteID.String(),
		Tags: []string{" go ", "mcp", "go"},
	})
	if err != nil {
		t.Fatalf("unexpected set tags error: %v", err)
	}
	if setResult.IsError || service.lastPatch.Tags == nil || len(*service.lastPatch.Tags) != 2 {
		t.Fatalf("unexpected set tags result=%+v patch=%+v", setResult, service.lastPatch)
	}

	listResult, err := api.handleListTags(context.Background(), mcp.CallToolRequest{}, struct{}{})
	if err != nil {
		t.Fatalf("unexpected list tags error: %v", err)
	}
	if listResult.IsError || service.lastOwner != ownerID {
		t.Fatalf("unexpected list tags result=%+v service=%+v", listResult, service)
	}
}

func TestSavedQueryHandlers(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	savedID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	service := &fakeNotesService{
		savedQuery: notes.SavedQuery{
			ID:          savedID,
			OwnerUserID: ownerID,
			Name:        "Ranked work",
			Query:       "search=golang&search_mode=fts&tag=work&sort=relevance",
		},
		savedList: []notes.SavedQuery{{
			ID:          savedID,
			OwnerUserID: ownerID,
			Name:        "Ranked work",
			Query:       "search=golang&search_mode=fts&tag=work&sort=relevance",
		}},
		listResult: notes.ListResult{
			Filters: notes.ListFilters{
				OwnerUserID:   ownerID,
				Page:          1,
				PageSize:      20,
				Status:        notes.StatusActive,
				SortField:     "relevance",
				SortDirection: "asc",
			},
		},
	}
	api := &Server{notes: service, ownerID: ownerID}

	listSaved, err := api.handleListSavedQueries(context.Background(), mcp.CallToolRequest{}, struct{}{})
	if err != nil || listSaved.IsError {
		t.Fatalf("expected saved query list success, result=%+v err=%v", listSaved, err)
	}

	saveResult, err := api.handleSaveQuery(context.Background(), mcp.CallToolRequest{}, saveQueryArgs{
		Name: "Work query",
		listNotesArgs: listNotesArgs{
			Search: stringPtr("golang"),
			Tag:    stringPtr("work"),
		},
	})
	if err != nil || saveResult.IsError {
		t.Fatalf("expected save query success, result=%+v err=%v", saveResult, err)
	}
	if service.lastSaved.Name != "Work query" || service.lastSaved.Query == "" {
		t.Fatalf("expected normalized saved query input, got %+v", service.lastSaved)
	}

	listNotesResult, err := api.handleListNotes(context.Background(), mcp.CallToolRequest{}, listNotesArgs{
		SavedQueryID: savedID.String(),
		Order:        stringPtr("asc"),
	})
	if err != nil || listNotesResult.IsError {
		t.Fatalf("expected saved-query-backed list success, result=%+v err=%v", listNotesResult, err)
	}
	if service.lastList.Search == nil || *service.lastList.Search != "golang" || service.lastList.SearchMode != notes.SearchModeFTS || len(service.lastList.Tags) != 1 || service.lastList.Tags[0] != "work" || service.lastList.SortField != "relevance" || service.lastList.SortDirection != "asc" {
		t.Fatalf("expected saved query filters plus override, got %+v", service.lastList)
	}

	deleteResult, err := api.handleDeleteSavedQuery(context.Background(), mcp.CallToolRequest{}, noteIDArgs{ID: savedID.String()})
	if err != nil || deleteResult.IsError {
		t.Fatalf("expected delete saved query success, result=%+v err=%v", deleteResult, err)
	}
	if service.lastSavedID != savedID {
		t.Fatalf("expected delete saved query to use %s, got %s", savedID, service.lastSavedID)
	}
}

func TestTagHelpers(t *testing.T) {
	got := normalizeTags([]string{" go ", "", "mcp", "go"})
	if len(got) != 2 || got[0] != "go" || got[1] != "mcp" {
		t.Fatalf("unexpected normalized tags: %#v", got)
	}

	remaining := removeTags([]string{"go", "mcp", "notes"}, []string{"mcp"})
	if len(remaining) != 2 || remaining[0] != "go" || remaining[1] != "notes" {
		t.Fatalf("unexpected remaining tags: %#v", remaining)
	}
}

func TestToolDefinitionsAndServerRegistration(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	api := &Server{notes: &fakeNotesService{}, ownerID: ownerID}

	listTool := api.listNotesTool()
	if listTool.Name != "list_notes" {
		t.Fatalf("expected list tool name, got %q", listTool.Name)
	}
	listSchema := listTool.InputSchema.Properties
	if _, ok := listSchema["search_mode"]; !ok {
		t.Fatalf("expected search_mode in list tool schema, got %#v", listSchema)
	}
	if _, ok := listSchema["has_title"]; !ok {
		t.Fatalf("expected has_title in list tool schema, got %#v", listSchema)
	}
	if _, ok := listSchema["tags"]; !ok {
		t.Fatalf("expected tags filter in list tool schema, got %#v", listSchema)
	}
	if _, ok := listSchema["tag_count_min"]; !ok {
		t.Fatalf("expected tag_count_min in list tool schema, got %#v", listSchema)
	}
	if _, ok := listSchema["tag_mode"]; !ok {
		t.Fatalf("expected tag_mode filter in list tool schema, got %#v", listSchema)
	}
	if _, ok := listSchema["sort"]; !ok {
		t.Fatalf("expected sort filter in list tool schema, got %#v", listSchema)
	}

	getTool := api.getNoteTool()
	if getTool.Name != "get_note" || !strings.Contains(getTool.Description, "configured local MCP owner") {
		t.Fatalf("unexpected get tool definition: %+v", getTool)
	}

	findRelatedTool := api.findRelatedNotesTool()
	if findRelatedTool.Name != "find_related_notes" {
		t.Fatalf("unexpected related tool definition: %+v", findRelatedTool)
	}
	if _, ok := findRelatedTool.InputSchema.Properties["limit"]; !ok {
		t.Fatalf("expected find_related_notes to expose limit, got %#v", findRelatedTool.InputSchema.Properties)
	}

	createTool := api.createNoteTool()
	if createTool.Name != "create_note" {
		t.Fatalf("unexpected create tool definition: %+v", createTool)
	}
	if _, ok := createTool.InputSchema.Properties["tags"]; !ok {
		t.Fatalf("expected create tool to accept tags, got %#v", createTool.InputSchema.Properties)
	}

	updateTool := api.updateNoteTool()
	if updateTool.Name != "update_note" {
		t.Fatalf("unexpected update tool definition: %+v", updateTool)
	}
	if _, ok := updateTool.InputSchema.Properties["replace_tags"]; !ok {
		t.Fatalf("expected update tool to expose replace_tags, got %#v", updateTool.InputSchema.Properties)
	}

	addTagsTool := api.addNoteTagsTool()
	if addTagsTool.Name != "add_note_tags" {
		t.Fatalf("unexpected add tags tool definition: %+v", addTagsTool)
	}

	deleteTool := api.deleteNoteTool()
	if deleteTool.Name != "delete_note" {
		t.Fatalf("unexpected delete tool definition: %+v", deleteTool)
	}

	listSavedQueriesTool := api.listSavedQueriesTool()
	if listSavedQueriesTool.Name != "list_saved_queries" {
		t.Fatalf("unexpected list saved queries tool definition: %+v", listSavedQueriesTool)
	}

	saveQueryTool := api.saveQueryTool()
	if saveQueryTool.Name != "save_query" {
		t.Fatalf("unexpected save query tool definition: %+v", saveQueryTool)
	}
	if _, ok := saveQueryTool.InputSchema.Properties["name"]; !ok {
		t.Fatalf("expected save_query to expose name, got %#v", saveQueryTool.InputSchema.Properties)
	}

	deleteSavedQueryTool := api.deleteSavedQueryTool()
	if deleteSavedQueryTool.Name != "delete_saved_query" {
		t.Fatalf("unexpected delete saved query tool definition: %+v", deleteSavedQueryTool)
	}

	listTagsTool := api.listTagsTool()
	if listTagsTool.Name != "list_tags" {
		t.Fatalf("unexpected list tags tool definition: %+v", listTagsTool)
	}

	setTagsTool := api.setNoteTagsTool()
	if setTagsTool.Name != "set_note_tags" {
		t.Fatalf("unexpected set tags tool definition: %+v", setTagsTool)
	}

	removeTagsTool := api.removeNoteTagsTool()
	if removeTagsTool.Name != "remove_note_tags" {
		t.Fatalf("unexpected remove tags tool definition: %+v", removeTagsTool)
	}

	server := NewServer(&fakeNotesService{}, ownerID)
	if server == nil {
		t.Fatal("expected MCP server to be created")
	}
	tools := server.ListTools()
	if len(tools) != 17 {
		t.Fatalf("expected 17 registered tools, got %d", len(tools))
	}
	if _, ok := tools["list_notes"]; !ok {
		t.Fatalf("expected list_notes to be registered, got %#v", tools)
	}
	if _, ok := tools["get_note"]; !ok {
		t.Fatalf("expected get_note to be registered, got %#v", tools)
	}
	if _, ok := tools["find_related_notes"]; !ok {
		t.Fatalf("expected find_related_notes to be registered, got %#v", tools)
	}
	if _, ok := tools["create_note"]; !ok {
		t.Fatalf("expected create_note to be registered, got %#v", tools)
	}
	if _, ok := tools["update_note"]; !ok {
		t.Fatalf("expected update_note to be registered, got %#v", tools)
	}
	if _, ok := tools["delete_note"]; !ok {
		t.Fatalf("expected delete_note to be registered, got %#v", tools)
	}
	if _, ok := tools["list_saved_queries"]; !ok {
		t.Fatalf("expected list_saved_queries to be registered, got %#v", tools)
	}
	if _, ok := tools["save_query"]; !ok {
		t.Fatalf("expected save_query to be registered, got %#v", tools)
	}
	if _, ok := tools["delete_saved_query"]; !ok {
		t.Fatalf("expected delete_saved_query to be registered, got %#v", tools)
	}
	if _, ok := tools["list_tags"]; !ok {
		t.Fatalf("expected list_tags to be registered, got %#v", tools)
	}
	if _, ok := tools["set_note_tags"]; !ok {
		t.Fatalf("expected set_note_tags to be registered, got %#v", tools)
	}
	if _, ok := tools["share_note"]; !ok {
		t.Fatalf("expected share_note to be registered, got %#v", tools)
	}
	if _, ok := tools["unshare_note"]; !ok {
		t.Fatalf("expected unshare_note to be registered, got %#v", tools)
	}
	if _, ok := tools["archive_note"]; !ok {
		t.Fatalf("expected archive_note to be registered, got %#v", tools)
	}
	if _, ok := tools["unarchive_note"]; !ok {
		t.Fatalf("expected unarchive_note to be registered, got %#v", tools)
	}
	if _, ok := tools["add_note_tags"]; !ok {
		t.Fatalf("expected add_note_tags to be registered, got %#v", tools)
	}
	if _, ok := tools["remove_note_tags"]; !ok {
		t.Fatalf("expected remove_note_tags to be registered, got %#v", tools)
	}
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
