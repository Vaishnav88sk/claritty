package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type scanMetrics struct {
	scanDuration *prometheus.HistogramVec
	reports      *prometheus.CounterVec
	lastReport   prometheus.Gauge
	leader       prometheus.Gauge
	handler      http.Handler
}

func newScanMetrics() *scanMetrics {
	metrics := &scanMetrics{
		scanDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "claritty_agent_scan_duration_seconds",
			Help:    "Duration of the Agent analysis pipeline.",
			Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600},
		}, []string{"outcome"}),
		reports: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "claritty_agent_reports_total",
			Help: "Hub report delivery attempts by outcome.",
		}, []string{"outcome"}),
		lastReport: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "claritty_agent_last_successful_delivery_timestamp_seconds",
			Help: "Unix time of the last successful report delivery by this process.",
		}),
		leader: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "claritty_agent_leader",
			Help: "Whether this Agent is eligible to scan.",
		}),
	}
	for _, outcome := range []string{"success", "error"} {
		metrics.scanDuration.WithLabelValues(outcome)
		metrics.reports.WithLabelValues(outcome)
	}
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.scanDuration, metrics.reports, metrics.lastReport, metrics.leader)
	metrics.handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{MaxRequestsInFlight: 2})
	return metrics
}
