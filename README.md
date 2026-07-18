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

## Development

```sh
go test ./...                    # backend
cd web && npm run build          # frontend typecheck + build
```

See [API_CONTRACT.md](API_CONTRACT.md) for the frozen REST contract between
`internal/server` and `web/`.
