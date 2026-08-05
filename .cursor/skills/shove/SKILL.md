---
name: shove
description: >-
  Integrate with Shove, an async push-notification server (APNS, FCM, WebPush,
  Telegram, Webhook, Email). Use when sending pushes via Shove HTTP API or Redis
  queue, wiring feedback for invalid tokens, configuring -queue-redis vs
  in-memory, docker-compose shove, or when the user mentions shove, /api/push,
  or push notification workers.
---

# Shove

Async fire-and-forget push gateway. Clients enqueue; workers deliver upstream.

Module: `codeberg.org/pennersr/shove`  
Default API: `http://localhost:8322`

## Mental model

1. Client POSTs to `/api/push/<service>` **or** RPUSHes raw JSON to Redis list `shove:<service>`.
2. Shove returns `202 OK` (HTTP) and workers push asynchronously.
3. Invalid tokens surface via `POST /api/feedback` — poll and prune your DB.
4. Metrics: `GET /metrics` (Prometheus).

## Queue backends

| Mode | Flag | Persistence | Direct client enqueue |
|------|------|-------------|------------------------|
| In-memory | omit `-queue-redis` | Lost on restart | No — HTTP only |
| Redis | `-queue-redis redis://host:6379` | Yes | Yes — `shove.NewRedisClient` |

Redis lists: `shove:fcm`, `shove:apns`, `shove:webpush`, `shove:telegram`, `shove:webhook`, `shove:email`.

Prefer Redis when many microservices enqueue and Shove may restart independently.

## HTTP API

All push endpoints: `POST /api/push/<service>` → `202` + body `OK` on accept.

| Service | Path | Notes |
|---------|------|-------|
| APNS | `/api/push/apns` | `headers`, `payload.aps`, `token` |
| FCM | `/api/push/fcm` | FCM HTTP v1 `message` object |
| WebPush | `/api/push/webpush` | `subscription`, optional `token` for feedback |
| Telegram | `/api/push/telegram` | `chat_id` must be a **string** |
| Webhook | `/api/push/webhook` | `url` + `body` or `data` |
| Email | `/api/push/email` | See [reference.md](reference.md); supports digests under rate limit |

Feedback:

```bash
curl -X POST http://localhost:8322/api/feedback
# { "feedback": [{ "service": "apns", "token": "...", "reason": "invalid" }] }
```

Payload examples: [reference.md](reference.md).

## Go client (Redis direct)

```go
import "codeberg.org/pennersr/shove/pkg/shove"

client := shove.NewRedisClient(os.Getenv("REDIS_URL")) // default redis://localhost:6379
err := client.PushRaw("fcm", rawJSON) // service id = list suffix
```

Client must send the **same raw payload** the HTTP endpoint expects for that service.

## When integrating in another project

1. Decide HTTP vs Redis enqueue (HTTP if shove is always up; Redis for decoupling).
2. Use correct service id (`fcm`, `apns`, `webpush`, `telegram`, `webhook`, `email`).
3. Always implement feedback polling for token cleanup.
4. Telegram `chat_id` must be a **string**.
5. Mount credentials at runtime (FCM JSON, APNS PEMs, VAPID keys) — never bake into images.
6. Email/Telegram support rate-limit squashing/digests when configured.

## Local run

```bash
docker compose up --build   # API :8322, Redis queue
```

`compose.yaml` uses `-queue-redis redis://redis:6379`. Omit that flag for in-memory queue (no Redis).

## Do not confuse

- Redis is only the job queue, not a cache.
- `202` means queued, not upstream-delivered.
- Feedback is pull-based (`POST /api/feedback`), not a webhook to your app.
