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
	"sync"
	"syscall"
	"time"

	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/ai"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/k8s"
)

var (
	watchMu      sync.Mutex
	watchCancel  context.CancelFunc
	watchRunning bool
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

	// ── HTTP API ────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// POST /trigger — run one immediate scan
	mux.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		log.Println("Manual scan triggered via /trigger")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			runScanAndSend(ctx, pipeline, cfg)
		}()
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"scan started"}`)
	})

	// POST /watch — start continuous scanning
	mux.HandleFunc("/watch", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			watchMu.Lock()
			defer watchMu.Unlock()
			if watchRunning {
				fmt.Fprint(w, `{"status":"already watching"}`)
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			watchCancel = cancel
			watchRunning = true
			go func() {
				log.Printf("Continuous watch started (interval=%s)", cfg.ScanInterval)
				for {
					scanCtx, scanCancel := context.WithTimeout(ctx, 10*time.Minute)
					runScanAndSend(scanCtx, pipeline, cfg)
					scanCancel()
					select {
					case <-time.After(cfg.ScanInterval):
					case <-ctx.Done():
						watchMu.Lock()
						watchRunning = false
						watchMu.Unlock()
						log.Println("Continuous watch stopped")
						return
					}
				}
			}()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"watch started","interval_secs":%d}`, int(cfg.ScanInterval.Seconds()))

		case http.MethodDelete:
			watchMu.Lock()
			defer watchMu.Unlock()
			if watchCancel != nil {
				watchCancel()
				watchCancel = nil
			}
			fmt.Fprint(w, `{"status":"watch stopped"}`)

		default:
			http.Error(w, "POST or DELETE only", http.StatusMethodNotAllowed)
		}
	})

	// GET /status — return agent status
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		watchMu.Lock()
		running := watchRunning
		watchMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"cluster":"%s","hub":"%s","watching":%v,"interval_secs":%d}`,
			cfg.ClusterName, cfg.HubURL, running, int(cfg.ScanInterval.Seconds()))
	})

	// ── Graceful shutdown ───────────────────────────────────────────────────
	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	log.Printf("Report sent to hub: %s", resp.Status)
}
