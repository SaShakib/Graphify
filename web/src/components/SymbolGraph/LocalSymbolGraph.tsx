import { useMemo } from "react"
import { useNavigate } from "react-router-dom"
import { Background, Controls, ReactFlow, type Edge as RFEdge, type NodeTypes } from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import type { CallEdgeEntry, Symbol } from "../../api/types"
import { edgeColor } from "../../utils/style"
import { SymbolNode, type SymbolFlowNode } from "./SymbolNode"
import "./LocalSymbolGraph.css"

const nodeTypes: NodeTypes = { symbol: SymbolNode }

const NODE_HEIGHT = 64
const COL_GAP = 260

export function LocalSymbolGraph({
  symbol,
  callers,
  calls,
}: {
  symbol: Symbol
  callers: CallEdgeEntry[]
  calls: CallEdgeEntry[]
}) {
  const navigate = useNavigate()

  const { nodes, edges } = useMemo(() => {
    const centerNode: SymbolFlowNode = {
      id: symbol.id,
      type: "symbol",
      position: { x: COL_GAP, y: Math.max(callers.length, calls.length, 1) * NODE_HEIGHT * 0.5 },
      data: { label: symbol.name, kind: symbol.kind, filePath: symbol.filePath, isCenter: true },
    }

    const callerNodes: SymbolFlowNode[] = callers.map((c, i) => ({
      id: c.symbol.id,
      type: "symbol",
      position: { x: 0, y: i * NODE_HEIGHT },
      data: { label: c.symbol.name, kind: c.symbol.kind, filePath: c.symbol.filePath },
    }))

    const calleeNodes: SymbolFlowNode[] = calls.map((c, i) => ({
      id: c.symbol.id,
      type: "symbol",
      position: { x: COL_GAP * 2, y: i * NODE_HEIGHT },
      data: { label: c.symbol.name, kind: c.symbol.kind, filePath: c.symbol.filePath },
    }))

    const rfEdges: RFEdge[] = [
      ...callers.map((c) => ({
        id: `${c.edge.source}->${c.edge.target}`,
        source: c.symbol.id,
        target: symbol.id,
        label: c.edge.kind,
        style: { stroke: edgeColor(c.edge.kind) },
        markerEnd: { type: "arrowclosed" as const, color: edgeColor(c.edge.kind) },
      })),
      ...calls.map((c) => ({
        id: `${c.edge.source}->${c.edge.target}`,
        source: symbol.id,
        target: c.symbol.id,
        label: c.edge.kind,
        style: { stroke: edgeColor(c.edge.kind) },
        markerEnd: { type: "arrowclosed" as const, color: edgeColor(c.edge.kind) },
      })),
    ]

    return { nodes: [...callerNodes, centerNode, ...calleeNodes], edges: rfEdges }
  }, [symbol, callers, calls])

  if (callers.length === 0 && calls.length === 0) {
    return <div className="local-graph-empty">No direct callers or callees for this symbol.</div>
  }

  return (
    <div className="local-symbol-graph">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => {
          if (node.id !== symbol.id) navigate(`/symbol/${encodeURIComponent(node.id)}`)
        }}
        fitView
        proOptions={{ hideAttribution: true }}
        minZoom={0.3}
      >
        <Background gap={20} size={1} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  )
}
