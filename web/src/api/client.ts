import {
  ALL_EDGES,
  ALL_SYMBOLS,
  MEMBERS_BY_PARENT,
  MOCK_STATS,
  MOCK_TREE,
  SYMBOLS_BY_FILE,
  SYMBOLS_BY_ID,
  mockCallers,
  mockCalls,
  mockSearch,
  mockSubgraph,
} from "../mock/data"
import { getMockSourceLines } from "../mock/source"
import type {
  CallEdgeEntry,
  Edge,
  GraphResponse,
  StatsResponse,
  SubgraphResponse,
  Symbol,
  SymbolKind,
  SymbolRef,
  SourceResponse,
  TreeNode,
} from "./types"

const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true"

// All real requests go through the Vite dev proxy at /api (see vite.config.ts),
// which forwards to http://localhost:8420.
const API_BASE = "/api"

const MOCK_DELAY_MS = 120

function delay<T>(value: T): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), MOCK_DELAY_MS))
}

class ApiRequestError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
    this.name = "ApiRequestError"
  }
}

async function request<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`)
  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json()
      if (body && typeof body.error === "string") message = body.error
    } catch {
      // ignore malformed error body
    }
    throw new ApiRequestError(res.status, message)
  }
  return res.json() as Promise<T>
}

export async function getTree(): Promise<TreeNode> {
  if (USE_MOCK) return delay(MOCK_TREE)
  return request<TreeNode>("/tree")
}

export async function getFileSymbols(filePath: string): Promise<SymbolRef[]> {
  if (USE_MOCK) return delay(SYMBOLS_BY_FILE[filePath] ?? [])
  return request<SymbolRef[]>(`/files/${encodeURIComponentPath(filePath)}/symbols`)
}

export async function getSymbol(id: string): Promise<Symbol> {
  if (USE_MOCK) {
    const symbol = SYMBOLS_BY_ID[id]
    if (!symbol) return Promise.reject(new ApiRequestError(404, `symbol not found: ${id}`))
    return delay(symbol)
  }
  return request<Symbol>(`/symbols/${encodeURIComponent(id)}`)
}

export async function getSymbolMembers(id: string): Promise<SymbolRef[]> {
  if (USE_MOCK) return delay(MEMBERS_BY_PARENT[id] ?? [])
  return request<SymbolRef[]>(`/symbols/${encodeURIComponent(id)}/members`)
}

export async function getSymbolCalls(id: string): Promise<CallEdgeEntry[]> {
  if (USE_MOCK) return delay(mockCalls(id))
  return request<CallEdgeEntry[]>(`/symbols/${encodeURIComponent(id)}/calls`)
}

export async function getSymbolCallers(id: string): Promise<CallEdgeEntry[]> {
  if (USE_MOCK) return delay(mockCallers(id))
  return request<CallEdgeEntry[]>(`/symbols/${encodeURIComponent(id)}/callers`)
}

export async function getGraph(): Promise<GraphResponse> {
  if (USE_MOCK) return delay({ nodes: ALL_SYMBOLS, edges: ALL_EDGES satisfies Edge[] })
  return request<GraphResponse>("/graph")
}

export async function getSubgraph(symbolId: string, depth = 2): Promise<SubgraphResponse> {
  if (USE_MOCK) return delay(mockSubgraph(symbolId, depth))
  return request<SubgraphResponse>(`/graph/subgraph?symbol=${encodeURIComponent(symbolId)}&depth=${depth}`)
}

export async function search(q: string, kind?: SymbolKind): Promise<SymbolRef[]> {
  if (USE_MOCK) return delay(mockSearch(q, kind))
  const params = new URLSearchParams({ q })
  if (kind) params.set("kind", kind)
  return request<SymbolRef[]>(`/search?${params.toString()}`)
}

export async function getSource(filePath: string, start: number, end: number): Promise<SourceResponse> {
  if (USE_MOCK) {
    return delay({ filePath, startLine: start, lines: getMockSourceLines(filePath, start, end) })
  }
  const params = new URLSearchParams({ file: filePath, start: String(start), end: String(end) })
  return request<SourceResponse>(`/source?${params.toString()}`)
}

export async function getStats(): Promise<StatsResponse> {
  if (USE_MOCK) return delay(MOCK_STATS)
  return request<StatsResponse>("/stats")
}

// Encodes a repo-relative path for use after `/files/` while preserving `/`
// separators, matching the `*path` wildcard route in the contract.
function encodeURIComponentPath(path: string): string {
  return path.split("/").map(encodeURIComponent).join("/")
}

export { ApiRequestError }
