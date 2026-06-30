package server

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tomozo6/gcpalert2slack/internal/notification"
)

type stubSender struct {
	err      error
	received []notification.MonitoringNotification
}

func (s *stubSender) PostNotification(_ context.Context, item notification.MonitoringNotification) error {
	if s.err != nil {
		return s.err
	}

	s.received = append(s.received, item)
	return nil
}

func TestHandlerSuccess(t *testing.T) {
	t.Parallel()

	sender := &stubSender{}
	handler := NewHandler(sender)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validPushBody()))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if len(sender.received) != 1 {
		t.Fatalf("len(sender.received) = %d, want 1", len(sender.received))
	}
}

func TestHandlerInvalidPayload(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&stubSender{})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"message":{"data":"***"}}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlerSlackFailure(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&stubSender{err: errors.New("slack down")})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validPushBody()))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandlerMethodAndNotFound(t *testing.T) {
	t.Parallel()

	handler := NewHandler(&stubSender{})

	methodReq := httptest.NewRequest(http.MethodGet, "/", nil)
	methodRec := httptest.NewRecorder()
	handler.ServeHTTP(methodRec, methodReq)
	if methodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET / status = %d, want %d", methodRec.Code, http.StatusMethodNotAllowed)
	}

	notFoundReq := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	notFoundRec := httptest.NewRecorder()
	handler.ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("POST /healthz status = %d, want %d", notFoundRec.Code, http.StatusNotFound)
	}
}

func validPushBody() string {
	payload := `{"incident":{"policy_name":"cpu high","state":"open","severity":"critical","url":"https://example.com","scoping_project_id":"demo","documentation":{"content":"hello"}}}`
	return `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(payload)) + `"}}`
}
