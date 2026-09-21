package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
	"testing"
)

func TestBuiltinBindingsAcrossNestedClauses(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`v:="outer";switch {case true:v:=1;_=v;switch {case true:case false:};case false:_=len(v)}`, true},
		{`v:="outer";switch {case true:switch {case true:v:=1;_=v;case false:};_=len(v);case false:}`, true},
		{`switch {case true:v:=1;switch {case true:v:="inner";_=len(v);case false:};_=len(v);case false:}`, false},
		{`v:=1;switch {case true:v:="inner";_=len(v);switch {case true:case false:};case false:_=len(v)}`, false},
		{`v:="outer";var ch chan int;select {case <-ch:v:=1;_=v;switch {case true:case false:};default:_=len(v)}`, true},
		{`v:=1;var ch chan int;select {case <-ch:v:="inner";_=len(v);default:_=len(v)}`, false},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main(){" + tc.body + "}")}})
		for _, pkg := range graph.Packages {
			for _, source := range pkg.Files {
				for _, fn := range source.File.Funcs {
					body := syntax.ParseFuncBodyStatements(source.File, fn)
					ends := localRuleScopeEnds(body)
					for i, stmt := range body.Stmts {
						if stmt.Kind != syntax.StmtBlock && ends[i] != localRuleScopeEnd(body, stmt.StartTok) {
							t.Fatalf("%s: scope endpoint for token %d = %d, want %d", tc.body, stmt.StartTok, ends[i], localRuleScopeEnd(body, stmt.StartTok))
						}
					}
				}
			}
		}
		result := CheckGraphCore(graph)
		if result.Ok != tc.valid || !tc.valid && result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d token=%d", tc.body, result.Ok, result.Error, result.ErrorToken)
		}
	}
}
