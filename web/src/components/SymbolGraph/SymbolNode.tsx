import { useState, type CSSProperties, type MouseEvent } from "react"
import { useNavigate } from "react-router-dom"
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react"
import type { Symbol, SymbolKind } from "../../api/types"
import { getSymbol } from "../../api/client"
import { KIND_LABEL, kindColor } from "../../utils/style"
import "./SymbolNode.css"

export type NodeRole = "entry" | "leaf" | "normal"

export type SymbolNodeData = {
  label: string
  kind: SymbolKind
  filePath: string
  isCenter?: boolean
  role?: NodeRole
  // True when a scope filter is active and this node was pulled in only
  // as context for an in-scope node (a "side dependency"), not because it
  // belongs to the selected scope itself.
  dimmed?: boolean
  // Pre-loaded full detail (e.g. the local graph already has it for its
  // center symbol) — skips the on-click fetch when present.
  preloaded?: Symbol
}

export type SymbolFlowNode = Node<SymbolNodeData, "symbol">

// Clicking a node expands it in place to show its signature/params/returns
// — "what it receives and emits" — without leaving the graph. Navigating to
// the full detail page (callers/callees/source) is a separate, explicit
// action so the two intents don't fight over a single click.
export function SymbolNode({ id, data }: NodeProps<SymbolFlowNode>) {
  const navigate = useNavigate()
  const [expanded, setExpanded] = useState(false)
  const [detail, setDetail] = useState<Symbol | null>(data.preloaded ?? null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)

  function toggle() {
    const next = !expanded
    setExpanded(next)
    if (next && !detail && !loading) {
      setLoading(true)
      setError(false)
      getSymbol(id)
        .then(setDetail)
        .catch(() => setError(true))
        .finally(() => setLoading(false))
    }
  }

  function openFull(e: MouseEvent) {
    e.stopPropagation()
    navigate(`/symbol/${encodeURIComponent(id)}`)
  }

  const roleClass = data.role && data.role !== "normal" ? `role-${data.role}` : ""

  return (
    <div
      className={`symbol-node ${data.isCenter ? "center" : ""} ${roleClass} ${expanded ? "expanded" : ""} ${data.dimmed ? "dimmed" : ""}`}
      style={{ "--node-color": kindColor(data.kind) } as CSSProperties}
      onClick={toggle}
    >
      <Handle type="target" position={Position.Left} />
      <div className="symbol-node-head">
        <div className="symbol-node-headtext">
          <div className="symbol-node-kind">
            {KIND_LABEL[data.kind] ?? data.kind}
            {data.role === "entry" && <span className="symbol-node-role-badge">ENTRY</span>}
          </div>
          <div className="symbol-node-name">{data.label}</div>
          <div className="symbol-node-file">{data.filePath}</div>
        </div>
        {!data.isCenter && (
          <button type="button" className="symbol-node-open" title="Open full detail" onClick={openFull}>
            ↗
          </button>
        )}
      </div>

      {expanded && (
        <div className="symbol-node-expanded" onClick={(e) => e.stopPropagation()}>
          {loading && <div className="symbol-node-expanded-status">Loading…</div>}
          {error && <div className="symbol-node-expanded-status error">Failed to load</div>}
          {detail && (
            <>
              {detail.doc && <div className="symbol-node-doc">{detail.doc}</div>}
              <div className="symbol-node-io">
                <div>
                  <div className="symbol-node-io-label">in</div>
                  {detail.params.length === 0 && <div className="symbol-node-io-empty">—</div>}
                  {detail.params.map((p, i) => (
                    <div key={i} className="symbol-node-io-row">
                      {p.name && <span className="symbol-node-io-name">{p.name}</span>}
                      {p.type && <span className="symbol-node-io-type">{p.type}</span>}
                    </div>
                  ))}
                </div>
                <div>
                  <div className="symbol-node-io-label">out</div>
                  {detail.returns.length === 0 && <div className="symbol-node-io-empty">—</div>}
                  {detail.returns.map((p, i) => (
                    <div key={i} className="symbol-node-io-row">
                      {p.name && <span className="symbol-node-io-name">{p.name}</span>}
                      {p.type && <span className="symbol-node-io-type">{p.type}</span>}
                    </div>
                  ))}
                </div>
              </div>
              {!data.isCenter && (
                <button type="button" className="symbol-node-expanded-open" onClick={openFull}>
                  Open full detail →
                </button>
              )}
            </>
          )}
        </div>
      )}

      <Handle type="source" position={Position.Right} />
    </div>
  )
}
