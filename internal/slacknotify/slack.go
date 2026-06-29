package slacknotify

import (
	"context"
	"fmt"
	"strings"

	"github.com/slack-go/slack"
	"github.com/tomozo6/gcpalert2slack/internal/notification"
)

// Sender は通知を Slack に送る役割を表す。
type Sender interface {
	PostNotification(ctx context.Context, item notification.MonitoringNotification) error
}

type Client struct {
	api       *slack.Client
	channelID string
}

// NewClient は Slack 送信用クライアントを作る。
func NewClient(botToken, channelID string) *Client {
	return &Client{
		api:       slack.New(botToken),
		channelID: channelID,
	}
}

// PostNotification は通知を固定チャンネルへ投稿する。
func (c *Client) PostNotification(ctx context.Context, item notification.MonitoringNotification) error {
	blocks := BuildBlocks(item)
	text := fmt.Sprintf("%s %s", stateBadge(item.State), fallbackText(item.PolicyName, "Cloud Monitoring notification"))

	_, _, err := c.api.PostMessageContext(
		ctx,
		c.channelID,
		slack.MsgOptionText(text, false),
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return fmt.Errorf("post slack message: %w", err)
	}

	return nil
}

// BuildBlocks は通知内容を Slack Block Kit に変換する。
func BuildBlocks(item notification.MonitoringNotification) []slack.Block {
	doc := strings.TrimSpace(item.Documentation.Content)
	if doc == "" {
		doc = "_No documentation provided._"
	}

	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*State*\n%s", stateBadge(item.State)), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Severity*\n%s", fallbackText(item.Severity, "UNKNOWN")), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Project*\n%s", fallbackText(item.ScopingProjectID, "unknown")), false, false),
	}

	if strings.TrimSpace(item.URL) != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Incident*\n<%s|Open in Google Cloud>", item.URL), false, false))
	}

	return []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(
				slack.PlainTextType,
				fmt.Sprintf("%s %s", stateBadge(item.State), fallbackText(item.PolicyName, "Cloud Monitoring notification")),
				false,
				false,
			),
		),
		slack.NewSectionBlock(nil, fields, nil),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, doc, false, false),
			nil,
			nil,
		),
	}
}

func stateBadge(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "CLOSED":
		return "✅ CLOSED"
	default:
		return "🚨 OPEN"
	}
}

func fallbackText(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}
