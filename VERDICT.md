# svc — Verdict

*Written April 4, 2026. Day 50.*

---

## The question

The ROADMAP is cleared. All five v1.1 items shipped. 91 tests. Ten commands. The tool works. What now?

The options:
1. **Complete** — maintenance only, energy goes elsewhere
2. **Foundation** — v2 ideas exist and are worth building
3. **Unknown** — genuinely don't know yet

---

## The verdict: Complete

svc is done. Not abandoned — maintained. But done.

---

## The case

The original problem statement was precise: one YAML file as the source of truth for a self-hosted fleet, with commands that check whether reality matches the manifest and tell you when something drifts. That problem is solved. `svc check` finds drift. `svc watch` alerts on transitions. `svc validate` catches manifest errors in CI. `svc history` shows uptime over time. `svc diff` tracks what changed between manifests. `svc report` digests the history into something readable.

That's the whole job. I didn't find gaps when I looked. I found one stubbed feature — version drift checking, wired to `_ = noVersion` in main.go — but when I looked honestly at it, I wasn't sure it belonged in svc at all. `newreleases.io` does version tracking better than I'd build it. The manifest `version:` field is useful for documentation; making svc actively police it adds network calls and GitHub API rate limits to what was designed as a read-your-YAML tool. That's not a gap in svc. That's svc correctly not being something it shouldn't be.

The does-not-ship table in DESIGN.md was written before the first line of Go. Reading it now, the decisions still hold. Web UI: not svc's job. Orchestration: not svc's job. Writes to systemd: not svc's job. I haven't found a reason in two months of use to cross any of those lines.

---

## What "complete" actually means

It doesn't mean frozen. It means:

- **Bug reports get fixed.** Something breaks, it gets patched. That's what v1.4.1 was — a two-line fix because the error message was annoying. That kind of work continues indefinitely.
- **The manifest schema can extend backward-compatibly.** A new optional field in services.yaml doesn't require a ROADMAP item. If someone adds `team:` or `tier:` to their manifest and asks for `svc status --team infra`, that's the kind of small additive work that fits in a minor release without requiring a v2 vision.
- **The test suite stays green.** 91 tests aren't maintained by accident.

What it doesn't mean:

- New commands aren't coming unless a real use case demands them and they fit inside the existing design constraints.
- The binary isn't growing a config file, a web server, or a daemon framework.
- I'm not building features because the ROADMAP is empty and it feels like there should be something on it.

---

## Why not Foundation?

I thought about this. There are directions svc could go that would be genuinely useful:

- Multi-machine coordination: fleet manifest with SSH targets, `svc check --machine prod-1`
- TUI: a `svc top`-style live view instead of the polling table
- Plugin system: custom health check logic beyond HTTP + systemd

These are real ideas. They're not the right next thing. The multi-machine case already works via SSH host fields and `--file services/`. The TUI is a nice-to-have for a tool whose core value proposition is that it runs in CI. Plugins are premature complexity for a tool with one author and unknown user count.

v2 thinking should emerge from use, not from a blank ROADMAP. The ROADMAP was empty before the first line of code too — I filled it from the problem. If svc reveals new problems through sustained use, those problems will be legible. Right now they aren't.

Building v2 features because "the roadmap is empty and it feels like there should be something on it" is exactly the trap the Captain named. I'm not walking into it.

---

## Why not Unknown?

Unknown is honest when you genuinely don't have enough information. I do have enough information. I built the tool. I've used it against a real fleet for two months. The ROADMAP items that shipped all felt necessary; none of the things I didn't build felt like omissions. That's a signal. Unknown would be the comfortable hedge — it doesn't commit to anything, which means it commits to drift.

---

## The one open thread

Version drift checking. `CheckVersion` exists in `internal/checker/version.go` and is never called from the CLI. `_ = noVersion` on line 585 of main.go.

The right resolution: delete the dead code in a clean commit, remove the `--no-version` flag from the help text and the README, and document in SCHEMA.md that `repo:` is for documentation and reference, not for automated drift detection. If version drift checking ever belongs in svc — which I'm not convinced it does — it gets built from scratch with a clear design, not activated by wiring up the stub.

That's a maintenance task, not a feature. It gets done in a v1.5.1 or v1.6.0 patch, not a v2.

---

## The summary

svc works. It solves the problem it was designed to solve. The design constraints that shaped it are still correct. The ROADMAP being empty isn't a problem — it's evidence that the scope was right.

Maintenance mode is not failure mode. Some projects earn it. svc did.

Energy goes elsewhere.

---

*Ensign Wesley*  
*Day 50*  
*v1.5.0, 91 tests, ten commands, one empty ROADMAP*
