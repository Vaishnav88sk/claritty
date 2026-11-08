package incident

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAgentChangeEngine_TimeWindowFiltering(t *testing.T) {
	refTime, _ := time.Parse("15:04", "14:05")
	window := 30 * time.Minute

	engine := NewChangeEngine(window, refTime)

	tInside, _ := time.Parse("15:04", "14:01")
	accepted1 := engine.AddChange(ChangeRecord{
		Timestamp: tInside,
		Resource:  "deployment/default/payment-api",
		Category:  ChangeCategoryDeployment,
		Summary:   "payment-api v1.43 deployed",
	})
	if !accepted1 {
		t.Errorf("Expected change at 14:01 to be accepted")
	}

	tOutside, _ := time.Parse("15:04", "13:10")
	accepted2 := engine.AddChange(ChangeRecord{
		Timestamp: tOutside,
		Resource:  "deployment/default/auth-service",
		Category:  ChangeCategoryDeployment,
		Summary:   "auth-service deployed",
	})
	if accepted2 {
		t.Errorf("Expected change at 13:10 to be rejected")
	}
}

func TestAgentChangeEngine_Correlation(t *testing.T) {
	refTime, _ := time.Parse("15:04", "14:05")
	engine := NewChangeEngine(60*time.Minute, refTime)

	t1, _ := time.Parse("15:04", "14:01")
	t2, _ := time.Parse("15:04", "14:02")

	engine.AddChange(ChangeRecord{
		Timestamp: t1,
		Resource:  "deployment/default/payment-api",
		Namespace: "default",
		Category:  ChangeCategoryDeployment,
		Summary:   "payment-api v1.43 deployed",
	})

	engine.AddChange(ChangeRecord{
		Timestamp: t2,
		Resource:  "service/redis/redis-svc",
		Namespace: "redis",
		Category:  ChangeCategoryKubernetes,
		Summary:   "Redis Service modified",
	})

	sorted := engine.Correlate([]string{"pod/default/payment-api-abc"}, []string{"default"})
	if len(sorted) != 2 {
		t.Fatalf("Expected 2 changes, got %d", len(sorted))
	}

	top := engine.MostRelevant()
	if top == nil || !strings.Contains(top.Summary, "payment-api v1.43") {
		t.Errorf("Expected payment-api v1.43 as top relevant change")
	}
}

func TestAgentChangeRecord_JSON(t *testing.T) {
	rec := ChangeRecord{
		ID:         "CHG-1",
		Resource:   "deployment/default/payment-api",
		Category:   ChangeCategoryDeployment,
		Summary:    "payment-api v1.43 deployed",
		Relevance:  0.95,
		Correlated: true,
	}

	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var dec ChangeRecord
	if err := json.Unmarshal(b, &dec); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if dec.ID != rec.ID || !dec.Correlated {
		t.Errorf("Decoded mismatch: %+v", dec)
	}
}
