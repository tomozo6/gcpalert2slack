package server

import (
	"io"
	"log"
	"net/http"

	"github.com/tomozo6/gcpalert2slack/internal/notification"
	"github.com/tomozo6/gcpalert2slack/internal/slacknotify"
)

type Handler struct {
	sender slacknotify.Sender
}

// NewHandler は Pub/Sub push を受け取る HTTP handler を作る。
func NewHandler(sender slacknotify.Sender) *Handler {
	return &Handler{sender: sender}
}

// ServeHTTP は POST / だけを受け付け、Pub/Sub payload を decode する。
// 通知 body が正しければ 204 を返す。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	log.Printf("read request body success. body: %s", string(body))

	item, err := notification.DecodePushRequest(body)
	if err != nil {
		http.Error(w, "invalid pubsub push payload", http.StatusBadRequest)
		return
	}

	log.Printf("received monitoring notification policy=%q state=%q severity=%q project=%q", item.PolicyName, item.State, item.Severity, item.ScopingProjectID)

	if err := h.sender.PostNotification(r.Context(), item); err != nil {
		log.Printf("post notification to slack: %v", err)
		http.Error(w, "failed to notify slack", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
