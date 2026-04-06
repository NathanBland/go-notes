package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
	"github.com/nathanbland/go-notes/internal/platform/db/sqlc"
)

type fakeQueryer struct {
	user            dbsqlc.User
	userErr         error
	note            dbsqlc.Note
	noteErr         error
	deleteRows      int64
	deleteErr       error
	notes           []dbsqlc.Note
	count           int64
	countErr        error
	tagRows         []dbsqlc.ListTagsForOwnerRow
	relatedRows     []dbsqlc.FindRelatedNotesForOwnerRow
	savedQuery      dbsqlc.SavedQuery
	savedQueries    []dbsqlc.SavedQuery
	lastCreate      dbsqlc.CreateNoteParams
	lastGetID       dbsqlc.GetNoteByIDForOwnerParams
	lastShareSlug   *string
	lastPatch       dbsqlc.UpdateNotePatchParams
	lastDelete      dbsqlc.DeleteNoteForOwnerParams
	lastList        dbsqlc.ListNotesForOwnerParams
	lastCount       dbsqlc.CountNotesForOwnerParams
	lastUpsertUser  dbsqlc.UpsertUserFromOIDCParams
	lastSavedCreate dbsqlc.CreateSavedQueryParams
	lastSavedGet    dbsqlc.GetSavedQueryForOwnerParams
	lastSavedDelete dbsqlc.DeleteSavedQueryForOwnerParams
	lastRelated     dbsqlc.FindRelatedNotesForOwnerParams
}

func (f *fakeQueryer) UpsertUserFromOIDC(ctx context.Context, arg dbsqlc.UpsertUserFromOIDCParams) (dbsqlc.User, error) {
	f.lastUpsertUser = arg
	return f.user, f.userErr
}
func (f *fakeQueryer) GetUserByID(ctx context.Context, id uuid.UUID) (dbsqlc.User, error) {
	return f.user, f.userErr
}
func (f *fakeQueryer) CreateNote(ctx context.Context, arg dbsqlc.CreateNoteParams) (dbsqlc.Note, error) {
	f.lastCreate = arg
	return f.note, f.noteErr
}
func (f *fakeQueryer) GetNoteByIDForOwner(ctx context.Context, arg dbsqlc.GetNoteByIDForOwnerParams) (dbsqlc.Note, error) {
	f.lastGetID = arg
	return f.note, f.noteErr
}
func (f *fakeQueryer) GetNoteByShareSlug(ctx context.Context, shareSlug *string) (dbsqlc.Note, error) {
	f.lastShareSlug = shareSlug
	return f.note, f.noteErr
}
func (f *fakeQueryer) UpdateNotePatch(ctx context.Context, arg dbsqlc.UpdateNotePatchParams) (dbsqlc.Note, error) {
	f.lastPatch = arg
	return f.note, f.noteErr
}
func (f *fakeQueryer) DeleteNoteForOwner(ctx context.Context, arg dbsqlc.DeleteNoteForOwnerParams) (int64, error) {
	f.lastDelete = arg
	return f.deleteRows, f.deleteErr
}
func (f *fakeQueryer) ListNotesForOwner(ctx context.Context, arg dbsqlc.ListNotesForOwnerParams) ([]dbsqlc.Note, error) {
	f.lastList = arg
	return f.notes, f.noteErr
}
func (f *fakeQueryer) CountNotesForOwner(ctx context.Context, arg dbsqlc.CountNotesForOwnerParams) (int64, error) {
	f.lastCount = arg
	return f.count, f.countErr
}
func (f *fakeQueryer) ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]dbsqlc.ListTagsForOwnerRow, error) {
	return f.tagRows, f.noteErr
}
func (f *fakeQueryer) FindRelatedNotesForOwner(ctx context.Context, arg dbsqlc.FindRelatedNotesForOwnerParams) ([]dbsqlc.FindRelatedNotesForOwnerRow, error) {
	f.lastRelated = arg
	return f.relatedRows, f.noteErr
}
func (f *fakeQueryer) CreateSavedQuery(ctx context.Context, arg dbsqlc.CreateSavedQueryParams) (dbsqlc.SavedQuery, error) {
	f.lastSavedCreate = arg
	return f.savedQuery, f.noteErr
}
func (f *fakeQueryer) GetSavedQueryForOwner(ctx context.Context, arg dbsqlc.GetSavedQueryForOwnerParams) (dbsqlc.SavedQuery, error) {
	f.lastSavedGet = arg
	return f.savedQuery, f.noteErr
}
func (f *fakeQueryer) ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]dbsqlc.SavedQuery, error) {
	return f.savedQueries, f.noteErr
}
func (f *fakeQueryer) DeleteSavedQueryForOwner(ctx context.Context, arg dbsqlc.DeleteSavedQueryForOwnerParams) (int64, error) {
	f.lastSavedDelete = arg
	return f.deleteRows, f.deleteErr
}

func TestToAuthUser(t *testing.T) {
	now := time.Now().UTC()
	user := toAuthUser(dbsqlc.User{
		ID:            uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		OidcIssuer:    "issuer",
		OidcSubject:   "subject",
		EmailVerified: true,
		CreatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Time: now, Valid: true},
	})

	if user.OIDCIssuer != "issuer" || user.OIDCSubject != "subject" || !user.CreatedAt.Equal(now) {
		t.Fatalf("unexpected auth user conversion: %+v", user)
	}
}

func TestToNote(t *testing.T) {
	now := time.Now().UTC()
	note := toNote(dbsqlc.Note{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Content:     "hello",
		Tags:        []string{"go"},
		CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
	})

	if note.Content != "hello" || len(note.Tags) != 1 || !note.CreatedAt.Equal(now) {
		t.Fatalf("unexpected note conversion: %+v", note)
	}
}

func TestHelperConverters(t *testing.T) {
	if derefBool(nil) {
		t.Fatal("expected nil bool to dereference to false")
	}
	truthy := true
	if !derefBool(&truthy) {
		t.Fatal("expected dereferenced bool to be true")
	}

	if len(derefStringSlice(nil)) != 0 {
		t.Fatal("expected nil slice to become empty slice")
	}
	values := []string{"a", "b"}
	if got := derefStringSlice(&values); len(got) != 2 {
		t.Fatalf("unexpected slice conversion: %#v", got)
	}
}

func TestTimestampHelpers(t *testing.T) {
	if got := timestamptz(nil); got.Valid {
		t.Fatalf("expected nil time to become invalid timestamptz, got %+v", got)
	}

	now := time.Now().UTC()
	got := timestamptz(&now)
	if !got.Valid || !got.Time.Equal(now) {
		t.Fatalf("unexpected timestamptz conversion: %+v", got)
	}

	if !timestampValue(got).Equal(now) {
		t.Fatalf("expected timestamp value to round-trip, got %v", timestampValue(got))
	}
	if !timestampValue(pgtype.Timestamptz{}).IsZero() {
		t.Fatal("expected invalid timestamptz to become zero time")
	}
}

func TestStoreMethodsAdaptQueryParams(t *testing.T) {
	now := time.Now().UTC()
	queryer := &fakeQueryer{
		user: dbsqlc.User{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			OidcIssuer:  "issuer",
			OidcSubject: "subject",
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		},
		note: dbsqlc.Note{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Content:     "hello",
			Tags:        []string{"go"},
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		},
		notes: []dbsqlc.Note{{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Content:     "hello",
			Tags:        []string{"go"},
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		}},
		count:      1,
		deleteRows: 1,
		tagRows: []dbsqlc.ListTagsForOwnerRow{
			{Tag: "go", Count: 2},
			{Tag: "mcp", Count: 1},
		},
		relatedRows: []dbsqlc.FindRelatedNotesForOwnerRow{{
			ID:             uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			OwnerUserID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Content:        "related",
			Tags:           []string{"go", "mcp"},
			SharedTags:     []string{"go"},
			SharedTagCount: 1,
			CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
		}},
	}
	store := &Store{queries: queryer}
	ctx := context.Background()

	if _, err := store.UpsertUserFromOIDC(ctx, auth.Identity{Issuer: "issuer", Subject: "subject"}); err != nil {
		t.Fatalf("unexpected upsert error: %v", err)
	}
	if queryer.lastUpsertUser.OidcIssuer != "issuer" {
		t.Fatalf("unexpected upsert params: %+v", queryer.lastUpsertUser)
	}

	if _, err := store.GetUserByID(ctx, queryer.user.ID); err != nil {
		t.Fatalf("unexpected get user error: %v", err)
	}

	if _, err := store.CreateNote(ctx, notes.CreateInput{OwnerUserID: queryer.note.OwnerUserID, Content: "hello", Tags: []string{"go"}}); err != nil {
		t.Fatalf("unexpected create note error: %v", err)
	}
	if queryer.lastCreate.Content != "hello" {
		t.Fatalf("unexpected create params: %+v", queryer.lastCreate)
	}

	if _, err := store.GetNoteByIDForOwner(ctx, queryer.note.ID, queryer.note.OwnerUserID); err != nil {
		t.Fatalf("unexpected get by id error: %v", err)
	}
	if queryer.lastGetID.ID != queryer.note.ID {
		t.Fatalf("unexpected get-by-id params: %+v", queryer.lastGetID)
	}

	slug := "share-me"
	if _, err := store.GetNoteByShareSlug(ctx, slug); err != nil {
		t.Fatalf("unexpected get share slug error: %v", err)
	}
	if queryer.lastShareSlug == nil || *queryer.lastShareSlug != slug {
		t.Fatalf("unexpected share slug params: %v", queryer.lastShareSlug)
	}

	shared := true
	if _, err := store.UpdateNotePatch(ctx, notes.PatchInput{
		ID:          queryer.note.ID,
		OwnerUserID: queryer.note.OwnerUserID,
		SharedSet:   true,
		Shared:      &shared,
	}); err != nil {
		t.Fatalf("unexpected patch error: %v", err)
	}
	if !queryer.lastPatch.SharedIsSet || !queryer.lastPatch.Shared {
		t.Fatalf("unexpected patch params: %+v", queryer.lastPatch)
	}

	if rows, err := store.DeleteNoteForOwner(ctx, queryer.note.ID, queryer.note.OwnerUserID); err != nil || rows != 1 {
		t.Fatalf("unexpected delete result rows=%d err=%v", rows, err)
	}

	filters := notes.ListFilters{
		OwnerUserID:   queryer.note.OwnerUserID,
		Page:          2,
		PageSize:      5,
		Status:        notes.StatusAll,
		SortField:     "title",
		SortDirection: "asc",
	}
	items, total, err := store.ListNotesForOwner(ctx, filters)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result items=%d total=%d", len(items), total)
	}
	if queryer.lastList.OffsetCount != 5 || queryer.lastCount.OwnerUserID != queryer.note.OwnerUserID {
		t.Fatalf("unexpected list/count params list=%+v count=%+v", queryer.lastList, queryer.lastCount)
	}

	tagSummary, err := store.ListTagsForOwner(ctx, queryer.note.OwnerUserID)
	if err != nil {
		t.Fatalf("unexpected list tags error: %v", err)
	}
	if len(tagSummary) != 2 || tagSummary[0].Tag != "go" || tagSummary[0].Count != 2 {
		t.Fatalf("unexpected tag summary: %+v", tagSummary)
	}

	related, err := store.FindRelatedNotesForOwner(ctx, queryer.note.OwnerUserID, queryer.note.ID, 4)
	if err != nil {
		t.Fatalf("unexpected related notes error: %v", err)
	}
	if len(related) != 1 || related[0].SharedTagCount != 1 || len(related[0].SharedTags) != 1 || related[0].SharedTags[0] != "go" {
		t.Fatalf("unexpected related notes: %+v", related)
	}
	if queryer.lastRelated.NoteID != queryer.note.ID || queryer.lastRelated.OwnerUserID != queryer.note.OwnerUserID || queryer.lastRelated.LimitCount != 4 {
		t.Fatalf("unexpected related query params: %+v", queryer.lastRelated)
	}
}

func TestStoreMethodsMapOptionalAndFilterFields(t *testing.T) {
	now := time.Now().UTC()
	title := "teaching title"
	content := "body"
	tags := []string{"go", "sqlc"}
	archived := true
	shared := false
	shareSlug := "existing-slug"
	createdAfter := now.Add(-2 * time.Hour)
	createdBefore := now.Add(-time.Hour)
	updatedAfter := now.Add(-30 * time.Minute)
	updatedBefore := now

	queryer := &fakeQueryer{
		note: dbsqlc.Note{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Title:       &title,
			Content:     content,
			Tags:        tags,
			Archived:    true,
			Shared:      true,
			ShareSlug:   &shareSlug,
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		},
		notes: []dbsqlc.Note{{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Title:       &title,
			Content:     content,
			Tags:        tags,
			Archived:    true,
			Shared:      true,
			ShareSlug:   &shareSlug,
			CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
		}},
		count: 1,
	}
	store := &Store{queries: queryer}

	updated, err := store.UpdateNotePatch(context.Background(), notes.PatchInput{
		ID:           queryer.note.ID,
		OwnerUserID:  queryer.note.OwnerUserID,
		TitleSet:     true,
		Title:        &title,
		ContentSet:   true,
		Content:      &content,
		TagsSet:      true,
		Tags:         &tags,
		ArchivedSet:  true,
		Archived:     &archived,
		SharedSet:    true,
		Shared:       &shared,
		ShareSlugSet: true,
		ShareSlug:    &shareSlug,
	})
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if updated.ShareSlug == nil || *updated.ShareSlug != shareSlug {
		t.Fatalf("expected share slug to round-trip, got %+v", updated)
	}
	if queryer.lastPatch.Title == nil || *queryer.lastPatch.Title != title {
		t.Fatalf("expected title to be forwarded, got %+v", queryer.lastPatch)
	}
	if len(queryer.lastPatch.Tags) != 2 || !queryer.lastPatch.Archived || queryer.lastPatch.Shared {
		t.Fatalf("unexpected patch params: %+v", queryer.lastPatch)
	}

	results, total, err := store.ListNotesForOwner(context.Background(), notes.ListFilters{
		OwnerUserID:   queryer.note.OwnerUserID,
		Page:          3,
		PageSize:      10,
		Status:        notes.StatusArchived,
		TagMatchMode:  notes.TagMatchAll,
		Shared:        &shared,
		Tags:          []string{tags[0], tags[1]},
		Search:        &content,
		SortField:     "created_at",
		SortDirection: "desc",
		CreatedAfter:  &createdAfter,
		CreatedBefore: &createdBefore,
		UpdatedAfter:  &updatedAfter,
		UpdatedBefore: &updatedBefore,
	})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(results) != 1 || total != 1 {
		t.Fatalf("unexpected list results=%d total=%d", len(results), total)
	}
	if !queryer.lastList.CreatedAfterFilter.Valid || !queryer.lastList.UpdatedBeforeFilter.Valid {
		t.Fatalf("expected timestamp filters to be valid, got %+v", queryer.lastList)
	}
	if !queryer.lastList.TagFilterEnabled || queryer.lastList.TagMatchMode != notes.TagMatchAll || len(queryer.lastList.TagFilters) != 2 {
		t.Fatalf("expected tag filters to be forwarded, got %+v", queryer.lastList)
	}
	if queryer.lastList.OffsetCount != 20 || queryer.lastCount.StatusFilter != notes.StatusArchived {
		t.Fatalf("unexpected list/count params list=%+v count=%+v", queryer.lastList, queryer.lastCount)
	}
}

func TestStoreMapsNoRowsToDomainErrors(t *testing.T) {
	store := &Store{queries: &fakeQueryer{userErr: pgx.ErrNoRows, noteErr: pgx.ErrNoRows}}
	ctx := context.Background()

	if _, err := store.GetUserByID(ctx, uuid.New()); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if _, err := store.GetNoteByIDForOwner(ctx, uuid.New(), uuid.New()); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected note not found, got %v", err)
	}
	if _, err := store.GetNoteByShareSlug(ctx, "missing"); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected share slug not found, got %v", err)
	}
	if _, err := store.UpdateNotePatch(ctx, notes.PatchInput{ID: uuid.New(), OwnerUserID: uuid.New()}); !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected patch not found, got %v", err)
	}
}

func TestStorePropagatesGeneralErrors(t *testing.T) {
	wantErr := errors.New("boom")
	store := &Store{queries: &fakeQueryer{
		userErr:   wantErr,
		noteErr:   wantErr,
		deleteErr: wantErr,
		countErr:  wantErr,
	}}
	ctx := context.Background()

	if _, err := store.UpsertUserFromOIDC(ctx, auth.Identity{}); !errors.Is(err, wantErr) {
		t.Fatalf("expected upsert error, got %v", err)
	}
	if _, err := store.GetUserByID(ctx, uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("expected get user error, got %v", err)
	}
	if _, err := store.CreateNote(ctx, notes.CreateInput{OwnerUserID: uuid.New()}); !errors.Is(err, wantErr) {
		t.Fatalf("expected create note error, got %v", err)
	}
	if _, err := store.GetNoteByIDForOwner(ctx, uuid.New(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("expected get note error, got %v", err)
	}
	if _, err := store.GetNoteByShareSlug(ctx, "x"); !errors.Is(err, wantErr) {
		t.Fatalf("expected get share error, got %v", err)
	}
	if _, err := store.UpdateNotePatch(ctx, notes.PatchInput{ID: uuid.New(), OwnerUserID: uuid.New()}); !errors.Is(err, wantErr) {
		t.Fatalf("expected patch error, got %v", err)
	}
	if _, err := store.DeleteNoteForOwner(ctx, uuid.New(), uuid.New()); !errors.Is(err, wantErr) {
		t.Fatalf("expected delete error, got %v", err)
	}
	if _, _, err := store.ListNotesForOwner(ctx, notes.ListFilters{OwnerUserID: uuid.New(), Page: 1, PageSize: 20, Status: notes.StatusActive, SortField: "updated_at", SortDirection: "desc"}); !errors.Is(err, wantErr) {
		t.Fatalf("expected list/count error, got %v", err)
	}
}
