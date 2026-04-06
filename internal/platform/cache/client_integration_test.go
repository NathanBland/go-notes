package cache

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewConnectsToValkeyAndPerformsKVOperations(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("integration coverage only")
	}

	address := os.Getenv("VALKEY_ADDR")
	if address == "" {
		address = "127.0.0.1:6379"
	}

	client, err := New(address, "")
	if err != nil {
		t.Fatalf("expected valkey client to connect, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Fatalf("close valkey client: %v", closeErr)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Set(ctx, "coverage:integration:key", "value", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	value, ok := client.Get(ctx, "coverage:integration:key")
	if !ok || value != "value" {
		t.Fatalf("unexpected get result ok=%v value=%q", ok, value)
	}
	if err := client.Delete(ctx, "coverage:integration:key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
