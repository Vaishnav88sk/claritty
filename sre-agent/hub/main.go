// main.go - Claritty SRE Hub Server
// Serves the REST API and web dashboard on port 8822.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/api"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/db"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/slack"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "apply database migrations and exit")
	flag.Parse()
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required. Set it to your PostgreSQL connection string.")
	}
	if !*migrateOnly && (cfg.RequireAPIKey || cfg.IngestAddr != "") && strings.TrimSpace(cfg.HubAPIKey) == "" {
		log.Fatal("CLARITTY_HUB_API_KEY is required for authenticated ingestion")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Database ─────────────────────────────────────────────────────────────
	connectCtx, cancelConnect := context.WithTimeout(ctx, 10*time.Second)
	store, err := db.Open(connectCtx, cfg.DatabaseURL)
	cancelConnect()
	if err != nil {
		log.Fatalf("Database connection failed: %v\nCheck your DATABASE_URL setting.", err)
	}
	defer store.Close()
	if *migrateOnly || cfg.AutoMigrate {
		migrationCtx, cancelMigration := context.WithTimeout(ctx, 5*time.Minute)
		err = store.Migrate(migrationCtx)
		cancelMigration()
		if err != nil {
			log.Fatalf("Database migration failed: %v", err)
		}
	}
	if *migrateOnly {
		return
	}
	schemaCtx, cancelSchema := context.WithTimeout(ctx, 5*time.Second)
	err = store.CheckSchema(schemaCtx)
	cancelSchema()
	if err != nil {
		log.Fatalf("Database schema unavailable: %v", err)
	}
	log.Println("✓ Connected to PostgreSQL")

	// ── Slack ─────────────────────────────────────────────────────────────────
	slackClient := slack.New(cfg.SlackWebhook, cfg.SlackChannel)
	if slackClient != nil {
		log.Printf("✓ Slack alerts enabled → %s", cfg.SlackChannel)
	}

	hubBaseURL := cfg.PublicURL
	if hubBaseURL == "" {
		hubBaseURL = fmt.Sprintf("http://localhost:%s", cfg.Port)
	}

	// ── Routes ────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	management := mux
	if cfg.ManagementAddr != "" {
		management = http.NewServeMux()
	}
	management.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	management.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := store.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, "ok")
	})
	management.Handle("GET /metrics", metricsHandler(store))

	// REST API
	apiHandler := api.New(store, slackClient, hubBaseURL, cfg.HubAPIKey)
	apiHandler.RegisterRoutes(mux)

	// Static dashboard — serve from dashboard/dist directory if built, otherwise dashboard/
	dashboardDir := "dashboard"
	if fi, err := os.Stat("dashboard/dist"); err == nil && fi.IsDir() {
		dashboardDir = "dashboard/dist"
	}
	fs := http.FileServer(http.Dir(dashboardDir))
	mux.Handle("/", fs)

	dashboardAddr := ":" + cfg.Port
	var dashboard http.Handler = mux
	var servers []*http.Server
	if cfg.IngestAddr != "" {
		// The SSO sidecar shares the pod network namespace and reaches loopback.
		dashboardAddr = "127.0.0.1:" + cfg.Port
		dashboard = http.NewCrossOriginProtection().Handler(mux)
		servers = append(servers, serve(cfg.IngestAddr, apiHandler.IngestionHandler()))
	}
	servers = append(servers, serve(dashboardAddr, dashboard))
	if cfg.ManagementAddr != "" {
		servers = append(servers, serve(cfg.ManagementAddr, management))
	}

	<-ctx.Done()
	log.Println("Shutdown signal received — stopping hub")
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, server := range servers {
		_ = server.Shutdown(shutCtx)
	}
}

func serve(addr string, handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Printf("Hub listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	return server
}
