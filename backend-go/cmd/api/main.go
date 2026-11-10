package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/elvove/photography/backend-go/internal/config"
	"github.com/elvove/photography/backend-go/internal/httpapi"
	"github.com/elvove/photography/backend-go/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	db, err := store.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := httpapi.BootstrapAdmin(context.Background(), db, cfg.AdminUsername, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + strings.TrimPrefix(cfg.Port, ":"),
		Handler:           httpapi.NewServer(cfg, db).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	stop, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()
	go func() {
		<-stop.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	log.Printf("Go API listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
