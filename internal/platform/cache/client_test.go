package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRawClient struct{ closed bool }

func (f *fakeRawClient) Close() { f.closed = true }

type fakeStringResult struct {
	value string
	err   error
}

func (f fakeStringResult) Result() (string, error) { return f.value, f.err }

type fakeStatusResult struct{ err error }

func (f fakeStatusResult) Err() error { return f.err }

type fakeIntResult struct{ err error }

func (f fakeIntResult) Err() error { return f.err }

type fakeCompat struct {
	getValue   string
	getErr     error
	setErr     error
	deleteErr  error
	lastGetKey string
	lastSetKey string
	lastSetTTL time.Duration
	lastDelete []string
}

func (f *fakeCompat) Get(ctx context.Context, key string) stringResult {
	f.lastGetKey = key
	return fakeStringResult{value: f.getValue, err: f.getErr}
}

func (f *fakeCompat) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) statusResult {
	f.lastSetKey = key
	f.lastSetTTL = expiration
	return fakeStatusResult{err: f.setErr}
}

func (f *fakeCompat) Del(ctx context.Context, keys ...string) intResult {
	f.lastDelete = append([]string(nil), keys...)
	return fakeIntResult{err: f.deleteErr}
}

func TestClientMethods(t *testing.T) {
	raw := &fakeRawClient{}
	compat := &fakeCompat{getValue: "value"}
	client := &Client{raw: raw, compat: compat}

	value, ok := client.Get(context.Background(), "notes:key")
	if !ok || value != "value" {
		t.Fatalf("unexpected get result ok=%v value=%q", ok, value)
	}
	if compat.lastGetKey != "notes:key" {
		t.Fatalf("expected get key to be recorded, got %q", compat.lastGetKey)
	}

	if err := client.Set(context.Background(), "notes:key", "payload", time.Minute); err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}
	if compat.lastSetKey != "notes:key" || compat.lastSetTTL != time.Minute {
		t.Fatalf("unexpected set call key=%q ttl=%s", compat.lastSetKey, compat.lastSetTTL)
	}

	if err := client.Delete(context.Background(), "a", "b"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if len(compat.lastDelete) != 2 {
		t.Fatalf("expected delete keys to be recorded, got %#v", compat.lastDelete)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if !raw.closed {
		t.Fatal("expected raw client to be closed")
	}
}

func TestClientGetMissAndCommandErrors(t *testing.T) {
	client := &Client{
		raw:    &fakeRawClient{},
		compat: &fakeCompat{getErr: errors.New("boom"), setErr: errors.New("set"), deleteErr: errors.New("delete")},
	}

	if _, ok := client.Get(context.Background(), "notes:key"); ok {
		t.Fatal("expected get error to look like a miss")
	}
	if err := client.Set(context.Background(), "notes:key", "payload", time.Second); err == nil {
		t.Fatal("expected set error")
	}
	if err := client.Delete(context.Background(), "notes:key"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestClientCloseWithoutRawClient(t *testing.T) {
	client := &Client{}
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil raw close to succeed, got %v", err)
	}
}
