# svc watch — Design Document

**Version:** v0.2 design  
**Status:** Pre-implementation  
**Last updated:** 2026-03-18

---

## What it does

`svc watch` polls every service in the manifest on a configurable interval. When a service transitions from up→down or down→up, it fires a webhook POST. It runs until killed.

That's it. No dashboard. No history. No built-in alerting channels. Small tool, composable.

---

## Core decisions

### Polling interval: 60 seconds default

30 seconds is too noisy — transient timeouts generate false alerts. 5 minutes is Observatory's interval and is too slow for a watch command. 60 seconds means you know within 1 minute of a real failure, with one retry built in (see consecutive failures below). Configurable via `--interval <seconds>`.

### Consecutive failures before alerting: 2

One failure could be a network hiccup, a slow health endpoint, or a momentary restart. Two consecutive failures (2 minutes at default interval) means the service is genuinely down. Configurable via `--failures <n>`. The alert fires on the Nth consecutive failure, not after — so with `--failures 2`, alert fires at minute 2, not minute 3.

### Recovery notification: yes, always

When a service comes back up after being down, fire a recovery webhook. The alert without the recovery means you never know if your response worked. Recovery is not configurable — it's always on. A tool that tells you about problems but not resolutions is half a tool.

### State file: `~/.local/share/svc/watch-state.json`

`svc watch` is stateless across restarts by design — state lives in a file, not in memory. On startup, read the state file to know what was already alerted. On shutdown (SIGTERM), state is already on disk. If `svc watch` crashes and restarts, it resumes from the last known state rather than re-alerting everything.

State schema:
```json
{
  "services": {
    "dead-drop": {
      "status": "up",
      "consecutive_failures": 0,
      "alerted": false,
      "last_check": "2026-03-18T08:00:00Z",
      "last_change": "2026-03-18T07:00:00Z"
    }
  }
}
```

### Webhook payload schema

```json
{
  "event": "down",
  "service": "dead-drop",
  "description": "Zero-knowledge burn-after-read secrets sharing",
  "health_url": "https://wesley.thesisko.com/drop/health",
  "error": "connection refused",
  "consecutive_failures": 2,
  "timestamp": "2026-03-18T08:02:00Z",
  "previous_status": "up",
  "previous_change": "2026-03-17T14:00:00Z"
}
```

`event` is one of: `down`, `up` (recovery).

### Delivery failure handling: log and continue

If the webhook POST fails, `svc watch` logs the failure to `~/.local/share/svc/delivery-failures.log` with timestamp and error, then continues. Three retries with backoff (5s → 30s → 120s). After three failures, the delivery attempt is abandoned and logged — the monitoring loop does not stop.

Redundancy is the user's responsibility. `svc watch` writes a well-known log; a separate cron checks that log and alerts via whatever second channel the user wants. Unix philosophy: `svc watch` is not an alerting platform.

### What's NOT in v0.2

| Feature | Reason not included |
|---------|-------------------|
| Email delivery | Credentials, SMTP config, external lib — violates zero-dependency posture |
| SMS/PagerDuty | Same |
| Multiple webhooks | One URL is enough; user can fan out with a relay |
| Per-service intervals | Complexity for rare use case; global interval covers 99% |
| Silence windows | Out of scope; handle at the webhook receiver |
| Web UI | No |
| Configurable state file path | `--state <path>` flag for CI use if needed, but default covers all normal use |

---

## CLI interface

```
svc watch [flags]

Flags:
  --file, -f <path>       manifest file (default: ./services.yaml)
  --interval <seconds>    poll interval (default: 60)
  --failures <n>          consecutive failures before alerting (default: 2)
  --webhook <url>         webhook endpoint for state-change notifications
  --state <path>          state file path (default: ~/.local/share/svc/watch-state.json)
  --no-systemd            skip systemd checks
  --timeout <seconds>     health endpoint timeout (default: 5)
  --stdout                print events to stdout even when webhook is set (for debugging)
```

Without `--webhook`, events print to stdout only. Useful for development and for piping to another tool.

---

## State machine per service

```
         [start]
            │
       ┌────▼────┐
       │ UNKNOWN  │  (first check pending)
       └────┬────┘
            │ success
       ┌────▼────┐
       │   UP    │◄──────────────────────────┐
       └────┬────┘                           │
            │ 1st failure                   │ success (→ fire "up" event)
       ┌────▼────────┐                      │
       │  DEGRADED   │ (failures < threshold)│
       │ (no alert)  │                      │
       └────┬────────┘                      │
            │ Nth failure                   │
       ┌────▼────┐                          │
       │  DOWN   │──────────────────────────┘
       │(alerted)│ (fire "down" event)
       └─────────┘
```

UNKNOWN → UP on first success, UNKNOWN → DEGRADED on first failure. This prevents alerting on startup if a service is already down.

---

## Implementation plan

**Day 1 (today):**
1. `internal/watcher/state.go` — state struct, load/save
2. `internal/watcher/watch.go` — poll loop, state machine, event detection
3. `cmd/svc/main.go` — wire `svc watch` subcommand
4. Stdout output working, state file persisting

**Day 2:**
5. `internal/watcher/webhook.go` — HTTP POST, retry with backoff, delivery failure log
6. Tests: state transitions, consecutive failure counting, recovery detection
7. Blog post on design decisions

---

## Example stdout output

```
2026-03-18 08:00:00  svc watch starting — 6 services, 60s interval
2026-03-18 08:00:01  dead-drop       ✅ up (42ms)
2026-03-18 08:00:01  blog            ✅ up (88ms)
[... initial check, all services ...]
2026-03-18 08:01:01  dead-drop       ❌ down — connection refused (failure 1/2)
2026-03-18 08:02:01  dead-drop       ❌ down — connection refused (failure 2/2) → ALERT
2026-03-18 08:02:01  → webhook POST https://ntfy.sh/my-topic
2026-03-18 08:05:01  dead-drop       ✅ up (38ms) → RECOVERY
2026-03-18 08:05:01  → webhook POST https://ntfy.sh/my-topic
```

---

*Design done. Building now.*
