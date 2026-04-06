package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeKV struct {
	values map[string]string
	setErr error
	delErr error
}

func (f *fakeKV) Get(ctx context.Context, key string) (string, bool) {
	value, ok := f.values[key]
	return value, ok
}

func (f *fakeKV) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

func (f *fakeKV) Delete(ctx context.Context, keys ...string) error {
	if f.delErr != nil {
		return f.delErr
	}
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func TestCacheSessionStorePendingAuthLifecycle(t *testing.T) {
	kv := &fakeKV{values: map[string]string{}}
	store := NewCacheSessionStore(kv)
	pending := PendingAuth{
		State:     "state-1",
		Nonce:     "nonce-1",
		Verifier:  "verifier-1",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}

	if err := store.StorePendingAuth(context.Background(), pending, time.Minute); err != nil {
		t.Fatalf("unexpected store error: %v", err)
	}
	got, err := store.ConsumePendingAuth(context.Background(), pending.State)
	if err != nil {
		t.Fatalf("unexpected consume error: %v", err)
	}
	if got.State != pending.State || got.Verifier != pending.Verifier {
		t.Fatalf("unexpected pending auth: %+v", got)
	}
	if _, ok := kv.values[pendingAuthKey(pending.State)]; ok {
		t.Fatal("expected pending auth to be single-use")
	}
}

func TestCacheSessionStoreConsumePendingAuthErrors(t *testing.T) {
	store := NewCacheSessionStore(&fakeKV{values: map[string]string{}})

	if _, err := store.ConsumePendingAuth(context.Background(), "missing"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state, got %v", err)
	}

	expiredStore := NewCacheSessionStore(&fakeKV{values: map[string]string{pendingAuthKey("state"): `{"state":"state","expires_at":"2000-01-01T00:00:00Z"}`}})
	if _, err := expiredStore.ConsumePendingAuth(context.Background(), "state"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state for expired token, got %v", err)
	}

	badJSONStore := NewCacheSessionStore(&fakeKV{values: map[string]string{pendingAuthKey("state"): `{"state"`}})
	if _, err := badJSONStore.ConsumePendingAuth(context.Background(), "state"); err == nil {
		t.Fatal("expected invalid json to be returned")
	}

	deleteErrStore := NewCacheSessionStore(&fakeKV{
		delErr: errors.New("delete failed"),
		values: map[string]string{pendingAuthKey("state"): `{"state":"state","expires_at":"2999-01-01T00:00:00Z"}`},
	})
	if _, err := deleteErrStore.ConsumePendingAuth(context.Background(), "state"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestCacheSessionStoreSessionLifecycle(t *testing.T) {
	kv := &fakeKV{values: map[string]string{}}
	store := NewCacheSessionStore(kv)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	session, err := store.CreateSession(context.Background(), userID, time.Hour)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if session.ID == "" || session.UserID != userID {
		t.Fatalf("unexpected session: %+v", session)
	}

	got, err := store.GetSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("unexpected session lookup: %+v", got)
	}

	if err := store.DeleteSession(context.Background(), session.ID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if _, ok := kv.values[sessionKey(session.ID)]; ok {
		t.Fatal("expected session to be deleted")
	}
}

func TestCacheSessionStoreGetSessionHandlesMissingAndExpired(t *testing.T) {
	store := NewCacheSessionStore(&fakeKV{values: map[string]string{}})
	if _, err := store.GetSession(context.Background(), "missing"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}

	expired := `{"id":"session-1","user_id":"11111111-1111-1111-1111-111111111111","created_at":"2020-01-01T00:00:00Z","expires_at":"2020-01-01T00:00:01Z"}`
	kv := &fakeKV{values: map[string]string{sessionKey("session-1"): expired}}
	store = NewCacheSessionStore(kv)
	if _, err := store.GetSession(context.Background(), "session-1"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized for expired session, got %v", err)
	}

	badJSONStore := NewCacheSessionStore(&fakeKV{values: map[string]string{sessionKey("session-1"): `{"id"`}})
	if _, err := badJSONStore.GetSession(context.Background(), "session-1"); err == nil {
		t.Fatal("expected invalid session json error")
	}
}

func TestCacheSessionStorePropagatesKVErrors(t *testing.T) {
	pending := PendingAuth{State: "state", ExpiresAt: time.Now().UTC().Add(time.Minute)}
	if err := NewCacheSessionStore(&fakeKV{setErr: errors.New("set failed")}).StorePendingAuth(context.Background(), pending, time.Minute); err == nil {
		t.Fatal("expected set error")
	}
	if _, err := NewCacheSessionStore(&fakeKV{setErr: errors.New("set failed")}).CreateSession(context.Background(), uuid.New(), time.Minute); err == nil {
		t.Fatal("expected create session set error")
	}
	if err := NewCacheSessionStore(&fakeKV{delErr: errors.New("delete failed"), values: map[string]string{pendingAuthKey("state"): `{"state":"state","expires_at":"2999-01-01T00:00:00Z"}`}}).DeleteSession(context.Background(), "session-1"); err == nil {
		t.Fatal("expected delete error")
	}
}
