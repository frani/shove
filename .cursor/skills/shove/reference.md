# Shove API reference

Default base URL: `http://localhost:8322`

## APNS

```bash
curl -i --data '{
  "headers": {"apns-priority": 10, "apns-topic": "com.shove.app"},
  "payload": {"aps": {"alert": "hi"}},
  "token": "81b8ecff8cb6d22154404d43b9aeaaf6219dfbef2abb2fe313f3725f4505cb47"
}' http://localhost:8322/api/push/apns
```

## FCM

```bash
curl -i --data '{
  "message": {
    "notification": {"body": "Hello world!", "title": "Test"},
    "token": "c7VmdNNHQaGTLkmi....15CmMs"
  }
}' http://localhost:8322/api/push/fcm
```

## Webhook

Raw body:

```bash
curl -i --data '{
  "url": "http://localhost:8000/api/webhook",
  "headers": {"foo": "bar"},
  "body": "Hello world!"
}' http://localhost:8322/api/push/webhook
```

JSON body via `data`:

```bash
curl -i --data '{
  "url": "http://localhost:8000/api/webhook",
  "headers": {"foo": "bar"},
  "data": {"hello": "world!"}
}' http://localhost:8322/api/push/webhook
```

## WebPush

```bash
curl -i --data '{
  "subscription": {
    "endpoint": "https://updates.push.services.mozilla.com/wpush/v2/...",
    "keys": {"auth": "...", "p256dh": "..."}
  },
  "headers": {"ttl": 3600, "urgency": "high"},
  "token": "use-this-for-feedback-instead-of-subscription",
  "payload": {"hello": "world"}
}' http://localhost:8322/api/push/webpush
```

Without `token`, the serialized subscription is used for feedback.

## Telegram

```bash
curl -i --data '{
  "method": "sendMessage",
  "payload": {"chat_id": "12345678", "text": "Hello!"}
}' http://localhost:8322/api/push/telegram
```

`chat_id` must be a string. Unreachable chats appear in feedback with that chat ID as the token.

## Email

Example shape (see also `scripts/email.json`):

```json
{
  "digest": {"subject": "Hello Digest"},
  "subject": "Hello world!",
  "from": "jane@doe.org",
  "to": ["john@doe.org"],
  "text": "Hello World!\n",
  "html": "<p>Hello <b>world</b>!</p>",
  "attachments": []
}
```

```bash
curl -i -X POST --data @./scripts/email.json http://localhost:8322/api/push/email
```

When `-email-rate-amount` / `-email-rate-per` are exceeded, messages are digested and sent later as one digest email.

## Feedback

```bash
curl -X POST http://localhost:8322/api/feedback
```

```json
{
  "feedback": [
    {
      "service": "apns-sandbox",
      "token": "881becff...",
      "reason": "invalid"
    }
  ]
}
```

## Redis enqueue (Go)

```go
package main

import (
	"encoding/json"
	"log"
	"os"

	"codeberg.org/pennersr/shove/pkg/shove"
)

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	client := shove.NewRedisClient(redisURL)

	raw, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"token": "device-token",
			"notification": map[string]string{
				"title": "Test",
				"body":  "Hello",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := client.PushRaw("fcm", raw); err != nil {
		log.Fatal(err)
	}
}
```

List names: `shove:<serviceID>` (e.g. `shove:fcm`).

## Server flags (common)

| Flag | Purpose |
|------|---------|
| `-api-addr` | Listen address (default `:8322`) |
| `-queue-redis` | Redis URL; omit for in-memory |
| `-fcm-credentials-file` | FCM service account JSON |
| `-apns-certificate-path` | APNS production PEM |
| `-apns-sandbox-certificate-path` | APNS sandbox PEM |
| `-webpush-vapid-keys-file` | VAPID keys JSON |
| `-telegram-bot-token` | Telegram bot token |
| `-email-host` / `-email-port` | SMTP |
| `-email-rate-amount` / `-email-rate-per` | Email rate limit |
| `*-workers` | Worker count per service |
