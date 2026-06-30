package notification

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type PushRequest struct {
	Message PushMessage `json:"message"`
}

type PushMessage struct {
	Data string `json:"data"`
}

// WebhookPayload はCloud MonitoringからのWebhook全体の構造体です
type WebhookPayload struct {
	Incident Incident `json:"incident"`
	Version  string   `json:"version"` // 現在は "1.2" が入ります
}

// Incident はインシデントの詳細情報を格納します
type Incident struct {
	IncidentID              string        `json:"incident_id"`
	Renotify                bool          `json:"renotify,omitempty"`
	ScopingProjectID        string        `json:"scoping_project_id"`
	ScopingProjectNumber    int64         `json:"scoping_project_number,omitempty"`
	URL                     string        `json:"url"`
	StartedAt               int64         `json:"started_at"` // Unixエポック秒
	EndedAt                 int64         `json:"ended_at,omitempty"`
	State                   string        `json:"state"` // "open" または "closed"
	Summary                 string        `json:"summary"`
	ApigeeURL               string        `json:"apigee_url,omitempty"`
	ObservedValue           string        `json:"observed_value,omitempty"`
	ResourceTypeDisplayName string        `json:"resource_type_display_name,omitempty"`
	ResourceID              string        `json:"resource_id,omitempty"`
	ResourceDisplayName     string        `json:"resource_display_name,omitempty"`
	ResourceName            string        `json:"resource_name,omitempty"`
	ConditionName           string        `json:"condition_name"`
	PolicyName              string        `json:"policy_name"`
	Severity                string        `json:"severity,omitempty"`
	ThresholdValue          string        `json:"threshold_value,omitempty"`
	Resource                Resource      `json:"resource"`
	Metric                  Metric        `json:"metric"`
	Metadata                Metadata      `json:"metadata,omitempty"`
	Condition               Condition     `json:"condition,omitempty"`
	Documentation           Documentation `json:"documentation,omitempty"`
}

type Resource struct {
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
}

type Metric struct {
	Type        string            `json:"type"`
	DisplayName string            `json:"displayName,omitempty"`
	Labels      map[string]string `json:"labels"`
}

type Metadata struct {
	SystemLabels map[string]string `json:"system_labels,omitempty"`
	UserLabels   map[string]string `json:"user_labels,omitempty"`
}

type Condition struct {
	Name               string              `json:"name"`
	DisplayName        string              `json:"displayName"`
	ConditionThreshold *ConditionThreshold `json:"conditionThreshold,omitempty"`
}

type ConditionThreshold struct {
	Aggregations   []Aggregation `json:"aggregations,omitempty"`
	Comparison     string        `json:"comparison,omitempty"`
	Duration       string        `json:"duration,omitempty"`
	Filter         string        `json:"filter,omitempty"`
	ThresholdValue float64       `json:"thresholdValue,omitempty"`
	Trigger        *Trigger      `json:"trigger,omitempty"`
}

type Aggregation struct {
	AlignmentPeriod  string   `json:"alignmentPeriod,omitempty"`
	GroupByFields    []string `json:"groupByFields,omitempty"`
	PerSeriesAligner string   `json:"perSeriesAligner,omitempty"`
}

type Trigger struct {
	Count int `json:"count,omitempty"`
}

type Documentation struct {
	Content string `json:"content,omitempty"`
}

// MonitoringNotification は、以前のコードとの互換性のために
// Incident 構造体への別名（Type Alias）として定義します。
type MonitoringNotification = Incident

// DecodePushRequest は Pub/Sub push のリクエスト body を読み取り、
// message.data に入っている Cloud Monitoring 通知を取り出す。
func DecodePushRequest(body []byte) (MonitoringNotification, error) {
	var pushRequest PushRequest
	// まずは HTTP body 全体を Pub/Sub push の外側の JSON として読む。
	// この段階では、まだ incident 本体ではなく message.data しか取り出していない。
	if err := json.Unmarshal(body, &pushRequest); err != nil {
		return MonitoringNotification{}, fmt.Errorf("unmarshal push request: %w", err)
	}

	// Pub/Sub push では、実際の通知内容は message.data に base64 文字列として入る。
	// ここが空だと、この先に読むべき本体が存在しない。
	if pushRequest.Message.Data == "" {
		return MonitoringNotification{}, fmt.Errorf("message.data is required")
	}

	// message.data は JSON そのものではなく base64 で包まれているので、
	// 先にデコードして元の JSON bytes に戻す。
	decoded, err := base64.StdEncoding.DecodeString(pushRequest.Message.Data)
	if err != nil {
		return MonitoringNotification{}, fmt.Errorf("decode message.data: %w", err)
	}

	var payload WebhookPayload
	// base64 を外したあとの bytes を、今度は Cloud Monitoring 通知の JSON として読む。
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return MonitoringNotification{}, fmt.Errorf("unmarshal incident: %w", err)
	}

	item := payload.Incident

	// 通知によって大文字小文字が揺れても後続処理で扱いやすいように整える。
	item.State = NormalizeState(item.State)
	item.Severity = strings.ToUpper(strings.TrimSpace(item.Severity))

	return item, nil
}

// NormalizeState は state の前後空白を除去し、大文字にそろえる。
// 後続処理で大文字小文字の違いを気にせず比較するために使う。
func NormalizeState(state string) string {
	normalized := strings.ToUpper(strings.TrimSpace(state))
	switch normalized {
	case "OPEN", "CLOSED":
		return normalized
	default:
		return normalized
	}
}
