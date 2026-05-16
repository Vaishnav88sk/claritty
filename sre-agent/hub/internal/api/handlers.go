// Package api implements the Hub REST API handlers.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/db"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/slack"
)

// Handler holds dependencies for all API routes.
type Handler struct {
	store      *db.Store
	slack      *slack.Client
	hubBaseURL string
	apiKey     string
}

// New creates the API handler.
func New(store *db.Store, slackClient *slack.Client, hubBaseURL, apiKey string) *Handler {
	return &Handler{store: store, slack: slackClient, hubBaseURL: hubBaseURL, apiKey: apiKey}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/incidents", h.handleIncidents)
	mux.HandleFunc("/api/v1/incidents/", h.handleIncidentByID)
	mux.HandleFunc("/api/v1/clusters", h.handleClusters)
	mux.HandleFunc("/api/v1/clusters/", h.handleClusterDetail)
	mux.HandleFunc("/api/v1/stats", h.handleStats)
}

// ─── Incidents ────────────────────────────────────────────────────────────────

func (h *Handler) handleIncidents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		h.receiveIncident(w, r)
	case http.MethodGet:
		h.listIncidents(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// receiveIncident accepts an incident report from a cluster agent.
func (h *Handler) receiveIncident(w http.ResponseWriter, r *http.Request) {
	// Optional API key check
	if h.apiKey != "" && r.Header.Get("X-Claritty-Key") != h.apiKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract fields
	inc := &db.IncidentRow{}
	jsonStr(payload["id"], &inc.ID)
	jsonStr(payload["cluster_name"], &inc.Cluster)
	jsonStr(payload["severity"], &inc.Severity)
	jsonStr(payload["status"], &inc.Status)
	jsonStr(payload["title"], &inc.Title)
	jsonStr(payload["category"], &inc.Category)
	jsonStr(payload["root_cause"], &inc.RootCause)
	jsonStr(payload["llm_model"], &inc.LLMModel)
	jsonBool(payload["has_issue"], &inc.HasIssue)
	jsonInt(payload["confidence_score"], &inc.Confidence)
	jsonFloat(payload["scan_duration_secs"], &inc.ScanDurationSecs)
	jsonTime(payload["detected_at"], &inc.DetectedAt)

	inc.ContributingFactors = nullableJSON(payload["contributing_factors"])
	inc.AffectedServices = nullableJSON(payload["affected_services"])
	inc.AffectedNamespaces = nullableJSON(payload["affected_namespaces"])
	inc.RemediationPlan = nullableJSON(payload["remediation_plan"])

	// Derive namespace from affected_namespaces
	var nsList []string
	if err := json.Unmarshal(inc.AffectedNamespaces, &nsList); err == nil && len(nsList) > 0 {
		inc.Namespace = nsList[0]
	}

	// Default status
	if inc.Status == "" {
		inc.Status = "INVESTIGATING"
	}

	// Upsert cluster from snapshot if present
	if snap, ok := payload["cluster_snapshot"]; ok {
		var snapMap map[string]json.RawMessage
		if err := json.Unmarshal(snap, &snapMap); err == nil {
			cr := &db.ClusterRow{Name: inc.Cluster}
			jsonStr(snapMap["cluster_name"], &cr.Name)
			jsonFloat(snapMap["health_score"], &cr.HealthScore)
			jsonInt(snapMap["total_nodes"], &cr.TotalNodes)
			jsonInt(snapMap["ready_nodes"], &cr.ReadyNodes)
			jsonInt(snapMap["running_pods"], &cr.RunningPods)
			jsonInt(snapMap["pending_pods"], &cr.PendingPods)
			jsonInt(snapMap["failed_pods"], &cr.FailedPods)
			jsonInt(snapMap["crashloop_pods"], &cr.Crashloop)
			var ns []string
			if err := json.Unmarshal(nullableJSON(snapMap["namespaces"]), &ns); err == nil {
				cr.Namespaces = ns
			}
			cr.LastSeen = inc.DetectedAt
			_ = h.store.UpsertCluster(cr)
		}
	} else {
		// Ensure cluster exists
		_ = h.store.UpsertCluster(&db.ClusterRow{
			Name:     inc.Cluster,
			LastSeen: inc.DetectedAt,
		})
	}

	if err := h.store.InsertIncident(inc); err != nil {
		http.Error(w, fmt.Sprintf("store error: %v", err), http.StatusInternalServerError)
		return
	}

	// Slack alert for SEV1/SEV2
	if inc.HasIssue && (inc.Severity == "SEV1" || inc.Severity == "SEV2") {
		h.slack.AlertIncident(slack.IncidentPayload{
			ID:        inc.ID,
			Cluster:   inc.Cluster,
			Severity:  inc.Severity,
			Title:     inc.Title,
			Namespace: inc.Namespace,
			RootCause: inc.RootCause,
			HubURL:    h.hubBaseURL,
		})
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"status":"accepted","id":"%s"}`, inc.ID)
}

func (h *Handler) listIncidents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := db.ListIncidentsFilter{
		Cluster:   q.Get("cluster"),
		Namespace: q.Get("namespace"),
		Severity:  q.Get("severity"),
		Status:    q.Get("status"),
	}
	incidents, err := h.store.ListIncidents(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if incidents == nil {
		incidents = []db.IncidentRow{}
	}
	json.NewEncoder(w).Encode(incidents)
}

// handleIncidentByID handles GET /api/v1/incidents/:id and PATCH /api/v1/incidents/:id/status
func (h *Handler) handleIncidentByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/incidents/")
	parts := strings.Split(path, "/")
	id := parts[0]

	if len(parts) == 2 && parts[1] == "status" && r.Method == http.MethodPatch {
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateIncidentStatus(id, body.Status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, `{"status":"updated"}`)
		return
	}

	inc, err := h.store.GetIncident(id)
	if err != nil || inc == nil {
		http.Error(w, "incident not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(inc)
}

// ─── Clusters ─────────────────────────────────────────────────────────────────

func (h *Handler) handleClusters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	clusters, err := h.store.ListClusters()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if clusters == nil {
		clusters = []db.ClusterRow{}
	}
	json.NewEncoder(w).Encode(clusters)
}

func (h *Handler) handleClusterDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/clusters/"), "/")
	clusterName := parts[0]

	// Return incidents for this cluster
	if len(parts) == 2 && parts[1] == "incidents" {
		incidents, err := h.store.ListIncidents(db.ListIncidentsFilter{Cluster: clusterName})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if incidents == nil {
			incidents = []db.IncidentRow{}
		}
		json.NewEncoder(w).Encode(incidents)
		return
	}

	// Get cluster detail
	clusters, err := h.store.ListClusters()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, c := range clusters {
		if c.Name == clusterName {
			json.NewEncoder(w).Encode(c)
			return
		}
	}
	http.Error(w, "cluster not found", http.StatusNotFound)
}

// ─── Stats ────────────────────────────────────────────────────────────────────

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	stats, err := h.store.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func jsonStr(raw json.RawMessage, dst *string) {
	if raw == nil {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

func jsonBool(raw json.RawMessage, dst *bool) {
	if raw == nil {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

func jsonInt(raw json.RawMessage, dst *int) {
	if raw == nil {
		return
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		*dst = int(f)
	}
}

func jsonFloat(raw json.RawMessage, dst *float64) {
	if raw == nil {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

func jsonTime(raw json.RawMessage, dst *time.Time) {
	if raw == nil {
		return
	}
	_ = json.Unmarshal(raw, dst)
}

func nullableJSON(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return json.RawMessage("null")
	}
	return raw
}
