package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/httpapi"
	"github.com/nathanbland/go-notes/internal/notes"
)

type coverageUserStore struct {
	user auth.User
}

func (s *coverageUserStore) UpsertUserFromOIDC(ctx context.Context, identity auth.Identity) (auth.User, error) {
	return s.user, nil
}

func (s *coverageUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	if id != s.user.ID {
		return auth.User{}, auth.ErrUnauthorized
	}
	return s.user, nil
}

type coverageOIDCClient struct{}

func (coverageOIDCClient) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	return "https://issuer.example/login?state=" + state, nil
}

func (coverageOIDCClient) Exchange(ctx context.Context, code, verifier, expectedNonce string) (auth.Identity, error) {
	return auth.Identity{Issuer: "https://issuer.example", Subject: "sub-1"}, nil
}

type coverageSessionStore struct {
	session auth.Session
}

func (s *coverageSessionStore) StorePendingAuth(ctx context.Context, pending auth.PendingAuth, ttl time.Duration) error {
	return nil
}

func (s *coverageSessionStore) ConsumePendingAuth(ctx context.Context, state string) (auth.PendingAuth, error) {
	return auth.PendingAuth{State: state, Nonce: "nonce", Verifier: "verifier", ExpiresAt: time.Now().UTC().Add(time.Minute)}, nil
}

func (s *coverageSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, ttl time.Duration) (auth.Session, error) {
	return s.session, nil
}

func (s *coverageSessionStore) GetSession(ctx context.Context, id string) (auth.Session, error) {
	if id != s.session.ID {
		return auth.Session{}, auth.ErrUnauthorized
	}
	return s.session, nil
}

func (s *coverageSessionStore) DeleteSession(ctx context.Context, id string) error {
	return nil
}

type coverageNotesStore struct {
	note       notes.Note
	sharedNote notes.Note
	savedQuery notes.SavedQuery
}

func (s *coverageNotesStore) CreateNote(ctx context.Context, input notes.CreateInput) (notes.Note, error) {
	s.note = notes.Note{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		OwnerUserID: input.OwnerUserID,
		Title:       input.Title,
		Content:     input.Content,
		Tags:        input.Tags,
		Shared:      input.Shared,
		ShareSlug:   input.ShareSlug,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	return s.note, nil
}

func (s *coverageNotesStore) GetNoteByIDForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (notes.Note, error) {
	if s.note.ID == id && s.note.OwnerUserID == ownerUserID {
		return s.note, nil
	}
	return notes.Note{}, notes.ErrNotFound
}

func (s *coverageNotesStore) GetNoteByShareSlug(ctx context.Context, shareSlug string) (notes.Note, error) {
	if s.sharedNote.ShareSlug != nil && *s.sharedNote.ShareSlug == shareSlug {
		return s.sharedNote, nil
	}
	return notes.Note{}, notes.ErrNotFound
}

func (s *coverageNotesStore) UpdateNotePatch(ctx context.Context, input notes.PatchInput) (notes.Note, error) {
	if s.note.ID != input.ID || s.note.OwnerUserID != input.OwnerUserID {
		return notes.Note{}, notes.ErrNotFound
	}
	if input.ContentSet && input.Content != nil {
		s.note.Content = *input.Content
	}
	return s.note, nil
}

func (s *coverageNotesStore) DeleteNoteForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	if s.note.ID == id && s.note.OwnerUserID == ownerUserID {
		return 1, nil
	}
	return 0, nil
}

func (s *coverageNotesStore) ListNotesForOwner(ctx context.Context, filters notes.ListFilters) ([]notes.Note, int64, error) {
	if s.note.OwnerUserID != filters.OwnerUserID {
		return []notes.Note{}, 0, nil
	}
	return []notes.Note{s.note}, 1, nil
}

func (s *coverageNotesStore) ListTagsForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]notes.TagSummary, error) {
	if s.note.OwnerUserID != ownerUserID {
		return []notes.TagSummary{}, nil
	}
	return []notes.TagSummary{{Tag: "go", Count: 1}}, nil
}

func (s *coverageNotesStore) FindRelatedNotesForOwner(ctx context.Context, ownerUserID, id uuid.UUID, limit int32) ([]notes.RelatedNote, error) {
	if s.note.OwnerUserID != ownerUserID || s.note.ID != id {
		return []notes.RelatedNote{}, nil
	}
	return []notes.RelatedNote{{
		Note:           s.note,
		SharedTags:     []string{"go"},
		SharedTagCount: 1,
	}}, nil
}

func (s *coverageNotesStore) CreateSavedQuery(ctx context.Context, input notes.CreateSavedQueryInput) (notes.SavedQuery, error) {
	s.savedQuery = notes.SavedQuery{
		ID:          uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		Query:       input.Query,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	return s.savedQuery, nil
}

func (s *coverageNotesStore) GetSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (notes.SavedQuery, error) {
	if s.savedQuery.ID == id && s.savedQuery.OwnerUserID == ownerUserID {
		return s.savedQuery, nil
	}
	return notes.SavedQuery{}, notes.ErrNotFound
}

func (s *coverageNotesStore) ListSavedQueriesForOwner(ctx context.Context, ownerUserID uuid.UUID) ([]notes.SavedQuery, error) {
	if s.savedQuery.ID == uuid.Nil || s.savedQuery.OwnerUserID != ownerUserID {
		return []notes.SavedQuery{}, nil
	}
	return []notes.SavedQuery{s.savedQuery}, nil
}

func (s *coverageNotesStore) DeleteSavedQueryForOwner(ctx context.Context, id, ownerUserID uuid.UUID) (int64, error) {
	if s.savedQuery.ID == id && s.savedQuery.OwnerUserID == ownerUserID {
		s.savedQuery = notes.SavedQuery{}
		return 1, nil
	}
	return 0, nil
}

type coverageCache struct{}

func (coverageCache) Get(ctx context.Context, key string) (string, bool) { return "", false }
func (coverageCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return nil
}
func (coverageCache) Delete(ctx context.Context, keys ...string) error { return nil }

func TestHandlerCoverageFromAppPackage(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	session := auth.Session{ID: "session-1", UserID: userID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)}
	user := auth.User{ID: userID}
	shareSlug := "public-note"
	store := &coverageNotesStore{
		note: notes.Note{
			ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			OwnerUserID: userID,
			Content:     "body",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		sharedNote: notes.Note{
			ID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			OwnerUserID: userID,
			Content:     "shared body",
			Shared:      true,
			ShareSlug:   &shareSlug,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}

	authService := auth.NewService(&coverageUserStore{user: user}, coverageOIDCClient{}, &coverageSessionStore{session: session}, time.Hour, time.Minute)
	notesService := notes.NewService(store, coverageCache{}, time.Minute, time.Minute)
	handler := httpapi.NewHandler(httpapi.Dependencies{
		Logger:              slog.New(slog.NewTextHandler(ioDiscard{}, nil)),
		AuthService:         authService,
		NotesService:        notesService,
		SessionCookieName:   "go_notes_session",
		SessionCookieSecure: false,
		ThrottleRequestsPS:  100,
		ThrottleBurst:       100,
	})

	makeAuthorized := func(method, path, body string) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: session.ID})
		return req
	}

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected healthz 200, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	handler.ServeHTTP(res, makeAuthorized(http.MethodGet, "/api/v1/notes", ""))
	if res.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	createReq := makeAuthorized(http.MethodPost, "/api/v1/notes", `{"content":"created body"}`)
	handler.ServeHTTP(res, createReq)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", res.Code)
	}

	noteID := store.note.ID.String()
	getReq := makeAuthorized(http.MethodGet, "/api/v1/notes/"+noteID, "")
	getReq.SetPathValue("id", noteID)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, getReq)
	if res.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d", res.Code)
	}

	patchReq := makeAuthorized(http.MethodPatch, "/api/v1/notes/"+noteID, `{"content":"patched body"}`)
	patchReq.SetPathValue("id", noteID)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, patchReq)
	if res.Code != http.StatusOK {
		t.Fatalf("expected patch 200, got %d", res.Code)
	}

	deleteReq := makeAuthorized(http.MethodDelete, "/api/v1/notes/"+noteID, "")
	deleteReq.SetPathValue("id", noteID)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, deleteReq)
	if res.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d", res.Code)
	}

	sharedReq := httptest.NewRequest(http.MethodGet, "/api/v1/notes/shared/"+shareSlug, nil)
	sharedReq.SetPathValue("slug", shareSlug)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, sharedReq)
	if res.Code != http.StatusOK {
		t.Fatalf("expected shared note 200, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	invalidCreate := makeAuthorized(http.MethodPost, "/api/v1/notes", `{"content":"one"}{"content":"two"}`)
	handler.ServeHTTP(res, invalidCreate)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid create 400, got %d", res.Code)
	}

	invalidPatch := makeAuthorized(http.MethodPatch, "/api/v1/notes/"+noteID, `{"content":"one"}{"content":"two"}`)
	invalidPatch.SetPathValue("id", noteID)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, invalidPatch)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid patch 400, got %d", res.Code)
	}
}

func TestHandlerSecurityBranchesFromAppPackage(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	session := auth.Session{ID: "session-1", UserID: userID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)}
	user := auth.User{ID: userID}
	store := &coverageNotesStore{}

	authService := auth.NewService(&coverageUserStore{user: user}, coverageOIDCClient{}, &coverageSessionStore{session: session}, time.Hour, time.Minute)
	notesService := notes.NewService(store, coverageCache{}, time.Minute, time.Minute)
	handler := httpapi.NewHandler(httpapi.Dependencies{
		Logger:              slog.New(slog.NewTextHandler(ioDiscard{}, nil)),
		AuthService:         authService,
		NotesService:        notesService,
		SessionCookieName:   "go_notes_session",
		SessionCookieSecure: false,
		ThrottleRequestsPS:  100,
		ThrottleBurst:       100,
	})

	makeAuthorized := func(method, path, body string) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "go_notes_session", Value: session.ID})
		return req
	}

	checkStatus := func(t *testing.T, req *http.Request, want int) {
		t.Helper()
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("expected %d, got %d for %s %s", want, res.Code, req.Method, req.URL.Path)
		}
	}

	checkStatus(t, httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil), http.StatusUnauthorized)
	checkStatus(t, makeAuthorized(http.MethodGet, "/api/v1/notes?page_size=500", ""), http.StatusBadRequest)

	getBadID := makeAuthorized(http.MethodGet, "/api/v1/notes/not-a-uuid", "")
	getBadID.SetPathValue("id", "not-a-uuid")
	checkStatus(t, getBadID, http.StatusBadRequest)

	getMissing := makeAuthorized(http.MethodGet, "/api/v1/notes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "")
	getMissing.SetPathValue("id", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	checkStatus(t, getMissing, http.StatusNotFound)

	createInvalid := makeAuthorized(http.MethodPost, "/api/v1/notes", `{"content":"   "}`)
	checkStatus(t, createInvalid, http.StatusBadRequest)

	patchBadID := makeAuthorized(http.MethodPatch, "/api/v1/notes/not-a-uuid", `{"content":"body"}`)
	patchBadID.SetPathValue("id", "not-a-uuid")
	checkStatus(t, patchBadID, http.StatusBadRequest)

	patchInvalid := makeAuthorized(http.MethodPatch, "/api/v1/notes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", `{"content":null}`)
	patchInvalid.SetPathValue("id", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	checkStatus(t, patchInvalid, http.StatusBadRequest)

	deleteBadID := makeAuthorized(http.MethodDelete, "/api/v1/notes/not-a-uuid", "")
	deleteBadID.SetPathValue("id", "not-a-uuid")
	checkStatus(t, deleteBadID, http.StatusBadRequest)

	sharedMissing := httptest.NewRequest(http.MethodGet, "/api/v1/notes/shared/missing", nil)
	sharedMissing.SetPathValue("slug", "missing")
	checkStatus(t, sharedMissing, http.StatusNotFound)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
