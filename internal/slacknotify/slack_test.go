package slacknotify

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/tomozo6/gcpalert2slack/internal/notification"
)

func TestBuildBlocks(t *testing.T) {
	t.Parallel()

	item := notification.MonitoringNotification{
		PolicyName:              "cpu high",
		State:                   "OPEN",
		Severity:                "CRITICAL",
		URL:                     "https://console.example.com",
		ScopingProjectID:        "demo",
		ScopingProjectNumber:    12345678,
		StartedAt:               1782799925,
		IncidentID:              "0.test_incident",
		ObservedValue:           "0.95",
		ThresholdValue:          "0.80",
		ResourceTypeDisplayName: "Compute Engine",
		Summary:                 "CPU latency is very high",
	}
	item.Documentation.Content = "Check server load"

	blocks := BuildBlocks(item)

	// Verify the blocks layout
	if len(blocks) == 0 {
		t.Fatal("expected non-empty blocks")
	}

	// 1st Block: Header
	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Fatalf("blocks[0] type = %T, want *slack.HeaderBlock", blocks[0])
	}
	if got := header.Text.Text; got != "🚨 OPEN: cpu high" {
		t.Fatalf("header text = %q, want %q", got, "🚨 OPEN: cpu high")
	}

	// 2nd Block: Context
	ctxBlock, ok := blocks[1].(*slack.ContextBlock)
	if !ok {
		t.Fatalf("blocks[1] type = %T, want *slack.ContextBlock", blocks[1])
	}
	if len(ctxBlock.ContextElements.Elements) == 0 {
		t.Fatal("expected context elements")
	}
	ctxText, ok := ctxBlock.ContextElements.Elements[0].(*slack.TextBlockObject)
	if !ok {
		t.Fatalf("context element type = %T, want *slack.TextBlockObject", ctxBlock.ContextElements.Elements[0])
	}
	expectedCtxSubstr := "📂 *Project:* demo"
	if !strings.Contains(ctxText.Text, expectedCtxSubstr) {
		t.Errorf("context text = %q, expected to contain %q", ctxText.Text, expectedCtxSubstr)
	}

	// Verify statusColor helper function
	if color := statusColor("OPEN", "CRITICAL"); color != "#E01E5A" {
		t.Errorf("statusColor(OPEN, CRITICAL) = %q, want #E01E5A", color)
	}
	if color := statusColor("CLOSED", "CRITICAL"); color != "#2EB67D" {
		t.Errorf("statusColor(CLOSED, CRITICAL) = %q, want #2EB67D", color)
	}
	if color := statusColor("OPEN", "WARNING"); color != "#ECB22E" {
		t.Errorf("statusColor(OPEN, WARNING) = %q, want #ECB22E", color)
	}
}
