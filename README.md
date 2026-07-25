# LoopBudget CLI

Public, zero-Node tooling for [LoopBudget](https://loopbudget.com).

| Binary | Role |
| --- | --- |
| `loopbudget-claude-code` | Claude Code **Stop** hook → `/api/ingest` |
| `loopbudget-cursor` | Cursor transcript sidecar → `/api/ingest` |

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/LoopBudget/cli/main/install.sh | VERSION=0.2.0 bash
export PATH="$HOME/.loopbudget/bin:$PATH"
loopbudget-claude-code init   # writes ~/.loopbudget/credentials (mode 600)
```

Install one tool only: `TOOLS=cursor VERSION=0.2.0 bash …`

## Claude Code

```json
{
  "hooks": {
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "loopbudget-claude-code stop-hook"
      }]
    }]
  }
}
```

## Cursor

```bash
# credentials already from init — optional filter:
PROJECT_FILTER=my-project loopbudget-cursor
```

## Threat model

[SECURITY.md](./SECURITY.md)

## Build

```bash
go test ./...
go build -o loopbudget-claude-code ./cmd/loopbudget-claude-code
go build -o loopbudget-cursor ./cmd/loopbudget-cursor
./scripts/release.sh
```
