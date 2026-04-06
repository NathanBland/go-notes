package mcpapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/nathanbland/go-notes/internal/notes"
)

// NotesService is the shared service boundary used by both REST and MCP.
type NotesService interface {
	Create(ctx context.Context, input notes.CreateInput) (notes.Note, error)
	CreateSavedQuery(ctx context.Context, input notes.CreateSavedQueryInput) (notes.SavedQuery, error)
	Delete(ctx context.Context, ownerUserID, noteID uuid.UUID) error
	DeleteSavedQuery(ctx context.Context, ownerUserID, savedQueryID uuid.UUID) error
	FindRelatedNotes(ctx context.Context, ownerUserID, noteID uuid.UUID, limit int32) ([]notes.RelatedNote, error)
	GetByIDForOwner(ctx context.Context, ownerUserID, noteID uuid.UUID) (notes.Note, error)
	GetSavedQuery(ctx context.Context, ownerUserID, savedQueryID uuid.UUID) (notes.SavedQuery, error)
	ListTags(ctx context.Context, ownerUserID uuid.UUID) ([]notes.TagSummary, error)
	List(ctx context.Context, filters notes.ListFilters) (notes.ListResult, error)
	ListSavedQueries(ctx context.Context, ownerUserID uuid.UUID) ([]notes.SavedQuery, error)
	Patch(ctx context.Context, input notes.PatchInput) (notes.Note, error)
	RenameTag(ctx context.Context, ownerUserID uuid.UUID, oldTag, newTag string) (notes.RenameTagResult, error)
}

// Server wraps the MCP-facing note tools with a fixed local-development owner.
// This first MCP slice is intentionally local-only and uses a configured owner
// UUID until the project adds a fuller MCP-specific auth story.
type Server struct {
	notes   NotesService
	ownerID uuid.UUID
}

type listNotesArgs struct {
	SavedQueryID  string   `json:"saved_query_id"`
	Page          *int     `json:"page,omitempty"`
	PageSize      *int     `json:"page_size,omitempty"`
	Search        *string  `json:"search,omitempty"`
	SearchMode    *string  `json:"search_mode,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Shared        *bool    `json:"shared,omitempty"`
	HasTitle      *bool    `json:"has_title,omitempty"`
	Tag           *string  `json:"tag,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	TagCountMin   *int     `json:"tag_count_min,omitempty"`
	TagCountMax   *int     `json:"tag_count_max,omitempty"`
	TagMode       *string  `json:"tag_mode,omitempty"`
	Sort          *string  `json:"sort,omitempty"`
	Order         *string  `json:"order,omitempty"`
	CreatedAfter  *string  `json:"created_after,omitempty"`
	CreatedBefore *string  `json:"created_before,omitempty"`
	UpdatedAfter  *string  `json:"updated_after,omitempty"`
	UpdatedBefore *string  `json:"updated_before,omitempty"`
}

type getNoteArgs struct {
	ID string `json:"id"`
}

type createNoteArgs struct {
	Title    *string  `json:"title,omitempty"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags,omitempty"`
	Archived bool     `json:"archived"`
	Shared   bool     `json:"shared"`
}

type updateNoteArgs struct {
	ID          string   `json:"id"`
	Title       *string  `json:"title,omitempty"`
	ClearTitle  bool     `json:"clear_title"`
	Content     *string  `json:"content,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ReplaceTags bool     `json:"replace_tags"`
	Archived    *bool    `json:"archived,omitempty"`
	Shared      *bool    `json:"shared,omitempty"`
}

type modifyNoteTagsArgs struct {
	ID   string   `json:"id"`
	Tags []string `json:"tags"`
}

type renameTagArgs struct {
	OldTag string `json:"old_tag"`
	NewTag string `json:"new_tag"`
}

type noteIDArgs struct {
	ID string `json:"id"`
}

type findRelatedNotesArgs struct {
	ID    string `json:"id"`
	Limit *int   `json:"limit,omitempty"`
}

type saveQueryArgs struct {
	Name string `json:"name"`
	listNotesArgs
}

type listTagsOutput struct {
	Tags []notes.TagSummary `json:"tags"`
}

type listSavedQueriesOutput struct {
	SavedQueries []notes.SavedQuery `json:"saved_queries"`
}

type relatedNotesOutput struct {
	RelatedNotes []notes.RelatedNote `json:"related_notes"`
}

type listNotesOutput struct {
	Notes    []notes.Note `json:"notes"`
	Total    int64        `json:"total"`
	Page     int32        `json:"page"`
	PageSize int32        `json:"page_size"`
	Sort     string       `json:"sort"`
	Order    string       `json:"order"`
}

// NewServer creates the first stdio-oriented MCP server for go-notes.
func NewServer(notesService NotesService, ownerID uuid.UUID) *mcpserver.MCPServer {
	api := &Server{notes: notesService, ownerID: ownerID}

	s := mcpserver.NewMCPServer(
		"go-notes MCP",
		"0.1.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithInstructions("Local MCP access to go-notes. This stdio slice exposes owner-scoped note lifecycle, tag discovery, and filtering tools for local agent workflows."),
	)

	s.AddTool(api.listNotesTool(), mcp.NewTypedToolHandler(api.handleListNotes))
	s.AddTool(api.getNoteTool(), mcp.NewTypedToolHandler(api.handleGetNote))
	s.AddTool(api.findRelatedNotesTool(), mcp.NewTypedToolHandler(api.handleFindRelatedNotes))
	s.AddTool(api.createNoteTool(), mcp.NewTypedToolHandler(api.handleCreateNote))
	s.AddTool(api.updateNoteTool(), mcp.NewTypedToolHandler(api.handleUpdateNote))
	s.AddTool(api.deleteNoteTool(), mcp.NewTypedToolHandler(api.handleDeleteNote))
	s.AddTool(api.listSavedQueriesTool(), mcp.NewTypedToolHandler(api.handleListSavedQueries))
	s.AddTool(api.saveQueryTool(), mcp.NewTypedToolHandler(api.handleSaveQuery))
	s.AddTool(api.deleteSavedQueryTool(), mcp.NewTypedToolHandler(api.handleDeleteSavedQuery))
	s.AddTool(api.listTagsTool(), mcp.NewTypedToolHandler(api.handleListTags))
	s.AddTool(api.renameTagTool(), mcp.NewTypedToolHandler(api.handleRenameTag))
	s.AddTool(api.setNoteTagsTool(), mcp.NewTypedToolHandler(api.handleSetNoteTags))
	s.AddTool(api.shareNoteTool(), mcp.NewTypedToolHandler(api.handleShareNote))
	s.AddTool(api.unshareNoteTool(), mcp.NewTypedToolHandler(api.handleUnshareNote))
	s.AddTool(api.archiveNoteTool(), mcp.NewTypedToolHandler(api.handleArchiveNote))
	s.AddTool(api.unarchiveNoteTool(), mcp.NewTypedToolHandler(api.handleUnarchiveNote))
	s.AddTool(api.addNoteTagsTool(), mcp.NewTypedToolHandler(api.handleAddNoteTags))
	s.AddTool(api.removeNoteTagsTool(), mcp.NewTypedToolHandler(api.handleRemoveNoteTags))
	return s
}

func (s *Server) listNotesTool() mcp.Tool {
	return mcp.NewTool("list_notes",
		mcp.WithDescription("List notes for the configured local MCP owner with filtering, sorting, and pagination."),
		mcp.WithString("saved_query_id", mcp.Description("Optional saved query UUID to preload before applying explicit overrides")),
		mcp.WithNumber("page", mcp.Description("Page number starting at 1"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("page_size", mcp.Description("Page size from 1 to 100"), mcp.DefaultNumber(20), mcp.Min(1), mcp.Max(100)),
		mcp.WithString("search", mcp.Description("Search term across note fields")),
		mcp.WithString("search_mode", mcp.Description("Search behavior"), mcp.Enum(notes.SearchModePlain, notes.SearchModeFTS)),
		mcp.WithString("status", mcp.Description("Note status filter"), mcp.Enum(notes.StatusActive, notes.StatusArchived, notes.StatusAll)),
		mcp.WithBoolean("shared", mcp.Description("Filter for shared or non-shared notes")),
		mcp.WithBoolean("has_title", mcp.Description("Filter for notes that do or do not have a title")),
		mcp.WithString("tag", mcp.Description("Single-tag filter kept for compatibility with the earlier MCP slice")),
		mcp.WithArray("tags", mcp.Description("Optional list of tags to filter by"), mcp.WithStringItems()),
		mcp.WithNumber("tag_count_min", mcp.Description("Minimum number of tags on a note"), mcp.Min(0)),
		mcp.WithNumber("tag_count_max", mcp.Description("Maximum number of tags on a note"), mcp.Min(0)),
		mcp.WithString("tag_mode", mcp.Description("How multi-tag filters match"), mcp.Enum(notes.TagMatchAny, notes.TagMatchAll)),
		mcp.WithString("sort", mcp.Description("Sort field"), mcp.Enum("created_at", "updated_at", "title", "tag_count", "primary_tag", "relevance")),
		mcp.WithString("order", mcp.Description("Sort order"), mcp.Enum("asc", "desc")),
		mcp.WithString("created_after", mcp.Description("RFC3339 lower bound for created_at")),
		mcp.WithString("created_before", mcp.Description("RFC3339 upper bound for created_at")),
		mcp.WithString("updated_after", mcp.Description("RFC3339 lower bound for updated_at")),
		mcp.WithString("updated_before", mcp.Description("RFC3339 upper bound for updated_at")),
	)
}

func (s *Server) getNoteTool() mcp.Tool {
	return mcp.NewTool("get_note",
		mcp.WithDescription("Get a single note by UUID for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
	)
}

func (s *Server) createNoteTool() mcp.Tool {
	return mcp.NewTool("create_note",
		mcp.WithDescription("Create a note for the configured local MCP owner."),
		mcp.WithString("title", mcp.Description("Optional note title")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Note body content")),
		mcp.WithArray("tags", mcp.Description("Optional note tags"), mcp.WithStringItems()),
		mcp.WithBoolean("archived", mcp.Description("Whether the note starts archived"), mcp.DefaultBool(false)),
		mcp.WithBoolean("shared", mcp.Description("Whether the note should be shared publicly"), mcp.DefaultBool(false)),
	)
}

func (s *Server) findRelatedNotesTool() mcp.Tool {
	return mcp.NewTool("find_related_notes",
		mcp.WithDescription("Find owner-scoped notes related to a source note by overlapping tags."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Source note UUID")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of related notes to return"), mcp.DefaultNumber(5), mcp.Min(1), mcp.Max(20)),
	)
}

func (s *Server) updateNoteTool() mcp.Tool {
	return mcp.NewTool("update_note",
		mcp.WithDescription("Apply a partial update to a note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
		mcp.WithString("title", mcp.Description("Optional new title for the note")),
		mcp.WithBoolean("clear_title", mcp.Description("Clear the current title instead of setting one")),
		mcp.WithString("content", mcp.Description("Optional replacement content for the note")),
		mcp.WithArray("tags", mcp.Description("Optional replacement tag list"), mcp.WithStringItems()),
		mcp.WithBoolean("replace_tags", mcp.Description("Replace the note tags with the provided tags array"), mcp.DefaultBool(false)),
		mcp.WithBoolean("archived", mcp.Description("Optional archived state")),
		mcp.WithBoolean("shared", mcp.Description("Optional shared state")),
	)
}

func (s *Server) addNoteTagsTool() mcp.Tool {
	return mcp.NewTool("add_note_tags",
		mcp.WithDescription("Add one or more tags to an existing note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
		mcp.WithArray("tags", mcp.Required(), mcp.Description("Tags to add"), mcp.WithStringItems()),
	)
}

func (s *Server) deleteNoteTool() mcp.Tool {
	return mcp.NewTool("delete_note",
		mcp.WithDescription("Delete an existing note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
	)
}

func (s *Server) listSavedQueriesTool() mcp.Tool {
	return mcp.NewTool("list_saved_queries",
		mcp.WithDescription("List owner-scoped saved queries for the configured local MCP owner."),
	)
}

func (s *Server) saveQueryTool() mcp.Tool {
	return mcp.NewTool("save_query",
		mcp.WithDescription("Save a named note-list query for the configured local MCP owner using the same filter surface as list_notes."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable saved query name")),
		mcp.WithString("saved_query_id", mcp.Description("Optional existing saved query to preload before applying explicit overrides")),
		mcp.WithNumber("page", mcp.Description("Page number starting at 1"), mcp.DefaultNumber(1), mcp.Min(1)),
		mcp.WithNumber("page_size", mcp.Description("Page size from 1 to 100"), mcp.DefaultNumber(20), mcp.Min(1), mcp.Max(100)),
		mcp.WithString("search", mcp.Description("Search term across note fields")),
		mcp.WithString("search_mode", mcp.Description("Search behavior"), mcp.Enum(notes.SearchModePlain, notes.SearchModeFTS)),
		mcp.WithString("status", mcp.Description("Note status filter"), mcp.Enum(notes.StatusActive, notes.StatusArchived, notes.StatusAll)),
		mcp.WithBoolean("shared", mcp.Description("Filter for shared or non-shared notes")),
		mcp.WithBoolean("has_title", mcp.Description("Filter for notes that do or do not have a title")),
		mcp.WithString("tag", mcp.Description("Single-tag filter kept for compatibility with the earlier MCP slice")),
		mcp.WithArray("tags", mcp.Description("Optional list of tags to filter by"), mcp.WithStringItems()),
		mcp.WithNumber("tag_count_min", mcp.Description("Minimum number of tags on a note"), mcp.Min(0)),
		mcp.WithNumber("tag_count_max", mcp.Description("Maximum number of tags on a note"), mcp.Min(0)),
		mcp.WithString("tag_mode", mcp.Description("How multi-tag filters match"), mcp.Enum(notes.TagMatchAny, notes.TagMatchAll)),
		mcp.WithString("sort", mcp.Description("Sort field"), mcp.Enum("created_at", "updated_at", "title", "tag_count", "primary_tag", "relevance")),
		mcp.WithString("order", mcp.Description("Sort order"), mcp.Enum("asc", "desc")),
		mcp.WithString("created_after", mcp.Description("RFC3339 lower bound for created_at")),
		mcp.WithString("created_before", mcp.Description("RFC3339 upper bound for created_at")),
		mcp.WithString("updated_after", mcp.Description("RFC3339 lower bound for updated_at")),
		mcp.WithString("updated_before", mcp.Description("RFC3339 upper bound for updated_at")),
	)
}

func (s *Server) deleteSavedQueryTool() mcp.Tool {
	return mcp.NewTool("delete_saved_query",
		mcp.WithDescription("Delete an owner-scoped saved query for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Saved query UUID")),
	)
}

func (s *Server) listTagsTool() mcp.Tool {
	return mcp.NewTool("list_tags",
		mcp.WithDescription("List the current owner-scoped tag vocabulary with note counts for the configured local MCP owner."),
	)
}

func (s *Server) renameTagTool() mcp.Tool {
	return mcp.NewTool("rename_tag",
		mcp.WithDescription("Rename one owner-scoped tag across matching notes for the configured local MCP owner."),
		mcp.WithString("old_tag", mcp.Required(), mcp.Description("Existing tag name to replace")),
		mcp.WithString("new_tag", mcp.Required(), mcp.Description("Replacement tag name")),
	)
}

func (s *Server) setNoteTagsTool() mcp.Tool {
	return mcp.NewTool("set_note_tags",
		mcp.WithDescription("Replace the full tag set on an existing note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
		mcp.WithArray("tags", mcp.Required(), mcp.Description("Replacement tag list"), mcp.WithStringItems()),
	)
}

func (s *Server) shareNoteTool() mcp.Tool {
	return mcp.NewTool("share_note",
		mcp.WithDescription("Mark a note as shared for the configured local MCP owner and return the resulting share slug."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
	)
}

func (s *Server) unshareNoteTool() mcp.Tool {
	return mcp.NewTool("unshare_note",
		mcp.WithDescription("Stop sharing a note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
	)
}

func (s *Server) archiveNoteTool() mcp.Tool {
	return mcp.NewTool("archive_note",
		mcp.WithDescription("Archive a note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
	)
}

func (s *Server) unarchiveNoteTool() mcp.Tool {
	return mcp.NewTool("unarchive_note",
		mcp.WithDescription("Unarchive a note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
	)
}

func (s *Server) removeNoteTagsTool() mcp.Tool {
	return mcp.NewTool("remove_note_tags",
		mcp.WithDescription("Remove one or more tags from an existing note for the configured local MCP owner."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Note UUID")),
		mcp.WithArray("tags", mcp.Required(), mcp.Description("Tags to remove"), mcp.WithStringItems()),
	)
}

func (s *Server) handleListNotes(ctx context.Context, _ mcp.CallToolRequest, args listNotesArgs) (*mcp.CallToolResult, error) {
	filters, err := s.resolveListFilters(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := s.notes.List(ctx, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list notes: %v", err)), nil
	}
	output := listNotesOutput{
		Notes:    result.Notes,
		Total:    result.Total,
		Page:     result.Filters.Page,
		PageSize: result.Filters.PageSize,
		Sort:     result.Filters.SortField,
		Order:    result.Filters.SortDirection,
	}
	return mcp.NewToolResultStructured(output, fmt.Sprintf("Returned %d notes (total=%d).", len(output.Notes), output.Total)), nil
}

func (s *Server) handleGetNote(ctx context.Context, _ mcp.CallToolRequest, args getNoteArgs) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}
	note, err := s.notes.GetByIDForOwner(ctx, s.ownerID, noteID)
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to load note: %v", err)), nil
	}
	return mcp.NewToolResultStructured(note, fmt.Sprintf("Loaded note %s.", note.ID)), nil
}

func (s *Server) handleFindRelatedNotes(ctx context.Context, _ mcp.CallToolRequest, args findRelatedNotesArgs) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}
	limit := int32(5)
	if args.Limit != nil {
		if *args.Limit < 1 || *args.Limit > 20 {
			return mcp.NewToolResultError("limit must be between 1 and 20"), nil
		}
		limit = int32(*args.Limit)
	}
	items, err := s.notes.FindRelatedNotes(ctx, s.ownerID, noteID, limit)
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to find related notes: %v", err)), nil
	}
	output := relatedNotesOutput{RelatedNotes: items}
	return mcp.NewToolResultStructured(output, fmt.Sprintf("Returned %d related notes.", len(items))), nil
}

func (s *Server) handleCreateNote(ctx context.Context, _ mcp.CallToolRequest, args createNoteArgs) (*mcp.CallToolResult, error) {
	if strings.TrimSpace(args.Content) == "" {
		return mcp.NewToolResultError("content is required"), nil
	}

	var title *string
	if args.Title != nil {
		trimmed := strings.TrimSpace(*args.Title)
		if trimmed != "" {
			title = &trimmed
		}
	}

	created, err := s.notes.Create(ctx, notes.CreateInput{
		OwnerUserID: s.ownerID,
		Title:       title,
		Content:     args.Content,
		Tags:        normalizeTags(args.Tags),
		Archived:    args.Archived,
		Shared:      args.Shared,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create note: %v", err)), nil
	}
	return mcp.NewToolResultStructured(created, fmt.Sprintf("Created note %s.", created.ID)), nil
}

func (s *Server) handleUpdateNote(ctx context.Context, _ mcp.CallToolRequest, args updateNoteArgs) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}

	input := notes.PatchInput{
		ID:          noteID,
		OwnerUserID: s.ownerID,
	}

	if args.ClearTitle {
		input.TitleSet = true
		input.Title = nil
	}
	if args.Title != nil {
		input.TitleSet = true
		trimmed := strings.TrimSpace(*args.Title)
		input.Title = &trimmed
	}
	if args.Content != nil {
		trimmed := strings.TrimSpace(*args.Content)
		if trimmed == "" {
			return mcp.NewToolResultError("content cannot be blank when provided"), nil
		}
		input.ContentSet = true
		input.Content = &trimmed
	}
	if args.ReplaceTags {
		normalized := normalizeTags(args.Tags)
		input.TagsSet = true
		input.Tags = &normalized
	}
	if args.Archived != nil {
		input.ArchivedSet = true
		input.Archived = args.Archived
	}
	if args.Shared != nil {
		input.SharedSet = true
		input.Shared = args.Shared
	}
	if !input.TitleSet && !input.ContentSet && !input.TagsSet && !input.ArchivedSet && !input.SharedSet {
		return mcp.NewToolResultError("update_note requires at least one field to change"), nil
	}

	updated, err := s.notes.Patch(ctx, input)
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to update note: %v", err)), nil
	}
	return mcp.NewToolResultStructured(updated, fmt.Sprintf("Updated note %s.", updated.ID)), nil
}

func (s *Server) handleDeleteNote(ctx context.Context, _ mcp.CallToolRequest, args noteIDArgs) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}
	if err := s.notes.Delete(ctx, s.ownerID, noteID); err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete note: %v", err)), nil
	}
	return mcp.NewToolResultStructured(map[string]any{"id": noteID.String(), "deleted": true}, fmt.Sprintf("Deleted note %s.", noteID)), nil
}

func (s *Server) handleListSavedQueries(ctx context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
	items, err := s.notes.ListSavedQueries(ctx, s.ownerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list saved queries: %v", err)), nil
	}
	output := listSavedQueriesOutput{SavedQueries: items}
	return mcp.NewToolResultStructured(output, fmt.Sprintf("Returned %d saved queries.", len(items))), nil
}

func (s *Server) handleSaveQuery(ctx context.Context, _ mcp.CallToolRequest, args saveQueryArgs) (*mcp.CallToolResult, error) {
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	filters, err := s.resolveListFilters(ctx, args.listNotesArgs)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	savedQuery, err := s.notes.CreateSavedQuery(ctx, notes.CreateSavedQueryInput{
		OwnerUserID: s.ownerID,
		Name:        name,
		Query:       notes.EncodeListFilters(filters),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save query: %v", err)), nil
	}
	return mcp.NewToolResultStructured(savedQuery, fmt.Sprintf("Saved query %s.", savedQuery.ID)), nil
}

func (s *Server) handleDeleteSavedQuery(ctx context.Context, _ mcp.CallToolRequest, args noteIDArgs) (*mcp.CallToolResult, error) {
	savedQueryID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}
	if err := s.notes.DeleteSavedQuery(ctx, s.ownerID, savedQueryID); err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("saved query not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete saved query: %v", err)), nil
	}
	return mcp.NewToolResultStructured(map[string]any{"id": savedQueryID.String(), "deleted": true}, fmt.Sprintf("Deleted saved query %s.", savedQueryID)), nil
}

func (s *Server) handleListTags(ctx context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
	tags, err := s.notes.ListTags(ctx, s.ownerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tags: %v", err)), nil
	}
	output := listTagsOutput{Tags: tags}
	return mcp.NewToolResultStructured(output, fmt.Sprintf("Returned %d tags.", len(tags))), nil
}

func (s *Server) handleRenameTag(ctx context.Context, _ mcp.CallToolRequest, args renameTagArgs) (*mcp.CallToolResult, error) {
	oldTag := strings.TrimSpace(args.OldTag)
	newTag := strings.TrimSpace(args.NewTag)
	if oldTag == "" || newTag == "" {
		return mcp.NewToolResultError("old_tag and new_tag are required"), nil
	}
	if oldTag == newTag {
		return mcp.NewToolResultError("new_tag must be different from old_tag"), nil
	}
	result, err := s.notes.RenameTag(ctx, s.ownerID, oldTag, newTag)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to rename tag: %v", err)), nil
	}
	return mcp.NewToolResultStructured(result, fmt.Sprintf("Renamed %q to %q on %d notes.", result.OldTag, result.NewTag, result.AffectedNotes)), nil
}

func (s *Server) handleSetNoteTags(ctx context.Context, _ mcp.CallToolRequest, args modifyNoteTagsArgs) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}
	tags := normalizeTags(args.Tags)
	updated, err := s.notes.Patch(ctx, notes.PatchInput{
		ID:          noteID,
		OwnerUserID: s.ownerID,
		TagsSet:     true,
		Tags:        &tags,
	})
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to replace note tags: %v", err)), nil
	}
	return mcp.NewToolResultStructured(updated, fmt.Sprintf("Replaced tags on note %s.", updated.ID)), nil
}

func (s *Server) handleShareNote(ctx context.Context, _ mcp.CallToolRequest, args noteIDArgs) (*mcp.CallToolResult, error) {
	return s.patchBoolState(ctx, args.ID, "share", func(input *notes.PatchInput, value bool) {
		input.SharedSet = true
		input.Shared = &value
	}, true)
}

func (s *Server) handleUnshareNote(ctx context.Context, _ mcp.CallToolRequest, args noteIDArgs) (*mcp.CallToolResult, error) {
	return s.patchBoolState(ctx, args.ID, "unshare", func(input *notes.PatchInput, value bool) {
		input.SharedSet = true
		input.Shared = &value
	}, false)
}

func (s *Server) handleArchiveNote(ctx context.Context, _ mcp.CallToolRequest, args noteIDArgs) (*mcp.CallToolResult, error) {
	return s.patchBoolState(ctx, args.ID, "archive", func(input *notes.PatchInput, value bool) {
		input.ArchivedSet = true
		input.Archived = &value
	}, true)
}

func (s *Server) handleUnarchiveNote(ctx context.Context, _ mcp.CallToolRequest, args noteIDArgs) (*mcp.CallToolResult, error) {
	return s.patchBoolState(ctx, args.ID, "unarchive", func(input *notes.PatchInput, value bool) {
		input.ArchivedSet = true
		input.Archived = &value
	}, false)
}

func (s *Server) patchBoolState(ctx context.Context, id string, action string, set func(*notes.PatchInput, bool), value bool) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), nil
	}
	input := notes.PatchInput{
		ID:          noteID,
		OwnerUserID: s.ownerID,
	}
	set(&input, value)
	updated, err := s.notes.Patch(ctx, input)
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to %s note: %v", action, err)), nil
	}
	var verb string
	switch action {
	case "share":
		verb = "Shared"
	case "unshare":
		verb = "Unshared"
	case "archive":
		verb = "Archived"
	case "unarchive":
		verb = "Unarchived"
	default:
		verb = "Updated"
	}
	return mcp.NewToolResultStructured(updated, fmt.Sprintf("%s note %s.", verb, updated.ID)), nil
}

func (s *Server) handleAddNoteTags(ctx context.Context, _ mcp.CallToolRequest, args modifyNoteTagsArgs) (*mcp.CallToolResult, error) {
	updated, err := s.modifyNoteTags(ctx, args, true)
	if err != nil {
		return updated, nil
	}
	return updated, nil
}

func (s *Server) handleRemoveNoteTags(ctx context.Context, _ mcp.CallToolRequest, args modifyNoteTagsArgs) (*mcp.CallToolResult, error) {
	updated, err := s.modifyNoteTags(ctx, args, false)
	if err != nil {
		return updated, nil
	}
	return updated, nil
}

func (s *Server) modifyNoteTags(ctx context.Context, args modifyNoteTagsArgs, adding bool) (*mcp.CallToolResult, error) {
	noteID, err := uuid.Parse(strings.TrimSpace(args.ID))
	if err != nil {
		return mcp.NewToolResultError("id must be a valid UUID"), err
	}
	tags := normalizeTags(args.Tags)
	if len(tags) == 0 {
		return mcp.NewToolResultError("at least one non-empty tag is required"), fmt.Errorf("empty tags")
	}

	current, err := s.notes.GetByIDForOwner(ctx, s.ownerID, noteID)
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), err
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to load note: %v", err)), err
	}

	nextTags := current.Tags
	action := "Added"
	if adding {
		nextTags = normalizeTags(append(current.Tags, tags...))
	} else {
		nextTags = removeTags(current.Tags, tags)
		action = "Removed"
	}

	updated, err := s.notes.Patch(ctx, notes.PatchInput{
		ID:          noteID,
		OwnerUserID: s.ownerID,
		TagsSet:     true,
		Tags:        &nextTags,
	})
	if err != nil {
		if err == notes.ErrNotFound {
			return mcp.NewToolResultError("note not found"), err
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to update note tags: %v", err)), err
	}
	return mcp.NewToolResultStructured(updated, fmt.Sprintf("%s tags on note %s.", action, updated.ID)), nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return uniqueStrings(normalized)
}

func removeTags(current, removals []string) []string {
	if len(current) == 0 {
		return []string{}
	}
	removeSet := make(map[string]struct{}, len(removals))
	for _, tag := range removals {
		removeSet[tag] = struct{}{}
	}
	result := make([]string, 0, len(current))
	for _, tag := range current {
		if _, ok := removeSet[tag]; ok {
			continue
		}
		result = append(result, tag)
	}
	return normalizeTags(result)
}

func (s *Server) resolveListFilters(ctx context.Context, args listNotesArgs) (notes.ListFilters, error) {
	if strings.TrimSpace(args.SavedQueryID) == "" {
		return normalizeListArgs(s.ownerID, args)
	}
	savedQueryID, err := uuid.Parse(strings.TrimSpace(args.SavedQueryID))
	if err != nil {
		return notes.ListFilters{}, fmt.Errorf("saved_query_id must be a valid UUID")
	}
	savedQuery, err := s.notes.GetSavedQuery(ctx, s.ownerID, savedQueryID)
	if err != nil {
		if err == notes.ErrNotFound {
			return notes.ListFilters{}, fmt.Errorf("saved_query_id was not found")
		}
		return notes.ListFilters{}, fmt.Errorf("failed to load saved query: %w", err)
	}
	baseValues, err := url.ParseQuery(savedQuery.Query)
	if err != nil {
		return notes.ListFilters{}, fmt.Errorf("saved query contains invalid stored filters")
	}
	values := overlayExplicitValues(baseValues, valuesFromListArgs(args))
	return normalizeListValues(s.ownerID, values)
}

func overlayExplicitValues(base, explicit url.Values) url.Values {
	result := url.Values{}
	for key, items := range base {
		for _, item := range items {
			result.Add(key, item)
		}
	}
	for key, items := range explicit {
		if key == "saved_query_id" {
			continue
		}
		result.Del(key)
		for _, item := range items {
			result.Add(key, item)
		}
	}
	return result
}
