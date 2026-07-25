# LoopBudget CLI

Public, zero-Node tooling for [LoopBudget](https://loopbudget.com).

The app platform is private; this repo is the public source + releases for client connectors.

## `loopbudget-claude-code`

Claude Code **Stop** hook → LoopBudget `/api/ingest`.

```bash
curl -fsSL https://raw.githubusercontent.com/LoopBudget/cli/main/install.sh | VERSION=0.1.0 bash
export PATH="$HOME/.loopbudget/bin:$PATH"
loopbudget-claude-code init
loopbudget-claude-code doctor
```

Wire `~/.claude/settings.json` (no secrets):

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

Threat model: [SECURITY.md](./SECURITY.md)

### Build from source

```bash
go build -o loopbudget-claude-code .
./scripts/release.sh
```
