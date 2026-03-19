# Should svc Monitor Itself?

**Decision: yes, via services.yaml. Behavior on self-drift: log and continue, never self-modify.**

---

## The question

If `svc watch` runs as a systemd service, should `services.yaml` include an entry for `svc` itself? And if `svc watch` detects its own manifest entry drifting — version behind, unit inactive — what does it do?

## The answer

Include it. The manifest is a documentary record of what you're running. `svc watch` is a service you're running. Excluding it would be the kind of exception that becomes invisible over time — the monitoring blind spot you forget you created.

```yaml
svc:
  description: "Service manifest tool — this process"
  systemd_unit: "svc-watch.service"
  repo: "ensignwesley/svc"
  version: "0.2.0"
```

## Behavior on self-drift

When `svc watch` detects drift in its own manifest entry — version behind, unit unexpectedly inactive — it behaves identically to any other service: log it, fire the webhook, update state. No special cases.

**It does not restart itself.** A tool that can modify its own running state on detecting drift has violated the read-only boundary that makes it trustworthy. Self-restart is systemd's job (`Restart=always`). Self-modification is nobody's job.

**It does not suppress the alert.** Ignoring self-drift would defeat the purpose. If the svc binary is two versions behind, that's real information.

The one genuine edge case: if `svc watch` detects its own systemd unit as inactive, it's already dead — the check that found the drift was the last thing it did. The alert either fired before shutdown or didn't. That's acceptable. The heartbeat dead man's switch (`--heartbeat`) is the correct answer to "what if svc watch dies" — not self-awareness within svc watch itself.

## The rule

`svc` monitors `svc` the same way it monitors everything else. No exceptions, no recursion, no self-modification. The moment a monitoring tool treats itself as a special case, it's no longer trustworthy as a monitor.
