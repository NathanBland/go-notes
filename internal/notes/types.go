package notes

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("notes: not found")

const (
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusAll      = "all"

	TagMatchAny = "any"
	TagMatchAll = "all"

	SearchModePlain = "plain"
	SearchModeFTS   = "fts"
)

// Note is the API-facing note model. Nullable fields stay as pointers so the
// JSON layer can show the difference between missing values and empty strings.
type Note struct {
	ID          uuid.UUID `json:"id"`
	OwnerUserID uuid.UUID `json:"-"`
	Title       *string   `json:"title"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags"`
	Archived    bool      `json:"archived"`
	Shared      bool      `json:"shared"`
	ShareSlug   *string   `json:"share_slug"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateInput is the validated input needed to create a note.
type CreateInput struct {
	OwnerUserID uuid.UUID
	Title       *string
	Content     string
	Tags        []string
	Archived    bool
	Shared      bool
	ShareSlug   *string
}

// PatchInput uses pointer fields plus explicit "Set" flags.
// In Go this is the clearest way to teach PATCH semantics because a nil pointer
// alone cannot distinguish an omitted field from an explicit JSON null.
type PatchInput struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID

	TitleSet bool
	Title    *string

	ContentSet bool
	Content    *string

	TagsSet bool
	Tags    *[]string

	ArchivedSet bool
	Archived    *bool

	SharedSet bool
	Shared    *bool

	ShareSlugSet bool
	ShareSlug    *string
}

// ListFilters is the normalized, validated filter object shared by handlers,
// services, cache keys, and SQL queries.
type ListFilters struct {
	OwnerUserID   uuid.UUID  `json:"owner_user_id"`
	Page          int32      `json:"page"`
	PageSize      int32      `json:"page_size"`
	Search        *string    `json:"search,omitempty"`
	SearchMode    string     `json:"search_mode"`
	Status        string     `json:"status"`
	Shared        *bool      `json:"shared,omitempty"`
	HasTitle      *bool      `json:"has_title,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	TagCountMin   *int32     `json:"tag_count_min,omitempty"`
	TagCountMax   *int32     `json:"tag_count_max,omitempty"`
	TagMatchMode  string     `json:"tag_mode"`
	SortField     string     `json:"sort"`
	SortDirection string     `json:"order"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`
}

func (f ListFilters) Offset() int32 {
	return (f.Page - 1) * f.PageSize
}

// ListResult keeps paginated notes together with the filters and total count
// used to produce them.
type ListResult struct {
	Notes   []Note
	Total   int64
	Filters ListFilters
}

// TagSummary is the owner-scoped tag vocabulary shape used by MCP and docs.
// Count represents how many notes currently include the tag.
type TagSummary struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// RelatedNote keeps a related note together with the tag overlap used to rank
// it. This gives MCP clients enough structure to explain why a note was
// returned without moving overlap logic out of PostgreSQL.
type RelatedNote struct {
	Note           Note     `json:"note"`
	SharedTags     []string `json:"shared_tags"`
	SharedTagCount int32    `json:"shared_tag_count"`
}

// SavedQuery keeps a named, owner-scoped list query that can be reused across
// REST, the web UI, and MCP without inventing a second filter grammar.
type SavedQuery struct {
	ID          uuid.UUID `json:"id"`
	OwnerUserID uuid.UUID `json:"-"`
	Name        string    `json:"name"`
	Query       string    `json:"query"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateSavedQueryInput is the validated data needed to persist a saved query.
type CreateSavedQueryInput struct {
	OwnerUserID uuid.UUID
	Name        string
	Query       string
}

// Store defines the PostgreSQL-backed note persistence operations.
type Store interface {
	CreateNote(ctx context.Context, input CreateInput) (Note, error)
	GetNoteByIDForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (Note, error)
	GetNoteByShareSlug(ctx context.Context, shareSlug string) (Note, error)
	UpdateNotePatch(ctx context.Context, input PatchInput) (Note, error)
	DeleteNoteForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error)
	ListNotesForOwner(ctx context.Context, filters ListFilters) ([]Note, int64, error)
	ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]TagSummary, error)
	FindRelatedNotesForOwner(ctx context.Context, ownerUserID, noteID uuid.UUID, limit int32) ([]RelatedNote, error)
	CreateSavedQuery(ctx context.Context, input CreateSavedQueryInput) (SavedQuery, error)
	GetSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (SavedQuery, error)
	ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]SavedQuery, error)
	DeleteSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error)
}

// Cache is the tiny Valkey-facing contract used by the notes service.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// EncodeListFilters keeps saved-query persistence transport-neutral by turning
// normalized list filters back into a canonical query string. It intentionally
// omits owner scoping because saved queries are already owner-bound records.
func EncodeListFilters(filters ListFilters) string {
	values := url.Values{}
	if filters.Page > 1 {
		values.Set("page", strconv.Itoa(int(filters.Page)))
	}
	if filters.PageSize > 0 && filters.PageSize != 20 {
		values.Set("page_size", strconv.Itoa(int(filters.PageSize)))
	}
	if filters.Search != nil && *filters.Search != "" {
		values.Set("search", *filters.Search)
	}
	if filters.SearchMode != "" && filters.SearchMode != SearchModePlain {
		values.Set("search_mode", filters.SearchMode)
	}
	if filters.Status != "" && filters.Status != StatusActive {
		values.Set("status", filters.Status)
	}
	if filters.Shared != nil {
		values.Set("shared", strconv.FormatBool(*filters.Shared))
	}
	if filters.HasTitle != nil {
		values.Set("has_title", strconv.FormatBool(*filters.HasTitle))
	}
	for _, tag := range filters.Tags {
		values.Add("tag", tag)
	}
	if filters.TagCountMin != nil {
		values.Set("tag_count_min", strconv.Itoa(int(*filters.TagCountMin)))
	}
	if filters.TagCountMax != nil {
		values.Set("tag_count_max", strconv.Itoa(int(*filters.TagCountMax)))
	}
	if filters.TagMatchMode != "" && filters.TagMatchMode != TagMatchAny {
		values.Set("tag_mode", filters.TagMatchMode)
	}
	if filters.SortField != "" && filters.SortField != "updated_at" {
		values.Set("sort", filters.SortField)
	}
	if filters.SortDirection != "" && filters.SortDirection != "desc" {
		values.Set("order", filters.SortDirection)
	}
	if filters.CreatedAfter != nil {
		values.Set("created_after", filters.CreatedAfter.UTC().Format(time.RFC3339))
	}
	if filters.CreatedBefore != nil {
		values.Set("created_before", filters.CreatedBefore.UTC().Format(time.RFC3339))
	}
	if filters.UpdatedAfter != nil {
		values.Set("updated_after", filters.UpdatedAfter.UTC().Format(time.RFC3339))
	}
	if filters.UpdatedBefore != nil {
		values.Set("updated_before", filters.UpdatedBefore.UTC().Format(time.RFC3339))
	}
	return values.Encode()
}
