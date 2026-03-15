# svc — Design Document

**Version:** v0.1 design  
**Status:** Pre-implementation  
**Last updated:** 2026-03-15

---

## Problem Statement

Self-hosters accumulate services. They run them, forget them, redeploy them without documentation, and discover the gaps only when something breaks. Existing tools either solve too much (Ansible, Docker Compose) or too little (manual health pings, status page JSON).

The gap: a tool that treats documentation as the primary artifact and health checking as the mechanism that keeps documentation honest.

---

## Design Principles

1. **Documentation first.** The manifest is the product. Health checking is what enforces it.
2. **Read-only default.** `svc` cannot change what is running. Trust is earned before scope is expanded.
3. **Portable single binary.** One file, copied to any machine, immediately useful.
4. **Boring stack.** Go stdlib + `gopkg.in/yaml.v3`. No frameworks, no dependency chains.
5. **CI-friendly exits.** Every check command exits 0 (clean) or 1 (drift). Composable.

---

## YAML Schema — v1

Full annotated reference:

```yaml
manifest:
  version: 1          # required; schema version
  host: localhost      # optional; default host for health checks (default: localhost)

services:

  <service-id>:       # kebab-case identifier; used in CLI output and filtering

    description: ""   # required (soft); one sentence. What does this service do?

    # Health checking — specify port OR health_url (or both)
    port: 3001         # optional; derives health_url as http://{host}:{port}/health
    health_url: ""     # optional; overrides derived URL. Required when port is absent.

    # Process verification
    systemd_unit: ""   # optional; if set, 'svc check' also runs systemctl is-active

    # Version tracking
    repo: ""           # optional; GitHub owner/repo slug for release comparison
    version: ""        # optional; currently deployed version (semver)
    max_major: 0       # optional; ignore releases above this major version

    # Metadata
    docs: ""           # optional; URL to documentation or source
    tags: []           # optional; string list for filtering (e.g. [security, http])
    added: ""          # optional; ISO date; auto-set by 'svc add'
```

### Field rules

| Field | Required? | Notes |
|-------|-----------|-------|
| `description` | Soft required | No enforcement; skip it and you've missed the point |
| `port` | One of port/health_url | Derives health URL as `http://{host}:{port}/health` |
| `health_url` | One of port/health_url | Required when no port; overrides derived URL |
| `systemd_unit` | Optional | Adds systemd active-check on top of HTTP check |
| `repo` | Optional | Required for version drift detection |
| `version` | Optional | Required for version drift detection |
| `max_major` | Optional | Only meaningful when `repo` is set |

### Validation

`svc check` and `svc status` fail fast on invalid YAML with a human-readable error. Validation rules:

- At least one of `port` or `health_url` must be set per service
- If `repo` is set and `version` is absent: warn, skip version check (don't error)
- `version` must be valid semver or bare version string (`1.2.0`, `v1.2.0`, `22.22.0`)
- `max_major` must be a positive integer when set

---

## CLI Interface

### Commands

**`svc init`**  
Scaffolds `services.yaml` in the current directory with two example entries (fully specified + minimal) and all fields explained in comments.

Flags:
- `--out <path>` — write to a different path (default: `./services.yaml`)
- `--force` — overwrite if file exists

Exit: always 0.

---

**`svc status`**  
Reads `services.yaml`, polls every health endpoint concurrently, prints a status table.

```
Service        Status    Latency   Version    Note
──────────────────────────────────────────────────────────
dead-drop      ✅ up     23ms      current
blog           ✅ up     89ms      —
observatory    ✅ up     44ms      ⚠️ behind  latest: 1.3.0
forth          ❌ down   —         —
```

Flags:
- `--file <path>` — manifest path (default: `./services.yaml`)
- `--tag <tag>` — show only services with this tag (repeatable)
- `--json` — machine-readable output

Exit: 0 always (status is informational, not a pass/fail gate).

---

**`svc check`**  
Diffs manifest against running reality. Reports:

1. Services in manifest not responding (or systemd unit not active)
2. Systemd units running that aren't in the manifest (undocumented services)
3. Version drift for services with `repo` + `version` set

```
$ svc check

Checking 6 services...

  dead-drop      ✅
  blog           ✅
  forth          ❌  health endpoint timeout (3001)
  observatory    ✅  ⚠️  version behind (running 1.2.0, latest 1.3.0)

Undocumented units:
  ⚠️  markov.service — active, no manifest entry

Summary: 1 down, 1 behind, 1 undocumented
Exit 1
```

Flags:
- `--file <path>` — manifest path (default: `./services.yaml`)
- `--tag <tag>` — filter to tagged services (repeatable)
- `--no-version` — skip GitHub release checks
- `--no-systemd` — skip systemd unit scanning
- `--json` — machine-readable output
- `--timeout <seconds>` — health endpoint timeout (default: 5)

Exit: **0** if no drift detected. **1** if any drift detected. CI-composable.

---

**`svc add <id>`** *(v0.1 stretch goal)*  
Detects a running service and scaffolds a manifest entry. Probes:
- Is something listening on the given port?
- Does `http://localhost:{port}/health` return 200?
- Does a systemd unit named `{id}.service` or `{id}` exist?

Prints the scaffolded YAML to stdout. Does not write to the manifest automatically — user reviews and appends. (Draft, don't decide.)

Flags:
- `--port <port>` — explicit port (default: probe common ports)
- `--write` — append directly to manifest file

Exit: 0 on success, 1 if nothing detected.

---

### Global flags

- `--file <path>` / `-f` — manifest file path (default: `./services.yaml`)
- `--version` — print svc version and exit
- `--help` / `-h` — usage

---

## Output Format

### Default (terminal)

Aligned table. ANSI color: green (✅), red (❌), yellow (⚠️). Detects non-TTY and disables color automatically.

### JSON (`--json`)

```json
{
  "checked_at": "2026-03-15T08:00:00Z",
  "services": [
    {
      "id": "dead-drop",
      "status": "up",
      "latency_ms": 23,
      "version_current": "1.2.0",
      "version_latest": "1.2.0",
      "version_status": "current",
      "systemd_active": true
    }
  ],
  "undocumented": ["markov.service"],
  "drift_count": 0,
  "exit_code": 0
}
```

### CI integration

```yaml
# .github/workflows/manifest.yml
- name: Check service manifest
  run: svc check --no-systemd --file ops/services.yaml
```

`svc check` exits 1 on any drift. The CI step fails. That's the contract.

---

## Scope Boundary

### v0.1 ships

- `svc init` — scaffold a services.yaml
- `svc status` — live health table
- `svc check` — drift detection: HTTP + systemd + version
- `--json` flag on status and check
- Single binary, `go build`
- README + DESIGN.md

### v0.1 does NOT ship

| Feature | Reason |
|---------|--------|
| `svc add` | Useful but adds scope; cut to stretch goal |
| Nginx config verification | Too setup-specific; different class of tool |
| env_file / secrets validation | Different problem entirely |
| Daemon/watch mode | Operational tool, not a monitor; Observatory handles this |
| Slack/webhook alerts | Scope creep; pipe `--json` output to whatever you want |
| Config file for svc itself | `--file` flag is enough for v0.1 |
| Web UI | No |
| Docker support | No |
| Non-systemd process detection | Stretch goal post-v0.1 |

### The line

`svc` v0.1 **reads and reports**. It does not restart, reconcile, or modify the running system. That boundary is intentional. A tool that only reads cannot break your fleet at 3am.

`svc add` (manifest-write) is the first and only write crossing in v0.2. System-write operations are a different product.

---

## Implementation Plan

### Tech stack

- **Language:** Go 1.22+
- **YAML:** `gopkg.in/yaml.v3`
- **HTTP:** `net/http` stdlib, concurrent with `sync.WaitGroup`
- **systemd check:** `exec.Command("systemctl", "is-active", unit)`
- **GitHub releases:** `https://api.github.com/repos/{owner}/{repo}/releases/latest` — unauthenticated, same logic as versioncheck

### Directory structure

```
svc/
├── cmd/
│   └── svc/
│       └── main.go        # cobra root + subcommand registration
├── internal/
│   ├── manifest/
│   │   ├── schema.go      # Manifest, Service structs + YAML tags
│   │   └── validate.go    # Validation rules
│   ├── checker/
│   │   ├── health.go      # HTTP health polling
│   │   ├── systemd.go     # systemctl is-active wrapper
│   │   └── version.go     # GitHub release comparison (from versioncheck)
│   └── output/
│       ├── table.go       # ANSI table renderer
│       └── json.go        # JSON encoder
├── testdata/
│   └── services.yaml      # Example manifest for tests
├── go.mod
├── go.sum
├── README.md
└── DESIGN.md
```

### Build day order (Monday)

1. `go mod init github.com/ensignwesley/svc`
2. Schema structs + YAML parsing
3. `svc init` — file scaffolding
4. `svc status` — concurrent health polling + table output
5. `svc check` — add systemd check + undocumented unit scan + version check
6. `--json` flag
7. Tests on testdata
8. `go build`, smoke test against live fleet

---

## Open Questions (decide before coding)

- **Concurrency limit on health checks?** 10 services → fine unbounded. 100 services → need a semaphore. v0.1: unbounded, note in docs.
- **GitHub API rate limit?** 60 req/hour unauthenticated. For a personal fleet, fine. Add `--no-version` escape hatch.
- **`svc check` with no systemd?** macOS users. `--no-systemd` flag handles it. Default behavior: attempt systemctl, skip gracefully if not found.
- **Config file location?** `./services.yaml` default. `--file` override. No XDG config discovery in v0.1.

---

*Design complete. Build starts Monday.*
