package parser

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"graphify/internal/graph"
)

var goCallKinds = map[string]bool{"call_expression": true}

// ParseGo extracts top-level func/method/type/const/var declarations from a
// Go source file, plus every call made from inside a function or method body.
func ParseGo(filePath string, src []byte) (*graph.FileGraph, error) {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}

	fg := &graph.FileGraph{FilePath: filePath, Language: "go"}

	for _, decl := range namedChildren(tree.RootNode()) {
		switch decl.Type() {
		case "function_declaration":
			sym, calls := goFunc(decl, src, filePath, "")
			fg.Symbols = append(fg.Symbols, sym)
			fg.UnresolvedCalls = append(fg.UnresolvedCalls, calls...)

		case "method_declaration":
			recvType := goReceiverType(decl.ChildByFieldName("receiver"), src)
			sym, calls := goFunc(decl, src, filePath, recvType)
			fg.Symbols = append(fg.Symbols, sym)
			fg.UnresolvedCalls = append(fg.UnresolvedCalls, calls...)

		case "type_declaration":
			for _, spec := range namedChildren(decl) {
				if spec.Type() != "type_spec" {
					continue
				}
				name := nodeText(spec.ChildByFieldName("name"), src)
				typ := spec.ChildByFieldName("type")
				kind := graph.KindClass
				if typ != nil && typ.Type() == "interface_type" {
					kind = graph.KindInterface
				}
				fg.Symbols = append(fg.Symbols, graph.Symbol{
					SymbolRef: graph.SymbolRef{
						ID:        symbolID(filePath, name),
						Name:      name,
						Kind:      kind,
						FilePath:  filePath,
						StartLine: line1(decl.StartPoint()),
						EndLine:   line1(decl.EndPoint()),
					},
					Signature: strings.TrimSpace(nodeText(decl, src)[:min(len(nodeText(decl, src)), 200)]),
					Language:  "go",
					Doc:       leadingComment(decl, src),
				})
			}

		case "const_declaration", "var_declaration":
			kind := graph.KindVariable
			if decl.Type() == "const_declaration" {
				kind = graph.KindConst
			}
			fg.Symbols = append(fg.Symbols, goSpecs(decl, src, filePath, kind)...)
		}
	}

	return fg, nil
}

func goReceiverType(recv *sitter.Node, src []byte) string {
	if recv == nil {
		return ""
	}
	for _, p := range namedChildren(recv) {
		if p.Type() != "parameter_declaration" {
			continue
		}
		t := p.ChildByFieldName("type")
		if t == nil {
			continue
		}
		name := nodeText(t, src)
		return strings.TrimPrefix(name, "*")
	}
	return ""
}

func goFunc(decl *sitter.Node, src []byte, filePath, recvType string) (graph.Symbol, []graph.UnresolvedCall) {
	name := nodeText(decl.ChildByFieldName("name"), src)
	qualified := name
	if recvType != "" {
		qualified = recvType + "." + name
	}

	var params []graph.Param
	if pl := decl.ChildByFieldName("parameters"); pl != nil {
		for _, p := range namedChildren(pl) {
			if p.Type() != "parameter_declaration" && p.Type() != "variadic_parameter_declaration" {
				continue
			}
			typ := nodeText(p.ChildByFieldName("type"), src)
			for _, nameNode := range fieldChildren(p, "name") {
				params = append(params, graph.Param{Name: nodeText(nameNode, src), Type: typ})
			}
			if len(fieldChildren(p, "name")) == 0 {
				params = append(params, graph.Param{Name: "", Type: typ})
			}
		}
	}

	var returns []graph.Param
	if res := decl.ChildByFieldName("result"); res != nil {
		if res.Type() == "parameter_list" {
			for _, p := range namedChildren(res) {
				typ := nodeText(p.ChildByFieldName("type"), src)
				if typ == "" {
					typ = nodeText(p, src)
				}
				returns = append(returns, graph.Param{Name: nodeText(p.ChildByFieldName("name"), src), Type: typ})
			}
		} else {
			returns = append(returns, graph.Param{Type: nodeText(res, src)})
		}
	}

	kind := graph.KindFunction
	if recvType != "" {
		kind = graph.KindMethod
	}

	id := symbolID(filePath, qualified)
	sym := graph.Symbol{
		SymbolRef: graph.SymbolRef{
			ID:        id,
			Name:      name,
			Kind:      kind,
			FilePath:  filePath,
			StartLine: line1(decl.StartPoint()),
			EndLine:   line1(decl.EndPoint()),
		},
		Signature: goSignature(decl, src),
		Params:    params,
		Returns:   returns,
		Receiver:  recvType,
		Language:  "go",
		Doc:       leadingComment(decl, src),
	}
	if recvType != "" {
		sym.ParentID = symbolID(filePath, recvType)
	}

	var calls []graph.UnresolvedCall
	if body := decl.ChildByFieldName("body"); body != nil {
		walk(body, goCallKinds, func(call *sitter.Node) {
			fn := call.ChildByFieldName("function")
			if fn == nil {
				return
			}
			target := ""
			qualified := false
			switch fn.Type() {
			case "identifier":
				target = nodeText(fn, src)
			case "selector_expression":
				target = nodeText(fn.ChildByFieldName("field"), src)
				qualified = true
			}
			if target != "" {
				calls = append(calls, graph.UnresolvedCall{FromID: id, TargetName: target, Kind: graph.EdgeCalls, Qualified: qualified})
			}
		})
	}

	return sym, calls
}

func goSignature(decl *sitter.Node, src []byte) string {
	body := decl.ChildByFieldName("body")
	end := decl.EndByte()
	if body != nil {
		end = body.StartByte()
	}
	sig := string(src[decl.StartByte():end])
	return strings.TrimSpace(sig)
}

func goSpecs(decl *sitter.Node, src []byte, filePath string, kind graph.SymbolKind) []graph.Symbol {
	var out []graph.Symbol
	specType := "const_spec"
	if kind == graph.KindVariable {
		specType = "var_spec"
	}
	for _, spec := range namedChildren(decl) {
		if spec.Type() != specType {
			continue
		}
		doc := leadingComment(spec, src)
		if doc == "" {
			doc = leadingComment(decl, src)
		}
		for _, nameNode := range fieldChildren(spec, "name") {
			name := nodeText(nameNode, src)
			out = append(out, graph.Symbol{
				SymbolRef: graph.SymbolRef{
					ID:        symbolID(filePath, name),
					Name:      name,
					Kind:      kind,
					FilePath:  filePath,
					StartLine: line1(spec.StartPoint()),
					EndLine:   line1(spec.EndPoint()),
				},
				Signature: strings.TrimSpace(nodeText(spec, src)),
				Language:  "go",
				Doc:       doc,
			})
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
