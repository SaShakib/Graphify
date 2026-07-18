// Types copied verbatim from /Users/default/Downloads/shakib/projects/graphify/API_CONTRACT.md
// Do NOT diverge from field names/types here without updating that file.

export type SymbolKind =
  | "module" // a source file
  | "package" // a directory / package
  | "class"
  | "interface"
  | "function"
  | "method"
  | "const"
  | "variable"

export interface Param {
  name: string
  type: string // "" if unknown/untyped
}

export interface SymbolRef {
  id: string // stable id, e.g. "go:internal/graph/builder.go:BuildGraph"
  name: string
  kind: SymbolKind
  filePath: string // repo-relative path
  startLine: number // 1-indexed
  endLine: number
}

export interface Symbol extends SymbolRef {
  signature: string // rendered signature, e.g. "func BuildGraph(files []File) (*Graph, error)"
  params: Param[] // inputs
  returns: Param[] // outputs
  receiver?: string // for methods, e.g. "*Builder"
  parentId?: string // enclosing class/module symbol id
  language: string // "go" | "typescript" | "javascript" | "python"
  doc?: string // leading comment/docstring, if present
}

export type EdgeKind = "calls" | "references" | "contains" | "implements" | "extends"

export interface Edge {
  source: string // symbol id
  target: string // symbol id
  kind: EdgeKind
}

export interface TreeNode {
  path: string // repo-relative path, "" for repo root
  name: string
  type: "dir" | "file"
  language?: string // set when type === "file"
  children?: TreeNode[] // set when type === "dir"
}

export interface CallEdgeEntry {
  edge: Edge
  symbol: SymbolRef
}

export interface GraphResponse {
  nodes: SymbolRef[]
  edges: Edge[]
}

export interface SubgraphResponse extends GraphResponse {
  center: string
}

export interface SourceResponse {
  filePath: string
  startLine: number
  lines: string[]
}

export interface StatsResponse {
  files: number
  symbols: number
  edges: number
  languages: Record<string, number>
}

export interface ApiError {
  error: string
}
