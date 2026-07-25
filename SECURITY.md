# Security model — `loopbudget-claude-code`

Static Go binary. No Node runtime. Treat it like any privileged local agent connector.

## What it can access

| Resource | Access |
| --- | --- |
| `~/.loopbudget/credentials` | Read (mode must be `600`) |
| Claude Code Stop stdin | `session_id`, `transcript_path` |
| Transcript file path from stdin | Read-only (`EvalSymlinks`) |
| `~/.loopbudget/claude-code-state.json` | Read/write token deltas |
| Network | `POST {allowlisted-origin}/api/ingest` only |

It does **not** shell out, install packages at runtime, or upload transcript text — only derived token counts and cost estimates.

## Trust chain (GA)

1. Install a **version-pinned** binary from [GitHub Releases](https://github.com/LoopBudget/cli/releases) (or the install script that verifies `SHA256SUMS`).
2. Prefer reviewing `SHA256SUMS` on the release page before piping install scripts.
3. **Never** put `LOOPBUDGET_API_KEY` in `.claude/settings.json` or on the hook `command` line.
4. Store secrets with `loopbudget-claude-code init` → `~/.loopbudget/credentials` (`chmod 600`). The CLI **refuses to start** if the key appears on argv.
5. **URL allowlist**: only `loopbudget.com` / `www.loopbudget.com` (HTTPS) and loopback for local dev.
6. Source: [`cli/loopbudget-claude-code`](https://github.com/LoopBudget/cli).

## What leaves your machine

JSON to `/api/ingest`: session id, profile name, connector kind, token deltas, cost estimate. No prompt/response bodies.

## Reporting issues

https://github.com/LoopBudget/loopbudget/issues
