package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
)

type fakeUserStore struct {
	user      auth.User
	upsertErr error
	getErr    error
}

func (f *fakeUserStore) UpsertUserFromOIDC(ctx context.Context, identity auth.Identity) (auth.User, error) {
	if f.upsertErr != nil {
		return auth.User{}, f.upsertErr
	}
	return f.user, nil
}
func (f *fakeUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	if f.getErr != nil {
		return auth.User{}, f.getErr
	}
	if f.user.ID == id {
		return f.user, nil
	}
	return auth.User{}, auth.ErrUnauthorized
}

type fakeOIDCClient struct {
	redirectURL string
	authErr     error
	identity    auth.Identity
	exchangeErr error
}

func (f *fakeOIDCClient) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	if f.redirectURL != "" {
		return f.redirectURL, nil
	}
	return "https://issuer.example/login?state=" + state, nil
}
func (f *fakeOIDCClient) Exchange(ctx context.Context, code, verifier, expectedNonce string) (auth.Identity, error) {
	if f.exchangeErr != nil {
		return auth.Identity{}, f.exchangeErr
	}
	if f.identity.Issuer != "" || f.identity.Subject != "" {
		return f.identity, nil
	}
	return auth.Identity{Issuer: "https://issuer.example", Subject: "sub-1"}, nil
}

type fakeSessionStore struct {
	session    auth.Session
	pending    auth.PendingAuth
	pendingErr error
	sessionErr error
	deleteErr  error
}

func (f *fakeSessionStore) StorePendingAuth(ctx context.Context, pending auth.PendingAuth, ttl time.Duration) error {
	f.pending = pending
	return nil
}
func (f *fakeSessionStore) ConsumePendingAuth(ctx context.Context, state string) (auth.PendingAuth, error) {
	if f.pendingErr != nil {
		return auth.PendingAuth{}, f.pendingErr
	}
	if f.pending.State != "" {
		return f.pending, nil
	}
	return auth.PendingAuth{State: state, Nonce: "nonce", Verifier: "verifier", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (f *fakeSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, ttl time.Duration) (auth.Session, error) {
	if f.sessionErr != nil {
		return auth.Session{}, f.sessionErr
	}
	f.session = auth.Session{ID: "session-1", UserID: userID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(ttl)}
	return f.session, nil
}
func (f *fakeSessionStore) GetSession(ctx context.Context, id string) (auth.Session, error) {
	if f.sessionErr != nil {
		return auth.Session{}, f.sessionErr
	}
	if id == f.session.ID {
		return f.session, nil
	}
	return auth.Session{}, auth.ErrUnauthorized
}
func (f *fakeSessionStore) DeleteSession(ctx context.Context, id string) error { return f.deleteErr }

type fakeNotesStore struct {
	createdNote     notes.Note
	currentNote     notes.Note
	sharedNote      notes.Note
	createErr       error
	getErr          error
	sharedErr       error
	patchErr        error
	deleteErr       error
	listErr         error
	deleteRows      int64
	patchResult     notes.Note
	listItems       []notes.Note
	listTotal       int64
	savedItems      []notes.SavedQuery
	savedItem       notes.SavedQuery
	savedErr        error
	renamedNotes    []notes.Note
	renameErr       error
	lastRenameOld   string
	lastRenameNew   string
	lastSavedCreate notes.CreateSavedQueryInput
	lastSavedDelete uuid.UUID
	lastListFilters notes.ListFilters
}

func (f *fakeNotesStore) CreateNote(ctx context.Context, input notes.CreateInput) (notes.Note, error) {
	if f.createErr != nil {
		return notes.Note{}, f.createErr
	}
	if f.createdNote.ID != uuid.Nil {
		return f.createdNote, nil
	}
	return notes.Note{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), OwnerUserID: input.OwnerUserID, Content: input.Content, Tags: input.Tags, Title: input.Title}, nil
}
func (f *fakeNotesStore) GetNoteByIDForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (notes.Note, error) {
	if f.getErr != nil {
		return notes.Note{}, f.getErr
	}
	if f.currentNote.ID != uuid.Nil {
		return f.currentNote, nil
	}
	return notes.Note{}, notes.ErrNotFound
}
func (f *fakeNotesStore) GetNoteByShareSlug(ctx context.Context, shareSlug string) (notes.Note, error) {
	if f.sharedErr != nil {
		return notes.Note{}, f.sharedErr
	}
	if f.sharedNote.ShareSlug != nil && *f.sharedNote.ShareSlug == shareSlug {
		return f.sharedNote, nil
	}
	return notes.Note{}, notes.ErrNotFound
}
func (f *fakeNotesStore) UpdateNotePatch(ctx context.Context, input notes.PatchInput) (notes.Note, error) {
	if f.patchErr != nil {
		return notes.Note{}, f.patchErr
	}
	if f.patchResult.ID != uuid.Nil {
		return f.patchResult, nil
	}
	return f.currentNote, nil
}
func (f *fakeNotesStore) DeleteNoteForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	if f.deleteRows != 0 {
		return f.deleteRows, nil
	}
	return 1, nil
}
func (f *fakeNotesStore) ListNotesForOwner(ctx context.Context, filters notes.ListFilters) ([]notes.Note, int64, error) {
	f.lastListFilters = filters
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listItems, f.listTotal, nil
}
func (f *fakeNotesStore) ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]notes.TagSummary, error) {
	return nil, nil
}
func (f *fakeNotesStore) RenameTagForOwner(ctx context.Context, ownerUserID uuid.UUID, oldTag, newTag string) ([]notes.Note, error) {
	f.lastRenameOld = oldTag
	f.lastRenameNew = newTag
	if f.renameErr != nil {
		return nil, f.renameErr
	}
	return f.renamedNotes, nil
}
func (f *fakeNotesStore) FindRelatedNotesForOwner(ctx context.Context, ownerUserID, id uuid.UUID, limit int32) ([]notes.RelatedNote, error) {
	return nil, nil
}
func (f *fakeNotesStore) CreateSavedQuery(ctx context.Context, input notes.CreateSavedQueryInput) (notes.SavedQuery, error) {
	f.lastSavedCreate = input
	if f.savedErr != nil {
		return notes.SavedQuery{}, f.savedErr
	}
	if f.savedItem.ID != uuid.Nil {
		return f.savedItem, nil
	}
	return notes.SavedQuery{ID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), OwnerUserID: input.OwnerUserID, Name: input.Name, Query: input.Query, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
}
func (f *fakeNotesStore) GetSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (notes.SavedQuery, error) {
	if f.savedErr != nil {
		return notes.SavedQuery{}, f.savedErr
	}
	if f.savedItem.ID != uuid.Nil {
		return f.savedItem, nil
	}
	return notes.SavedQuery{}, notes.ErrNotFound
}
func (f *fakeNotesStore) ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]notes.SavedQuery, error) {
	if f.savedErr != nil {
		return nil, f.savedErr
	}
	return f.savedItems, nil
}
func (f *fakeNotesStore) DeleteSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	f.lastSavedDelete = id
	if f.savedErr != nil {
		return 0, f.savedErr
	}
	return 1, nil
}

type fakeCache struct{ values map[string]string }

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
func (f *fakeCache) Delete(ctx context.Context, keys ...string) error { return nil }

func newTestHandler() http.Handler {
	return newTestHandlerWithDeps(Dependencies{})
}

func newTestHandlerWithDeps(overrides Dependencies) http.Handler {
	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	userStore := &fakeUserStore{user: user}
	sessionStore := &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: user.ID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)}}
	authService := auth.NewService(userStore, &fakeOIDCClient{}, sessionStore, time.Hour, time.Minute)
	slug := "shared-1"
	noteStore := &fakeNotesStore{sharedNote: notes.Note{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), OwnerUserID: user.ID, Content: "public", Shared: true, ShareSlug: &slug}}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	logger := slog.New(slog.NewTextHandler(ioDiscard{}, nil))
	deps := Dependencies{
		Logger:              logger,
		AuthService:         authService,
		NotesService:        notesService,
		SessionCookieName:   "go_notes_session",
		ThrottleRequestsPS:  100,
		ThrottleBurst:       100,
		SessionCookieSecure: false,
	}
	if overrides.Logger != nil {
		deps.Logger = overrides.Logger
	}
	if overrides.AuthService != nil {
		deps.AuthService = overrides.AuthService
	}
	if overrides.NotesService != nil {
		deps.NotesService = overrides.NotesService
	}
	if overrides.SessionCookieName != "" {
		deps.SessionCookieName = overrides.SessionCookieName
	}
	if overrides.ThrottleRequestsPS != 0 {
		deps.ThrottleRequestsPS = overrides.ThrottleRequestsPS
	}
	if overrides.ThrottleBurst != 0 {
		deps.ThrottleBurst = overrides.ThrottleBurst
	}
	if overrides.SessionCookieSecure {
		deps.SessionCookieSecure = true
	}
	return NewHandler(deps)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestHealthEndpoint(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestSavedQueryEndpoints(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	savedID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	store := &fakeNotesStore{
		savedItem: notes.SavedQuery{
			ID:          savedID,
			OwnerUserID: userID,
			Name:        "Work search",
			Query:       "search=golang&tag=work",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		savedItems: []notes.SavedQuery{{
			ID:          savedID,
			OwnerUserID: userID,
			Name:        "Work search",
			Query:       "search=golang&tag=work",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}},
	}
	handler := newTestHandlerWithDeps(Dependencies{
		NotesService: notes.NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-queries", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "saved_queries") {
		t.Fatalf("expected saved query list response, got status=%d body=%q", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/saved-queries", strings.NewReader(`{"name":"Work query","query":"search=golang&tag=work"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected saved query create 201, got %d", res.Code)
	}
	if store.lastSavedCreate.Name != "Work query" || store.lastSavedCreate.Query == "" {
		t.Fatalf("expected normalized saved query create input, got %+v", store.lastSavedCreate)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/saved-queries/"+savedID.String(), nil)
	req.SetPathValue("id", savedID.String())
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || store.lastSavedDelete != savedID {
		t.Fatalf("expected saved query delete, got status=%d lastDeleted=%s", res.Code, store.lastSavedDelete)
	}
}

func TestRenameTagEndpoint(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	store := &fakeNotesStore{
		renamedNotes: []notes.Note{{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: userID,
			Content:     "hello",
			Tags:        []string{"roadmap"},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}},
	}
	handler := newTestHandlerWithDeps(Dependencies{
		NotesService: notes.NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags/rename", strings.NewReader(`{"old_tag":"planning","new_tag":"roadmap"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected rename tag 200, got %d body=%q", res.Code, res.Body.String())
	}
	if store.lastRenameOld != "planning" || store.lastRenameNew != "roadmap" {
		t.Fatalf("expected rename inputs to reach notes service, got old=%q new=%q", store.lastRenameOld, store.lastRenameNew)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tags/rename", strings.NewReader(`{"old_tag":"same","new_tag":"same"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected rename tag validation error, got %d body=%q", res.Code, res.Body.String())
	}
}

func TestGetNoteTranslatesNotFoundIntoErrorEnvelope(t *testing.T) {
	handler := newTestHandlerWithDeps(Dependencies{
		NotesService: notes.NewService(&fakeNotesStore{getErr: notes.ErrNotFound}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	req.SetPathValue("id", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), `"code":"not_found"`) {
		t.Fatalf("expected structured not_found envelope, got %q", res.Body.String())
	}
}

func TestListNotesAppliesSavedQueryAndExplicitOverride(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	savedID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	store := &fakeNotesStore{
		savedItem: notes.SavedQuery{
			ID:          savedID,
			OwnerUserID: userID,
			Name:        "Ranked work",
			Query:       "search=golang&search_mode=fts&tag=work&sort=relevance",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		listItems: []notes.Note{{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), OwnerUserID: userID, Content: "hello"}},
		listTotal: 1,
	}
	handler := newTestHandlerWithDeps(Dependencies{
		NotesService: notes.NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes?saved_query_id="+savedID.String()+"&order=asc", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", res.Code)
	}
	if store.lastListFilters.Search == nil || *store.lastListFilters.Search != "golang" || store.lastListFilters.SearchMode != notes.SearchModeFTS || len(store.lastListFilters.Tags) != 1 || store.lastListFilters.Tags[0] != "work" || store.lastListFilters.SortField != "relevance" || store.lastListFilters.SortDirection != "asc" {
		t.Fatalf("expected saved query filters plus override to reach the list service, got %+v", store.lastListFilters)
	}
}

func TestLoginRedirects(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", res.Code)
	}
	if location := res.Header().Get("Location"); !strings.HasPrefix(location, "https://issuer.example/login?state=") {
		t.Fatalf("unexpected location: %s", location)
	}
}

func TestLoginStoresSafeRedirectTarget(t *testing.T) {
	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	sessionStore := &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: user.ID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)}}
	authService := auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, sessionStore, time.Hour, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{AuthService: authService})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login?redirect_to=/", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", res.Code)
	}
	if sessionStore.pending.RedirectTo != "/" {
		t.Fatalf("expected safe redirect target to be stored, got %q", sessionStore.pending.RedirectTo)
	}
}

func TestLoginFailureIsLogged(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	authService := auth.NewService(
		&fakeUserStore{},
		&fakeOIDCClient{authErr: errors.New("oidc: issuer did not match the issuer returned by provider")},
		&fakeSessionStore{},
		time.Hour,
		time.Minute,
	)
	handler := newTestHandlerWithDeps(Dependencies{
		Logger:      logger,
		AuthService: authService,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
	if !strings.Contains(logs.String(), "failed to begin login") {
		t.Fatalf("expected login failure log entry, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), "issuer did not match") {
		t.Fatalf("expected underlying oidc error in logs, got %q", logs.String())
	}
}

func TestHomeGuestPage(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Continue with OIDC") {
		t.Fatalf("expected landing page login action, got %q", res.Body.String())
	}
}

func TestHomeWorkspacePage(t *testing.T) {
	title := "Weekly plan"
	note := notes.Note{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Title:       &title,
		Content:     "Plan the week",
		Tags:        []string{"planning"},
	}
	noteStore := &fakeNotesStore{
		listItems:   []notes.Note{note},
		listTotal:   1,
		currentNote: note,
	}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{NotesService: notesService})
	req := httptest.NewRequest(http.MethodGet, "/?note="+note.ID.String(), nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Notes workspace") || !strings.Contains(body, "Weekly plan") {
		t.Fatalf("expected workspace page with selected note, got %q", body)
	}
}

func TestUICreateNoteHTMX(t *testing.T) {
	title := "Draft agenda"
	created := notes.Note{
		ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Title:       &title,
		Content:     "Talk through launch topics",
		Tags:        []string{"launch"},
	}
	noteStore := &fakeNotesStore{createdNote: created, listItems: []notes.Note{created}, listTotal: 1}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{NotesService: notesService})
	body := strings.NewReader("title=Draft+agenda&content=Talk+through+launch+topics&tags=launch&shared=true")
	req := httptest.NewRequest(http.MethodPost, "/app/notes", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Draft agenda") || !strings.Contains(res.Body.String(), "Talk through launch topics") {
		t.Fatalf("expected created note to render in workspace, got %q", res.Body.String())
	}
}

func TestUIShowNoteFragment(t *testing.T) {
	title := "Call notes"
	note := notes.Note{
		ID:          uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Title:       &title,
		Content:     "# Heading\n\n- first item\n- second item\n\n`code`",
		Tags:        []string{"meeting"},
	}
	noteStore := &fakeNotesStore{currentNote: note}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{NotesService: notesService})
	req := httptest.NewRequest(http.MethodGet, "/app/notes/"+note.ID.String(), nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "note-markdown") || !strings.Contains(body, "<h1") || !strings.Contains(body, "<ul>") || !strings.Contains(body, "<code>code</code>") || strings.Contains(body, "<html") {
		t.Fatalf("expected note fragment, got %q", res.Body.String())
	}
}

func TestUIUpdateNoteHTMX(t *testing.T) {
	oldTitle := "Old title"
	newTitle := "Fresh title"
	current := notes.Note{
		ID:          uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Title:       &oldTitle,
		Content:     "Before",
		Tags:        []string{"draft"},
	}
	updated := notes.Note{
		ID:          current.ID,
		OwnerUserID: current.OwnerUserID,
		Title:       &newTitle,
		Content:     "After",
		Tags:        []string{"published"},
		Shared:      true,
	}
	noteStore := &fakeNotesStore{
		currentNote: current,
		patchResult: updated,
		listItems:   []notes.Note{updated},
		listTotal:   1,
	}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{NotesService: notesService})
	body := strings.NewReader("title=Fresh+title&content=After&tags=published&shared=true")
	req := httptest.NewRequest(http.MethodPost, "/app/notes/"+current.ID.String(), body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Fresh title") || !strings.Contains(res.Body.String(), "After") {
		t.Fatalf("expected updated note in workspace, got %q", res.Body.String())
	}
}

func TestUILogoutRedirects(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/app/logout", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", res.Code)
	}
	if location := res.Header().Get("Location"); location != "/" {
		t.Fatalf("expected redirect to /, got %q", location)
	}
}

func TestUIEditNoteFragment(t *testing.T) {
	title := "Edit me"
	note := notes.Note{
		ID:          uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Title:       &title,
		Content:     "# Original body",
		Tags:        []string{"draft"},
	}
	noteStore := &fakeNotesStore{currentNote: note}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{NotesService: notesService})
	req := httptest.NewRequest(http.MethodGet, "/app/notes/"+note.ID.String()+"/edit", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Save changes") || !strings.Contains(body, "name=\"content\"") || !strings.Contains(body, "Original body") {
		t.Fatalf("expected edit fragment, got %q", res.Body.String())
	}
}

func TestUICreateNoteValidationError(t *testing.T) {
	handler := newTestHandler()
	body := strings.NewReader("title=No+content")
	req := httptest.NewRequest(http.MethodPost, "/app/notes", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Content is required.") {
		t.Fatalf("expected validation message, got %q", res.Body.String())
	}
}

func TestUIUpdateNoteRedirectsWithoutHTMX(t *testing.T) {
	oldTitle := "Old title"
	newTitle := "Fresh title"
	current := notes.Note{
		ID:          uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		OwnerUserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Title:       &oldTitle,
		Content:     "Before",
	}
	updated := notes.Note{
		ID:          current.ID,
		OwnerUserID: current.OwnerUserID,
		Title:       &newTitle,
		Content:     "After",
	}
	noteStore := &fakeNotesStore{currentNote: current, patchResult: updated}
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{NotesService: notesService})
	body := strings.NewReader("title=Fresh+title&content=After")
	req := httptest.NewRequest(http.MethodPost, "/app/notes/"+current.ID.String(), body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", res.Code)
	}
	if location := res.Header().Get("Location"); location != "/?note="+current.ID.String() {
		t.Fatalf("expected redirect to selected note, got %q", location)
	}
}

func TestUIHelpers(t *testing.T) {
	if got := parseTags(" one, two ,, three "); len(got) != 3 || got[0] != "one" || got[2] != "three" {
		t.Fatalf("unexpected parsed tags: %#v", got)
	}
	if got := parseTags(""); len(got) != 0 {
		t.Fatalf("expected empty tags, got %#v", got)
	}

	if got := checkedAttr(true); got != "checked" {
		t.Fatalf("expected checked attr, got %q", got)
	}
	if got := checkedAttr(false); got != "" {
		t.Fatalf("expected empty unchecked attr, got %q", got)
	}

	if got := joinTags(nil); got != "No tags yet" {
		t.Fatalf("unexpected empty tag label: %q", got)
	}
	if got := tagCSV([]string{"a", "b"}); got != "a, b" {
		t.Fatalf("unexpected tag csv: %q", got)
	}

	title := " Hello "
	userName := " Casey "
	email := " person@example.com "
	note := notes.Note{ID: uuid.MustParse("abababab-abab-abab-abab-abababababab"), Title: &title}
	if got := noteToForm(note).Title; got != "Hello" {
		t.Fatalf("unexpected note form title: %q", got)
	}
	if got := noteTitle(note); got != "Hello" {
		t.Fatalf("unexpected note title: %q", got)
	}
	if got := displayName(auth.User{DisplayName: &userName}); got != "Casey" {
		t.Fatalf("unexpected display name: %q", got)
	}
	if got := displayName(auth.User{Email: &email}); got != "person@example.com" {
		t.Fatalf("unexpected email display name: %q", got)
	}
	if got := displayName(auth.User{}); got != "there" {
		t.Fatalf("unexpected anonymous display name: %q", got)
	}

	values := dict("a", 1, "b", "two")
	if values["a"] != 1 || values["b"] != "two" {
		t.Fatalf("unexpected dict contents: %#v", values)
	}

	api := &API{}
	if status := api.noteHTMLStatus(notes.ErrNotFound); status != http.StatusNotFound {
		t.Fatalf("expected 404 for note not found, got %d", status)
	}
	if status := api.noteHTMLStatus(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("expected 500 for generic error, got %d", status)
	}
}

func TestMeRequiresCookie(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestMeWithValidCookie(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unexpected json: %v", err)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatalf("expected data envelope, got %s", res.Body.String())
	}
}

func TestSharedNoteEndpoint(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/shared/shared-1", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestListValidationHappensBeforeStore(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes?page_size=500", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCallbackSuccessSetsCookie(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state&code=code", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if cookie := res.Result().Cookies(); len(cookie) == 0 || cookie[0].Name != "go_notes_session" {
		t.Fatalf("expected session cookie, got %+v", cookie)
	}
}

func TestCallbackSuccessRedirectsToWorkspaceWhenRequested(t *testing.T) {
	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	authService := auth.NewService(
		&fakeUserStore{user: user},
		&fakeOIDCClient{},
		&fakeSessionStore{
			pending: auth.PendingAuth{State: "state", Nonce: "nonce", Verifier: "verifier", RedirectTo: "/"},
			session: auth.Session{ID: "session-1", UserID: user.ID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)},
		},
		time.Hour,
		time.Minute,
	)
	handler := newTestHandlerWithDeps(Dependencies{AuthService: authService})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=state&code=code", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", res.Code)
	}
	if location := res.Header().Get("Location"); location != "/" {
		t.Fatalf("expected redirect to workspace, got %q", location)
	}
	if cookie := res.Result().Cookies(); len(cookie) == 0 || cookie[0].Name != "go_notes_session" {
		t.Fatalf("expected session cookie, got %+v", cookie)
	}
}

func TestSafeRedirectTarget(t *testing.T) {
	if got := safeRedirectTarget("/"); got != "/" {
		t.Fatalf("expected safe local redirect, got %q", got)
	}
	if got := safeRedirectTarget("https://example.com"); got != "" {
		t.Fatalf("expected external redirect to be rejected, got %q", got)
	}
	if got := safeRedirectTarget("//evil.example"); got != "" {
		t.Fatalf("expected scheme-relative redirect to be rejected, got %q", got)
	}
}

func TestCallbackHandlesProviderAndValidationErrors(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?error=access_denied", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for provider error, got %d", res.Code)
	}

	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	authService := auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "s", UserID: user.ID}, pendingErr: auth.ErrInvalidState}, time.Hour, time.Minute)
	handler = newTestHandlerWithDeps(Dependencies{AuthService: authService})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?state=bad&code=code", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid callback, got %d", res.Code)
	}
}

func TestLogoutExpiresCookie(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	cookies := res.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge != -1 {
		t.Fatalf("expected expired cookie, got %+v", cookies)
	}
}

func TestCreateGetPatchDeleteAndNotFoundHandlers(t *testing.T) {
	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	title := "hello"
	noteStore := &fakeNotesStore{
		currentNote: notes.Note{ID: noteID, OwnerUserID: user.ID, Title: &title, Content: "body"},
		patchResult: notes.Note{ID: noteID, OwnerUserID: user.ID, Title: &title, Content: "patched"},
		listItems:   []notes.Note{{ID: noteID, OwnerUserID: user.ID, Title: &title, Content: "body"}},
		listTotal:   1,
	}
	authService := auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: user.ID}}, time.Hour, time.Minute)
	notesService := notes.NewService(noteStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{AuthService: authService, NotesService: notesService})

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"body"}`))
	createReq.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/notes/"+noteID.String(), nil)
	getReq.SetPathValue("id", noteID.String())
	getReq.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	getRes := httptest.NewRecorder()
	handler.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRes.Code)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/notes/"+noteID.String(), strings.NewReader(`{"content":"patched"}`))
	patchReq.SetPathValue("id", noteID.String())
	patchReq.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	patchRes := httptest.NewRecorder()
	handler.ServeHTTP(patchRes, patchReq)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", patchRes.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	listReq.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	listRes := httptest.NewRecorder()
	handler.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRes.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/"+noteID.String(), nil)
	deleteReq.SetPathValue("id", noteID.String())
	deleteReq.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	deleteRes := httptest.NewRecorder()
	handler.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteRes.Code)
	}

	missingStore := &fakeNotesStore{getErr: notes.ErrNotFound}
	missingService := notes.NewService(missingStore, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute)
	missingHandler := newTestHandlerWithDeps(Dependencies{AuthService: authService, NotesService: missingService})
	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/notes/"+noteID.String(), nil)
	missingReq.SetPathValue("id", noteID.String())
	missingReq.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	missingRes := httptest.NewRecorder()
	missingHandler.ServeHTTP(missingRes, missingReq)
	if missingRes.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", missingRes.Code)
	}
}

func TestHandlerValidationAndErrorPaths(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"   "}`))
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid create, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"body","extra":true}`))
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid create json, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/notes/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid note id, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/not-a-uuid", strings.NewReader(`{"content":"body"}`))
	req.SetPathValue("id", "not-a-uuid")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid patch note id, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", strings.NewReader(`{"content":null}`))
	req.SetPathValue("id", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid patch payload, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/notes/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid delete note id, got %d", res.Code)
	}

	api := &API{}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/notes/shared/", nil)
	req.SetPathValue("slug", "")
	res = httptest.NewRecorder()
	api.handleGetSharedNote(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank slug, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/missing", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown route, got %d", res.Code)
	}

	sharedMissingHandler := newTestHandlerWithDeps(Dependencies{
		NotesService: notes.NewService(&fakeNotesStore{sharedErr: notes.ErrNotFound}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/notes/shared/missing", nil)
	req.SetPathValue("slug", "missing")
	res = httptest.NewRecorder()
	sharedMissingHandler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing shared note, got %d", res.Code)
	}
}

func TestSharedNoteResponseDoesNotExposeInternalIDs(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	slug := "public-note"
	title := "Shared title"
	sharedHandler := newTestHandlerWithDeps(Dependencies{
		NotesService: notes.NewService(&fakeNotesStore{sharedNote: notes.Note{
			ID:          noteID,
			OwnerUserID: userID,
			Title:       &title,
			Content:     "public body",
			Tags:        []string{"share"},
			Shared:      true,
			ShareSlug:   &slug,
			CreatedAt:   time.Date(2026, time.April, 5, 6, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, time.April, 5, 7, 0, 0, 0, time.UTC),
		}}, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/shared/"+slug, nil)
	req.SetPathValue("slug", slug)
	res := httptest.NewRecorder()
	sharedHandler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 for shared note, got %d", res.Code)
	}

	var payload map[string]map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode shared note response: %v", err)
	}
	data := payload["data"]
	if _, ok := data["id"]; ok {
		t.Fatalf("expected shared note response to omit internal id, got %v", data)
	}
	if _, ok := data["owner_user_id"]; ok {
		t.Fatalf("expected shared note response to omit owner id, got %v", data)
	}
	if data["share_slug"] != slug {
		t.Fatalf("expected shared note response to keep share slug, got %v", data)
	}
}

func TestHandlerServiceFailures(t *testing.T) {
	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	authService := auth.NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: user.ID}}, time.Hour, time.Minute)

	makeHandler := func(store *fakeNotesStore) http.Handler {
		return newTestHandlerWithDeps(Dependencies{
			AuthService:  authService,
			NotesService: notes.NewService(store, &fakeCache{values: map[string]string{}}, time.Minute, time.Minute),
		})
	}

	createHandler := makeHandler(&fakeNotesStore{createErr: errors.New("boom")})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"body"}`))
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	createHandler.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for create failure, got %d", res.Code)
	}

	listHandler := makeHandler(&fakeNotesStore{listErr: errors.New("boom")})
	req = httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	listHandler.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for list failure, got %d", res.Code)
	}

	noteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	patchHandler := makeHandler(&fakeNotesStore{currentNote: notes.Note{ID: noteID, OwnerUserID: user.ID}, patchErr: errors.New("boom")})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/notes/"+noteID.String(), strings.NewReader(`{"content":"body"}`))
	req.SetPathValue("id", noteID.String())
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	patchHandler.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for patch failure, got %d", res.Code)
	}

	deleteHandler := makeHandler(&fakeNotesStore{currentNote: notes.Note{ID: noteID, OwnerUserID: user.ID}, deleteErr: errors.New("boom")})
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/notes/"+noteID.String(), nil)
	req.SetPathValue("id", noteID.String())
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res = httptest.NewRecorder()
	deleteHandler.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for delete failure, got %d", res.Code)
	}
}

func TestRequireUserClearsInvalidSessionCookie(t *testing.T) {
	user := auth.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	authService := auth.NewService(&fakeUserStore{user: user, getErr: auth.ErrUnauthorized}, &fakeOIDCClient{}, &fakeSessionStore{session: auth.Session{ID: "session-1", UserID: user.ID}}, time.Hour, time.Minute)
	handler := newTestHandlerWithDeps(Dependencies{AuthService: authService})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: "session-1"})
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
	cookies := res.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge != -1 {
		t.Fatalf("expected invalid session to expire cookie, got %+v", cookies)
	}
}

func TestMiddlewareHelpers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(ioDiscard{}, nil))

	panicHandler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}), recoverMiddleware(logger))
	res := httptest.NewRecorder()
	panicHandler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected recovered panic to return 500, got %d", res.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.10:1234"
	if got := throttleKey(req); got != "198.51.100.10" {
		t.Fatalf("unexpected throttle key from remote addr: %q", got)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.20, 10.0.0.1")
	if got := throttleKey(req); got != "203.0.113.20" {
		t.Fatalf("unexpected throttle key from forwarded header: %q", got)
	}

	api := &API{sessionCookieName: "go_notes_session", sessionCookieSecure: true}
	if cookie := api.sessionCookie("abc", time.Now().UTC().Add(time.Hour)); !cookie.HttpOnly || !cookie.Secure {
		t.Fatalf("expected secure session cookie, got %+v", cookie)
	}
	if cookie := api.expiredSessionCookie(); cookie.MaxAge != -1 {
		t.Fatalf("expected expired cookie, got %+v", cookie)
	}

	res = httptest.NewRecorder()
	api.writeNotesError(res, notes.ErrNotFound, "fallback")
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected notes not found to map to 404, got %d", res.Code)
	}
}

func TestThrottleOnPublicRoutes(t *testing.T) {
	handler := newTestHandlerWithDeps(Dependencies{ThrottleRequestsPS: 1000, ThrottleBurst: 1})

	first := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	first.RemoteAddr = "203.0.113.10:1234"
	firstRes := httptest.NewRecorder()
	handler.ServeHTTP(firstRes, first)
	if firstRes.Code != http.StatusFound {
		t.Fatalf("expected first request to pass, got %d", firstRes.Code)
	}

	second := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	second.RemoteAddr = "203.0.113.10:1234"
	secondRes := httptest.NewRecorder()
	handler.ServeHTTP(secondRes, second)
	if secondRes.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be throttled, got %d", secondRes.Code)
	}
	if !strings.Contains(secondRes.Body.String(), "rate_limited") {
		t.Fatalf("expected rate-limited error envelope, got %s", secondRes.Body.String())
	}
}

func TestThrottleDoesNotApplyToUnlistedRoutes(t *testing.T) {
	handler := newTestHandlerWithDeps(Dependencies{ThrottleRequestsPS: 1000, ThrottleBurst: 1})

	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected health route to stay unthrottled, got %d", res.Code)
		}
	}
}
