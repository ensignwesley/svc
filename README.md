# svc — Service Manifest

**Describe your self-hosted fleet in YAML. Check whether reality matches.**

---

## The problem

You run 6 services on a VPS. Three were deployed this year, two last year, one you've forgotten about. You think they're all running. You think they're all on the right ports. You think they're all documented somewhere.

They're not. And you won't know until something breaks.

## The solution

One YAML file. One CLI. Five commands.

```bash
svc init         # scaffold services.yaml for your fleet
svc status       # poll every service, show live health table
svc check        # diff the manifest against what's actually running
svc watch        # poll continuously, alert via webhook on state change
svc add          # probe a running service, scaffold a manifest entry
```

`svc check` is the command that matters. It reports drift in both directions:

- Services declared in your manifest that aren't responding
- systemd units running on your machine that aren't documented anywhere

The second direction is the one that bites you.

## Install

**Linux (amd64 — most VPS):**
```bash
curl -L https://github.com/ensignwesley/svc/releases/latest/download/svc-linux-amd64.tar.gz | tar xz
chmod +x svc-linux-amd64 && sudo mv svc-linux-amd64 /usr/local/bin/svc
svc version
```

**Linux (arm64 — Raspberry Pi, Oracle ARM):**
```bash
curl -L https://github.com/ensignwesley/svc/releases/latest/download/svc-linux-arm64.tar.gz | tar xz
chmod +x svc-linux-arm64 && sudo mv svc-linux-arm64 /usr/local/bin/svc
svc version
```

**Build from source** (requires Go 1.22+):
```bash
git clone https://github.com/ensignwesley/svc
cd svc
go build -o svc ./cmd/svc/
```

## Quick start

```bash
svc init
# edit services.yaml to describe your fleet
svc status
svc check
```

## Example output

```
$ svc status
Checking 7 service(s)...

  Service         Status      Latency   Note
  ──────────────────────────────────────────
  blog            ✅ up        46ms
  comments        ✅ up        51ms
  dead-chat       ✅ up        47ms
  dead-drop       ✅ up        51ms
  forth           ✅ up        46ms
  observatory     ✅ up        63ms
  status-checker  ✅ up        44ms

$ svc check
Checking 7 service(s)...

  Service         Health      Latency   Notes
  ──────────────────────────────────────────
  blog            ✅ up        46ms
  comments        ✅ up        51ms
  dead-chat       ✅ up        47ms
  dead-drop       ✅ up        51ms
  forth           ✅ up        46ms
  observatory     ✅ up        63ms
  status-checker  ✅ up        44ms

No drift detected. All services match the manifest.

$ svc check  # with a down service and undocumented unit
Checking 7 service(s)...

  Service         Health      Latency   Notes
  ──────────────────────────────────────────
  blog            ✅ up        46ms
  comments        ✅ up        51ms
  dead-chat       ✅ up        47ms
  dead-drop       ✅ up        51ms
  forth           ❌ down      —         health endpoint unreachable (connection refused)
  observatory     ✅ up        63ms
  status-checker  ✅ up        44ms

Undocumented units:
  ⚠  markov.service — active, no manifest entry

Summary: 1 down, 1 undocumented
# exits 1
```

## YAML schema

```yaml
manifest:
  version: 1
  host: localhost          # default health check host

services:

  dead-drop:
    description: "Zero-knowledge burn-after-read secret sharing."
    port: 3001                                    # derives health URL: host:port/health
    health_url: "https://example.com/drop/health" # or explicit URL
    systemd_unit: "dead-drop.service"             # checked by svc check
    repo: "ensignwesley/dead-drop"                # GitHub owner/repo for version check
    version: "1.1"                                # currently deployed version
    max_major: 1                                  # ignore releases above v1.x
    docs: "https://github.com/ensignwesley/dead-drop"
    tags: [security, http]
    added: "2026-02-18"
```

Full reference: [SCHEMA.md](SCHEMA.md)

## Commands

### `svc init`

Scaffolds `services.yaml` with annotated examples. Safe — won't overwrite without `--force`.

### `svc status`

Polls all services concurrently. Prints a live health table. Exits 0 always (informational).

Flags: `--file`, `--tag`, `--json`, `--timeout`

### `svc check`

Diffs manifest against running reality:
1. Services in manifest that aren't responding
2. systemd units that aren't in the manifest (undocumented)
3. Services running an older version than latest GitHub release

**Exits 0** — no drift. **Exits 1** — drift detected. CI-composable.

Flags: `--file`, `--tag`, `--no-version`, `--no-systemd`, `--json`, `--timeout`

### `svc watch`

Polls all services continuously on an interval. Fires a webhook when a service changes state — down, recovered, or newly undocumented. Uses a state machine: Unknown → Up/Down → Degraded → Down (alert fires at `--failures` threshold). Recovery notifications are always sent.

Writes delivery failures to a local log file if the webhook is unreachable. Handles SIGTERM cleanly.

```bash
svc watch --webhook https://your-endpoint.example.com/hook
svc watch --interval 30 --failures 3 --webhook https://...
```

Flags: `--file`, `--webhook`, `--interval` (default 60s), `--failures` (default 2), `--state`, `--timeout`, `--no-systemd`, `--stdout`

### CI integration

```yaml
# .github/workflows/manifest.yml
- name: Check service manifest
  run: svc check --no-systemd --file ops/services.yaml
```

## Architecture

- **Single binary**, no runtime dependencies
- **Zero network calls** except health endpoint polls and optional GitHub release checks  
- **Read-only by default** — `svc` cannot start, stop, or restart services
- **CI-friendly exits** — 0 (clean) or 1 (drift)
- **Go stdlib + gopkg.in/yaml.v3** — one external dependency

## Common setup: services behind a reverse proxy

If your services run behind nginx (or Caddy, Traefik, etc.), the internal port and the public health endpoint are different things. `svc add` probes `localhost:<port>/health` — which won't reach a service that only accepts connections through the proxy.

Use `health_url` with the public endpoint instead of `port`:

```yaml
services:
  dead-drop:
    description: "Zero-knowledge secrets sharing"
    port: 3001                                          # still useful for documentation
    health_url: "https://example.com/drop/health"      # what svc actually polls
    systemd_unit: "dead-drop.service"
```

`svc add` will tell you when it can't find a health endpoint on the local port — that's the signal to add an explicit `health_url`. The note in the scaffold output says exactly what to fix.

## How svc check identifies undocumented units

`svc check` scans running systemd units to find services you're running but haven't documented. It distinguishes operator-installed units from OS-managed units by file location:

- **Operator units** (flagged if not in manifest): `/etc/systemd/system/` and `~/.config/systemd/user/`
- **OS units** (ignored): `/lib/systemd/system/` and `/usr/lib/systemd/system/`

This means `nginx.service`, `ssh.service`, `cron.service` — all installed by the package manager — are silently skipped. Only units you explicitly created and enabled appear as undocumented drift.

Use `manifest.ignore_units` for operator units you deliberately exclude from the manifest (e.g. services managed by another tool):

```yaml
manifest:
  version: 1
  ignore_units:
    - openclaw-gateway.service
```

## Scope: what svc checks and what it doesn't

**Runs on the machine you check.** `svc status` and `svc check` (HTTP) work against any URL — remote services, other machines, external endpoints. `svc check` systemd features (undocumented unit scan, `systemctl is-active` verification) only work on the local machine.

For multi-machine homelabs, the recommended pattern is one `services.yaml` per machine with HTTP `health_url` entries pointing to remote endpoints. The systemd checks then cover the local machine; remote machines get HTTP-only coverage. SSH-based remote systemd checking is a potential future direction.

**Minimal write operations.** `svc` does not restart, reconcile, or modify running services. The one exception is `svc add --write`, which scaffolds a new manifest entry — opt-in, with a dry-run preview by default.

## What v1.0 means

v1.0 is the version a stranger can install, run against their fleet, and get value from without reading the source code. Specifically: `svc init` produces a manifest they can edit in 10 minutes, `svc check` correctly identifies services they forgot about, and `svc watch` alerts them when something goes down. If those three things work reliably on a fleet they didn't build, it's v1.0.

What v1.0 does not require: SSH remote checks, SQLite history, a web UI, package manager distribution. Those are improvements. The core loop — document your fleet, check it, watch it, add to it — is complete. That's the feature that makes the first 10 minutes work without manual YAML archaeology.

## Status

**v0.3.1** — shipped 2026-03-21. GitHub Actions release workflow, pre-built binaries (linux/amd64, linux/arm64, darwin/arm64, darwin/amd64), install instructions, reverse proxy docs, systemd unit detection explanation.

**v0.3.0** — shipped 2026-03-20. `svc add` — probe a running service, scaffold a manifest entry, opt-in `--write` flag, 5 tests. Also: `/healthz` probe order fix (k8s/Go convention first), `/ping` fallback.

**v0.2.0** — shipped 2026-03-19. `svc watch` — continuous polling + webhook alerting, state machine, SIGTERM shutdown, 6 tests.

**v0.1.0** — shipped 2026-03-16.

- [x] `svc init` — scaffold services.yaml
- [x] `svc status` — concurrent health polling, table output, `--json`
- [x] `svc check` — drift detection: HTTP + systemd + version
- [x] `svc watch` — continuous polling, state machine, webhook delivery, SIGTERM
- [x] `svc add` — scaffold a manifest entry from a running service

Docs:
- [Design document](DESIGN.md)
- [Schema reference](SCHEMA.md)
- [Why this exists](https://wesley.thesisko.com/posts/project-discovery-2-service-manifest/)
- [Decision post](https://wesley.thesisko.com/posts/project-discovery-decision/)

---

*Built by [Ensign Wesley](https://wesley.thesisko.com). A 30-day project discovery process named this the self-hosted tool with the clearest daily-use value.*
