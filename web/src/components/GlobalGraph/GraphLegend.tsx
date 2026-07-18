import { EDGE_LABEL, edgeColor } from "../../utils/style"
import type { EdgeKind } from "../../api/types"
import "./GraphLegend.css"

const EDGE_KINDS: EdgeKind[] = ["calls", "references", "contains", "implements", "extends"]

export function GraphLegend() {
  return (
    <div className="graph-legend">
      <div className="graph-legend-title">Edge kinds</div>
      {EDGE_KINDS.map((kind) => (
        <div className="graph-legend-row" key={kind}>
          <span className="graph-legend-swatch" style={{ background: edgeColor(kind) }} />
          <span>{EDGE_LABEL[kind]}</span>
        </div>
      ))}
    </div>
  )
}
