// main.go — Claritty SRE Agent
// Runs inside a Kubernetes cluster as a Deployment with one active scanner.
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
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/tools/leaderelection"

	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/ai"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/k8s"
)

func main() {
	cfg := config.Load()
	if cfg.LeaderLease != "" && (!cfg.AutoWatch || cfg.EnableControl || cfg.PodNamespace == "" || cfg.PodName == "") {
		log.Fatal("Lease election requires autowatch, disabled scan controls, POD_NAMESPACE, and POD_NAME")
	}

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
	metrics := newScanMetrics()
	hubClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	scanner := &scanController{
		interval: cfg.ScanInterval,
		scanSlot: make(chan struct{}, 1),
		run: func(ctx context.Context) {
			runScanAndSend(ctx, pipeline, cfg, hubClient, metrics)
		},
	}
	if cfg.LeaderLease != "" {
		elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
			Lock:          k8sCli.LeaseLock(cfg.PodNamespace, cfg.LeaderLease, cfg.PodName),
			LeaseDuration: 30 * time.Second,
			RenewDeadline: 20 * time.Second,
			RetryPeriod:   5 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(leaderCtx context.Context) {
					metrics.leader.Set(1)
					defer metrics.leader.Set(0)
					scanner.startWatch(leaderCtx)
					<-leaderCtx.Done()
				},
				OnStoppedLeading: stop,
			},
		})
		if err != nil {
			log.Fatalf("Configure Lease election: %v", err)
		}
		go elector.Run(ctx)
	} else {
		metrics.leader.Set(1)
		if cfg.AutoWatch {
			scanner.startWatch(ctx)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.handler)

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
func runScanAndSend(ctx context.Context, pipeline *ai.Pipeline, cfg *config.Config, client *http.Client, metrics *scanMetrics) {
	log.Printf("[%s] Starting AI scan...", cfg.ClusterName)
	started := time.Now()
	report, err := pipeline.RunScan(ctx)
	duration := time.Since(started).Seconds()
	if err != nil {
		metrics.scanDuration.WithLabelValues("error").Observe(duration)
		log.Printf("Scan error: %v", err)
		return
	}
	metrics.scanDuration.WithLabelValues("success").Observe(duration)
	log.Printf("[%s] Scan complete: %s — %s (confidence %d%%)",
		cfg.ClusterName, report.Severity, report.Title, report.ConfidenceScore)

	if err := sendToHub(ctx, client, cfg, report); err != nil {
		metrics.reports.WithLabelValues("error").Inc()
		log.Printf("Failed to send report to Hub: %v", err)
		return
	}
	metrics.reports.WithLabelValues("success").Inc()
	metrics.lastReport.SetToCurrentTime()
}

func sendToHub(ctx context.Context, client *http.Client, cfg *config.Config, report any) error {
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	url := strings.TrimRight(cfg.HubURL, "/") + "/api/v1/incidents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.HubAPIKey != "" {
		req.Header.Set("X-Claritty-Key", cfg.HubAPIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Hub rejected report: %s", resp.Status)
	}
	log.Printf("Report sent to hub: %s", resp.Status)
	return nil
}
