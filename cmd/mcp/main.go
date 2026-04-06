package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/uuid"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/nathanbland/go-notes/internal/app"
	"github.com/nathanbland/go-notes/internal/mcpapi"
	"github.com/nathanbland/go-notes/internal/notes"
	cacheclient "github.com/nathanbland/go-notes/internal/platform/cache"
	"github.com/nathanbland/go-notes/internal/platform/db"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func versionString() string {
	return fmt.Sprintf("go-notes-mcp version=%s commit=%s date=%s", version, commit, date)
}

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString())
		return
	}

	ctx := context.Background()

	config, err := app.LoadMCPConfig()
	if err != nil {
		log.Fatal(err)
	}
	ownerID, err := uuid.Parse(config.OwnerUserID)
	if err != nil {
		log.Fatalf("MCP_OWNER_USER_ID must be a UUID: %v", err)
	}

	pool, err := db.NewPool(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	cache, err := cacheclient.New(config.ValkeyAddr, config.ValkeyPassword)
	if err != nil {
		log.Fatalf("failed to connect to valkey: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			log.Printf("failed to close valkey client: %v", closeErr)
		}
	}()

	noteService := notes.NewService(
		db.NewStore(pool),
		cache,
		config.NoteCacheTTL,
		config.ListCacheTTL,
	)

	server := mcpapi.NewServer(noteService, ownerID)
	if err := mcpserver.ServeStdio(server); err != nil {
		log.Fatal(err)
	}
}
