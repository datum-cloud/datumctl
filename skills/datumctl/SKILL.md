---
name: datumctl
description: >
  Lightweight routing skill for direct `datumctl` usage. Use when an agent
  needs to run Datum Cloud CLI commands, understand the authentication and
  context model, or decide whether to use the broader `datum-cloud/skills`
  repository or `datum-mcp` instead.
metadata:
  homepage: https://github.com/datum-cloud/datumctl
  source: https://github.com/datum-cloud/datumctl
---

# Datumctl

Use this skill when the task is specifically about running `datumctl`
commands directly.

## Choose the right integration

- For direct CLI execution, use `datumctl`.
- For a broader set of Datum-focused skills and agent installation targets, use
  `datum-cloud/skills`.
- For tool-based integrations in MCP-capable environments, use `datum-mcp`.

## Datumctl-specific rules

- Authenticate with `datumctl login` for interactive use.
- For headless or automation workflows, prefer machine-account authentication
  via `datumctl auth login --credentials <file>`.
- Most resource commands need either an active context or explicit
  `--organization` / `--project` flags.
- Prefer `-o json` or `-o yaml` when another tool or agent needs structured
  output.

## Safe starting points

```bash
datumctl version
datumctl login
datumctl get organizations
datumctl ctx
datumctl whoami
```
