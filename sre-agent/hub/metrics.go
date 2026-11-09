package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/db"
)

var reportReceipt = prometheus.NewDesc(
	"claritty_hub_cluster_last_report_received_timestamp_seconds",
	"Unix time when the Hub received the most recent persisted report for a cluster.",
	[]string{"cluster"}, nil,
)

type reportCollector struct{ store *db.Store }

func (c reportCollector) Describe(ch chan<- *prometheus.Desc) { ch <- reportReceipt }

func (c reportCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	receipts, err := c.store.ReportReceipts(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(reportReceipt, err)
		return
	}
	for _, receipt := range receipts {
		ch <- prometheus.MustNewConstMetric(reportReceipt, prometheus.GaugeValue, receipt.Timestamp, receipt.Cluster)
	}
}

func metricsHandler(store *db.Store) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(reportCollector{store: store})
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{MaxRequestsInFlight: 2})
}
