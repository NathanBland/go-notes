package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/notes"
)

// sharedNoteResponse is the intentionally public shape for shared note reads.
// It omits internal identifiers so a shared link does not leak the note UUID or
// owner UUID back to unauthenticated callers.
type sharedNoteResponse struct {
	Title     *string   `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	Archived  bool      `json:"archived"`
	Shared    bool      `json:"shared"`
	ShareSlug *string   `json:"share_slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	input, fields, err := parseCreateNoteRequest(r, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}
	if len(fields) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request", fields)
		return
	}
	note, err := a.notesService.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_failed", "failed to create note", nil)
		return
	}
	writeData(w, http.StatusCreated, note)
}

func (a *API) handleListNotes(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	filters, fields := a.parseListFiltersWithSavedQuery(r.Context(), r.URL.Query(), user.ID)
	if len(fields) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid query parameters", fields)
		return
	}
	result, err := a.notesService.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_failed", "failed to list notes", nil)
		return
	}
	writeList(w, result.Notes, listMeta{
		Page:     result.Filters.Page,
		PageSize: result.Filters.PageSize,
		Total:    result.Total,
		Sort:     result.Filters.SortField,
		Order:    result.Filters.SortDirection,
	})
}

func (a *API) handleListSavedQueries(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	items, err := a.notesService.ListSavedQueries(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "saved_queries_error", "failed to list saved queries", nil)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"saved_queries": items})
}

func (a *API) handleCreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	input, fields, err := parseCreateSavedQueryRequest(r, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}
	if len(fields) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request", fields)
		return
	}
	values, err := url.ParseQuery(input.Query)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request", map[string]string{"query": "must be a valid query string"})
		return
	}
	filters, filterFields := a.parseListFiltersWithSavedQuery(r.Context(), values, user.ID)
	if len(filterFields) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request", filterFields)
		return
	}
	input.Query = notes.EncodeListFilters(filters)
	savedQuery, err := a.notesService.CreateSavedQuery(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "saved_queries_error", "failed to create saved query", nil)
		return
	}
	writeData(w, http.StatusCreated, savedQuery)
}

func (a *API) handleRenameTag(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	oldTag, newTag, fields, err := parseRenameTagRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}
	if len(fields) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request", fields)
		return
	}
	result, err := a.notesService.RenameTag(r.Context(), user.ID, oldTag, newTag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tags_error", "failed to rename tag", nil)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (a *API) handleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	savedQueryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "saved query id must be a UUID", nil)
		return
	}
	if err := a.notesService.DeleteSavedQuery(r.Context(), user.ID, savedQueryID); err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "saved query not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "saved_queries_error", "failed to delete saved query", nil)
		return
	}
	writeData(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (a *API) handleGetNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "note id must be a UUID", nil)
		return
	}
	note, err := a.notesService.GetByIDForOwner(r.Context(), user.ID, noteID)
	if err != nil {
		a.writeNotesError(w, err, "failed to load note")
		return
	}
	writeData(w, http.StatusOK, note)
}

func (a *API) handlePatchNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "note id must be a UUID", nil)
		return
	}
	input, fields, err := parsePatchNoteRequest(r, noteID, user.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", nil)
		return
	}
	if len(fields) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid request", fields)
		return
	}
	note, err := a.notesService.Patch(r.Context(), input)
	if err != nil {
		a.writeNotesError(w, err, "failed to update note")
		return
	}
	writeData(w, http.StatusOK, note)
}

func (a *API) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	noteID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "note id must be a UUID", nil)
		return
	}
	if err := a.notesService.Delete(r.Context(), user.ID, noteID); err != nil {
		a.writeNotesError(w, err, "failed to delete note")
		return
	}
	writeData(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (a *API) handleGetSharedNote(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid_slug", "share slug is required", nil)
		return
	}
	note, err := a.notesService.GetByShareSlug(r.Context(), slug)
	if err != nil {
		a.writeNotesError(w, err, "failed to load shared note")
		return
	}
	writeData(w, http.StatusOK, toSharedNoteResponse(note))
}

func (a *API) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "endpoint not found", nil)
}

func (a *API) writeNotesError(w http.ResponseWriter, err error, fallbackMessage string) {
	if errors.Is(err, notes.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "note not found", nil)
		return
	}
	writeError(w, http.StatusInternalServerError, "notes_error", fallbackMessage, nil)
}

func toSharedNoteResponse(note notes.Note) sharedNoteResponse {
	return sharedNoteResponse{
		Title:     note.Title,
		Content:   note.Content,
		Tags:      note.Tags,
		Archived:  note.Archived,
		Shared:    note.Shared,
		ShareSlug: note.ShareSlug,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}
}
