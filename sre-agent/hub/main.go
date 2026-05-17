// main.go - Claritty SRE Hub Server
// Serves the REST API and web dashboard on port 8822.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/api"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/db"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/slack"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required. Set it to your PostgreSQL connection string.")
	}

	// ── Database ─────────────────────────────────────────────────────────────
	store, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v\nCheck your DATABASE_URL setting.", err)
	}
	log.Println("✓ Connected to PostgreSQL")

	// ── Slack ─────────────────────────────────────────────────────────────────
	slackClient := slack.New(cfg.SlackWebhook, cfg.SlackChannel)
	if slackClient != nil {
		log.Printf("✓ Slack alerts enabled → %s", cfg.SlackChannel)
	}

	hubBaseURL := fmt.Sprintf("http://localhost:%s", cfg.Port)

	// ── Routes ────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// REST API
	apiHandler := api.New(store, slackClient, hubBaseURL, cfg.HubAPIKey)
	apiHandler.RegisterRoutes(mux)

	// Static dashboard — serve from embedded dashboard/ directory
	fs := http.FileServer(http.Dir("dashboard"))
	mux.Handle("/", fs)

	// ── Server ────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("🚀 Claritty Hub running → http://localhost:%s", cfg.Port)
		log.Printf("   Dashboard  : http://localhost:%s/", cfg.Port)
		log.Printf("   API        : http://localhost:%s/api/v1/", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received — stopping hub")
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}
