package graph

import "testing"

func TestBuildResolvesCallsAndContains(t *testing.T) {
	files := []*FileGraph{
		{
			FilePath: "a.go",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "a.go:Caller", Name: "Caller", Kind: KindFunction}},
				{SymbolRef: SymbolRef{ID: "a.go:Server", Name: "Server", Kind: KindClass}},
				{SymbolRef: SymbolRef{ID: "a.go:Server.Start", Name: "Start", Kind: KindMethod}, ParentID: "a.go:Server"},
			},
			UnresolvedCalls: []UnresolvedCall{
				{FromID: "a.go:Caller", TargetName: "Helper", Kind: EdgeCalls},
			},
		},
		{
			FilePath: "b.go",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "b.go:Helper", Name: "Helper", Kind: KindFunction}},
			},
		},
	}

	g := Build(files)

	if len(g.Symbols) != 4 {
		t.Fatalf("expected 4 symbols, got %d", len(g.Symbols))
	}

	var gotCalls, gotContains bool
	for _, e := range g.Edges {
		if e.Source == "a.go:Caller" && e.Target == "b.go:Helper" && e.Kind == EdgeCalls {
			gotCalls = true
		}
		if e.Source == "a.go:Server" && e.Target == "a.go:Server.Start" && e.Kind == EdgeContains {
			gotContains = true
		}
	}
	if !gotCalls {
		t.Errorf("expected resolved calls edge to unique repo-wide match, edges: %+v", g.Edges)
	}
	if !gotContains {
		t.Errorf("expected contains edge from parentID, edges: %+v", g.Edges)
	}
}

func TestResolveCallNeverCrossesLanguageFamilies(t *testing.T) {
	// Regression test: Go's "os.Stat(...)" call site reduces to the bare
	// name "Stat" — same as an unrelated React "Stat" component in the web/
	// frontend. Before the language-family filter, resolveCall's
	// unique-repo-wide-match fallback wired these together into a bogus
	// cross-language edge.
	files := []*FileGraph{
		{
			FilePath: "cmd/common.go",
			Language: "go",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "cmd/common.go:resolveRepoPath", Name: "resolveRepoPath", Kind: KindFunction}, Language: "go"},
			},
			UnresolvedCalls: []UnresolvedCall{
				{FromID: "cmd/common.go:resolveRepoPath", TargetName: "Stat", Kind: EdgeCalls},
			},
		},
		{
			FilePath: "web/src/components/Header/Header.tsx",
			Language: "tsx",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "web/src/components/Header/Header.tsx:Stat", Name: "Stat", Kind: KindFunction}, Language: "tsx"},
			},
		},
	}
	g := Build(files)
	for _, e := range g.Edges {
		if e.Source == "cmd/common.go:resolveRepoPath" {
			t.Fatalf("expected os.Stat call to stay unresolved (no Go \"Stat\" symbol exists), got a cross-language edge to %s", e.Target)
		}
	}
}

func TestResolveCallMatchesAcrossJSFamily(t *testing.T) {
	// TypeScript/TSX/JavaScript compile to one runtime and commonly import
	// across those extensions, so calls between them must still resolve.
	files := []*FileGraph{
		{
			FilePath: "web/src/App.tsx",
			Language: "tsx",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "web/src/App.tsx:App", Name: "App", Kind: KindFunction}, Language: "tsx"},
			},
			UnresolvedCalls: []UnresolvedCall{
				{FromID: "web/src/App.tsx:App", TargetName: "helper", Kind: EdgeCalls},
			},
		},
		{
			FilePath: "web/src/utils/helper.ts",
			Language: "typescript",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "web/src/utils/helper.ts:helper", Name: "helper", Kind: KindFunction}, Language: "typescript"},
			},
		},
	}
	g := Build(files)
	for _, e := range g.Edges {
		if e.Source == "web/src/App.tsx:App" && e.Target == "web/src/utils/helper.ts:helper" {
			return
		}
	}
	t.Fatal("expected tsx -> ts call to resolve within the JS family")
}

func TestResolveCallPrefersSameFileThenSameDir(t *testing.T) {
	files := []*FileGraph{
		{
			FilePath: "pkg/a.go",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "pkg/a.go:Caller", Name: "Caller", Kind: KindFunction}},
				{SymbolRef: SymbolRef{ID: "pkg/a.go:log", Name: "log", Kind: KindFunction}},
			},
			UnresolvedCalls: []UnresolvedCall{
				{FromID: "pkg/a.go:Caller", TargetName: "log", Kind: EdgeCalls},
			},
		},
		{
			FilePath: "pkg/b.go",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "pkg/b.go:log", Name: "log", Kind: KindFunction}},
			},
		},
		{
			FilePath: "other/c.go",
			Symbols: []Symbol{
				{SymbolRef: SymbolRef{ID: "other/c.go:log", Name: "log", Kind: KindFunction}},
			},
		},
	}
	g := Build(files)
	for _, e := range g.Edges {
		if e.Source == "pkg/a.go:Caller" {
			if e.Target != "pkg/a.go:log" {
				t.Fatalf("expected same-file match, got %s", e.Target)
			}
			return
		}
	}
	t.Fatal("expected a resolved edge from Caller")
}
