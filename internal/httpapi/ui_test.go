package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
)

func TestDefaultUIFilters(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	filters := defaultUIFilters(ownerID)

	if filters.OwnerUserID != ownerID {
		t.Fatalf("expected owner %s, got %s", ownerID, filters.OwnerUserID)
	}
	if filters.Page != 1 || filters.PageSize != 20 {
		t.Fatalf("unexpected paging defaults: %+v", filters)
	}
	if filters.Status != notes.StatusActive || filters.TagMatchMode != notes.TagMatchAny {
		t.Fatalf("unexpected status defaults: %+v", filters)
	}
	if filters.SortField != "updated_at" || filters.SortDirection != "desc" {
		t.Fatalf("unexpected sort defaults: %+v", filters)
	}
}

func TestParseUIListFiltersUsesUIPreservedFields(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	api := &API{notesService: notes.NewService(&fakeNotesStore{}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)}
	req := httptest.NewRequest(http.MethodPost, "/app/notes", strings.NewReader(url.Values{
		"ui_search":        []string{"ranked notes"},
		"ui_search_mode":   []string{"fts"},
		"ui_has_title":     []string{"true"},
		"ui_tags":          []string{"work, planning"},
		"ui_tag_count_min": []string{"1"},
		"ui_tag_count_max": []string{"5"},
		"ui_tag_mode":      []string{"all"},
		"ui_sort":          []string{"primary_tag"},
		"ui_order":         []string{"asc"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	filters, form, fields := api.parseUIListFilters(req, ownerID)

	if len(fields) != 0 {
		t.Fatalf("expected no filter validation errors, got %#v", fields)
	}
	if filters.Search == nil || *filters.Search != "ranked notes" || filters.SearchMode != notes.SearchModeFTS {
		t.Fatalf("unexpected search filters: %+v", filters)
	}
	if filters.HasTitle == nil || !*filters.HasTitle || filters.TagCountMin == nil || *filters.TagCountMin != 1 || filters.TagCountMax == nil || *filters.TagCountMax != 5 {
		t.Fatalf("unexpected advanced UI filters: %+v", filters)
	}
	if len(filters.Tags) != 2 || filters.Tags[0] != "work" || filters.Tags[1] != "planning" {
		t.Fatalf("unexpected tag filters: %+v", filters.Tags)
	}
	if filters.TagMatchMode != notes.TagMatchAll || filters.SortField != "primary_tag" || filters.SortDirection != "asc" {
		t.Fatalf("unexpected list filters: %+v", filters)
	}
	if form.Search != "ranked notes" || form.SearchMode != notes.SearchModeFTS || form.HasTitle != "true" || form.TagCountMin != "1" || form.TagCountMax != "5" || form.Tags != "work, planning" || form.Mode != notes.TagMatchAll || form.Sort != "primary_tag" || form.Order != "asc" {
		t.Fatalf("unexpected form values: %+v", form)
	}
}

func TestParseUIListFiltersFallsBackToDefaults(t *testing.T) {
	ownerID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	api := &API{notesService: notes.NewService(&fakeNotesStore{}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)}
	req := httptest.NewRequest(http.MethodGet, "/?tag_mode=bad", nil)

	_, form, fields := api.parseUIListFilters(req, ownerID)

	if fields["tag_mode"] == "" {
		t.Fatalf("expected tag mode validation error, got %#v", fields)
	}
	if form.Mode != notes.TagMatchAny || form.Sort != "updated_at" || form.Order != "desc" {
		t.Fatalf("expected fallback form defaults, got %+v", form)
	}
}

func TestCloneValuesAndCopyIfMissing(t *testing.T) {
	values := url.Values{
		"ui_tags": []string{"planning"},
	}
	cloned := cloneValues(values)
	cloned.Add("ui_tags", "retro")
	if len(values["ui_tags"]) != 1 {
		t.Fatalf("expected original values to stay unchanged, got %#v", values)
	}

	copyIfMissing(cloned, "tags", "ui_tags")
	if got := cloned["tags"]; len(got) != 2 || got[0] != "planning" || got[1] != "retro" {
		t.Fatalf("expected fallback copy to preserve values, got %#v", got)
	}

	cloned.Set("sort", "updated_at")
	copyIfMissing(cloned, "sort", "ui_sort")
	if got := cloned.Get("sort"); got != "updated_at" {
		t.Fatalf("expected existing target value to be kept, got %q", got)
	}
}

func TestFilterQueryHelpers(t *testing.T) {
	form := noteListFormValues{
		SavedQueryID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Search:       "ranked notes",
		SearchMode:   notes.SearchModeFTS,
		HasTitle:     "true",
		Tags:         "work, planning",
		TagCountMin:  "1",
		TagCountMax:  "5",
		Mode:         notes.TagMatchAll,
		Sort:         "relevance",
		Order:        "desc",
	}
	query := buildFilterQuery(form)
	if !strings.Contains(query, "search=ranked+notes") || !strings.Contains(query, "search_mode=fts") || !strings.Contains(query, "has_title=true") {
		t.Fatalf("unexpected search query: %q", query)
	}
	if !strings.Contains(query, "saved_query_id=bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb") {
		t.Fatalf("expected saved query id in filter query, got %q", query)
	}
	if !strings.Contains(query, "tags=work%2C+planning") || !strings.Contains(query, "tag_mode=all") {
		t.Fatalf("unexpected filter query: %q", query)
	}
	if !strings.Contains(query, "tag_count_min=1") || !strings.Contains(query, "tag_count_max=5") || !strings.Contains(query, "sort=relevance") {
		t.Fatalf("unexpected sort query: %q", query)
	}

	combined := combineQuery(url.Values{"note": []string{"abc"}}, query)
	if !strings.Contains(combined, "note=abc") || !strings.Contains(combined, "tag_mode=all") {
		t.Fatalf("unexpected combined query: %q", combined)
	}

	if got := combineQuery(url.Values{"note": []string{"abc"}}, "%ZZ"); got != "note=abc" {
		t.Fatalf("expected invalid encoded query fallback, got %q", got)
	}
}

func TestUILinkHelpers(t *testing.T) {
	note := notes.Note{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}

	if got := selectedNoteLink(note, "tags=work"); got != "/?note=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa&tags=work" {
		t.Fatalf("unexpected selected note link: %q", got)
	}
	if got := editNoteLink(note, "tags=work"); got != "/?edit=1&note=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa&tags=work" && got != "/?note=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa&edit=1&tags=work" {
		t.Fatalf("unexpected edit note link: %q", got)
	}
	if got := tagFilterLink("planning", "primary_tag", "asc"); got != "/?order=asc&sort=primary_tag&tag_mode=any&tags=planning" && got != "/?sort=primary_tag&order=asc&tag_mode=any&tags=planning" && got != "/?tag_mode=any&sort=primary_tag&order=asc&tags=planning" && got != "/?tags=planning&tag_mode=any&sort=primary_tag&order=asc" {
		if !strings.Contains(got, "tags=planning") || !strings.Contains(got, "tag_mode=any") {
			t.Fatalf("unexpected tag filter link: %q", got)
		}
	}
}

func TestRenderMarkdownAndFormattingHelpers(t *testing.T) {
	rendered := string(renderMarkdown("# Heading\n\n`code`"))
	if !strings.Contains(rendered, "<h1") || !strings.Contains(rendered, "<code>code</code>") {
		t.Fatalf("expected markdown HTML, got %q", rendered)
	}

	timestamp := time.Date(2026, time.April, 4, 10, 30, 0, 0, time.UTC)
	if got := formatTimestamp(timestamp); !strings.Contains(got, "Apr 4, 2026") {
		t.Fatalf("unexpected formatted timestamp: %q", got)
	}
	if got := formatTimestamp("not-a-time"); got != "" {
		t.Fatalf("expected empty fallback formatting, got %q", got)
	}
}

func TestMaybeUserAndSelectionHelpers(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := auth.User{ID: userID}
	note := notes.Note{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		OwnerUserID: userID,
		Content:     "hello",
	}
	api := &API{
		authService:       auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: userID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)}}, time.Hour, time.Minute),
		notesService:      notes.NewService(&fakeNotesStore{currentNote: note, listItems: []notes.Note{note}, listTotal: 1}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
		sessionCookieName: "go_notes_session",
	}

	req := httptest.NewRequest(http.MethodGet, "/?note="+note.ID.String()+"&edit=1", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})

	gotUser, ok, invalidate := api.maybeUser(req)
	if !ok || invalidate || gotUser.ID != userID {
		t.Fatalf("expected authenticated user, got user=%+v ok=%v invalidate=%v", gotUser, ok, invalidate)
	}

	selected, editing, status := api.selectedNoteFromQuery(req, userID)
	if status != http.StatusOK || !editing || selected == nil || selected.ID != note.ID {
		t.Fatalf("unexpected selected note state: selected=%+v editing=%v status=%d", selected, editing, status)
	}
}

func TestWorkspaceHelpersAndRedirect(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := auth.User{ID: userID}
	title := "Tagged note"
	note := notes.Note{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		OwnerUserID: userID,
		Title:       &title,
		Content:     "hello",
		Tags:        []string{"work", "planning"},
	}
	api := &API{
		notesService: notes.NewService(&fakeNotesStore{currentNote: note, listItems: []notes.Note{note}, listTotal: 1}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/?tags=work,planning&tag_mode=all&sort=primary_tag&order=asc&note="+note.ID.String(), nil)
	vm, status := api.workspaceForRequest(req, user, saveQueryFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
	if status != http.StatusOK {
		t.Fatalf("expected workspace status 200, got %d", status)
	}
	if vm.SelectedNote == nil || vm.SelectedNote.ID != note.ID {
		t.Fatalf("expected selected note in workspace, got %+v", vm.SelectedNote)
	}
	if vm.Filters.Mode != notes.TagMatchAll || vm.Filters.Sort != "primary_tag" || vm.Filters.Order != "asc" {
		t.Fatalf("unexpected workspace filter form: %+v", vm.Filters)
	}

	res := httptest.NewRecorder()
	api.redirectOrRenderWorkspace(res, req, user, &note, false)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status, got %d", res.Code)
	}
	location := res.Header().Get("Location")
	if !strings.Contains(location, "note="+note.ID.String()) || !strings.Contains(location, "tags=work%2C+planning") {
		t.Fatalf("expected redirect to preserve note and filters, got %q", location)
	}
}

func TestWorkspaceTemplatePrioritizesCreateSectionAndRemovesDecorativeChips(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := auth.User{ID: userID}
	title := "Tagged note"
	note := notes.Note{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		OwnerUserID: userID,
		Title:       &title,
		Content:     "hello",
		Tags:        []string{"work"},
	}
	api := &API{
		notesService: notes.NewService(&fakeNotesStore{
			currentNote: note,
			listItems:   []notes.Note{note},
			listTotal:   1,
			savedItems: []notes.SavedQuery{{
				ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				OwnerUserID: userID,
				Name:        "Recent work",
				Query:       "tags=work",
			}},
		}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	vm, status := api.workspaceForRequest(req, user, saveQueryFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
	if status != http.StatusOK {
		t.Fatalf("expected workspace status 200, got %d", status)
	}

	res := httptest.NewRecorder()
	api.renderHTML(res, http.StatusOK, "workspace", vm)
	body := res.Body.String()

	createIndex := strings.Index(body, "New note")
	savedIndex := strings.Index(body, "Saved queries")
	filterIndex := strings.Index(body, "Filters")
	if createIndex == -1 || savedIndex == -1 || filterIndex == -1 {
		t.Fatalf("expected create, saved query, and filter sections in body=%q", body)
	}
	if !(createIndex < savedIndex && savedIndex < filterIndex) {
		t.Fatalf("expected sidebar order New note -> Saved queries -> Filters, got body=%q", body)
	}
	if strings.Contains(body, ">HTMX<") || strings.Contains(body, ">Reusable<") || strings.Contains(body, ">SQL first<") {
		t.Fatalf("expected decorative chips to be removed, got body=%q", body)
	}
}

func TestUIHomeBranchesAndCookieInvalidation(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	api := &API{
		authService:       auth.NewService(&fakeUserStore{user: auth.User{ID: userID}, getErr: auth.ErrUnauthorized}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: userID}}, time.Hour, time.Minute),
		sessionCookieName: "go_notes_session",
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/nope", nil)
	notFoundRes := httptest.NewRecorder()
	api.handleHome(notFoundRes, notFoundReq)
	if notFoundRes.Code != http.StatusNotFound {
		t.Fatalf("expected non-root home request to render 404, got %d", notFoundRes.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	api.handleHome(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected landing page on invalid session, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Continue with OIDC") {
		t.Fatalf("expected landing page body, got %q", res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge != -1 {
		t.Fatalf("expected invalid session cookie to be expired, got %+v", cookies)
	}
}

func TestUINoteHandlersBranchCoverage(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := auth.User{ID: userID}
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	note := notes.Note{ID: noteID, OwnerUserID: userID, Content: "body"}
	api := &API{
		authService:       auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: userID, ExpiresAt: time.Now().UTC().Add(time.Hour)}}, time.Hour, time.Minute),
		notesService:      notes.NewService(&fakeNotesStore{currentNote: note, getErr: notes.ErrNotFound}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
		sessionCookieName: "go_notes_session",
	}

	req := httptest.NewRequest(http.MethodGet, "/app/notes/"+noteID.String(), nil)
	res := httptest.NewRecorder()
	api.handleUIShowNote(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected unauthenticated show note redirect, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/notes/bad", nil)
	req.SetPathValue("id", "bad")
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUIShowNote(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid note id error, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/notes/"+noteID.String(), nil)
	req.SetPathValue("id", noteID.String())
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUIShowNote(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected missing note status, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/notes/"+noteID.String()+"/edit", nil)
	res = httptest.NewRecorder()
	api.handleUIEditNote(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected unauthenticated edit redirect, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/notes/bad/edit", nil)
	req.SetPathValue("id", "bad")
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUIEditNote(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid edit note id error, got %d", res.Code)
	}
}

func TestUICreateAndUpdateBranches(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := auth.User{ID: userID}
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	noteStore := &fakeNotesStore{
		createErr:   errors.New("boom"),
		currentNote: notes.Note{ID: noteID, OwnerUserID: userID, Content: "before"},
		patchErr:    notes.ErrNotFound,
	}
	api := &API{
		authService:       auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: userID, ExpiresAt: time.Now().UTC().Add(time.Hour)}}, time.Hour, time.Minute),
		notesService:      notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
		sessionCookieName: "go_notes_session",
	}

	req := httptest.NewRequest(http.MethodPost, "/app/notes", strings.NewReader("content=body"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	api.handleUICreateNote(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected unauthenticated create redirect, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/notes", strings.NewReader("content=body"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUICreateNote(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected create failure status, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/notes/"+noteID.String(), strings.NewReader("content=body"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res = httptest.NewRecorder()
	api.handleUIUpdateNote(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected unauthenticated update redirect, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/notes/bad", nil)
	req.SetPathValue("id", "bad")
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUIUpdateNote(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid update id error, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/notes/"+noteID.String(), strings.NewReader("title=No+content"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", noteID.String())
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUIUpdateNote(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid update form status, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/notes/"+noteID.String(), strings.NewReader("content=after"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", noteID.String())
	req = req.WithContext(withUser(context.Background(), user))
	res = httptest.NewRecorder()
	api.handleUIUpdateNote(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected patch error status to surface, got %d", res.Code)
	}
}

func TestWorkspaceAndRenderHelpers(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := auth.User{ID: userID}
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	title := "Title"
	store := &fakeNotesStore{
		listErr:     errors.New("boom"),
		currentNote: notes.Note{ID: noteID, OwnerUserID: userID, Title: &title, Content: "body"},
	}
	api := &API{
		notesService: notes.NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/?tag_mode=bad&note=bad", nil)
	vm, status := api.workspaceForRequest(req, user, saveQueryFormValues{}, nil, noteFormValues{}, nil, noteFormValues{}, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected invalid workspace status, got %d", status)
	}
	if vm.Filters.Sort != "updated_at" {
		t.Fatalf("expected filter form to be preserved with defaults, got %+v", vm.Filters)
	}

	req = httptest.NewRequest(http.MethodGet, "/?note=bad", nil)
	selected, editing, selectedStatus := api.selectedNoteFromQuery(req, userID)
	if selected != nil || editing || selectedStatus != http.StatusBadRequest {
		t.Fatalf("unexpected invalid selected note result: selected=%+v editing=%v status=%d", selected, editing, selectedStatus)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	res := httptest.NewRecorder()
	api.redirectOrRenderWorkspace(res, req, user, nil, false)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Pick a note to read it here.") {
		t.Fatalf("expected HTMX workspace render, got status=%d body=%q", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	api.renderWorkspaceOrPage(res, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusCreated, workspaceViewModel{User: user})
	if res.Code != http.StatusCreated || !strings.Contains(res.Body.String(), "<!doctype html>") {
		t.Fatalf("expected full page render, got status=%d body=%q", res.Code, res.Body.String())
	}
}
