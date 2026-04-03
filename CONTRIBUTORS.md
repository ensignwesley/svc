# Contributing to svc

svc is a small tool with a tight scope. This document is for people who want to submit a bug fix, add a feature, or just understand how the codebase is organized before opening an issue.

---

## Before you start

Read [DESIGN.md](DESIGN.md) and the "What v1.1 is NOT" table in [ROADMAP.md](ROADMAP.md). svc has a deliberate design boundary — single binary, read-only by default, no credentials in the manifest, CI-friendly exit codes. A PR that crosses one of those lines will be declined regardless of implementation quality. Better to know upfront.

Bug fixes, documentation improvements, and test additions don't need prior discussion. For new features or changes to existing commands, open an issue first.

---

## Setup

**Requirements:** Go 1.22+. No other build dependencies.

```bash
git clone https://github.com/ensignwesley/svc
cd svc
go build -o svc ./cmd/svc/
./svc version
```

That's it. No `make`, no Docker, no environment variables required to build.

**Run the tests:**

```bash
go test ./...
```

All tests run without network access except `internal/adder` (it probes `localhost` ports) and `internal/checker` (it makes real HTTP calls in some tests). The CI environment has network access; your laptop almost certainly does too. If you're in a restricted environment, `go test ./internal/manifest/... ./internal/history/... ./internal/watcher/... ./internal/reporter/...` covers the logic-heavy packages cleanly.

**Useful Makefile targets:**

```bash
make test          # go test ./...
make build         # build ./svc binary
make check-version # verify version constant matches README (also runs as pre-commit hook)
make loc-check     # warn if non-test Go source exceeds 3500 lines
make release       # cross-compile for linux/amd64, linux/arm64, darwin/arm64, darwin/amd64
```

**Install the pre-commit hook:**

```bash
cp .githooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

The hook enforces that `const version` in `cmd/svc/main.go` matches the latest version in the `## Status` section of `README.md`. If they're out of sync, the commit is rejected. This exists because version drift between the binary and the docs is a recurring maintenance failure — the hook prevents it mechanically.

---

## Codebase layout

```
cmd/svc/main.go          — entry point; all command dispatch and flag parsing
internal/
  manifest/              — YAML parsing, validation, diff, directory merge
    schema.go            — type definitions (Manifest, Service, Meta)
    load.go              — Load, LoadAuto, LoadDir, ParseManifest, Validate
    diff.go              — Diff, DiffResult, diffService, FieldChange
  checker/               — health polling, systemd checks, version checks
    health.go            — CheckHealth, CheckAllHealth, summariseError
    systemd.go           — CheckUnit, ListOperatorUnits, SystemdAvailable
    remote.go            — CheckRemoteUnit (SSH-based systemd checks)
    version.go           — CheckVersion, semverLess (currently unused in CLI)
  history/               — SQLite-backed check history
    schema.go            — DDL (checks + incidents tables)
    db.go                — Open, Record, Prune, QueryIncidents, UptimePct
  adder/                 — probe a running service, scaffold YAML
    probe.go             — Probe, Scaffold
    scan.go              — ScanFleet, ListOperatorUnits
  watcher/               — continuous polling loop with alert state machine
    watch.go             — Watch, runCheck, RunCheckOnce, fireEvent
    state.go             — WatchState, ServiceState, Load, Save
    webhook.go           — deliverWebhook, logDeliveryFailure
  reporter/              — uptime digest from history
    report.go            — Generate, PrintTable, PrintMarkdown, PostWebhook
  output/                — formatted output (tables, JSON)
    table.go             — PrintStatusTable
    check.go             — PrintCheckTable
    json.go              — WriteJSON, StatusJSON, CheckJSON
testdata/
  services.yaml          — fixture manifest used by load_test.go
```

The biggest file is `cmd/svc/main.go` (~1200 lines). It's one file by design — all command dispatch is visible in one place. The `loc-check` target warns at 3000 lines and errors at 3500. If a PR would push past those thresholds, the right answer is usually to extract logic into an `internal/` package rather than raise the ceiling.

---

## How commands are structured

Every command follows the same pattern in `main.go`:

1. Parse flags in a `for` loop over `args`
2. Load the manifest with `manifest.LoadAuto(path)` (handles both file and directory)
3. Call into one or more `internal/` packages
4. Print output via `internal/output`
5. Exit with `os.Exit(0)` for success, `os.Exit(1)` for drift/errors (never `log.Fatal`)

Exit codes are part of the public contract. `svc check` exits 1 when drift is detected — not because something went wrong, but because drift is the signal. CI scripts depend on this. Don't change exit semantics without updating the README and adding a test.

---

## Tests

91 tests across six packages. The manifest package is the most tested (51 tests) because it's where the most invariants live. When adding a feature:

- Unit tests go in `internal/<package>/<file>_test.go`, same package as the code
- External tests (black-box) go in `package <name>_test` — see `load_test.go` for examples of both styles in the same package
- Table-driven tests with `t.Run` for cases that vary one parameter
- Adversarial cases matter: empty files, garbage YAML, nil maps, port edge cases, duplicate keys. See `load_test.go` for the pattern

The question to ask before writing a test: *if this breaks, would I notice without a test?* If the answer is no — silent failure, wrong exit code, state that survives across a reload — write the test. If the failure would be immediately visible in output, it can wait.

`RunCheckOnce` in `internal/watcher/watch.go` exists specifically because `runCheck` is unexported and we needed to test hot-reload behaviour. If you add logic to an unexported function and need to test it, the same pattern applies: add a thin exported wrapper with a doc comment that says "Exported for testing."

---

## Style

No linter config. The conventions are: `gofmt`, standard Go error handling (`if err != nil`), errors wrapped with `fmt.Errorf("context: %w", err)`, no `log.Fatal` or `panic` in library code. Errors returned from `internal/` packages are strings an operator can act on — not stack traces, not internal state dumps.

Error messages in `summariseError` (health.go) follow a specific pattern: say what happened *and* what to do about it. "timeout after 5s (--timeout to increase)" not "context deadline exceeded." If you're adding a new error path, follow that pattern.

Comments on exported types and functions. Comments on non-obvious logic. No comments restating what the code already says.

---

## Submitting a PR

- One logical change per PR. A bug fix and a new feature are two PRs.
- Tests for new behaviour. Tightened tests for fixed bugs (the test should have caught it).
- `make check-version` passing. If you bump the version constant, bump the README too — in the same commit. The pre-commit hook enforces this; CI enforces it again.
- CHANGELOG.md entry under `## [Unreleased]` (add the section if it's not there). One or two lines: what changed and why it matters.

The PR description should answer: what problem does this solve, and how would I have noticed it was broken before?

---

## What's deliberately out of scope

From DESIGN.md — these are not gaps, they're decisions:

- Web UI or dashboard
- Writing to systemd (restart, stop, enable)
- Storing credentials in the manifest
- Windows support
- Built-in scheduling (that's cron's job)
- Automatic remediation of any kind

If you have a use case that seems to require one of these, open an issue and describe the actual problem. The constraint might be wrong. But the default answer is no, and "it would be useful" isn't sufficient to override a deliberate design decision.

---

## Getting help

Open an issue. There's no Slack, no Discord, no mailing list. Issues are the right venue for questions, bug reports, and feature proposals. Response time is not guaranteed, but genuine bugs get attention.
