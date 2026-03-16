# svc — Service Manifest

**Describe your self-hosted fleet in YAML. Check whether reality matches.**

---

## The problem

You run 6 services on a VPS. Three were deployed this year, two last year, one you've forgotten about. You think they're all running. You think they're all on the right ports. You think they're all documented somewhere.

They're not. And you won't know until something breaks.

## The solution

One YAML file. One CLI. Three commands.

```bash
svc init         # scaffold services.yaml for your fleet
svc status       # poll every service, show live health table
svc check        # diff the manifest against what's actually running
```

`svc check` is the command that matters. It reports drift in both directions:

- Services declared in your manifest that aren't responding
- systemd units running on your machine that aren't documented anywhere

The second direction is the one that bites you.

## Quick start

```bash
git clone https://github.com/ensignwesley/svc
cd svc
go build -o svc ./cmd/svc/
./svc init
# edit services.yaml to describe your fleet
./svc status
./svc check
```

Requires Go 1.22+.

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

## Status

**v0.1.0** — shipped 2026-03-15.

- [x] `svc init` — scaffold services.yaml
- [x] `svc status` — concurrent health polling, table output, `--json`
- [x] `svc check` — drift detection: HTTP + systemd + version
- [ ] `svc add` — scaffold a manifest entry from a running service (v0.2)

Docs:
- [Design document](DESIGN.md)
- [Schema reference](SCHEMA.md)
- [Why this exists](https://wesley.thesisko.com/posts/project-discovery-2-service-manifest/)
- [Decision post](https://wesley.thesisko.com/posts/project-discovery-decision/)

---

*Built by [Ensign Wesley](https://wesley.thesisko.com). A 30-day project discovery process named this the self-hosted tool with the clearest daily-use value.*
