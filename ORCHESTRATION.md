# Graphify Orchestration — Status

The AI engineering-intelligence platform built on top of graphify's code
graph. This maps every requirement to its concrete, verifiable status.

Legend: ✅ done & verified · 🔒 blocked on your `claude` login (code complete)

## 0. Foundation — the code graph ✅

| Capability | Where |
|---|---|
| Parse repo into a symbol/call graph (Go, TS, TSX, JS, Python) | `internal/parser` |
| Top-level symbols only — no library/system internals | `internal/indexer` (skips deps), parsers emit only declared symbols |
| Route-level detection (Go `net/http`) | `internal/parser/golang.go` |
| Cross-file method→type resolution | `internal/graph` (`resolveParents`) |
| Graph in SQLite, incremental re-index by content hash | `internal/store`, `internal/indexer` |
| MCP server: AI searches the graph instead of reading files | `internal/mcp` |
| Web graph visualizer | `web/` (React + @xyflow/react) |

## 1. "App can connect a Claude session from the terminal" ✅ / 🔒 credential

Proven and self-verifying. Run:

```sh
graphify bot doctor
```

It checks the whole chain (python, gh, claude, and graphify's own MCP
server) and reports each link. On this machine: **5/6 green** — the MCP
server responds with all its tools (the app side works); the one red is the
expired local `claude` login, which only you can refresh (`claude`
interactively, or set `ANTHROPIC_API_KEY`). No code can fix a login.

## 2. Bots — ✅ all 8 built

Run any from the CLI (`graphify bot <name>`) or the web dashboard. Go-native
bots need no auth; AI bots preflight-check and fail gracefully until
`claude` auth is present.

| Bot | Does | Auth | Status |
|---|---|---|---|
| **doctor** | preflight: verify the whole chain connects | none | ✅ |
| **graph-sync** | re-index + integrity check (dangling edges, resolution) | none | ✅ verified in CI |
| **pr-review** | review a PR diff: breaks, quality, duplication, rewrites, pattern-mismatch, cross-boundary contract mismatches → posts a comment | claude | ✅ built · 🔒 |
| **commit-check** | same dimensions, per-commit | claude | ✅ built · 🔒 |
| **test-writer** | generate tests for a symbol/file using real callers | claude | ✅ built · 🔒 |
| **anomaly-scan** | whole-codebase audit for breakages, contract mismatches, duplication, rule violations | claude | ✅ built · 🔒 |
| **feature-verdict** | plan a feature: rules it breaks (memory), code to reuse, options, PRD, tests to stay safe | claude | ✅ built · 🔒 |
| **triage** | correlate a GitHub issue to code (graph+memory), suggest fixes | claude+gh | ✅ built · 🔒 |
| **Data Supply** | feed graph+memory context to a coding agent | — | ✅ *is* `graphify mcp` (point any MCP client at it) |

AI bots are stdlib-only Python in `bots/`, sharing `bots/common.py` (index →
hand claude the graphify MCP tools → run prompt → parse output). Go-native
bots live in `cmd/graphify`. All are in one registry (`internal/bots`), so
CLI and dashboard list the same set.

> graph-sync earned its keep immediately: on first run it found a real bug
> in graphify — Go methods spread across a package's files were orphaned
> onto a phantom parent (12 dangling edges). Fixed + regression-tested.

## 3. Memory system — ✅

The counterpart to the graph: the graph answers *where code lives*, memory
answers *what we know that isn't in the code* — primary rules, lessons,
business logic, overviews. Stored as embeddings, searched semantically.

- **Code graph** → SQLite (`internal/store`).
- **Vector memory** → SQLite + embeddings (`internal/memory`).
  - Default embedder: offline, dependency-free lexical hashing (feature
    hashing + light stemming + cosine, with a calibrated noise floor).
  - Optional neural embedder: set `GRAPHIFY_EMBED_URL`/`MODEL` (any
    OpenAI-compatible endpoint, e.g. a local Ollama) for true semantic
    recall — no code change, no cloud key required.
- Surfaces: CLI (`graphify memory add/search/list/rm`), MCP
  (`memory_search`/`memory_add` — bots and agents recall and persist
  knowledge), REST (`/api/memory`), and the web **Memory** tab.

## 4. Control surfaces — ✅

| Surface | Status |
|---|---|
| CLI to run bots + manage memory | ✅ `graphify bot ...`, `graphify memory ...` |
| Web graph view | ✅ `graphify serve` → Graph |
| Web bot dashboard (trigger, live output, run history) | ✅ Bots tab — verified end-to-end |
| Web memory view (search, add, delete) | ✅ Memory tab — verified end-to-end |

## The one thing left for you

Every AI bot is code-complete and stops at the same single wall: the local
`claude` login is expired. Refresh it (`claude` interactively, or export
`ANTHROPIC_API_KEY`), then any AI bot runs for real — e.g.
`graphify bot pr-review 1 --dry-run`, or the dashboard Run buttons. CI runs
require `ANTHROPIC_API_KEY` as a repo secret (personal logins can't run
unattended).

## Verify everything yourself

```sh
go build -o graphify ./cmd/graphify
go test ./...                             # all green
cd web && npm run build && cd ..          # frontend, zero TS errors
graphify bot doctor                       # the orchestration chain
graphify serve .                          # Graph + Bots + Memory tabs
graphify memory search "how are call edges resolved"   # semantic recall
graphify bot graph-sync                   # no-auth bot, runs fully
```

## Possible next steps (not required)

- Consolidate `bots/pr_review.py` onto `bots/common.py` (it predates the
  shared toolkit and duplicates a few helpers).
- Route detection for more frameworks (TS/Next.js, chi/gin) — currently Go
  `net/http`.
- Wire Intercom as a triage data source once it's authorized.
- Persist bot run history (currently in-memory per `serve` process).
