# Graphify Orchestration — Status & Roadmap

This is the single source of truth for the AI engineering-intelligence
platform being built on top of graphify: what's done, what's verified, and
what's next. It maps every stated requirement to its concrete status.

Legend: ✅ done & verified · 🟡 partial/scaffolded · ⬜ not started · 🔒 blocked (needs you)

## 0. Foundation (the graph itself) — ✅

| Capability | Status | Where |
|---|---|---|
| Parse repo into symbol/call graph (Go, TS, TSX, JS, Python) | ✅ | `internal/parser` |
| Top-level symbols only — no library/system internals | ✅ | `internal/indexer` skips deps; parsers only emit declared top-level symbols |
| Route-level detection (Go `net/http`) | ✅ | `internal/parser/golang.go` |
| Graph stored in a DB | ✅ SQLite | `internal/store` |
| Incremental re-index (content hash) | ✅ | `internal/indexer` |
| MCP server so AI searches the graph instead of reading files | ✅ | `internal/mcp` — 8 tools |
| Web view of the graph | ✅ | `web/` (React + @xyflow/react) |
| CLI | ✅ | `cmd/graphify` — parse/serve/mcp/bot |

## 1. "App can connect a Claude session from the terminal" — ✅ mechanism / 🔒 credential

The core loop — spawn a headless `claude -p` session, hand it graphify's
MCP server, let it query the graph — is built and proven. Verify it any
time with:

```sh
graphify bot doctor
```

Current result on this machine: **5/6 green**. The graphify MCP server
responds with all 8 tools (the app side of the connection is fully
working). The one red check is the local `claude` CLI login being expired
— an interactive re-auth only you can do (`claude` once interactively, or
set `ANTHROPIC_API_KEY`). Nothing in graphify's code can fix a login.

## 2. Bots — 🟡 (1 of 8 built, mechanism reusable for the rest)

| # | Bot | Does | Status |
|---|---|---|---|
| 0 | **Doctor** | preflight: verify the whole chain connects | ✅ |
| 1 | **PR Review** | review a PR diff for breaking changes, quality, duplication, unnecessary rewrites, pattern-mismatch, cross-boundary contract ("hallucinated") mismatches → posts a comment | ✅ built, 🔒 on claude auth |
| 2 | **Commit Check** | same checks, per-commit granularity | ⬜ |
| 3 | **Test Writer** | generate test cases for changed code | ⬜ |
| 4 | **Graph Sync** | re-index the graph on push; verify graph integrity (dangling edges, resolution quality) | ✅ Go-native, no auth needed |
| 5 | **Anomaly Detector** | scan whole codebase for abnormalities / possible breakings | ⬜ |

> The Graph Sync bot already earned its keep: on first run it found a real
> correctness bug in graphify itself — a Go type's methods declared across
> multiple files of a package were orphaned onto a phantom same-file parent
> (12 dangling edges; `Store`'s member list was split 8/12). Fixed in
> `internal/graph/builder.go` (`resolveParents`), with a regression test.
| 6 | **Data Supply** | feed graph context to a coding agent | 🟡 this is the existing `graphify mcp` server; needs a documented "agent bootstrap" wrapper |
| 7 | **Support Triage (MCP)** | pull GitHub issues/PRs + chat/Intercom, correlate to code, suggest fixes | ⬜ (GitHub ready; Intercom 🔒 needs auth) |
| 8 | **Feature Verdict** | given a proposed feature: what rules it might break, more optimal options, final PRD, what to test so nothing breaks | ⬜ (needs the memory system below) |

Each bot is a stdlib-only Python script in `bots/`, invoked via `graphify
bot <name>` and wireable to CI. The PR Review bot is the reference
implementation for all the review-style ones (2, 5).

## 3. Memory system — ⬜ (designed, not built)

Two stores, as specified:
- **Code graph** → SQLite. ✅ already exists (`internal/store`).
- **Vector memory** → lessons, codebase primary rules, "what this software
  does", per-business-logic data + logical dependencies. ⬜ Not started.
  Decision needed: embedding model + vector store (see Open Decisions).

The intended payoff — "optimize by giving less read: AI searches for where
code lies instead of reading it all" — is *already delivered for code* by
the MCP server. Vector memory extends the same idea to non-code knowledge
(rules, lessons, business logic).

## 4. Control surfaces — 🟡

| Surface | Status |
|---|---|
| CLI session to run/control bots | ✅ `graphify bot ...` |
| Web view of the graph | ✅ `graphify serve` |
| Web view to **control bots** (dashboard: trigger, see runs/output) | ⬜ |

## Open decisions (need your input before building)

1. **claude auth**: re-login personal session (simplest, local-only) vs
   `ANTHROPIC_API_KEY` (works in CI). CI runs *require* the key regardless.
2. **Vector memory stack**: since bots are Python, the natural choice is a
   Python embedding lib + a local vector store (e.g. sqlite-vec, or
   Chroma/LanceDB). Which embedding provider?
3. **Bot build order**: Commit Check, Graph Sync, and the Support-Triage
   MCP are the cheapest next steps; Feature Verdict and Anomaly Detector
   depend on the vector memory existing first.

## How to verify everything yourself

```sh
go build -o graphify ./cmd/graphify   # build
go test ./...                          # backend tests (all green)
graphify bot doctor                    # verify the orchestration chain
graphify serve .                       # open the graph web view
graphify bot pr-review <pr> --dry-run  # run the PR bot (once claude auth is fixed)
```
