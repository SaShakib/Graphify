import { Link } from "react-router-dom"
import "./HomeView.css"

export function HomeView() {
  return (
    <div className="home-view">
      <h1>graphify</h1>
      <p>AI code knowledge graph &amp; visual debugger.</p>
      <p className="home-hint">
        Pick a file from the tree on the left, press <kbd>⌘K</kbd> to search for a symbol, or open the{" "}
        <Link to="/graph">full call graph</Link>.
      </p>
    </div>
  )
}
