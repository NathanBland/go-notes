package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeUserStore struct {
	user      User
	upsertErr error
	getErr    error
	lastID    uuid.UUID
}

func (f *fakeUserStore) UpsertUserFromOIDC(ctx context.Context, identity Identity) (User, error) {
	if f.upsertErr != nil {
		return User{}, f.upsertErr
	}
	return f.user, nil
}

func (f *fakeUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	f.lastID = id
	if f.getErr != nil {
		return User{}, f.getErr
	}
	return f.user, nil
}

type fakeOIDCClient struct {
	redirectURL string
	authErr     error
	identity    Identity
	exchangeErr error
	lastState   string
	lastNonce   string
	lastCode    string
}

func (f *fakeOIDCClient) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	f.lastState = state
	f.lastNonce = nonce
	if f.authErr != nil {
		return "", f.authErr
	}
	return f.redirectURL, nil
}

func (f *fakeOIDCClient) Exchange(ctx context.Context, code, verifier, expectedNonce string) (Identity, error) {
	f.lastCode = code
	f.lastNonce = expectedNonce
	if f.exchangeErr != nil {
		return Identity{}, f.exchangeErr
	}
	return f.identity, nil
}

type fakeSessionStore struct {
	pending      PendingAuth
	pendingErr   error
	session      Session
	sessionErr   error
	deleteErr    error
	storedTTL    time.Duration
	storedState  string
	createUserID uuid.UUID
}

func (f *fakeSessionStore) StorePendingAuth(ctx context.Context, pending PendingAuth, ttl time.Duration) error {
	f.pending = pending
	f.storedTTL = ttl
	f.storedState = pending.State
	return f.pendingErr
}

func (f *fakeSessionStore) ConsumePendingAuth(ctx context.Context, state string) (PendingAuth, error) {
	if f.pendingErr != nil {
		return PendingAuth{}, f.pendingErr
	}
	return f.pending, nil
}

func (f *fakeSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, ttl time.Duration) (Session, error) {
	f.createUserID = userID
	if f.sessionErr != nil {
		return Session{}, f.sessionErr
	}
	if f.session.ID == "" {
		f.session = Session{
			ID:        "session-1",
			UserID:    userID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(ttl),
		}
	}
	return f.session, nil
}

func (f *fakeSessionStore) GetSession(ctx context.Context, id string) (Session, error) {
	if f.sessionErr != nil {
		return Session{}, f.sessionErr
	}
	return f.session, nil
}

func (f *fakeSessionStore) DeleteSession(ctx context.Context, id string) error {
	return f.deleteErr
}

func TestBeginLoginStoresPendingStateAndReturnsRedirect(t *testing.T) {
	store := &fakeSessionStore{}
	oidc := &fakeOIDCClient{redirectURL: "https://issuer.example/login"}
	service := NewService(&fakeUserStore{}, oidc, store, time.Hour, 5*time.Minute)

	redirectURL, err := service.BeginLogin(context.Background(), "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if redirectURL != "https://issuer.example/login" {
		t.Fatalf("unexpected redirect URL: %s", redirectURL)
	}
	if store.storedTTL != 5*time.Minute {
		t.Fatalf("expected state ttl to be stored, got %s", store.storedTTL)
	}
	if store.pending.State == "" || store.pending.Nonce == "" || store.pending.Verifier == "" {
		t.Fatalf("expected pending auth to be populated, got %+v", store.pending)
	}
	if store.pending.RedirectTo != "/" {
		t.Fatalf("expected redirect target to be stored, got %q", store.pending.RedirectTo)
	}
	if oidc.lastState != store.pending.State || oidc.lastNonce != store.pending.Nonce {
		t.Fatalf("expected OIDC client to receive pending state, got state=%q nonce=%q", oidc.lastState, oidc.lastNonce)
	}
}

func TestBeginLoginReturnsOIDCError(t *testing.T) {
	wantErr := errors.New("oidc unavailable")
	service := NewService(&fakeUserStore{}, &fakeOIDCClient{authErr: wantErr}, &fakeSessionStore{}, time.Hour, time.Minute)

	_, err := service.BeginLogin(context.Background(), "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected oidc error, got %v", err)
	}
}

func TestFinishLoginValidationErrors(t *testing.T) {
	service := NewService(&fakeUserStore{}, &fakeOIDCClient{}, &fakeSessionStore{}, time.Hour, time.Minute)

	if _, _, _, err := service.FinishLogin(context.Background(), "", "code"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state, got %v", err)
	}
	if _, _, _, err := service.FinishLogin(context.Background(), "state", ""); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected invalid code, got %v", err)
	}
}

func TestFinishLoginSuccess(t *testing.T) {
	user := User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	sessionStore := &fakeSessionStore{
		pending: PendingAuth{State: "state", Nonce: "nonce", Verifier: "verifier", RedirectTo: "/"},
		session: Session{ID: "session-1", UserID: user.ID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)},
	}
	oidc := &fakeOIDCClient{identity: Identity{Issuer: "issuer", Subject: "subject"}}
	service := NewService(&fakeUserStore{user: user}, oidc, sessionStore, time.Hour, time.Minute)

	gotUser, gotSession, gotPending, err := service.FinishLogin(context.Background(), "state", "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser.ID != user.ID || gotSession.ID != "session-1" {
		t.Fatalf("unexpected login result user=%+v session=%+v", gotUser, gotSession)
	}
	if gotPending.RedirectTo != "/" {
		t.Fatalf("expected pending redirect target, got %+v", gotPending)
	}
	if sessionStore.createUserID != user.ID {
		t.Fatalf("expected session creation for %s, got %s", user.ID, sessionStore.createUserID)
	}
}

func TestFinishLoginPropagatesIntermediateErrors(t *testing.T) {
	user := User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}

	service := NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{pendingErr: ErrInvalidState}, time.Hour, time.Minute)
	if _, _, _, err := service.FinishLogin(context.Background(), "state", "code"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected pending auth error, got %v", err)
	}

	service = NewService(&fakeUserStore{user: user}, &fakeOIDCClient{exchangeErr: errors.New("exchange failed")}, &fakeSessionStore{
		pending: PendingAuth{State: "state", Nonce: "nonce", Verifier: "verifier"},
	}, time.Hour, time.Minute)
	if _, _, _, err := service.FinishLogin(context.Background(), "state", "code"); err == nil {
		t.Fatal("expected exchange error")
	}

	service = NewService(&fakeUserStore{upsertErr: errors.New("upsert failed")}, &fakeOIDCClient{identity: Identity{Issuer: "issuer", Subject: "subject"}}, &fakeSessionStore{
		pending: PendingAuth{State: "state", Nonce: "nonce", Verifier: "verifier"},
	}, time.Hour, time.Minute)
	if _, _, _, err := service.FinishLogin(context.Background(), "state", "code"); err == nil {
		t.Fatal("expected upsert error")
	}

	service = NewService(&fakeUserStore{user: user}, &fakeOIDCClient{identity: Identity{Issuer: "issuer", Subject: "subject"}}, &fakeSessionStore{
		pending:    PendingAuth{State: "state", Nonce: "nonce", Verifier: "verifier"},
		sessionErr: errors.New("session failed"),
	}, time.Hour, time.Minute)
	if _, _, _, err := service.FinishLogin(context.Background(), "state", "code"); err == nil {
		t.Fatal("expected session creation error")
	}
}

func TestAuthenticateHandlesUnauthorizedCases(t *testing.T) {
	service := NewService(&fakeUserStore{}, &fakeOIDCClient{}, &fakeSessionStore{sessionErr: ErrUnauthorized}, time.Hour, time.Minute)

	if _, _, err := service.Authenticate(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized for blank session id, got %v", err)
	}
	if _, _, err := service.Authenticate(context.Background(), "session-1"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized for missing session, got %v", err)
	}
}

func TestAuthenticateSuccess(t *testing.T) {
	user := User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}
	session := Session{ID: "session-1", UserID: user.ID}
	service := NewService(&fakeUserStore{user: user}, &fakeOIDCClient{}, &fakeSessionStore{session: session}, time.Hour, time.Minute)

	gotUser, gotSession, err := service.Authenticate(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser.ID != user.ID || gotSession.ID != session.ID {
		t.Fatalf("unexpected auth result user=%+v session=%+v", gotUser, gotSession)
	}
}

func TestAuthenticateReturnsUnauthorizedWhenUserLookupFails(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	service := NewService(&fakeUserStore{getErr: errors.New("lookup failed")}, &fakeOIDCClient{}, &fakeSessionStore{
		session: Session{ID: "session-1", UserID: userID},
	}, time.Hour, time.Minute)

	if _, _, err := service.Authenticate(context.Background(), "session-1"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized when user lookup fails, got %v", err)
	}
}

func TestLogout(t *testing.T) {
	sessionStore := &fakeSessionStore{}
	service := NewService(&fakeUserStore{}, &fakeOIDCClient{}, sessionStore, time.Hour, time.Minute)

	if err := service.Logout(context.Background(), ""); err != nil {
		t.Fatalf("blank logout should be a no-op, got %v", err)
	}
	sessionStore.deleteErr = errors.New("delete failed")
	if err := service.Logout(context.Background(), "session-1"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestRandomTokenReturnsURLSafeValue(t *testing.T) {
	token, err := randomToken(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}
