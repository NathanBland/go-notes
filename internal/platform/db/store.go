package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
	dbsqlc "github.com/nathanbland/go-notes/internal/platform/db/sqlc"
)

// Store is the small repository layer between validated API/service inputs and
// sqlc-generated queries.
//
// The goal is not to hide SQL. Instead, this layer adapts request-shaped Go
// values into typed sqlc parameter structs, while PostgreSQL does the heavy
// lifting for filtering, sorting, pagination, and counting.
//
// Keeping dynamic behavior constrained here also helps prevent SQL injection:
// request validation chooses allowed sort/filter values first, then Store passes
// them into bound sqlc parameters instead of building raw SQL strings.
type Store struct {
	queries queryer
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{queries: dbsqlc.New(pool)}
}

type queryer interface {
	UpsertUserFromOIDC(ctx context.Context, arg dbsqlc.UpsertUserFromOIDCParams) (dbsqlc.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (dbsqlc.User, error)
	CreateNote(ctx context.Context, arg dbsqlc.CreateNoteParams) (dbsqlc.Note, error)
	GetNoteByIDForOwner(ctx context.Context, arg dbsqlc.GetNoteByIDForOwnerParams) (dbsqlc.Note, error)
	GetNoteByShareSlug(ctx context.Context, shareSlug *string) (dbsqlc.Note, error)
	UpdateNotePatch(ctx context.Context, arg dbsqlc.UpdateNotePatchParams) (dbsqlc.Note, error)
	DeleteNoteForOwner(ctx context.Context, arg dbsqlc.DeleteNoteForOwnerParams) (int64, error)
	ListNotesForOwner(ctx context.Context, arg dbsqlc.ListNotesForOwnerParams) ([]dbsqlc.Note, error)
	CountNotesForOwner(ctx context.Context, arg dbsqlc.CountNotesForOwnerParams) (int64, error)
	ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]dbsqlc.ListTagsForOwnerRow, error)
	FindRelatedNotesForOwner(ctx context.Context, arg dbsqlc.FindRelatedNotesForOwnerParams) ([]dbsqlc.FindRelatedNotesForOwnerRow, error)
	CreateSavedQuery(ctx context.Context, arg dbsqlc.CreateSavedQueryParams) (dbsqlc.SavedQuery, error)
	GetSavedQueryForOwner(ctx context.Context, arg dbsqlc.GetSavedQueryForOwnerParams) (dbsqlc.SavedQuery, error)
	ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]dbsqlc.SavedQuery, error)
	DeleteSavedQueryForOwner(ctx context.Context, arg dbsqlc.DeleteSavedQueryForOwnerParams) (int64, error)
}

func (s *Store) UpsertUserFromOIDC(ctx context.Context, identity auth.Identity) (auth.User, error) {
	user, err := s.queries.UpsertUserFromOIDC(ctx, dbsqlc.UpsertUserFromOIDCParams{
		OidcIssuer:    identity.Issuer,
		OidcSubject:   identity.Subject,
		Email:         identity.Email,
		EmailVerified: identity.EmailVerified,
		DisplayName:   identity.DisplayName,
		PictureUrl:    identity.PictureURL,
	})
	if err != nil {
		return auth.User{}, err
	}
	return toAuthUser(user), nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, auth.ErrUnauthorized
		}
		return auth.User{}, err
	}
	return toAuthUser(user), nil
}

func (s *Store) CreateNote(ctx context.Context, input notes.CreateInput) (notes.Note, error) {
	note, err := s.queries.CreateNote(ctx, dbsqlc.CreateNoteParams{
		OwnerUserID: input.OwnerUserID,
		Title:       input.Title,
		Content:     input.Content,
		Tags:        input.Tags,
		Archived:    input.Archived,
		Shared:      input.Shared,
		ShareSlug:   input.ShareSlug,
	})
	if err != nil {
		return notes.Note{}, err
	}
	return toNote(note), nil
}

func (s *Store) GetNoteByIDForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (notes.Note, error) {
	note, err := s.queries.GetNoteByIDForOwner(ctx, dbsqlc.GetNoteByIDForOwnerParams{ID: id, OwnerUserID: ownerUserID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notes.Note{}, notes.ErrNotFound
		}
		return notes.Note{}, err
	}
	return toNote(note), nil
}

func (s *Store) GetNoteByShareSlug(ctx context.Context, shareSlug string) (notes.Note, error) {
	note, err := s.queries.GetNoteByShareSlug(ctx, &shareSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notes.Note{}, notes.ErrNotFound
		}
		return notes.Note{}, err
	}
	return toNote(note), nil
}

func (s *Store) UpdateNotePatch(ctx context.Context, input notes.PatchInput) (notes.Note, error) {
	params := dbsqlc.UpdateNotePatchParams{
		ID:             input.ID,
		OwnerUserID:    input.OwnerUserID,
		TitleIsSet:     input.TitleSet,
		Title:          input.Title,
		ContentIsSet:   input.ContentSet,
		Content:        input.Content,
		TagsIsSet:      input.TagsSet,
		Tags:           derefStringSlice(input.Tags),
		ArchivedIsSet:  input.ArchivedSet,
		Archived:       derefBool(input.Archived),
		SharedIsSet:    input.SharedSet,
		Shared:         derefBool(input.Shared),
		ShareSlugIsSet: input.ShareSlugSet,
		ShareSlug:      input.ShareSlug,
	}
	note, err := s.queries.UpdateNotePatch(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notes.Note{}, notes.ErrNotFound
		}
		return notes.Note{}, err
	}
	return toNote(note), nil
}

func (s *Store) DeleteNoteForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	return s.queries.DeleteNoteForOwner(ctx, dbsqlc.DeleteNoteForOwnerParams{ID: id, OwnerUserID: ownerUserID})
}

func (s *Store) ListNotesForOwner(ctx context.Context, filters notes.ListFilters) ([]notes.Note, int64, error) {
	// The repository maps validated filters into SQL params, but the actual
	// search/filter/sort/pagination work stays in PostgreSQL where it can be
	// tested once and used consistently by every caller.
	params := dbsqlc.ListNotesForOwnerParams{
		OwnerUserID:         filters.OwnerUserID,
		StatusFilter:        filters.Status,
		SharedFilter:        filters.Shared,
		HasTitleFilter:      filters.HasTitle,
		TagFilterEnabled:    len(filters.Tags) > 0,
		TagMatchMode:        filters.TagMatchMode,
		TagFilters:          filters.Tags,
		TagCountMinFilter:   filters.TagCountMin,
		TagCountMaxFilter:   filters.TagCountMax,
		SearchFilter:        filters.Search,
		SearchMode:          filters.SearchMode,
		CreatedAfterFilter:  timestamptz(filters.CreatedAfter),
		CreatedBeforeFilter: timestamptz(filters.CreatedBefore),
		UpdatedAfterFilter:  timestamptz(filters.UpdatedAfter),
		UpdatedBeforeFilter: timestamptz(filters.UpdatedBefore),
		SortField:           filters.SortField,
		SortDirection:       filters.SortDirection,
		LimitCount:          filters.PageSize,
		OffsetCount:         filters.Offset(),
	}
	items, err := s.queries.ListNotesForOwner(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.queries.CountNotesForOwner(ctx, dbsqlc.CountNotesForOwnerParams{
		OwnerUserID:         filters.OwnerUserID,
		StatusFilter:        filters.Status,
		SharedFilter:        filters.Shared,
		HasTitleFilter:      filters.HasTitle,
		TagFilterEnabled:    len(filters.Tags) > 0,
		TagMatchMode:        filters.TagMatchMode,
		TagFilters:          filters.Tags,
		TagCountMinFilter:   filters.TagCountMin,
		TagCountMaxFilter:   filters.TagCountMax,
		SearchFilter:        filters.Search,
		SearchMode:          filters.SearchMode,
		CreatedAfterFilter:  timestamptz(filters.CreatedAfter),
		CreatedBeforeFilter: timestamptz(filters.CreatedBefore),
		UpdatedAfterFilter:  timestamptz(filters.UpdatedAfter),
		UpdatedBeforeFilter: timestamptz(filters.UpdatedBefore),
	})
	if err != nil {
		return nil, 0, err
	}
	results := make([]notes.Note, 0, len(items))
	for _, item := range items {
		results = append(results, toNote(item))
	}
	return results, total, nil
}

func (s *Store) ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]notes.TagSummary, error) {
	rows, err := s.queries.ListTagsForOwner(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	results := make([]notes.TagSummary, 0, len(rows))
	for _, row := range rows {
		results = append(results, notes.TagSummary{
			Tag:   row.Tag,
			Count: row.Count,
		})
	}
	return results, nil
}

func (s *Store) FindRelatedNotesForOwner(ctx context.Context, ownerUserID, noteID uuid.UUID, limit int32) ([]notes.RelatedNote, error) {
	rows, err := s.queries.FindRelatedNotesForOwner(ctx, dbsqlc.FindRelatedNotesForOwnerParams{
		NoteID:      noteID,
		OwnerUserID: ownerUserID,
		LimitCount:  limit,
	})
	if err != nil {
		return nil, err
	}
	results := make([]notes.RelatedNote, 0, len(rows))
	for _, row := range rows {
		results = append(results, toRelatedNote(row))
	}
	return results, nil
}

func (s *Store) CreateSavedQuery(ctx context.Context, input notes.CreateSavedQueryInput) (notes.SavedQuery, error) {
	saved, err := s.queries.CreateSavedQuery(ctx, dbsqlc.CreateSavedQueryParams{
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		QueryString: input.Query,
	})
	if err != nil {
		return notes.SavedQuery{}, err
	}
	return toSavedQuery(saved), nil
}

func (s *Store) GetSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (notes.SavedQuery, error) {
	saved, err := s.queries.GetSavedQueryForOwner(ctx, dbsqlc.GetSavedQueryForOwnerParams{
		ID:          id,
		OwnerUserID: ownerUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notes.SavedQuery{}, notes.ErrNotFound
		}
		return notes.SavedQuery{}, err
	}
	return toSavedQuery(saved), nil
}

func (s *Store) ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]notes.SavedQuery, error) {
	rows, err := s.queries.ListSavedQueriesForOwner(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	results := make([]notes.SavedQuery, 0, len(rows))
	for _, row := range rows {
		results = append(results, toSavedQuery(row))
	}
	return results, nil
}

func (s *Store) DeleteSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	return s.queries.DeleteSavedQueryForOwner(ctx, dbsqlc.DeleteSavedQueryForOwnerParams{
		ID:          id,
		OwnerUserID: ownerUserID,
	})
}

func toAuthUser(user dbsqlc.User) auth.User {
	return auth.User{
		ID:            user.ID,
		OIDCIssuer:    user.OidcIssuer,
		OIDCSubject:   user.OidcSubject,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		DisplayName:   user.DisplayName,
		PictureURL:    user.PictureUrl,
		CreatedAt:     timestampValue(user.CreatedAt),
		UpdatedAt:     timestampValue(user.UpdatedAt),
	}
}

func toNote(note dbsqlc.Note) notes.Note {
	return notes.Note{
		ID:          note.ID,
		OwnerUserID: note.OwnerUserID,
		Title:       note.Title,
		Content:     note.Content,
		Tags:        note.Tags,
		Archived:    note.Archived,
		Shared:      note.Shared,
		ShareSlug:   note.ShareSlug,
		CreatedAt:   timestampValue(note.CreatedAt),
		UpdatedAt:   timestampValue(note.UpdatedAt),
	}
}

func toSavedQuery(saved dbsqlc.SavedQuery) notes.SavedQuery {
	return notes.SavedQuery{
		ID:          saved.ID,
		OwnerUserID: saved.OwnerUserID,
		Name:        saved.Name,
		Query:       saved.QueryString,
		CreatedAt:   timestampValue(saved.CreatedAt),
		UpdatedAt:   timestampValue(saved.UpdatedAt),
	}
}

func toRelatedNote(row dbsqlc.FindRelatedNotesForOwnerRow) notes.RelatedNote {
	return notes.RelatedNote{
		Note: notes.Note{
			ID:          row.ID,
			OwnerUserID: row.OwnerUserID,
			Title:       row.Title,
			Content:     row.Content,
			Tags:        row.Tags,
			Archived:    row.Archived,
			Shared:      row.Shared,
			ShareSlug:   row.ShareSlug,
			CreatedAt:   timestampValue(row.CreatedAt),
			UpdatedAt:   timestampValue(row.UpdatedAt),
		},
		SharedTags:     row.SharedTags,
		SharedTagCount: row.SharedTagCount,
	}
}

func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func derefStringSlice(value *[]string) []string {
	if value == nil {
		return []string{}
	}
	return *value
}

// timestamptz keeps nullable filter inputs explicit for sqlc/pgx.
// In Go we model an omitted timestamp as nil, then convert it here into the
// pgx type that can represent either “set” or “not provided”.
func timestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func timestampValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}
