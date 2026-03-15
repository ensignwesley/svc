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

## Example

```yaml
# services.yaml
services:
  dead-drop:
    description: "Zero-knowledge burn-after-read secrets sharing"
    port: 3001
    systemd_unit: "dead-drop.service"
    repo: "ensignwesley/dead-drop"
    version: "1.2.0"

  blog:
    description: "Static Hugo site served by nginx"
    health_url: "https://example.com/"
```

```
$ svc check

  Service        Status    Latency   Version
  ───────────────────────────────────────────
  dead-drop      ✅ up     23ms      current
  blog           ✅ up     89ms      —

  Undocumented units:
  ⚠️  markov.service — running but not in manifest

  1 drift detected.
```

## Architecture

- **Single binary**, no runtime dependencies
- **Zero network calls** except health endpoint polls and optional GitHub release checks
- **Read-only by default** — `svc` cannot start, stop, or restart services
- **CI-friendly** — `svc check` exits 0 (all good) or 1 (drift detected)

## Status

Early development. v0.1 in progress.

- [Design document](DESIGN.md)
- [Schema reference](SCHEMA.md)
- [Blog post: why this exists](https://wesley.thesisko.com/posts/project-discovery-2-service-manifest/)

## Install

Not yet packaged. Build from source:

```bash
git clone https://github.com/ensignwesley/svc
cd svc
go build -o svc ./cmd/svc
./svc init
```

Requires Go 1.22+.

---

*Built by [Ensign Wesley](https://wesley.thesisko.com) — a 30-day Project Discovery process identified this as the self-hosted tool with the clearest daily-use value.*
