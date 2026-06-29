package notification

import (
	"encoding/base64"
	"testing"
)

func TestDecodePushRequest(t *testing.T) {
	t.Parallel()

	payload := `{"policy_name":"cpu high","state":"open","severity":"critical","url":"https://example.com","scoping_project_id":"demo","documentation":{"content":"hello"}}`
	body := `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(payload)) + `"}}`

	notification, err := DecodePushRequest([]byte(body))
	if err != nil {
		t.Fatalf("DecodePushRequest() error = %v", err)
	}

	if notification.State != "OPEN" {
		t.Fatalf("state = %q, want OPEN", notification.State)
	}
	if notification.Severity != "CRITICAL" {
		t.Fatalf("severity = %q, want CRITICAL", notification.Severity)
	}
	if notification.Documentation.Content != "hello" {
		t.Fatalf("documentation = %q, want hello", notification.Documentation.Content)
	}
}

func TestDecodePushRequestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{"message":`},
		{name: "missing data", body: `{"message":{"data":""}}`},
		{name: "invalid base64", body: `{"message":{"data":"***"}}`},
		{name: "invalid incident json", body: `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(`{`)) + `"}}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := DecodePushRequest([]byte(tt.body)); err == nil {
				t.Fatal("DecodePushRequest() error = nil, want error")
			}
		})
	}
}

func TestNormalizeState(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"open":     "OPEN",
		"OPEN":     "OPEN",
		" closed ": "CLOSED",
		"unknown":  "UNKNOWN",
	}

	for input, want := range tests {
		if got := NormalizeState(input); got != want {
			t.Fatalf("NormalizeState(%q) = %q, want %q", input, got, want)
		}
	}
}
