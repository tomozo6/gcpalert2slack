package slacknotify

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	attachment := slack.Attachment{
		Color:  statusColor(item.State, item.Severity),
		Blocks: slack.Blocks{BlockSet: blocks},
	}

	_, _, err := c.api.PostMessageContext(
		ctx,
		c.channelID,
		slack.MsgOptionText(text, false),
		slack.MsgOptionAttachments(attachment),
	)
	if err != nil {
		return fmt.Errorf("post slack message: %w", err)
	}

	return nil
}

// BuildBlocks は通知内容を Slack Block Kit に変換する。
func BuildBlocks(item notification.MonitoringNotification) []slack.Block {
	var blocks []slack.Block

	// 1. Header (Plain text only, doesn't support Markdown formatting)
	headerText := fmt.Sprintf("%s: %s", stateBadge(item.State), fallbackText(item.PolicyName, "Cloud Monitoring Notification"))
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, false, false),
	))

	// 2. Context Block (Project and Event Time)
	timeStr := "N/A"
	if item.StartedAt > 0 {
		timeStr = time.Unix(item.StartedAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}
	contextText := fmt.Sprintf("📂 *Project:* %s  |  📅 *Time:* %s", fallbackText(item.ScopingProjectID, "unknown"), timeStr)
	blocks = append(blocks, slack.NewContextBlock(
		"",
		slack.NewTextBlockObject(slack.MarkdownType, contextText, false, false),
	))

	blocks = append(blocks, slack.NewDividerBlock())

	// 3. Section Block with two-column key-value fields
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*State:*\n%s", stateBadge(item.State)), false, false),
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Severity:*\n%s", severityBadge(item.Severity)), false, false),
	}

	if item.ObservedValue != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Observed Value:*\n`%s`", item.ObservedValue), false, false))
	}
	if item.ThresholdValue != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Threshold:*\n`%s`", item.ThresholdValue), false, false))
	}

	resourceType := fallbackText(item.ResourceTypeDisplayName, item.Resource.Type)
	if resourceType != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Resource Type:*\n%s", resourceType), false, false))
	}
	if item.IncidentID != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Incident ID:*\n`%s`", item.IncidentID), false, false))
	}

	if len(fields) > 0 {
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
		blocks = append(blocks, slack.NewDividerBlock())
	}

	// 4. Summary / Description (if present)
	if strings.TrimSpace(item.Summary) != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Summary:*\n%s", item.Summary), false, false),
			nil,
			nil,
		))
		blocks = append(blocks, slack.NewDividerBlock())
	}

	// 5. Documentation Block
	doc := strings.TrimSpace(item.Documentation.Content)
	if doc == "" {
		doc = "_No documentation provided._"
	}
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Documentation:*\n%s", doc), false, false),
		nil,
		nil,
	))

	// 6. Action Link (if URL is present)
	if strings.TrimSpace(item.URL) != "" {
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("🔗 *<%s|Open Incident in Google Cloud Console>*", item.URL), false, false),
			nil,
			nil,
		))
	}

	return blocks
}

func stateBadge(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "CLOSED":
		return "✅ CLOSED"
	default:
		return "🚨 OPEN"
	}
}

func severityBadge(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "CRITICAL":
		return "🔴 CRITICAL"
	case "WARNING":
		return "⚠️ WARNING"
	case "INFO":
		return "ℹ️ INFO"
	default:
		return "🔔 " + fallbackText(severity, "UNKNOWN")
	}
}

func statusColor(state, severity string) string {
	if strings.ToUpper(strings.TrimSpace(state)) == "CLOSED" {
		return "#2EB67D" // Green
	}

	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "CRITICAL":
		return "#E01E5A" // Red
	case "WARNING":
		return "#ECB22E" // Yellow
	case "INFO":
		return "#36C5F0" // Blue
	default:
		return "#1D9BD1" // Default Blue
	}
}

func fallbackText(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}
