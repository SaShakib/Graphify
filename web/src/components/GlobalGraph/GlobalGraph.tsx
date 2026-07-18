import { useMemo } from "react"
import { useNavigate } from "react-router-dom"
import { Background, Controls, MiniMap, ReactFlow, type Edge as RFEdge, type NodeTypes } from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import type { Edge, SymbolKind, SymbolRef } from "../../api/types"
import { layoutWithDagre } from "../../utils/dagreLayout"
import { edgeColor, kindHexColor } from "../../utils/style"
import { SymbolNode, type SymbolFlowNode } from "../SymbolGraph/SymbolNode"
import { GraphLegend } from "./GraphLegend"
import "./GlobalGraph.css"

const nodeTypes: NodeTypes = { symbol: SymbolNode }

export function GlobalGraph({
  nodes: symbolNodes,
  edges: symbolEdges,
  centerId,
}: {
  nodes: SymbolRef[]
  edges: Edge[]
  centerId?: string
}) {
  const navigate = useNavigate()

  const { nodes, edges } = useMemo(() => {
    const rawNodes: SymbolFlowNode[] = symbolNodes.map((s) => ({
      id: s.id,
      type: "symbol",
      position: { x: 0, y: 0 },
      data: { label: s.name, kind: s.kind, filePath: s.filePath, isCenter: s.id === centerId },
    }))
    const rfEdges: RFEdge[] = symbolEdges.map((e, i) => ({
      id: `${e.source}->${e.target}-${i}`,
      source: e.source,
      target: e.target,
      label: e.kind,
      style: { stroke: edgeColor(e.kind) },
      markerEnd: { type: "arrowclosed" as const, color: edgeColor(e.kind) },
    }))
    const laidOut = layoutWithDagre(rawNodes, rfEdges, "LR")
    return { nodes: laidOut, edges: rfEdges }
  }, [symbolNodes, symbolEdges, centerId])

  return (
    <div className="global-graph">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => navigate(`/symbol/${encodeURIComponent(node.id)}`)}
        fitView
        proOptions={{ hideAttribution: true }}
        minZoom={0.1}
        maxZoom={2}
      >
        <Background gap={24} size={1} />
        <Controls showInteractive={false} />
        <MiniMap
          pannable
          zoomable
          nodeColor={(n) => {
            const data = n.data as { kind?: SymbolKind } | undefined
            return data?.kind ? kindHexColor(data.kind) : "#888"
          }}
        />
      </ReactFlow>
      <GraphLegend />
    </div>
  )
}
