// main.go — Claritty SRE Agent
// Runs inside a Kubernetes cluster as a Deployment (1 replica).
// Exposes an HTTP API for the hub dashboard to trigger scans.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/ai"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/k8s"
)

func main() {
	cfg := config.Load()

	log.Printf("Claritty SRE Agent starting | cluster=%s hub=%s", cfg.ClusterName, cfg.HubURL)

	k8sCli, err := k8s.New()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	pipeline, err := ai.New(cfg, k8sCli)
	if err != nil {
		log.Fatalf("Failed to create AI pipeline: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scanner := &scanController{
		interval: cfg.ScanInterval,
		scanSlot: make(chan struct{}, 1),
		run: func(ctx context.Context) {
			runScanAndSend(ctx, pipeline, cfg)
		},
	}
	if cfg.AutoWatch {
		scanner.startWatch(ctx)
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	if cfg.EnableControl {
		mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, r *http.Request) {
			if !scanner.trigger(ctx) {
				http.Error(w, "scan unavailable or already running", http.StatusConflict)
				return
			}
			log.Println("Manual scan triggered via /trigger")
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"status":"scan started"}`)
		})
		mux.HandleFunc("POST /watch", func(w http.ResponseWriter, r *http.Request) {
			if !scanner.startWatch(ctx) {
				fmt.Fprint(w, `{"status":"already watching"}`)
				return
			}
			fmt.Fprintf(w, `{"status":"watch started","interval_secs":%d}`, int(cfg.ScanInterval.Seconds()))
		})
		mux.HandleFunc("DELETE /watch", func(w http.ResponseWriter, r *http.Request) {
			scanner.stopWatch()
			fmt.Fprint(w, `{"status":"watch stopped"}`)
		})
	}
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"cluster": cfg.ClusterName, "hub": cfg.HubURL,
			"watching": scanner.watching(), "interval_secs": int(cfg.ScanInterval.Seconds()),
		})
	})

	// ── Graceful shutdown ───────────────────────────────────────────────────
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("Agent HTTP server listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received — stopping agent")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	scanner.waitForScan(shutCtx)
}

// runScanAndSend runs the AI pipeline and POSTs the result to the hub.
func runScanAndSend(ctx context.Context, pipeline *ai.Pipeline, cfg *config.Config) {
	log.Printf("[%s] Starting AI scan...", cfg.ClusterName)
	report, err := pipeline.RunScan(ctx)
	if err != nil {
		log.Printf("Scan error: %v", err)
		return
	}
	log.Printf("[%s] Scan complete: %s — %s (confidence %d%%)",
		cfg.ClusterName, report.Severity, report.Title, report.ConfidenceScore)

	sendToHub(ctx, cfg, report)
}

func sendToHub(ctx context.Context, cfg *config.Config, report interface{}) {
	body, err := json.Marshal(report)
	if err != nil {
		log.Printf("Failed to marshal report: %v", err)
		return
	}

	url := cfg.HubURL + "/api/v1/incidents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("Failed to create hub request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.HubAPIKey != "" {
		req.Header.Set("X-Claritty-Key", cfg.HubAPIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send report to hub: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("Hub rejected report: %s", resp.Status)
		return
	}
	log.Printf("Report sent to hub: %s", resp.Status)
}
