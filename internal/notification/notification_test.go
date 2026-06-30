package notification

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"testing"
)

//go:embed notification_test.json
var testIncidentJSON []byte

func TestDecodePushRequest_Success_WithRealProductionLog(t *testing.T) {
	t.Parallel()

	pushBody := map[string]interface{}{
		"message": map[string]interface{}{
			"data": base64.StdEncoding.EncodeToString(testIncidentJSON),
		},
	}

	bodyBytes, err := json.Marshal(pushBody)
	if err != nil {
		t.Fatalf("failed to marshal push body: %v", err)
	}

	got, err := DecodePushRequest(bodyBytes)
	if err != nil {
		t.Fatalf("DecodePushRequest failed: %v", err)
	}

	// 1. Primitive fields
	wantPolicy := "重要 endpoint の応答が遅くなっています"
	if got.PolicyName != wantPolicy {
		t.Errorf("PolicyName = %q, want %q", got.PolicyName, wantPolicy)
	}

	if got.State != "OPEN" {
		t.Errorf("State = %q, want OPEN", got.State)
	}

	if got.Severity != "WARNING" {
		t.Errorf("Severity = %q, want WARNING", got.Severity)
	}

	if got.ScopingProjectID != "your-gcp-project-id" {
		t.Errorf("ScopingProjectID = %q, want your-gcp-project-id", got.ScopingProjectID)
	}

	if got.ScopingProjectNumber != 123456789012 {
		t.Errorf("ScopingProjectNumber = %d, want 123456789012", got.ScopingProjectNumber)
	}

	if got.StartedAt != 1782799925 {
		t.Errorf("StartedAt = %d, want 1782799925", got.StartedAt)
	}

	if got.IncidentID != "0.dummy_incident_id" {
		t.Errorf("IncidentID = %q, want 0.dummy_incident_id", got.IncidentID)
	}

	// 2. Resource structure assertions
	if got.Resource.Type != "consumed_api" {
		t.Errorf("Resource.Type = %q, want consumed_api", got.Resource.Type)
	}
	if got.Resource.Labels["project_id"] != "your-gcp-project-id" {
		t.Errorf("Resource.Labels[project_id] = %q, want your-gcp-project-id", got.Resource.Labels["project_id"])
	}

	// 3. Metric structure assertions
	if got.Metric.Type != "serviceruntime.googleapis.com/api/request_latencies" {
		t.Errorf("Metric.Type = %q, want serviceruntime.googleapis.com/api/request_latencies", got.Metric.Type)
	}

	// 4. Condition structure assertions
	if got.Condition.DisplayName != "重要 endpoint の応答が遅くなっています" {
		t.Errorf("Condition.DisplayName = %q, want ...", got.Condition.DisplayName)
	}
	if got.Condition.ConditionThreshold == nil {
		t.Fatal("Condition.ConditionThreshold is nil")
	}
	if got.Condition.ConditionThreshold.ThresholdValue != 0.75 {
		t.Errorf("ConditionThreshold.ThresholdValue = %f, want 0.75", got.Condition.ConditionThreshold.ThresholdValue)
	}
	if len(got.Condition.ConditionThreshold.Aggregations) != 1 {
		t.Fatalf("len(Aggregations) = %d, want 1", len(got.Condition.ConditionThreshold.Aggregations))
	}
	if got.Condition.ConditionThreshold.Aggregations[0].PerSeriesAligner != "ALIGN_PERCENTILE_95" {
		t.Errorf("PerSeriesAligner = %q, want ALIGN_PERCENTILE_95", got.Condition.ConditionThreshold.Aggregations[0].PerSeriesAligner)
	}

	// 5. Documentation structure assertions
	if got.Documentation.Content == "" {
		t.Error("Documentation.Content is empty")
	}
}

func TestDecodePushRequest_InvalidBase64(t *testing.T) {
	t.Parallel()

	body := []byte(`{"message":{"data":"invalid-base64!"}}`)
	_, err := DecodePushRequest(body)
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestDecodePushRequest_MissingData(t *testing.T) {
	t.Parallel()

	body := []byte(`{"message":{}}`)
	_, err := DecodePushRequest(body)
	if err == nil {
		t.Error("expected error for missing message.data, got nil")
	}
}
