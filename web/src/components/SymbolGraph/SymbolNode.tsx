import type { CSSProperties } from "react"
import { Handle, Position, type NodeProps, type Node } from "@xyflow/react"
import type { SymbolKind } from "../../api/types"
import { KIND_LABEL, kindColor } from "../../utils/style"
import "./SymbolNode.css"

export type SymbolNodeData = {
  label: string
  kind: SymbolKind
  filePath: string
  isCenter?: boolean
}

export type SymbolFlowNode = Node<SymbolNodeData, "symbol">

export function SymbolNode({ data }: NodeProps<SymbolFlowNode>) {
  return (
    <div className={`symbol-node ${data.isCenter ? "center" : ""}`} style={{ "--node-color": kindColor(data.kind) } as CSSProperties}>
      <Handle type="target" position={Position.Left} />
      <div className="symbol-node-kind">{KIND_LABEL[data.kind] ?? data.kind}</div>
      <div className="symbol-node-name">{data.label}</div>
      <div className="symbol-node-file">{data.filePath}</div>
      <Handle type="source" position={Position.Right} />
    </div>
  )
}
