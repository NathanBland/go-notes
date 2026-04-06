package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/notes"
)

type createNoteRequest struct {
	Title    *string  `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
	Archived bool     `json:"archived"`
	Shared   bool     `json:"shared"`
}

type createSavedQueryRequest struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

func decodeJSONBody(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return rejectTrailingJSON(decoder)
}

func parseCreateNoteRequest(r *http.Request, ownerUserID uuid.UUID) (notes.CreateInput, map[string]string, error) {
	var payload createNoteRequest
	if err := decodeJSONBody(r, &payload); err != nil {
		return notes.CreateInput{}, nil, err
	}
	fields := map[string]string{}
	if strings.TrimSpace(payload.Content) == "" {
		fields["content"] = "is required"
	}
	if len(fields) > 0 {
		return notes.CreateInput{}, fields, nil
	}
	return notes.CreateInput{
		OwnerUserID: ownerUserID,
		Title:       payload.Title,
		Content:     payload.Content,
		Tags:        payload.Tags,
		Archived:    payload.Archived,
		Shared:      payload.Shared,
	}, nil, nil
}

func parsePatchNoteRequest(r *http.Request, noteID, ownerUserID uuid.UUID) (notes.PatchInput, map[string]string, error) {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()

	// We decode into raw JSON first so PATCH can distinguish:
	// - field omitted entirely
	// - field sent as null
	// - field sent with a concrete value
	// That difference matters in Go because a nil pointer alone is not enough to
	// tell “missing” from “explicitly cleared”.
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return notes.PatchInput{}, nil, err
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return notes.PatchInput{}, nil, err
	}

	input := notes.PatchInput{ID: noteID, OwnerUserID: ownerUserID}
	fields := map[string]string{}
	for key, value := range raw {
		switch key {
		case "title":
			input.TitleSet = true
			if string(value) == "null" {
				input.Title = nil
				continue
			}
			var title string
			if err := json.Unmarshal(value, &title); err != nil {
				fields["title"] = "must be a string or null"
				continue
			}
			input.Title = &title
		case "content":
			input.ContentSet = true
			if string(value) == "null" {
				fields["content"] = "may not be null"
				continue
			}
			var content string
			if err := json.Unmarshal(value, &content); err != nil {
				fields["content"] = "must be a string"
				continue
			}
			input.Content = &content
		case "tags":
			input.TagsSet = true
			if string(value) == "null" {
				fields["tags"] = "may not be null"
				continue
			}
			var tags []string
			if err := json.Unmarshal(value, &tags); err != nil {
				fields["tags"] = "must be an array of strings"
				continue
			}
			input.Tags = &tags
		case "archived":
			input.ArchivedSet = true
			if string(value) == "null" {
				fields["archived"] = "may not be null"
				continue
			}
			var archived bool
			if err := json.Unmarshal(value, &archived); err != nil {
				fields["archived"] = "must be a boolean"
				continue
			}
			input.Archived = &archived
		case "shared":
			input.SharedSet = true
			if string(value) == "null" {
				fields["shared"] = "may not be null"
				continue
			}
			var shared bool
			if err := json.Unmarshal(value, &shared); err != nil {
				fields["shared"] = "must be a boolean"
				continue
			}
			input.Shared = &shared
		default:
			fields[key] = "is not supported"
		}
	}

	if len(raw) == 0 {
		fields["body"] = "must include at least one field"
	}
	if input.ContentSet && input.Content != nil && strings.TrimSpace(*input.Content) == "" {
		fields["content"] = "may not be blank"
	}
	return input, fields, nil
}

func parseCreateSavedQueryRequest(r *http.Request, ownerUserID uuid.UUID) (notes.CreateSavedQueryInput, map[string]string, error) {
	var payload createSavedQueryRequest
	if err := decodeJSONBody(r, &payload); err != nil {
		return notes.CreateSavedQueryInput{}, nil, err
	}
	fields := map[string]string{}
	if strings.TrimSpace(payload.Name) == "" {
		fields["name"] = "is required"
	}
	return notes.CreateSavedQueryInput{
		OwnerUserID: ownerUserID,
		Name:        strings.TrimSpace(payload.Name),
		Query:       strings.TrimSpace(payload.Query),
	}, fields, nil
}

func parseListFilters(values url.Values, ownerUserID uuid.UUID) (notes.ListFilters, map[string]string) {
	filters := notes.ListFilters{
		OwnerUserID:   ownerUserID,
		Page:          1,
		PageSize:      20,
		SearchMode:    notes.SearchModePlain,
		Status:        notes.StatusActive,
		TagMatchMode:  notes.TagMatchAny,
		SortField:     "updated_at",
		SortDirection: "desc",
	}
	fields := map[string]string{}

	// These validations are part of the query-safety boundary. We only allow
	// known sort fields/directions and bounded page sizes so request input never
	// turns into raw ORDER BY fragments or unbounded scans in SQL.

	if page := strings.TrimSpace(values.Get("page")); page != "" {
		parsed, err := strconv.Atoi(page)
		if err != nil || parsed < 1 {
			fields["page"] = "must be a positive integer"
		} else {
			filters.Page = int32(parsed)
		}
	}
	if pageSize := strings.TrimSpace(values.Get("page_size")); pageSize != "" {
		parsed, err := strconv.Atoi(pageSize)
		if err != nil || parsed < 1 || parsed > 100 {
			fields["page_size"] = "must be between 1 and 100"
		} else {
			filters.PageSize = int32(parsed)
		}
	}
	if search := strings.TrimSpace(values.Get("search")); search != "" {
		filters.Search = &search
	}
	if searchMode := strings.TrimSpace(values.Get("search_mode")); searchMode != "" {
		switch searchMode {
		case notes.SearchModePlain, notes.SearchModeFTS:
			filters.SearchMode = searchMode
		default:
			fields["search_mode"] = "must be plain or fts"
		}
	}
	if status := strings.TrimSpace(values.Get("status")); status != "" {
		switch status {
		case notes.StatusActive, notes.StatusArchived, notes.StatusAll:
			filters.Status = status
		default:
			fields["status"] = "must be active, archived, or all"
		}
	}
	if shared := strings.TrimSpace(values.Get("shared")); shared != "" {
		parsed, err := strconv.ParseBool(shared)
		if err != nil {
			fields["shared"] = "must be true or false"
		} else {
			filters.Shared = &parsed
		}
	}
	if hasTitle := strings.TrimSpace(values.Get("has_title")); hasTitle != "" {
		parsed, err := strconv.ParseBool(hasTitle)
		if err != nil {
			fields["has_title"] = "must be true or false"
		} else {
			filters.HasTitle = &parsed
		}
	}
	if tags := parseTagFilters(values); len(tags) > 0 {
		filters.Tags = tags
	}
	parseOptionalInt32(values, "tag_count_min", &filters.TagCountMin, fields, 0)
	parseOptionalInt32(values, "tag_count_max", &filters.TagCountMax, fields, 0)
	if mode := strings.TrimSpace(values.Get("tag_mode")); mode != "" {
		switch mode {
		case notes.TagMatchAny, notes.TagMatchAll:
			filters.TagMatchMode = mode
		default:
			fields["tag_mode"] = "must be any or all"
		}
	}
	if sort := strings.TrimSpace(values.Get("sort")); sort != "" {
		switch sort {
		case "created_at", "updated_at", "title", "tag_count", "primary_tag", "relevance":
			filters.SortField = sort
		default:
			fields["sort"] = "must be created_at, updated_at, title, tag_count, primary_tag, or relevance"
		}
	}
	if order := strings.TrimSpace(values.Get("order")); order != "" {
		switch order {
		case "asc", "desc":
			filters.SortDirection = order
		default:
			fields["order"] = "must be asc or desc"
		}
	}
	parseTimeQuery(values, "created_after", &filters.CreatedAfter, fields)
	parseTimeQuery(values, "created_before", &filters.CreatedBefore, fields)
	parseTimeQuery(values, "updated_after", &filters.UpdatedAfter, fields)
	parseTimeQuery(values, "updated_before", &filters.UpdatedBefore, fields)
	if filters.TagCountMin != nil && filters.TagCountMax != nil && *filters.TagCountMin > *filters.TagCountMax {
		fields["tag_count_max"] = "must be greater than or equal to tag_count_min"
	}
	if filters.SortField == "relevance" {
		if filters.Search == nil || strings.TrimSpace(*filters.Search) == "" {
			fields["search"] = "is required when sort is relevance"
		}
		if filters.SearchMode != notes.SearchModeFTS {
			fields["search_mode"] = "must be fts when sort is relevance"
		}
	}
	if filters.SearchMode == notes.SearchModeFTS && (filters.Search == nil || strings.TrimSpace(*filters.Search) == "") {
		fields["search"] = "is required when search_mode is fts"
	}
	return filters, fields
}

func parseTagFilters(values url.Values) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)
	appendTag := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	for _, value := range values["tag"] {
		appendTag(value)
	}
	for _, value := range values["tags"] {
		for _, part := range strings.Split(value, ",") {
			appendTag(part)
		}
	}
	return result
}

func parseTimeQuery(values url.Values, key string, dest **time.Time, fields map[string]string) {
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		fields[key] = fmt.Sprintf("must be an RFC3339 timestamp, got %q", value)
		return
	}
	parsed = parsed.UTC()
	*dest = &parsed
}

func parseOptionalInt32(values url.Values, key string, target **int32, fields map[string]string, min int32) {
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || int32(parsed) < min {
		fields[key] = fmt.Sprintf("must be an integer greater than or equal to %d", min)
		return
	}
	converted := int32(parsed)
	*target = &converted
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain a single JSON value")
		}
		return err
	}
	return nil
}
