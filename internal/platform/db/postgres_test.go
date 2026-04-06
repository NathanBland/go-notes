package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewPoolInvalidConfig(t *testing.T) {
	if _, err := NewPool(context.Background(), "://bad"); err == nil {
		t.Fatal("expected parse config error")
	}
}

func TestNewPoolConstructorAndPingFailure(t *testing.T) {
	origParse := parsePoolConfig
	origNew := newPoolWithConfig
	t.Cleanup(func() {
		parsePoolConfig = origParse
		newPoolWithConfig = origNew
	})

	parsePoolConfig = func(databaseURL string) (*pgxpool.Config, error) {
		return nil, errors.New("parse failed")
	}
	if _, err := NewPool(context.Background(), "postgres://example"); err == nil {
		t.Fatal("expected parse failure to be returned")
	}
}

func TestNewPoolConfiguresAndConnectsInUTC(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") == "" {
		t.Skip("integration coverage only")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected pool to connect, got %v", err)
	}
	t.Cleanup(pool.Close)

	var timezone string
	if err := pool.QueryRow(ctx, "SHOW TIME ZONE").Scan(&timezone); err != nil {
		t.Fatalf("show time zone: %v", err)
	}
	if timezone != "UTC" {
		t.Fatalf("expected UTC session timezone, got %q", timezone)
	}
}

func TestNewPoolPropagatesConstructorFailures(t *testing.T) {
	origParse := parsePoolConfig
	origNew := newPoolWithConfig
	t.Cleanup(func() {
		parsePoolConfig = origParse
		newPoolWithConfig = origNew
	})

	parsePoolConfig = func(databaseURL string) (*pgxpool.Config, error) {
		return &pgxpool.Config{ConnConfig: &pgx.ConnConfig{}}, nil
	}
	newPoolWithConfig = func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
		if config.MaxConns != 10 || config.MinConns != 1 || config.MinIdleConns != 1 {
			t.Fatalf("expected pool sizing to be applied, got %+v", config)
		}
		return nil, errors.New("construct failed")
	}

	if _, err := NewPool(context.Background(), "postgres://example"); err == nil {
		t.Fatal("expected constructor failure to be returned")
	}
}
