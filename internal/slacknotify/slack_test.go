package slacknotify

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/tomozo6/gcpalert2slack/internal/notification"
)

func TestBuildBlocks(t *testing.T) {
	t.Parallel()

	item := notification.MonitoringNotification{
		PolicyName:       "cpu high",
		State:            "OPEN",
		Severity:         "CRITICAL",
		URL:              "https://console.example.com",
		ScopingProjectID: "demo",
	}

	blocks := BuildBlocks(item)
	if len(blocks) != 3 {
		t.Fatalf("len(blocks) = %d, want 3", len(blocks))
	}

	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Fatalf("blocks[0] type = %T, want *slack.HeaderBlock", blocks[0])
	}
	if got := header.Text.Text; got != "🚨 OPEN cpu high" {
		t.Fatalf("header text = %q, want %q", got, "🚨 OPEN cpu high")
	}

	doc, ok := blocks[2].(*slack.SectionBlock)
	if !ok {
		t.Fatalf("blocks[2] type = %T, want *slack.SectionBlock", blocks[2])
	}
	if doc.Text == nil || doc.Text.Text != "_No documentation provided._" {
		t.Fatalf("documentation text = %#v, want fallback", doc.Text)
	}
}
