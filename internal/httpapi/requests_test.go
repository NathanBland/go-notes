package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/notes"
)

func TestParseListFiltersDefaults(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	filters, fields := parseListFilters(httptest.NewRequest("GET", "/api/v1/notes", nil).URL.Query(), ownerID)
	if len(fields) != 0 {
		t.Fatalf("expected no validation errors, got %v", fields)
	}
	if filters.Page != 1 || filters.PageSize != 20 {
		t.Fatalf("unexpected defaults: %+v", filters)
	}
	if filters.SearchMode != notes.SearchModePlain || filters.Status != "active" || filters.TagMatchMode != "any" || filters.SortField != "updated_at" || filters.SortDirection != "desc" {
		t.Fatalf("unexpected sort/status defaults: %+v", filters)
	}
}

func TestParseListFiltersValidation(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	req := httptest.NewRequest("GET", "/api/v1/notes?page=0&page_size=200&status=nope&search_mode=bad&has_title=nope&tag_count_min=-1&tag_count_max=1&tag_mode=weird&sort=relevance&order=sideways&created_after=bad", nil)
	_, fields := parseListFilters(req.URL.Query(), ownerID)
	for _, key := range []string{"page", "page_size", "status", "search_mode", "has_title", "tag_count_min", "tag_mode", "order", "created_after", "search"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("expected validation error for %s, got %v", key, fields)
		}
	}
}

func TestParseListFiltersValidValues(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	req := httptest.NewRequest("GET", "/api/v1/notes?page=2&page_size=10&search=golang+api&search_mode=fts&status=archived&shared=true&has_title=true&tag=work&tags=alpha,beta&tag_count_min=1&tag_count_max=4&tag_mode=all&sort=relevance&order=asc&created_after=2026-04-01T01:02:03-07:00&created_before=2026-04-02T01:02:03-07:00&updated_after=2026-04-03T01:02:03-07:00&updated_before=2026-04-04T01:02:03-07:00", nil)
	filters, fields := parseListFilters(req.URL.Query(), ownerID)
	if len(fields) != 0 {
		t.Fatalf("expected no validation errors, got %v", fields)
	}
	if filters.Page != 2 || filters.PageSize != 10 || filters.Status != "archived" {
		t.Fatalf("unexpected pagination/status filters: %+v", filters)
	}
	if filters.Search == nil || *filters.Search != "golang api" || len(filters.Tags) != 3 || filters.Tags[0] != "work" || filters.Tags[2] != "beta" {
		t.Fatalf("expected search and tags to be populated, got %+v", filters)
	}
	if filters.Search == nil || *filters.Search != "golang api" || filters.SearchMode != notes.SearchModeFTS {
		t.Fatalf("expected search mode and query to be populated, got %+v", filters)
	}
	if filters.Shared == nil || !*filters.Shared || filters.HasTitle == nil || !*filters.HasTitle || filters.TagCountMin == nil || *filters.TagCountMin != 1 || filters.TagCountMax == nil || *filters.TagCountMax != 4 || filters.TagMatchMode != "all" || filters.SortField != "relevance" || filters.SortDirection != "asc" {
		t.Fatalf("unexpected shared/sort filters: %+v", filters)
	}
	if filters.CreatedAfter == nil || filters.CreatedBefore == nil || filters.UpdatedAfter == nil || filters.UpdatedBefore == nil {
		t.Fatalf("expected all timestamp filters to be populated, got %+v", filters)
	}
}

func TestParseListFiltersRejectsFTSRequirements(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	req := httptest.NewRequest("GET", "/api/v1/notes?search_mode=fts", nil)
	_, fields := parseListFilters(req.URL.Query(), ownerID)
	if fields["search"] == "" {
		t.Fatalf("expected search requirement error, got %v", fields)
	}

	req = httptest.NewRequest("GET", "/api/v1/notes?search=go&sort=relevance", nil)
	_, fields = parseListFilters(req.URL.Query(), ownerID)
	if fields["search_mode"] == "" {
		t.Fatalf("expected relevance search_mode error, got %v", fields)
	}
}

func TestParseTagFiltersDeduplicatesValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/?tag=work&tag=work&tags=alpha,beta,alpha", nil)
	tags := parseTagFilters(req.URL.Query())
	if len(tags) != 3 || tags[0] != "work" || tags[1] != "alpha" || tags[2] != "beta" {
		t.Fatalf("unexpected tag filters: %#v", tags)
	}
}

func TestParsePatchNoteRequestTracksNullAndPresence(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	body := strings.NewReader(`{"title":null,"content":" updated ","shared":true}`)
	req := httptest.NewRequest("PATCH", "/api/v1/notes/1", body)
	input, fields, err := parsePatchNoteRequest(req, noteID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("unexpected validation errors: %v", fields)
	}
	if !input.TitleSet || input.Title != nil {
		t.Fatalf("expected title to be explicitly set to null, got %+v", input)
	}
	if !input.ContentSet || input.Content == nil || *input.Content != " updated " {
		t.Fatalf("expected content to be present, got %+v", input)
	}
	if !input.SharedSet || input.Shared == nil || !*input.Shared {
		t.Fatalf("expected shared to be true, got %+v", input)
	}
}

func TestParseCreateNoteRequest(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"title":"hello","content":" body ","tags":["go"],"shared":true}`))
	input, fields, err := parseCreateNoteRequest(req, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("unexpected validation errors: %v", fields)
	}
	if input.OwnerUserID != ownerID || input.Title == nil || *input.Title != "hello" {
		t.Fatalf("unexpected create input: %+v", input)
	}
}

func TestParseCreateNoteRequestValidationAndDecodeErrors(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"   "}`))
	_, fields, err := parseCreateNoteRequest(req, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["content"] == "" {
		t.Fatalf("expected content validation error, got %v", fields)
	}

	var dst struct {
		Content string `json:"content"`
	}
	badJSON := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":`))
	if err := decodeJSONBody(badJSON, &dst); err == nil {
		t.Fatal("expected invalid JSON decode error")
	}

	unknown := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"ok","extra":true}`))
	err = decodeJSONBody(unknown, &dst)
	if err == nil {
		t.Fatal("expected unknown field decode error")
	}

	trailing := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"ok"}{"content":"again"}`))
	err = decodeJSONBody(trailing, &dst)
	if err == nil {
		t.Fatal("expected trailing JSON decode error")
	}
}

func TestParsePatchNoteRequestValidationCases(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notes/1", strings.NewReader(`{}`))
	_, fields, err := parsePatchNoteRequest(req, noteID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["body"] == "" {
		t.Fatalf("expected empty-body validation error, got %v", fields)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/1", strings.NewReader(`{"content":null,"tags":null,"archived":"nope","wat":"x"}`))
	_, fields, err = parsePatchNoteRequest(req, noteID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, key := range []string{"content", "tags", "archived", "wat"} {
		if fields[key] == "" {
			t.Fatalf("expected validation error for %s, got %v", key, fields)
		}
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/1", strings.NewReader(`{"content":"   "}`))
	_, fields, err = parsePatchNoteRequest(req, noteID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["content"] == "" {
		t.Fatalf("expected blank content validation error, got %v", fields)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/1", strings.NewReader(`{"title":123,"content":true,"tags":"nope","shared":null}`))
	_, fields, err = parsePatchNoteRequest(req, noteID, ownerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, key := range []string{"title", "content", "tags", "shared"} {
		if fields[key] == "" {
			t.Fatalf("expected validation error for %s, got %v", key, fields)
		}
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/1", strings.NewReader(`{"content":"ok"}{"content":"again"}`))
	_, _, err = parsePatchNoteRequest(req, noteID, ownerID)
	if err == nil {
		t.Fatal("expected trailing JSON decode error for patch request")
	}
}

func TestParseTimeQueryUsesUTC(t *testing.T) {
	values := httptest.NewRequest("GET", "/?created_after=2026-04-01T01:02:03-07:00", nil).URL.Query()
	fields := map[string]string{}
	var parsed *time.Time
	parseTimeQuery(values, "created_after", &parsed, fields)
	if len(fields) != 0 {
		t.Fatalf("unexpected fields: %v", fields)
	}
	if parsed == nil || parsed.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp, got %v", parsed)
	}
}

func TestDecodeJSONBodyHonorsSizeLimit(t *testing.T) {
	tooLarge := map[string]string{"content": strings.Repeat("a", (1<<20)+1)}
	payload, err := json.Marshal(tooLarge)
	if err != nil {
		t.Fatalf("failed to build payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(string(payload)))
	var dst map[string]any
	if err := decodeJSONBody(req, &dst); err == nil {
		t.Fatal("expected oversized body decode error")
	}
}
