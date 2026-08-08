// Package bots defines the bot roster and a runner that executes bots and
// tracks their runs, backing the CLI's `graphify bot` commands and the web
// dashboard. Bots are executed by re-invoking this same graphify binary
// (`graphify bot <name> ...`), so the CLI and web paths run identical code.
package bots

// ArgDef declares one input a bot accepts, so the dashboard can render a
// form and the runner can validate before starting.
type ArgDef struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder,omitempty"`
}

// Def is a bot's static definition (mirrors API_CONTRACT.md's BotDef).
type Def struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`      // "go-native" | "python"
	NeedsAuth   bool     `json:"needsAuth"` // needs a working claude session
	Args        []ArgDef `json:"args"`

	// subcommand is how this bot is invoked as `graphify bot <subcommand>`.
	// Positional args are appended in argOrder; flag args are passed as
	// --name value. Kept unexported — it's an execution detail, not API.
	subcommand string
	// positional lists ArgDef names passed positionally (in order) rather
	// than as flags. Everything else is passed as --name value.
	positional []string
}

// Registry is the built-in bot roster. Order here is the display order.
var Registry = []Def{
	{
		Name:        "doctor",
		Title:       "Doctor",
		Description: "Preflight: verify the whole orchestration chain — python, gh, claude, and graphify's MCP server all connect.",
		Kind:        "go-native", // it shells to python, but from the UI's view it needs no args and no auth
		NeedsAuth:   false,
		subcommand:  "doctor",
	},
	{
		Name:        "graph-sync",
		Title:       "Graph Sync",
		Description: "Re-index the code graph and check its integrity (dangling edges, resolution quality). No AI, no auth.",
		Kind:        "go-native",
		NeedsAuth:   false,
		subcommand:  "graph-sync",
	},
	{
		Name:        "pr-review",
		Title:       "PR Review",
		Description: "Review an open PR's diff for breaking changes, quality, duplication, rewrites, pattern-mismatch, and cross-boundary contract mismatches. Posts a comment (or use dry-run).",
		Kind:        "python",
		NeedsAuth:   true,
		Args: []ArgDef{
			{Name: "pr_number", Label: "PR number", Required: true, Placeholder: "e.g. 12"},
			{Name: "dry_run", Label: "Dry run (print instead of posting) — 'true'/'false'", Required: false, Placeholder: "true"},
		},
		subcommand: "pr-review",
		positional: []string{"pr_number"},
	},
}

// Lookup returns the bot definition for name, or ok=false.
func Lookup(name string) (Def, bool) {
	for _, d := range Registry {
		if d.Name == name {
			return d, true
		}
	}
	return Def{}, false
}
