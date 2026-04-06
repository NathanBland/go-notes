package notes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	createInput   CreateInput
	createdNote   Note
	createErr     error
	getByIDNote   Note
	getByIDErr    error
	sharedNote    Note
	sharedErr     error
	patchInput    PatchInput
	patchedNote   Note
	patchErr      error
	deleteRows    int64
	deleteRowsSet bool
	deleteErr     error
	lastDelete    uuid.UUID
	listCalls     int
	listItems     []Note
	listTotal     int64
	listErr       error
	tagItems      []TagSummary
	tagErr        error
	renamedNotes  []Note
	renameErr     error
	lastRenameOld string
	lastRenameNew string
	relatedItems  []RelatedNote
	relatedErr    error
	lastRelatedID uuid.UUID
	lastLimit     int32
	savedItems    []SavedQuery
	savedItem     SavedQuery
	savedErr      error
	lastSaved     CreateSavedQueryInput
	lastSavedID   uuid.UUID
}

func (f *fakeStore) CreateNote(ctx context.Context, input CreateInput) (Note, error) {
	if f.createErr != nil {
		return Note{}, f.createErr
	}
	f.createInput = input
	if f.createdNote.ID == uuid.Nil {
		f.createdNote = Note{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: input.OwnerUserID,
			Title:       input.Title,
			Content:     input.Content,
			Tags:        input.Tags,
			Archived:    input.Archived,
			Shared:      input.Shared,
			ShareSlug:   input.ShareSlug,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
	}
	return f.createdNote, nil
}
func (f *fakeStore) GetNoteByIDForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (Note, error) {
	if f.getByIDErr != nil {
		return Note{}, f.getByIDErr
	}
	if f.getByIDNote.ID != uuid.Nil {
		return f.getByIDNote, nil
	}
	return Note{}, ErrNotFound
}
func (f *fakeStore) GetNoteByShareSlug(ctx context.Context, shareSlug string) (Note, error) {
	if f.sharedErr != nil {
		return Note{}, f.sharedErr
	}
	if f.sharedNote.ID != uuid.Nil {
		return f.sharedNote, nil
	}
	return Note{}, ErrNotFound
}
func (f *fakeStore) UpdateNotePatch(ctx context.Context, input PatchInput) (Note, error) {
	f.patchInput = input
	if f.patchErr != nil {
		return Note{}, f.patchErr
	}
	if f.patchedNote.ID != uuid.Nil {
		return f.patchedNote, nil
	}
	return Note{}, nil
}
func (f *fakeStore) DeleteNoteForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	f.lastDelete = id
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	if f.deleteRowsSet {
		return f.deleteRows, nil
	}
	return 1, nil
}
func (f *fakeStore) ListNotesForOwner(ctx context.Context, filters ListFilters) ([]Note, int64, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listItems, f.listTotal, nil
}
func (f *fakeStore) ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]TagSummary, error) {
	if f.tagErr != nil {
		return nil, f.tagErr
	}
	return f.tagItems, nil
}
func (f *fakeStore) RenameTagForOwner(ctx context.Context, ownerUserID uuid.UUID, oldTag, newTag string) ([]Note, error) {
	f.lastRenameOld = oldTag
	f.lastRenameNew = newTag
	if f.renameErr != nil {
		return nil, f.renameErr
	}
	return f.renamedNotes, nil
}
func (f *fakeStore) FindRelatedNotesForOwner(ctx context.Context, ownerUserID, id uuid.UUID, limit int32) ([]RelatedNote, error) {
	f.lastRelatedID = id
	f.lastLimit = limit
	if f.relatedErr != nil {
		return nil, f.relatedErr
	}
	return f.relatedItems, nil
}
func (f *fakeStore) CreateSavedQuery(ctx context.Context, input CreateSavedQueryInput) (SavedQuery, error) {
	f.lastSaved = input
	if f.savedErr != nil {
		return SavedQuery{}, f.savedErr
	}
	if f.savedItem.ID != uuid.Nil {
		return f.savedItem, nil
	}
	return SavedQuery{ID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), OwnerUserID: input.OwnerUserID, Name: input.Name, Query: input.Query, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
}
func (f *fakeStore) GetSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (SavedQuery, error) {
	f.lastSavedID = id
	if f.savedErr != nil {
		return SavedQuery{}, f.savedErr
	}
	if f.savedItem.ID != uuid.Nil {
		return f.savedItem, nil
	}
	return SavedQuery{}, ErrNotFound
}
func (f *fakeStore) ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]SavedQuery, error) {
	if f.savedErr != nil {
		return nil, f.savedErr
	}
	return f.savedItems, nil
}
func (f *fakeStore) DeleteSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	f.lastSavedID = id
	if f.savedErr != nil {
		return 0, f.savedErr
	}
	return 1, nil
}

type fakeCache struct {
	values map[string]string
}

func (f *fakeCache) Get(ctx context.Context, key string) (string, bool) {
	value, ok := f.values[key]
	return value, ok
}
func (f *fakeCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}
func (f *fakeCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func TestCreateSharedNoteGeneratesSlug(t *testing.T) {
	store := &fakeStore{}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	note, err := service.Create(context.Background(), CreateInput{
		OwnerUserID: ownerID,
		Content:     "hello",
		Shared:      true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.createInput.Shared || store.createInput.ShareSlug == nil {
		t.Fatalf("expected shared create to include a share slug, got %+v", store.createInput)
	}
	if note.ShareSlug == nil || *note.ShareSlug == "" {
		t.Fatalf("expected note to expose share slug, got %+v", note)
	}
}

func TestListUsesCacheAfterFirstRead(t *testing.T) {
	store := &fakeStore{listItems: []Note{{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}}, listTotal: 1}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)
	filters := ListFilters{
		OwnerUserID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Page:          1,
		PageSize:      20,
		Status:        StatusActive,
		SortField:     "updated_at",
		SortDirection: "desc",
	}
	if _, err := service.List(context.Background(), filters); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := service.List(context.Background(), filters); err != nil {
		t.Fatalf("unexpected error on cached read: %v", err)
	}
	if store.listCalls != 1 {
		t.Fatalf("expected store to be called once, got %d", store.listCalls)
	}
}

func TestGetByIDForOwnerUsesCacheAfterFirstLookup(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	store := &fakeStore{
		getByIDNote: Note{ID: noteID, OwnerUserID: ownerID, Content: "cached later"},
	}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)

	first, err := service.GetByIDForOwner(context.Background(), ownerID, noteID)
	if err != nil {
		t.Fatalf("unexpected error on first lookup: %v", err)
	}
	second, err := service.GetByIDForOwner(context.Background(), ownerID, noteID)
	if err != nil {
		t.Fatalf("unexpected error on second lookup: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected cached note on second lookup, got %+v and %+v", first, second)
	}
}

func TestGetByShareSlugUsesCacheAfterFirstLookup(t *testing.T) {
	slug := "shared-1"
	store := &fakeStore{
		sharedNote: Note{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), Content: "shared", Shared: true, ShareSlug: &slug},
	}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)

	first, err := service.GetByShareSlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("unexpected error on first shared lookup: %v", err)
	}
	store.sharedErr = errors.New("should not be called again")
	second, err := service.GetByShareSlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("unexpected error on cached shared lookup: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same shared note, got %+v and %+v", first, second)
	}
}

func TestSavedQueryHelpers(t *testing.T) {
	query := EncodeListFilters(ListFilters{
		OwnerUserID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Page:          1,
		PageSize:      20,
		SearchMode:    SearchModeFTS,
		Status:        StatusAll,
		TagMatchMode:  TagMatchAll,
		SortField:     "relevance",
		SortDirection: "asc",
		Tags:          []string{"work", "planning"},
		Search:        strPtr("golang"),
	})
	if strings.Contains(query, "owner_user_id") || !strings.Contains(query, "search=golang") || !strings.Contains(query, "sort=relevance") {
		t.Fatalf("unexpected encoded saved query: %q", query)
	}
}

func TestFindRelatedNotesUsesStore(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	relatedID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	store := &fakeStore{
		getByIDNote: Note{ID: noteID, OwnerUserID: ownerID, Content: "source", Tags: []string{"go", "mcp"}},
		relatedItems: []RelatedNote{{
			Note:           Note{ID: relatedID, OwnerUserID: ownerID, Content: "related", Tags: []string{"go", "sql"}},
			SharedTags:     []string{"go"},
			SharedTagCount: 1,
		}},
	}
	service := NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)

	items, err := service.FindRelatedNotes(context.Background(), ownerID, noteID, 7)
	if err != nil {
		t.Fatalf("unexpected related notes error: %v", err)
	}
	if len(items) != 1 || items[0].Note.ID != relatedID {
		t.Fatalf("unexpected related notes result: %+v", items)
	}
	if store.lastRelatedID != noteID || store.lastLimit != 7 {
		t.Fatalf("unexpected related notes store inputs: id=%s limit=%d", store.lastRelatedID, store.lastLimit)
	}
}

func TestFindRelatedNotesReturnsEmptyWhenSourceHasNoTags(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	store := &fakeStore{
		getByIDNote: Note{ID: noteID, OwnerUserID: ownerID, Content: "source", Tags: []string{}},
	}
	service := NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)

	items, err := service.FindRelatedNotes(context.Background(), ownerID, noteID, 0)
	if err != nil {
		t.Fatalf("unexpected related notes error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty result for tagless source note, got %+v", items)
	}
	if store.lastRelatedID != uuid.Nil {
		t.Fatalf("expected store lookup to be skipped for tagless note, got %s", store.lastRelatedID)
	}
}

func TestCreateSavedQueryValidatesName(t *testing.T) {
	store := &fakeStore{}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)

	if _, err := service.CreateSavedQuery(context.Background(), CreateSavedQueryInput{
		OwnerUserID: uuid.New(),
		Name:        "   ",
		Query:       "",
	}); err == nil {
		t.Fatal("expected blank saved query name to fail")
	}
}

func TestSavedQueryCRUDUsesStore(t *testing.T) {
	store := &fakeStore{
		savedItem: SavedQuery{
			ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:        "Ranked work",
			Query:       "search=golang",
		},
		savedItems: []SavedQuery{{
			ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:        "Ranked work",
			Query:       "search=golang",
		}},
	}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	created, err := service.CreateSavedQuery(context.Background(), CreateSavedQueryInput{
		OwnerUserID: ownerID,
		Name:        "Ranked work",
		Query:       "search=golang",
	})
	if err != nil || created.Name != "Ranked work" {
		t.Fatalf("expected create saved query success, got %+v err=%v", created, err)
	}
	if store.lastSaved.Name != "Ranked work" {
		t.Fatalf("expected create saved query input to reach store, got %+v", store.lastSaved)
	}

	items, err := service.ListSavedQueries(context.Background(), ownerID)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected saved query list success, got %+v err=%v", items, err)
	}

	loaded, err := service.GetSavedQuery(context.Background(), ownerID, store.savedItem.ID)
	if err != nil || loaded.ID != store.savedItem.ID {
		t.Fatalf("expected get saved query success, got %+v err=%v", loaded, err)
	}

	if err := service.DeleteSavedQuery(context.Background(), ownerID, store.savedItem.ID); err != nil {
		t.Fatalf("expected delete saved query success, got %v", err)
	}
}

func TestPatchGeneratesAndClearsShareSlug(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	current := Note{ID: noteID, OwnerUserID: ownerID, Content: "before"}
	store := &fakeStore{
		getByIDNote: current,
		patchedNote: Note{ID: noteID, OwnerUserID: ownerID, Content: "after", Shared: true},
	}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)

	shared := true
	content := " after "
	updated, err := service.Patch(context.Background(), PatchInput{
		ID:          noteID,
		OwnerUserID: ownerID,
		ContentSet:  true,
		Content:     &content,
		SharedSet:   true,
		Shared:      &shared,
	})
	if err != nil {
		t.Fatalf("unexpected patch error: %v", err)
	}
	if !store.patchInput.ShareSlugSet || store.patchInput.ShareSlug == nil || *store.patchInput.ShareSlug == "" {
		t.Fatalf("expected patch to generate share slug, got %+v", store.patchInput)
	}
	if store.patchInput.Content == nil || *store.patchInput.Content != "after" {
		t.Fatalf("expected patch content to be trimmed, got %+v", store.patchInput)
	}
	if updated.ID != noteID {
		t.Fatalf("unexpected updated note: %+v", updated)
	}

	previousSlug := "keep-me"
	current.ShareSlug = &previousSlug
	store.getByIDNote = current
	store.patchedNote = Note{ID: noteID, OwnerUserID: ownerID, Content: "after", Shared: false}
	cache.values[sharedCacheKey(previousSlug)] = "payload"

	shared = false
	_, err = service.Patch(context.Background(), PatchInput{
		ID:          noteID,
		OwnerUserID: ownerID,
		SharedSet:   true,
		Shared:      &shared,
	})
	if err != nil {
		t.Fatalf("unexpected unshare patch error: %v", err)
	}
	if !store.patchInput.ShareSlugSet || store.patchInput.ShareSlug != nil {
		t.Fatalf("expected patch to clear share slug, got %+v", store.patchInput)
	}
	if _, ok := cache.values[sharedCacheKey(previousSlug)]; ok {
		t.Fatal("expected previous shared cache entry to be deleted")
	}
}

func TestListTagsDelegatesToStore(t *testing.T) {
	store := &fakeStore{
		tagItems: []TagSummary{
			{Tag: "go", Count: 2},
			{Tag: "mcp", Count: 1},
		},
	}
	service := NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tags, err := service.ListTags(context.Background(), ownerID)
	if err != nil {
		t.Fatalf("unexpected list tags error: %v", err)
	}
	if len(tags) != 2 || tags[0].Tag != "go" || tags[0].Count != 2 {
		t.Fatalf("unexpected tag list: %+v", tags)
	}
}

func TestRenameTagUsesStoreAndRefreshesCaches(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	slug := "shared-1"
	store := &fakeStore{
		renamedNotes: []Note{{
			ID:          noteID,
			OwnerUserID: ownerID,
			Content:     "body",
			Tags:        []string{"roadmap", "mcp"},
			Shared:      true,
			ShareSlug:   &slug,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}},
	}
	cache := &fakeCache{values: map[string]string{}}
	service := NewService(store, cache, time.Minute, time.Minute)

	result, err := service.RenameTag(context.Background(), ownerID, " planning ", " roadmap ")
	if err != nil {
		t.Fatalf("unexpected rename error: %v", err)
	}
	if result.OldTag != "planning" || result.NewTag != "roadmap" || result.AffectedNotes != 1 {
		t.Fatalf("unexpected rename result: %+v", result)
	}
	if store.lastRenameOld != "planning" || store.lastRenameNew != "roadmap" {
		t.Fatalf("expected trimmed tags to reach store, got old=%q new=%q", store.lastRenameOld, store.lastRenameNew)
	}
	if _, ok := cache.values[noteCacheKey(ownerID.String(), noteID.String())]; !ok {
		t.Fatal("expected renamed note to refresh the note cache")
	}
	if _, ok := cache.values[sharedCacheKey(slug)]; !ok {
		t.Fatal("expected renamed shared note to refresh the shared cache")
	}
}

func TestDeleteHandlesNotFoundAndInvalidatesCache(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	slug := "shared-1"
	store := &fakeStore{
		getByIDNote:   Note{ID: noteID, OwnerUserID: ownerID, Content: "hello", ShareSlug: &slug},
		deleteRows:    1,
		deleteRowsSet: true,
	}
	cache := &fakeCache{values: map[string]string{
		noteCacheKey(ownerID.String(), noteID.String()): "payload",
		sharedCacheKey(slug):                            "payload",
	}}
	service := NewService(store, cache, time.Minute, time.Minute)

	if err := service.Delete(context.Background(), ownerID, noteID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if _, ok := cache.values[noteCacheKey(ownerID.String(), noteID.String())]; ok {
		t.Fatal("expected note cache key to be deleted")
	}
	if _, ok := cache.values[sharedCacheKey(slug)]; ok {
		t.Fatal("expected shared cache key to be deleted")
	}

	store.getByIDNote = Note{ID: noteID, OwnerUserID: ownerID}
	store.deleteRows = 0
	store.deleteRowsSet = true
	if err := service.Delete(context.Background(), ownerID, noteID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found when delete affects zero rows, got %v", err)
	}
}

func TestListCacheKeyAndHelpers(t *testing.T) {
	filters := ListFilters{
		OwnerUserID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Page:          2,
		PageSize:      5,
		Status:        StatusAll,
		SortField:     "title",
		SortDirection: "asc",
	}
	if filters.Offset() != 5 {
		t.Fatalf("expected offset 5, got %d", filters.Offset())
	}
	if got := trimStringPtr(nil); got != nil {
		t.Fatalf("expected nil trim result, got %v", got)
	}
	if got := trimStringPtr(strPtr(" hi ")); got == nil || *got != "hi" {
		t.Fatalf("unexpected trimmed string ptr: %v", got)
	}
	key, err := listCacheKey(filters)
	if err != nil || key == "" {
		t.Fatalf("expected cache key, got key=%q err=%v", key, err)
	}
	slug, err := randomSlug(8)
	if err != nil || len(slug) == 0 {
		t.Fatalf("expected random slug, got %q err=%v", slug, err)
	}
	payload, err := json.Marshal(filters)
	if err != nil || len(payload) == 0 {
		t.Fatalf("expected filters to marshal, got %q err=%v", payload, err)
	}
}

func strPtr(value string) *string {
	return &value
}
