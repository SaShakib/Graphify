// Builds an "Open in editor" link from a repo-relative filePath + line.
//
// The API only ever returns repo-relative paths (see API_CONTRACT.md), so the
// absolute repo root prefix below is a placeholder. In a real deployment this
// should be injected via config (e.g. VITE_REPO_ROOT) so the link resolves to
// the developer's actual checkout on disk.
const REPO_ROOT_PLACEHOLDER = "/absolute/path/to/repo"

export function editorLink(filePath: string, line: number): string {
  return `vscode://file/${REPO_ROOT_PLACEHOLDER}/${filePath}:${line}`
}

// Contract also documents a generic `graphify://open?file=..&line=..` scheme
// for tools/agents that aren't specifically VS Code.
export function graphifyOpenLink(filePath: string, line: number): string {
  const params = new URLSearchParams({ file: filePath, line: String(line) })
  return `graphify://open?${params.toString()}`
}
