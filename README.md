# gcpalert2slack

Pub/Sub push 経由で Cloud Monitoring 通知を受け取るアプリケーションです。

## 環境変数

- `PORT`: Cloud Run またはローカル実行時に待ち受ける HTTP ポート。デフォルトは `8080`
- `SLACK_BOT_TOKEN`: Slack Bot User OAuth Token
- `SLACK_CHANNEL_ID`: 通知を投稿する固定チャンネル ID

## ローカル起動

まず `.env.example` を参考に `.env` を作成します。

```env
PORT=8080
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_CHANNEL_ID=C0123456789
```

`.env` は起動時に自動で読み込まれます。

```bash
go run ./cmd/gcpalert2slack
```

`POST /` に Pub/Sub push 形式のリクエストを送ります。

リクエスト例:

```json
{
  "message": {
    "data": "eyJwb2xpY3lfbmFtZSI6ImNwdSBoaWdoIiwic3RhdGUiOiJvcGVuIn0="
  }
}
```

サーバーは `message.data` を base64 decode し、中に入っている Cloud Monitoring 通知 JSON を読み取ります。`state` は大文字に正規化され、正常な payload なら `204`、不正な payload なら `400` を返します。

decode に成功すると、通知内容を Slack の固定チャンネルへ投稿します。Slack 投稿に失敗した場合は `500` を返します。

## Pub/Sub エミュレータでの integration test

まずエミュレータを起動します。

```bash
gcloud beta emulators pubsub start --project=test-project
```

別ターミナルでエミュレータ接続用の環境変数を設定します。

```bash
$(gcloud beta emulators pubsub env-init)
```

その後、integration test を実行します。

```bash
go test ./integration -run TestPubSubPushWithEmulator -v
```

## Slack への実送信テスト

本物の Slack API に対して通知を 1 件投稿する live integration test も用意しています。

このテストは実際のチャンネルにメッセージを投稿するため、通常の `go test ./...` では自動実行されません。
明示的に `SLACK_LIVE_TEST=1` を付けた時だけ動きます。

```bash
SLACK_LIVE_TEST=1 go test ./integration -run TestSlackLive_PostNotification -v
```

前提:

- `.env` に `SLACK_BOT_TOKEN` と `SLACK_CHANNEL_ID` が入っていること
- integration test 実行時は repo ルートの `.env` を自動で読む
- Bot が投稿先チャンネルに参加していること
- Bot Token に `chat:write` 権限があること
