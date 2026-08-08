# graphify bots

AI bots that operate on this repo (or any repo graphify has indexed),
using graphify's MCP server as their source of code context — search the
graph, don't read whole files — instead of a raw file-reading agent loop.

Each bot is a standalone Python script (stdlib only, no `pip install`
needed) that:
1. Shells out to `graphify parse`/`graphify mcp` for repo context.
2. Shells out to `claude -p` (headless Claude Code) with a task-specific
   prompt and an MCP config scoped to graphify's tools.
3. Does something with the result — usually posting to GitHub via `gh`.

## Auth

Bots authenticate however the `claude` CLI on the machine running them is
authenticated:
- **Local/manual runs**: your existing `claude login` session. If it's
  expired, run `claude` interactively once to refresh it.
- **CI (GitHub Actions)**: an interactive login can't work unattended, so
  CI sets `ANTHROPIC_API_KEY` as a repo secret instead — the `claude` CLI
  picks it up automatically over OAuth when present.

`gh` (GitHub CLI) must also be authenticated — locally that's whatever
`gh auth login` you already have; in Actions it's the built-in
`GITHUB_TOKEN`.

## Bots

| Bot | Script | Trigger | Status |
|---|---|---|---|
| PR Review | `pr_review.py` | `graphify bot pr-review <pr>`, or automatically on push via `.github/workflows/pr-review-bot.yml` | ✅ working |
| Commit Check | — | — | planned |
| Test Writer | — | — | planned |
| Graph Sync | — | — | planned |
| Anomaly Detector | — | — | planned |
| Data Supply (MCP) | — | — | planned — this is graphify's existing `graphify mcp` server, already usable by any MCP client |
| Support Triage (GitHub issues + chat/Intercom) | — | — | planned — Intercom needs to be authorized first |
| Feature Verdict (PRD/impact analysis) | — | — | planned |

### PR Review

Reviews an open PR's diff for: breaking changes (via `get_callers` on
anything touched), code quality, duplication (via `search_symbol` before
flagging), unnecessary rewrites, pattern-consistency with the rest of the
codebase, and cross-boundary contract mismatches (e.g. a frontend assuming
a shape the backend doesn't actually send).

```sh
# from the repo root, with graphify built (go build -o graphify ./cmd/graphify)
graphify bot pr-review 12          # posts a comment on PR #12
graphify bot pr-review 12 --dry-run  # prints the review instead of posting
```

Runs automatically on every PR push once `ANTHROPIC_API_KEY` is set as a
repo secret — see `.github/workflows/pr-review-bot.yml`.
