package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/nathanbland/go-notes/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx)
	if err != nil {
		panic(err)
	}

	application.Logger.Info("starting go-notes", "addr", application.Config.HTTPAddr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := application.Shutdown(shutdownCtx); err != nil {
			application.Logger.Error("shutdown failed", "error", err)
		}
	}()

	if err := application.Server.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
		panic(err)
	}
}
