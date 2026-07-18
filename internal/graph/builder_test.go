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
