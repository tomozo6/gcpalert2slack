package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/tomozo6/gcpalert2slack/internal/notification"
	"github.com/tomozo6/gcpalert2slack/internal/slacknotify"
)

// TestSlackLive_PostNotification は、本物の Slack API に対して
// 通知を 1 件投稿できるかを確認する live integration test。
// 実際のチャンネルにメッセージが投稿されるため、明示的に有効化した時だけ実行する。
func TestSlackLive_PostNotification(t *testing.T) {
	t.Parallel()

	if os.Getenv("SLACK_LIVE_TEST") != "1" {
		t.Skip("SLACK_LIVE_TEST is not enabled")
	}

	loadDotEnvForIntegrationTest()

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	channelID := os.Getenv("SLACK_CHANNEL_ID")
	if botToken == "" || channelID == "" {
		t.Fatal("SLACK_BOT_TOKEN and SLACK_CHANNEL_ID are required when SLACK_LIVE_TEST=1")
	}

	client := slacknotify.NewClient(botToken, channelID)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	item := notification.MonitoringNotification{
		PolicyName:       "[TEST] gcpalert2slack Slack live test",
		State:            "OPEN",
		Severity:         "INFO",
		URL:              "https://example.com",
		ScopingProjectID: "local-test",
	}
	item.Documentation.Content = "This is a test message posted by TestSlackLive_PostNotification. You can safely ignore it."

	if err := client.PostNotification(ctx, item); err != nil {
		t.Fatalf("PostNotification() error = %v", err)
	}
}

func loadDotEnvForIntegrationTest() {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
	}

	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			return
		}
	}
}
