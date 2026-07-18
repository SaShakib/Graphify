package graph

import "strings"

// Build merges every parsed file's symbols into one repo-wide Graph and
// resolves each UnresolvedCall to a concrete edge.
//
// Call sites only carry the bare identifier written at the call (e.g.
// "log" for "s.log(port)") — there's no type checker here, so resolution
// is name-based and deliberately conservative: candidates are restricted to
// the same language as the call site (a Go call can never resolve to a
// TypeScript symbol, however the names happen to collide — e.g. Go's
// "os.Stat" call and an unrelated React "Stat" component both reduce to
// the bare name "Stat"), then prefer a candidate in the same file, then the
// same directory (package), then — only for unqualified/bare calls — fall
// back to a unique match within that language. Ambiguous names with no
// same-file/dir winner are left unresolved rather than guessing and
// drawing a misleading edge.
func Build(files []*FileGraph) *Graph {
	var symbols []Symbol
	var calls []UnresolvedCall
	for _, fg := range files {
		symbols = append(symbols, fg.Symbols...)
		calls = append(calls, fg.UnresolvedCalls...)
	}
	return BuildFlat(symbols, calls)
}

// BuildFlat is the same resolution logic as Build but takes symbols/calls
// directly rather than grouped by file — used by the store, which persists
// and reloads them as flat rows rather than FileGraph batches.
func BuildFlat(symbols []Symbol, calls []UnresolvedCall) *Graph {
	g := &Graph{Symbols: make(map[string]Symbol, len(symbols))}
	byName := make(map[string][]Symbol)

	for _, sym := range symbols {
		g.Symbols[sym.ID] = sym
		byName[sym.Name] = append(byName[sym.Name], sym)
		if sym.ParentID != "" {
			g.Edges = append(g.Edges, Edge{Source: sym.ParentID, Target: sym.ID, Kind: EdgeContains})
		}
	}

	for _, call := range calls {
		from, ok := g.Symbols[call.FromID]
		if !ok {
			continue
		}
		target := resolveCall(from, call.TargetName, call.Qualified, byName)
		if target == "" {
			continue
		}
		g.Edges = append(g.Edges, Edge{Source: call.FromID, Target: target, Kind: call.Kind})
	}

	return g
}

func resolveCall(from Symbol, name string, qualified bool, byName map[string][]Symbol) string {
	family := languageFamily(from.Language)
	var candidates []Symbol
	for _, c := range byName[name] {
		if languageFamily(c.Language) == family {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return ""
	}

	fromDir := dirOf(from.FilePath)
	var sameFile, sameDir *Symbol
	for i := range candidates {
		c := &candidates[i]
		if c.FilePath == from.FilePath {
			sameFile = c
			break
		}
		if sameDir == nil && dirOf(c.FilePath) == fromDir {
			sameDir = c
		}
	}
	if sameFile != nil {
		return sameFile.ID
	}
	if sameDir != nil {
		return sameDir.ID
	}
	if !qualified && len(candidates) == 1 {
		return candidates[0].ID
	}
	return ""
}

// languageFamily groups languages that can actually call into each other at
// runtime. TypeScript/TSX/JavaScript all compile to the same JS runtime and
// commonly import across those extensions; Go and Python never call into
// that runtime (or each other) directly, so a same-named symbol there is
// always a coincidence, not a real call target.
func languageFamily(lang string) string {
	switch lang {
	case "typescript", "tsx", "javascript":
		return "js"
	default:
		return lang
	}
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return ""
}
