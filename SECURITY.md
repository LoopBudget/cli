# Security model — LoopBudget CLI

Static Go binaries. No Node runtime.

## Shared

| Resource | Access |
| --- | --- |
| `~/.loopbudget/credentials` | Read (mode must be `600`) |
| Network | `POST {allowlisted-origin}/api/ingest` only |

URL allowlist: `loopbudget.com` / `www` (HTTPS) + loopback. API keys must not appear on argv (`init` writes the credentials file).

## `loopbudget-claude-code`

| Resource | Access |
| --- | --- |
| Claude Stop stdin | `session_id`, `transcript_path` |
| Transcript path from stdin | Read-only |
| `~/.loopbudget/claude-code-state.json` | Token deltas |

## `loopbudget-cursor`

| Resource | Access |
| --- | --- |
| `~/.cursor/projects/**/agent-transcripts/*.jsonl` | Read (byte-offset tail) |
| `~/.loopbudget/cursor-sidecar-state.json` | Offsets |

Estimates tokens from transcript text (chars÷4). Does **not** upload transcript bodies — only derived counts.

## Trust chain

1. Install a **version-pinned** release from [GitHub Releases](https://github.com/LoopBudget/cli/releases); verify `SHA256SUMS`.
2. Prefer `loopbudget-claude-code init` over putting keys in shell history / settings JSON.
3. Source: https://github.com/LoopBudget/cli

## Reporting

https://github.com/LoopBudget/cli/issues
