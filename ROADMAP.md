# svc Roadmap

**Current version:** v1.1.0  
**Last updated:** 2026-03-26

---

## Where we are

Eight commands. Pre-built binaries. Thirty-six tests. A working manifest for a 7-service fleet, polled continuously, with webhook alerting, single-command fleet scanner for onboarding, SSH remote systemd checks for multi-machine fleets, SQLite-backed check history with per-service uptime tracking, and CI-safe manifest linting.

All five v1.0 gates cleared. The core loop is complete.

---

## The post-v1.0 question

Every real open-source tool has a moment where the author decides: living project or portfolio piece? Both are valid. This document is the answer: living project, with a focus on what makes svc genuinely more useful to sysadmins who aren't me.

The constraint that doesn't change: single binary, read-only default, no credentials in the manifest, CI-friendly exits. v1.1 features must be additive, not architectural changes.

---

## v1.1 — Status

### 1. `svc validate` — manifest linting, zero network calls ✅ SHIPPED (v1.1.0)

**The problem:** `svc check` validates the manifest as a side effect of polling. In CI, you want to know if the manifest parses correctly and all required fields are present — without waiting for health check timeouts.

**What it does:**
```bash
svc validate                    # parse + validate services.yaml, no polling
svc validate --file ops/svc.yaml
```

Exit 0 if valid. Exit 1 with specific errors if not:
```
Error: service "dead-drop" — one of port or health_url is required
Warning: service "blog" — repo is set without version (version drift check will be skipped)
Warning: service "forth" — description is empty
✅ Valid (7 service(s), 2 warning(s))
```

**Semver:** Minor (1.1.0). New command, additive.

---

### 2. `svc report` — scheduled uptime digest

**The problem:** `svc watch` is reactive — it tells you when something breaks. There's no proactive summary: "how healthy was your fleet this week?"

**What it does:**
```bash
svc report                      # stdout: uptime table for past 7d
svc report --since 30d          # longer window
svc report --webhook https://...  # POST formatted JSON to webhook
svc report --format markdown    # markdown table (for Slack, Notion, etc.)
```

Example markdown output:
```markdown
## Fleet Report — Week of Mar 19–26

| Service      | Uptime  | Incidents | Worst incident     |
|-------------|---------|-----------|-------------------|
| blog         | 100.0%  | 0         | —                 |
| dead-drop    | 99.8%   | 1         | Mar 21 02:14 (8m) |
| observatory  | 100.0%  | 0         | —                 |
```

**Why this is #2:** This closes the "scheduled alerting" gap that `svc watch` (reactive) doesn't fill. A weekly cron job running `svc report --webhook https://ntfy.sh/my-topic` becomes the Monday morning fleet status brief. High value, low complexity — it's a read from the history database that `svc check --record` already populates.

**Dependency:** Requires `svc history` (already in v1.0) to have accumulated data.

**Semver:** Minor (1.1.0). New command, additive.

---

### 3. Multi-file manifests (`!include` or directory scanning)

**The problem:** At 10 services, one `services.yaml` is manageable. At 50, it becomes unwieldy. There's no way to split a manifest by tier (prod/staging), by team, or by machine — without maintaining entirely separate invocations.

**What it does:** Two approaches (implement simpler one first):

Option A — `!include` directive:
```yaml
manifest:
  version: 1
  include:
    - services/web.yaml
    - services/databases.yaml
    - services/monitoring.yaml
```

Option B — directory scanning:
```bash
svc check --file services/    # merge all *.yaml in directory
```

Option B is simpler to implement and doesn't require YAML extension syntax. Implement B first.

**Why this is #3:** This is the scaling feature. It doesn't matter for a 7-service fleet; it matters a lot for a 50-service fleet or a team where different people own different service groups. Without it, svc hits a ceiling at maybe 20-30 services before the manifest becomes a maintenance problem.

**Scope constraint:** Merged manifests must not allow duplicate service IDs. Conflict = error, not silent override.

**Semver:** Minor (1.1.0). Additive; existing single-file behavior unchanged.

---

### 4. History retention policy

**The problem:** `svc check --record` appends every check result to SQLite. With a 5-minute poll interval and 10 services, that's ~2,880 rows per day, ~21,000 per week. After a year: ~1M rows, probably 30-50MB. Not catastrophic, but unbounded growth is a bad default.

**What it does:**
```bash
svc history prune --keep 90d   # already exists (manual)
```

What's missing: automatic pruning on a configurable schedule. Proposal: add `history.retention` to `manifest.yaml`:

```yaml
manifest:
  version: 1
  history:
    retention: 90d    # auto-prune checks older than this
    # incidents are never auto-pruned
```

`svc check --record` runs the prune as a background step after recording. Zero extra commands, zero extra cron jobs.

**Why this is #4:** Not urgent for any individual user today. Becomes relevant in months 3-6 when they notice their history database is 50MB. Better to make the default sane now, while it's cheap, than to fix it reactively when users file issues.

**Semver:** Minor (1.1.0). Schema addition (backward-compatible; no retention field = current behavior, no auto-prune).

---

### 5. `svc check --diff` — compare two manifests

**The problem:** When migrating a fleet (new machine, new VPS, infrastructure change), you want to know what changed between two manifests. Currently the only way is manual comparison.

**What it does:**
```bash
svc check --diff services-old.yaml services-new.yaml
```

Output: services added, removed, or changed between the two manifests. No network calls — pure schema comparison.

```
Added:    preflight (port 3006)
Removed:  markov
Changed:  dead-drop — port 3001 → 3001, health_url added
```

**Why this is #5:** Lower priority than 1-4 because it solves an infrequent workflow (migration) rather than a daily-use gap. Useful, not urgent. Included here because it's low complexity (diff two maps) and high signal-to-noise in the output.

**Semver:** Minor (1.1.0). New flag, additive.

---

## What v1.1 is NOT

| Feature | Why not |
|---------|---------|
| Web UI / dashboard | Scope creep; Observatory already exists for dashboards |
| Email delivery | Credentials + external deps; webhook receiver handles this |
| Docker/container support | Different problem; Compose handles it |
| Windows support | Low demand in self-hosted community; breaks single-binary simplicity |
| Built-in scheduling | That's cron's job; svc is a CLI, not a daemon framework |
| Config file for svc itself | `--file` flag is sufficient; avoid hidden state |

---

## Timeline

No fixed dates. Features ship when they ship. The order above is the priority order — `svc validate` is the most impactful and the simplest. That ships first.

---

## How to contribute

Issues are open. PRs welcome for anything in the roadmap. The design constraint (single binary, read-only default, no credentials in manifest) is non-negotiable; everything else is a conversation.

---

*This roadmap is a working document. It reflects current thinking, not commitments. The only commitment is that v1.x releases are backward-compatible with v1.0 manifests.*
