# graphify

**AI Code Knowledge Graph & Visual Debugger.**

graphify parses a source repo into a symbol/call graph — modules, classes,
functions, methods, consts, vars, with types, call edges, and file:line
references — instead of treating source as flat text. It's built for two
consumers:

- **Humans**, via an interactive graph visualizer: browse the file tree,
  open a module to see its declared symbols, drill into a function to see
  its signature, doc comment, callers/callees as a connected graph, and the
  exact source lines, all with line-level links back to the file.
- **AI agents**, via an MCP server exposing `search_symbol`, `get_symbol`,
  `get_file_symbols`, `get_callers`, `get_callees`, `get_file_slice`,
  `get_tree`, `get_stats` — so debugging or reviewing AI-written code means
  *querying the graph* (search → inspect → trace callers/callees → read the
  exact lines) instead of re-reading whole files to rebuild context.

Everything is derived from the same underlying graph — the web UI and the
MCP tools both read from one SQLite-backed index built by the same parser.

## Architecture

```
cmd/graphify/       CLI (cobra): parse / serve / mcp subcommands
internal/parser/    tree-sitter based extraction: Go, TypeScript, TSX, JavaScript, Python
internal/graph/      language-agnostic symbol/edge model + call resolution
internal/indexer/    walks a repo, parses changed files (content-hash), prunes deleted ones
internal/store/      SQLite persistence + read queries backing both the API and MCP server
internal/server/    REST API (see API_CONTRACT.md) + serves the built web UI
internal/mcp/        MCP server exposing the graph as tools for AI agents
web/                 React + TypeScript + Vite graph visualizer (@xyflow/react)
```

Call graph edges are resolved by name: a call site's bare identifier is
matched to a declared symbol preferring the same file, then the same
directory, then falling back to a unique repo-wide match. Ambiguous names
with no same-file/dir winner are left unresolved rather than guessing.

## Quick start

Requires Go 1.22+ and Node 18+.

```sh
# build the CLI
go build -o graphify ./cmd/graphify

# build the web UI once (graphify serve auto-detects web/dist)
cd web && npm install && npm run build && cd ..

# parse + serve a repo (defaults to the current directory)
./graphify serve /path/to/some/repo
# -> opens http://localhost:8420
```

Or point an MCP-capable agent (e.g. Claude Code) at:

```sh
./graphify mcp /path/to/some/repo
```

as the launch command for a stdio MCP server.

### CLI commands

- `graphify parse [path]` — index a repo into `.graphify/graph.db` and exit.
- `graphify serve [path] [--port 8420] [--web-dir ...] [--open]` — index, then
  serve the REST API and (if built) the web UI.
- `graphify mcp [path]` — index, then run an MCP server over stdio.

All three share `--db` to override the default `.graphify/graph.db` path,
and re-run incrementally: unchanged files (by content hash) aren't
re-parsed, and files deleted since the last run are pruned.

### Bots

AI and graph bots that operate on the indexed repo:

- `graphify bot doctor` — verify the whole chain (claude, gh, MCP) connects.
- `graphify bot graph-sync` — re-index + integrity check (no AI, no auth).
- `graphify bot pr-review <pr> [--dry-run]` — review a PR diff, post a comment.
- `graphify bot commit-check [ref]` — review a single commit's diff.
- `graphify bot test-writer <symbol|file> [--write]` — generate tests.
- `graphify bot anomaly-scan [--focus X]` — audit the codebase for risks.
- `graphify bot feature-verdict "<feature>"` — plan a feature vs. the codebase.
- `graphify bot triage <issue#> [--comment]` — correlate an issue to code.

All also runnable from the web **Bots** dashboard. The AI bots use graphify's
MCP tools (graph + memory) as their repo context. See
[bots/README.md](bots/README.md) for the full roster and auth, and
[ORCHESTRATION.md](ORCHESTRATION.md) for the overall status.

### Vector memory

Non-code knowledge — codebase rules, lessons, business logic, overviews —
stored as embeddings and searched semantically (the counterpart to the code
graph). Kept in `.graphify/memory.db`.

- `graphify memory add --kind rule --title "..." --text "..."`
- `graphify memory search "how are call edges resolved"`
- `graphify memory list` · `graphify memory rm <id>`

Also available via the web **Memory** tab, the REST API (`/api/memory`), and
MCP tools (`memory_search`/`memory_add`) so agents recall and persist
knowledge. The default embedder is offline and dependency-free; set
`GRAPHIFY_EMBED_URL`/`GRAPHIFY_EMBED_MODEL` (any OpenAI-compatible endpoint,
e.g. a local Ollama) to upgrade to neural embeddings — no code change.

## Development

```sh
go test ./...                    # backend
cd web && npm run build          # frontend typecheck + build
```

See [API_CONTRACT.md](API_CONTRACT.md) for the frozen REST contract between
`internal/server` and `web/`.
