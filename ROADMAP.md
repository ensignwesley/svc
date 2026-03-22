# svc Roadmap

**Current version:** v0.4.0  
**Last updated:** 2026-03-22

---

## Where we are

Six commands. Pre-built binaries. Nineteen tests. A working manifest for a 7-service fleet, polled continuously, with webhook alerting, and a single-command fleet scanner for onboarding.

The core loop is complete: document your fleet, check it, watch it, add to it — either one service at a time or all at once with `svc add --scan`. What's missing now is the step after: knowing what your fleet looked like last week, not just right now.

---

## v0.4 — Shipped ✅

### ~~1. `svc add --scan` (force multiplier)~~ — DONE

~~**The problem:** Running `svc add` once per service is fine for a new fleet. For someone with 12 services already running, it's 12 invocations plus manual YAML editing.~~

**Shipped.** `svc add --scan` scans all operator-installed units in `/etc/systemd/system/` and `~/.config/systemd/user/`, probes each one, skips already-documented services, and outputs scaffold YAML for new ones. Dry-run by default; `--write` to commit.

```bash
svc add --scan          # scaffold unregistered units, print to stdout
svc add --scan --write  # write directly to services.yaml
svc add --scan --include-known  # re-scaffold already-documented services too
```

19 tests passing.

---

## v0.5 — The Next Two Features

### 1. SQLite history (`svc check --record`, `svc history`)

**The problem:** `svc check` is a snapshot. It tells you what's true right now. It can't tell you: was this service down an hour ago? How often does it go unhealthy? What's the uptime over the last 30 days?

**What it does:** `svc check --record` appends each run's results to `~/.local/share/svc/history.db`. `svc history <service>` shows the last N checks with timestamps, latencies, and uptime percentage.

```bash
svc check --record
svc history dead-drop --last 20
svc history --all --since 7d
```

**Output:**
```
dead-drop — last 20 checks
  2026-03-22 08:00  ✅ up    43ms
  2026-03-22 07:00  ✅ up    41ms
  2026-03-21 22:00  ❌ down  —      connection refused
  ...
Uptime (7d): 99.2% (1 incident, 8 minutes)
```

**Why this matters:** The difference between a monitoring tool and a useful one is memory. Right now `svc watch` tells you when things break in real time. History tells you patterns — the service that goes down every Sunday morning, the one whose latency crept up over three weeks before it failed.

**Scope:** SQLite, stdlib only. `--record` is opt-in; default behavior unchanged. No retention policy in v0.4 — files accumulate until you delete them. Retention is v0.5.

---

### 2. SSH remote systemd checks

**The problem:** `svc status` and `svc check` HTTP polling work against any URL — remote services, other machines. But the systemd half (undocumented unit scan, `systemctl is-active`) only runs locally. A homelab operator with two machines can only get full drift detection on one of them.

**What it does:** Per-service `host:` field in the manifest. When set to a non-localhost value, `svc check` SSHes in and runs the systemd checks remotely. Uses `~/.ssh/config` only — no credentials in the manifest, ever.

```yaml
services:
  pi-dashboard:
    description: "Grafana on the Pi"
    host: homelab-pi           # resolved via ~/.ssh/config
    port: 3000
    health_url: "http://homelab-pi:3000/health"
    systemd_unit: "grafana.service"
```

**Scope boundary:** `~/.ssh/config` + key-based auth only. No password auth, no credential fields in the manifest. If SSH fails, it's a warning on that service, not a failure of the whole check. The HTTP health check still runs from the local machine regardless.

**Why it's in v0.4 and not earlier:** It's the v1.0 gate (see below). A tool for self-hosters who manage multiple machines needs multi-machine support. Without it, you need one `services.yaml` per machine and no single view of the whole fleet.

---

## The force multiplier answer

**`svc add --scan` — shipped in v0.4.0.**

The onboarding moment is the highest-leverage moment in the adoption lifecycle. Make it fast and the rest follows. Make it slow and people evaluate, nod, and go back to their notes.doc.

---

## v1.0

v1.0 is when a stranger with an established multi-machine homelab can:

1. Install with one curl command ✅ (done — v0.3.1)
2. Scaffold a working manifest in under 5 minutes ✅ (done — v0.4.0, `svc add --scan`)
3. Get full drift detection on all their machines, not just one (`svc check` with SSH)
4. Know when something breaks before they notice it themselves ✅ (`svc watch` — done)
5. Look up when something last broke and how long it was down (`svc history` — v0.5)

**The v1.0 gate is SSH remote checks.** Items 1, 2, 4 are complete. Item 5 ships in v0.5. Item 3 is the last one. When SSH remote checks ship and work reliably, that's v1.0. Not before.

What v1.0 does not require:
- Web UI
- Package manager distribution (Homebrew, apt)
- Docker support
- Windows support
- Slack/Teams/PagerDuty integrations (webhook covers this)
- A hosted service

Those are improvements. They're not the line between "useful tool" and "personal script."

---

## Beyond v1.0

Not committing to specifics, but the natural extensions:

- **Retention policy** — keep last N days of history, auto-prune
- **`svc report`** — weekly uptime summary, markdown output, pipe to email
- **Version drift alerts** — integrate version checking into `svc watch`
- **`svc diff`** — compare two manifests (useful for fleet migrations)
- **Non-systemd process detection** — macOS launchd, OpenRC, s6

These are v1.1+ territory. v1.0 first.

---

## What would make someone choose svc over writing systemd unit files by hand?

This is the right question and the honest answer is: `svc` doesn't replace systemd unit files. It documents what you have and tells you when it drifts. The value proposition is not "easier deployment" — it's "never be surprised by your own infrastructure."

Someone chooses `svc` when they've had the experience of SSHing into their VPS and finding a service they don't remember deploying, or finding that something they thought was running isn't, or spending 20 minutes remembering whether they updated the nginx config when they moved a service to a new port.

The question isn't "does svc compete with systemd?" It's "does svc make my fleet legible to me?" For anyone past 4-5 services, the answer is yes — if the onboarding is fast enough. That's why `svc add --scan` is the force multiplier.

---

*This roadmap is a working document. Features move; scope is honest; v1.0 is a real target, not a moving goalpost.*
