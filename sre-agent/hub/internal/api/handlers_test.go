package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/db"
	"github.com/Vaishnav88sk/claritty/sre-agent/hub/internal/slack"
)

func setupTestApp(t *testing.T) (*http.ServeMux, sqlmock.Sqlmock) {
	// Create sqlmock database connection
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	// Create store using the mock DB
	store := db.NewWithDB(mockDB)

	// Create a dummy slack client
	slackClient := slack.New("dummy-token", "dummy-channel")

	// Create handler and register routes
	handler := New(store, slackClient, "http://dummy-hub", "secret-key")
	mux := http.ServeMux{}
	handler.RegisterRoutes(&mux)

	return &mux, mock
}

func TestReceiveIncident_Valid(t *testing.T) {
	mux, mock := setupTestApp(t)

	// Mock the DB expectations for UpsertCluster
	mock.ExpectExec(`INSERT INTO clusters`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock the DB expectations for InsertIncident (including cluster upsert check)
	mock.ExpectExec(`INSERT INTO clusters`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO incidents`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	payload := map[string]interface{}{
		"id":               "inc-123",
		"cluster_name":     "test-cluster",
		"severity":         "SEV1",
		"title":            "CrashLoopBackOff in DB",
		"llm_model":        "gpt-4",
		"has_issue":        true,
		"confidence_score": 95,
		"detected_at":      time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewBuffer(body))
	req.Header.Set("X-Claritty-Key", "secret-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled DB expectations: %s", err)
	}
}

func TestReceiveIncident_InvalidJSON(t *testing.T) {
	mux, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewBufferString("{invalid-json}"))
	req.Header.Set("X-Claritty-Key", "secret-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestReceiveIncident_Unauthorized(t *testing.T) {
	mux, _ := setupTestApp(t)

	payload := map[string]interface{}{"id": "inc-123"}
	body, _ := json.Marshal(payload)
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewBuffer(body))
	// NOT setting X-Claritty-Key
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGetStats(t *testing.T) {
	mux, mock := setupTestApp(t)

	// Mock DB queries in store.GetStats()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM incidents$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM incidents WHERE status='INVESTIGATING'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	mock.ExpectQuery(`SELECT COALESCE`).
		WillReturnRows(sqlmock.NewRows([]string{"mttr"}).AddRow(3600.5))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["total_incidents"] != float64(10) {
		t.Errorf("Expected total_incidents=10, got %v", response["total_incidents"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled DB expectations: %s", err)
	}
}
