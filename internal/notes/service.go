package notes

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service owns business rules plus the teaching-friendly cache-aside behavior.
type Service struct {
	store        Store
	cache        Cache
	noteCacheTTL time.Duration
	listCacheTTL time.Duration
}

func NewService(store Store, cache Cache, noteCacheTTL, listCacheTTL time.Duration) *Service {
	return &Service{
		store:        store,
		cache:        cache,
		noteCacheTTL: noteCacheTTL,
		listCacheTTL: listCacheTTL,
	}
}

// Create normalizes note input, generates a share slug when needed, persists
// the note, and seeds the single-note cache.
func (s *Service) Create(ctx context.Context, input CreateInput) (Note, error) {
	input.Content = strings.TrimSpace(input.Content)
	if input.Tags == nil {
		input.Tags = []string{}
	}
	if input.Shared {
		slug, err := randomSlug(8)
		if err != nil {
			return Note{}, err
		}
		input.ShareSlug = &slug
	}

	created, err := s.store.CreateNote(ctx, CreateInput{
		OwnerUserID: input.OwnerUserID,
		Title:       trimStringPtr(input.Title),
		Content:     input.Content,
		Tags:        input.Tags,
		Archived:    input.Archived,
		Shared:      input.Shared,
		ShareSlug:   input.ShareSlug,
	})
	if err != nil {
		return Note{}, err
	}
	s.cacheNote(ctx, created)
	return created, nil
}

// GetByIDForOwner reads through the note cache first, then falls back to
// PostgreSQL for the authoritative owner-scoped lookup.
func (s *Service) GetByIDForOwner(ctx context.Context, ownerUserID, noteID uuid.UUID) (Note, error) {
	key := noteCacheKey(ownerUserID.String(), noteID.String())
	if payload, ok := s.cache.Get(ctx, key); ok {
		var cached Note
		if err := json.Unmarshal([]byte(payload), &cached); err == nil {
			return cached, nil
		}
	}

	note, err := s.store.GetNoteByIDForOwner(ctx, noteID, ownerUserID)
	if err != nil {
		return Note{}, err
	}
	s.cacheNote(ctx, note)
	return note, nil
}

// GetByShareSlug serves intentionally public shared notes by share slug.
func (s *Service) GetByShareSlug(ctx context.Context, slug string) (Note, error) {
	key := sharedCacheKey(slug)
	if payload, ok := s.cache.Get(ctx, key); ok {
		var cached Note
		if err := json.Unmarshal([]byte(payload), &cached); err == nil {
			return cached, nil
		}
	}

	note, err := s.store.GetNoteByShareSlug(ctx, slug)
	if err != nil {
		return Note{}, err
	}
	s.cacheSharedNote(ctx, note)
	return note, nil
}

// Patch applies partial updates while preserving the difference between omitted
// fields, explicit nulls, and concrete values.
func (s *Service) Patch(ctx context.Context, input PatchInput) (Note, error) {
	current, err := s.store.GetNoteByIDForOwner(ctx, input.ID, input.OwnerUserID)
	if err != nil {
		return Note{}, err
	}

	if input.ContentSet && input.Content != nil {
		trimmed := strings.TrimSpace(*input.Content)
		input.Content = &trimmed
	}
	if input.TagsSet && input.Tags == nil {
		return Note{}, ErrNotFound
	}
	input.Title = trimStringPtr(input.Title)

	if input.SharedSet {
		if input.Shared != nil && *input.Shared {
			if current.ShareSlug == nil {
				slug, err := randomSlug(8)
				if err != nil {
					return Note{}, err
				}
				input.ShareSlugSet = true
				input.ShareSlug = &slug
			} else {
				input.ShareSlugSet = true
				input.ShareSlug = current.ShareSlug
			}
		} else {
			input.ShareSlugSet = true
			input.ShareSlug = nil
		}
	}

	updated, err := s.store.UpdateNotePatch(ctx, input)
	if err != nil {
		return Note{}, err
	}

	_ = s.cache.Delete(ctx, noteCacheKey(input.OwnerUserID.String(), input.ID.String()))
	if current.ShareSlug != nil {
		_ = s.cache.Delete(ctx, sharedCacheKey(*current.ShareSlug))
	}
	s.cacheNote(ctx, updated)
	s.cacheSharedNote(ctx, updated)
	return updated, nil
}

// Delete removes an owner-scoped note and evicts any related cache entries.
func (s *Service) Delete(ctx context.Context, ownerUserID, noteID uuid.UUID) error {
	current, err := s.store.GetNoteByIDForOwner(ctx, noteID, ownerUserID)
	if err != nil {
		return err
	}
	rows, err := s.store.DeleteNoteForOwner(ctx, noteID, ownerUserID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	_ = s.cache.Delete(ctx, noteCacheKey(ownerUserID.String(), noteID.String()))
	if current.ShareSlug != nil {
		_ = s.cache.Delete(ctx, sharedCacheKey(*current.ShareSlug))
	}
	return nil
}

// List returns a page of owner-scoped notes plus total-count metadata, using a
// short-lived cached result when the normalized filters match.
func (s *Service) List(ctx context.Context, filters ListFilters) (ListResult, error) {
	if filters.SearchMode == "" {
		filters.SearchMode = SearchModePlain
	}
	key, err := listCacheKey(filters)
	if err == nil {
		if payload, ok := s.cache.Get(ctx, key); ok {
			var cached ListResult
			if err := json.Unmarshal([]byte(payload), &cached); err == nil {
				return cached, nil
			}
		}
	}

	items, total, err := s.store.ListNotesForOwner(ctx, filters)
	if err != nil {
		return ListResult{}, err
	}
	result := ListResult{Notes: items, Total: total, Filters: filters}
	if payload, err := json.Marshal(result); err == nil && key != "" {
		_ = s.cache.Set(ctx, key, string(payload), s.listCacheTTL)
	}
	return result, nil
}

// ListTags returns the current owner-scoped tag vocabulary and note counts.
// The SQL layer does the aggregation so MCP and other callers all see the same
// deterministic ordering and counting rules.
func (s *Service) ListTags(ctx context.Context, ownerUserID uuid.UUID) ([]TagSummary, error) {
	return s.store.ListTagsForOwner(ctx, ownerUserID)
}

// RenameTag rewrites one owner-scoped tag across matching notes.
// PostgreSQL performs the ordered array rewrite so REST, UI, and MCP all teach
// the same normalization rules instead of rebuilding tag-set logic in Go.
func (s *Service) RenameTag(ctx context.Context, ownerUserID uuid.UUID, oldTag, newTag string) (RenameTagResult, error) {
	oldTag = strings.TrimSpace(oldTag)
	newTag = strings.TrimSpace(newTag)
	if oldTag == "" || newTag == "" {
		return RenameTagResult{}, errors.New("both old_tag and new_tag are required")
	}
	if oldTag == newTag {
		return RenameTagResult{OldTag: oldTag, NewTag: newTag, AffectedNotes: 0}, nil
	}

	updated, err := s.store.RenameTagForOwner(ctx, ownerUserID, oldTag, newTag)
	if err != nil {
		return RenameTagResult{}, err
	}
	for _, note := range updated {
		s.cacheNote(ctx, note)
		s.cacheSharedNote(ctx, note)
	}
	return RenameTagResult{
		OldTag:        oldTag,
		NewTag:        newTag,
		AffectedNotes: int64(len(updated)),
	}, nil
}

// FindRelatedNotes returns other owner-scoped notes ranked by overlapping tags.
// The SQL layer computes overlap and deterministic ordering so MCP gets a
// stable explanation of "related" without rebuilding set logic in Go.
func (s *Service) FindRelatedNotes(ctx context.Context, ownerUserID, noteID uuid.UUID, limit int32) ([]RelatedNote, error) {
	current, err := s.store.GetNoteByIDForOwner(ctx, noteID, ownerUserID)
	if err != nil {
		return nil, err
	}
	if len(current.Tags) == 0 {
		return []RelatedNote{}, nil
	}
	if limit <= 0 {
		limit = 5
	}
	return s.store.FindRelatedNotesForOwner(ctx, ownerUserID, noteID, limit)
}

// CreateSavedQuery stores a named, owner-scoped saved query. The query string
// is expected to already be normalized from validated filters so all transports
// can share the same saved-query semantics.
func (s *Service) CreateSavedQuery(ctx context.Context, input CreateSavedQueryInput) (SavedQuery, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return SavedQuery{}, errors.New("saved query name is required")
	}
	return s.store.CreateSavedQuery(ctx, input)
}

// GetSavedQuery returns one owner-scoped saved query by UUID.
func (s *Service) GetSavedQuery(ctx context.Context, ownerUserID, savedQueryID uuid.UUID) (SavedQuery, error) {
	return s.store.GetSavedQueryForOwner(ctx, savedQueryID, ownerUserID)
}

// ListSavedQueries returns the owner-scoped saved query list in deterministic
// repository order so REST, UI, and MCP all present the same saved filters.
func (s *Service) ListSavedQueries(ctx context.Context, ownerUserID uuid.UUID) ([]SavedQuery, error) {
	return s.store.ListSavedQueriesForOwner(ctx, ownerUserID)
}

// DeleteSavedQuery removes one owner-scoped saved query by UUID.
func (s *Service) DeleteSavedQuery(ctx context.Context, ownerUserID, savedQueryID uuid.UUID) error {
	rows, err := s.store.DeleteSavedQueryForOwner(ctx, savedQueryID, ownerUserID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) cacheNote(ctx context.Context, note Note) {
	payload, err := json.Marshal(note)
	if err == nil {
		_ = s.cache.Set(ctx, noteCacheKey(note.OwnerUserID.String(), note.ID.String()), string(payload), s.noteCacheTTL)
	}
	s.cacheSharedNote(ctx, note)
}

func (s *Service) cacheSharedNote(ctx context.Context, note Note) {
	if !note.Shared || note.ShareSlug == nil {
		return
	}
	payload, err := json.Marshal(note)
	if err == nil {
		_ = s.cache.Set(ctx, sharedCacheKey(*note.ShareSlug), string(payload), s.noteCacheTTL)
	}
}

func noteCacheKey(ownerID, noteID string) string {
	return fmt.Sprintf("notes:id:%s:%s", ownerID, noteID)
}

func sharedCacheKey(slug string) string {
	return fmt.Sprintf("notes:shared:%s", slug)
}

func listCacheKey(filters ListFilters) (string, error) {
	payload, err := json.Marshal(filters)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(payload)
	return fmt.Sprintf("notes:list:%s:%s", filters.OwnerUserID.String(), hex.EncodeToString(hash[:])), nil
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func randomSlug(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := strings.ToLower(base64.RawURLEncoding.EncodeToString(buf))
	if len(encoded) > length {
		return encoded[:length], nil
	}
	return encoded, nil
}
