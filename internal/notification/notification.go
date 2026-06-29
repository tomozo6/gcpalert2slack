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

type MonitoringNotification struct {
	PolicyName       string `json:"policy_name"`
	State            string `json:"state"`
	Severity         string `json:"severity"`
	URL              string `json:"url"`
	ScopingProjectID string `json:"scoping_project_id"`
	Documentation    struct {
		Content string `json:"content"`
	} `json:"documentation"`
}

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

	var item MonitoringNotification
	// base64 を外したあとの bytes を、今度は Cloud Monitoring 通知の JSON として読む。
	if err := json.Unmarshal(decoded, &item); err != nil {
		return MonitoringNotification{}, fmt.Errorf("unmarshal incident: %w", err)
	}

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
