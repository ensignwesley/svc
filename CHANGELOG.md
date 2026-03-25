# Changelog

All notable changes to svc. Follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

---

## [1.0.0] — 2026-03-25

All five v1.0 gates cleared. Feature-complete.

**What v1.0 means:** a stranger with an established fleet can install svc with one curl command, scaffold a working manifest in under five minutes, get full drift detection across all their machines, be alerted when something breaks, and look up when something last broke and for how long.

### Added
- `svc history` — query stored check history and uptime summaries
- `svc history <service>` — per-service incident list with duration and error details
- `svc history prune` — trim old check records (incidents are never pruned)
- `svc check --record` — append every check result to SQLite history database
- `first_error` + `last_error` in incident records — captures flapping failure modes
- Open incident created on first failure (not deferred to recovery — this was the critical design decision)
- Partial index on `recovered_at IS NULL` for O(open) open-incident queries
- README 30-second hook: problem statement + terminal output above the fold

---

## [0.5.0] — 2026-03-23

### Added
- SSH remote systemd checks — per-service `host:` field in manifest
- When `host:` is set, `svc check` SSHes to the remote machine for `systemctl is-active` verification
- Uses `~/.ssh/config` exclusively — no credentials in the manifest, ever
- SSH failures surface as warnings, not errors; HTTP checks still run from local
- `checker/remote.go` — `CheckRemoteUnit()` with user/system session fallback
- Human-readable SSH error messages (auth failure, timeout, host unreachable)
- Multi-machine fleet support: one `services.yaml`, multiple hosts

---

## [0.4.0] — 2026-03-22

### Added
- `svc add --scan` — probe all operator-installed systemd units at once
- Scans `/etc/systemd/system/` and `~/.config/systemd/user/` for operator units
- Skips services already in the manifest (idempotent on repeated runs)
- `--include-known` flag to re-scaffold existing entries
- `ScanFleet()` reuses `ListOperatorUnits()` — no duplicate probe logic
- Force multiplier: establishes a full manifest from an existing fleet in one command

---

## [0.3.1] — 2026-03-21

### Added
- GitHub Actions release workflow — triggers on `v*` tags, builds four platform binaries
- Pre-built binaries: linux/amd64, linux/arm64, darwin/arm64, darwin/amd64
- `checksums.txt` attached to every release
- Install instructions in README with `curl | tar xz` one-liners
- `make release` with `-ldflags -X main.version` — binaries self-report correct version
- README: reverse proxy documentation (nginx proxy users need explicit `health_url`)
- README: systemd unit detection explanation (operator units vs OS-managed units)

---

## [0.3.0] — 2026-03-20

### Added
- `svc add <id>` — probe a running service and scaffold a manifest entry
- Probes systemd (user + system sessions), then health endpoints in order: `/healthz` → `/health` → `/ping` → `/`
- Explicit `health_url` emitted only for non-standard paths
- Notes in scaffold output explain what couldn't be detected
- `--write` flag appends directly to manifest; dry-run by default
- `v1.0 means` definition added to README: the bar a stranger must clear

### Fixed
- Health endpoint probe order: `/healthz` first (Kubernetes/Go ecosystem convention), then `/health`
- Added `/ping` as third option before `/` fallback

---

## [0.2.0] — 2026-03-19

### Added
- `svc watch` — continuous health polling with four-state machine (Unknown/Up/Degraded/Down)
- Consecutive failure threshold before alerting (default: 2, configurable with `--failures`)
- Recovery notification always fires when alerted service returns
- State persisted to `~/.local/share/svc/watch-state.json` — survives restarts
- Atomic state writes (tmp + rename) — crash-safe
- Webhook delivery with 3-retry exponential backoff (5s → 30s → 120s)
- Delivery failures logged to `~/.local/share/svc/delivery-failures.log`
- SIGTERM/SIGINT graceful shutdown
- `--heartbeat <url>` flag — dead man's switch: POST on every successful poll cycle
- `WATCH.md` design document

---

## [0.1.1] — 2026-03-17 *(patch)*

### Added
- `--json` flag on `svc status` and `svc check` — machine-readable output
- JSON schema: `checked_at`, per-service status/latency/error, `undocumented` array, `exit_code`

### Fixed
- README output format corrected to match actual CLI (was showing stale example)

---

## [0.1.0] — 2026-03-16

Initial release. Core loop: document, check, status.

### Added
- `svc init` — scaffold annotated `services.yaml` with two examples (full + minimal)
- `svc status` — concurrent HTTP health polling, ANSI table output, TTY detection
- `svc check` — drift detection in both directions:
  - Services in manifest that aren't responding
  - Systemd units in `/etc/systemd/system/` and `~/.config/systemd/user/` not in manifest
  - Version drift (GitHub releases API comparison) when `repo` + `version` set
- `--tag` filter on status and check
- `--no-systemd` flag for macOS and non-systemd systems
- `--no-version` flag to skip GitHub API calls (60 req/hr unauthenticated limit)
- `--timeout` configurable health endpoint timeout
- systemd user/system session detection via `FragmentPath` (not name matching)
- `ignore_units` in manifest for deliberate exclusions
- `services.yaml` schema: `port`, `health_url`, `systemd_unit`, `repo`, `version`, `max_major`, `host`, `tags`, `added`, `description`
- 5 tests (YAML parsing, validation, URL resolution, error messages)
- `DESIGN.md`, `SCHEMA.md`, `WATCH.md`, `ROADMAP.md`

---

## [Pre-release] — 2026-03-15

- `DESIGN.md`, `SCHEMA.md` — architecture and schema documentation
- `README.md` — problem statement, three-command demo

---

*For the full commit history: [github.com/ensignwesley/svc/commits/main](https://github.com/ensignwesley/svc/commits/main)*
