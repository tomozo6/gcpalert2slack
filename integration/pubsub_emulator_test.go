package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/tomozo6/gcpalert2slack/internal/notification"
	"github.com/tomozo6/gcpalert2slack/internal/server"
)

type testCase struct {
	name  string
	input notification.MonitoringNotification
	want  notification.MonitoringNotification
}

type pushResult struct {
	status              int
	decodedNotification notification.MonitoringNotification
}

// TestPubSubPushWithEmulator_OpenNotificationIsDeliveredAndNormalized は、
// open 状態の通知を Pub/Sub エミュレータ経由で送ったときに、
// アプリまで push で届き、正規化後の値として decode されるかを確認する。
func TestPubSubPushWithEmulator_OpenNotificationIsDeliveredAndNormalized(t *testing.T) {
	t.Parallel()

	tc := testCase{
		name: "open notification is delivered and normalized",
		input: notification.MonitoringNotification{
			PolicyName:       "cpu high",
			State:            "open",
			Severity:         "critical",
			ScopingProjectID: "demo",
		},
		want: notification.MonitoringNotification{
			PolicyName:       "cpu high",
			State:            "OPEN",
			Severity:         "CRITICAL",
			ScopingProjectID: "demo",
		},
	}

	// このテストは Pub/Sub エミュレータから push subscription 経由で
	// HTTP handler に通知が届き、handler が 204 を返せることを確認する。
	// あわせて、decode 後の通知内容が期待どおりかも検証する。
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		t.Skip("PUBSUB_EMULATOR_HOST is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// push された HTTP リクエストを 1 回だけ受け取り、
	// そのときの decode 結果と HTTP ステータスを結果チャネルに返す。
	results := make(chan pushResult, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := cloneRequestBody(r)
		if err != nil {
			t.Errorf("cloneRequestBody() error = %v", err)
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}

		recorder := newStatusRecorder(w)
		server.NewHandler(&stubSender{}).ServeHTTP(recorder, r)

		select {
		case results <- pushResult{
			status:              recorder.status,
			decodedNotification: mustDecodeNotificationForTest(t, body),
		}:
		default:
		}
	})

	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	// エミュレータに対して topic / push subscription を作る。
	// push 先は httptest で起動したローカル HTTP サーバーに向ける。
	projectID := "test-project"
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("pubsub.NewClient() error = %v", err)
	}
	defer client.Close()

	suffix := time.Now().UnixNano()
	topicID := fmt.Sprintf("topic-%d", suffix)
	subscriptionID := fmt.Sprintf("sub-%d", suffix)

	topic, err := client.CreateTopic(ctx, topicID)
	if err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}
	defer topic.Stop()

	subscription, err := client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{
		Topic: topic,
		PushConfig: pubsub.PushConfig{
			Endpoint: httpServer.URL,
		},
		AckDeadline: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}
	defer subscription.Delete(ctx)

	// 送信する入力データは testCase にまとめておき、
	// Pub/Sub に渡す直前だけ JSON に変換する。
	payload, err := marshalNotification(tc.input)
	if err != nil {
		t.Fatalf("marshalNotification() error = %v", err)
	}

	result := topic.Publish(ctx, &pubsub.Message{
		Data: payload,
	})
	if _, err := result.Get(ctx); err != nil {
		t.Fatalf("Publish().Get() error = %v", err)
	}

	// push 配送が終わると、HTTP ステータスと decode 結果が results に入る。
	// 期限までに来なければ integration 失敗とみなす。
	var got pushResult
	select {
	case got = <-results:
	case <-ctx.Done():
		t.Fatal("timed out waiting for push delivery from emulator")
	}

	if got.status != http.StatusNoContent {
		t.Fatalf("%s: status = %d, want %d", tc.name, got.status, http.StatusNoContent)
	}
	if got.decodedNotification != tc.want {
		t.Fatalf("%s: decodedNotification = %#v, want %#v", tc.name, got.decodedNotification, tc.want)
	}
}

// statusRecorder は、handler が返した HTTP ステータスコードを覚えておくための補助型。
// このテストでは「push された通知をアプリが正常に受け付けて 204 を返したか」を見るために使う。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// newStatusRecorder は statusRecorder を作る関数。
// まだ何も返していない状態では 200 を初期値にしておき、
// handler が実際にどのステータスを返したかをあとで確認できるようにする。
func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

// WriteHeader は、handler が返したステータスコードを記録しつつ、
// その内容を本来の ResponseWriter にもそのまま渡す。
// これにより「テスト用の記録」と「実際の HTTP 応答」の両方を成立させる。
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// cloneRequestBody は、HTTP body を読み出したうえで、
// アプリ本体が同じ内容をもう一度読めるように詰め直す関数。
// HTTP の body は通常 1 回読むと空になるため、
// テスト側で先読みしたあとでもアプリ本体が同じ内容を読めるようにしている。
func cloneRequestBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// mustDecodeNotificationForTest は、受け取った body を decode して返す。
// integration test では decode 失敗はその時点で異常なので、テストを失敗させる。
func mustDecodeNotificationForTest(t *testing.T, body []byte) notification.MonitoringNotification {
	t.Helper()

	item, err := notification.DecodePushRequest(body)
	if err != nil {
		t.Fatalf("DecodePushRequest() error = %v", err)
	}

	return item
}

// marshalNotification は、testCase に書いた入力データを
// Pub/Sub に publish できる JSON bytes に変換する関数。
func marshalNotification(input notification.MonitoringNotification) ([]byte, error) {
	return json.Marshal(input)
}

type stubSender struct{}

func (s *stubSender) PostNotification(_ context.Context, _ notification.MonitoringNotification) error {
	return nil
}
