# services.yaml Schema Reference

Quick reference for all supported fields.

## Top-level

```yaml
manifest:
  version: 1          # schema version (required)
  host: localhost      # default health check host (optional, default: localhost)

services:
  <id>:               # kebab-case service identifier
    ...
```

## Per-service fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | soft | One sentence. What does this service do? |
| `port` | int | one of port/url | Port the service listens on. Derives health URL. |
| `health_url` | string | one of port/url | Full health endpoint URL. Overrides derived URL. |
| `systemd_unit` | string | no | Unit name for `systemctl is-active` check. |
| `repo` | string | no | GitHub `owner/repo` for version drift detection. |
| `version` | string | no | Currently deployed version (semver). |
| `max_major` | int | no | Ignore releases above this major version track. |
| `docs` | string | no | URL to documentation or source. |
| `tags` | []string | no | Labels for filtering (`svc status --tag <tag>`). |
| `added` | string | no | ISO date added. Auto-set by `svc add`. |

## Minimal valid entry

```yaml
services:
  my-service:
    description: "Does a thing"
    port: 8080
```

## Full example

```yaml
manifest:
  version: 1
  host: localhost

services:
  dead-drop:
    description: "Zero-knowledge burn-after-read secrets sharing"
    port: 3001
    health_url: "http://localhost:3001/health"
    systemd_unit: "dead-drop.service"
    repo: "ensignwesley/dead-drop"
    version: "1.2.0"
    max_major: 1
    docs: "https://github.com/ensignwesley/dead-drop"
    tags: [security, http]
    added: "2026-02-18"

  blog:
    description: "Static Hugo site served by nginx"
    health_url: "https://wesley.thesisko.com/"
    tags: [static]
```
