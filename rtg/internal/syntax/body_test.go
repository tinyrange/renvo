package syntax

import "testing"

func TestParseFuncBodyStatements(t *testing.T) {
	file := parseOneFuncBodyTestFile(t, `package main

func appMain() int {
	total := 0
	if total == 0 {
		total = add(total, 1)
	} else if total < 0 {
		return -1
	} else {
		total = total + 2
	}
	for i := 0; i < 3; i++ {
		total += i
	}
	return total
}
`)
	body := ParseFuncBody(file, file.Funcs[0])
	if !body.Ok {
		t.Fatalf("ParseFuncBody failed: err=%d tok=%d", body.Error, body.ErrorTok)
	}
	want := []int{
		StmtBlock,
		StmtAssign,
		StmtIf,
		StmtBlock,
		StmtAssign,
		StmtIf,
		StmtBlock,
		StmtReturn,
		StmtBlock,
		StmtAssign,
		StmtFor,
		StmtBlock,
		StmtAssign,
		StmtReturn,
	}
	assertStmtKinds(t, body, want)
	if body.Stmts[2].ElseStart < 0 || body.Stmts[2].ElseEnd <= body.Stmts[2].ElseStart {
		t.Fatalf("if else range = %d:%d", body.Stmts[2].ElseStart, body.Stmts[2].ElseEnd)
	}
	if exprText(file, body.Exprs[1]) != "total == 0" || body.Exprs[1].Kind != ExprBinary {
		t.Fatalf("if expr = %q kind %d", exprText(file, body.Exprs[1]), body.Exprs[1].Kind)
	}
	if !hasExpr(body, ExprCall, "add(total, 1)", file) {
		t.Fatalf("call expression not found: %#v", body.Exprs)
	}
}

func TestParseFuncBodySwitchAndLabels(t *testing.T) {
	file := parseOneFuncBodyTestFile(t, `package main

func appMain(v int) int {
start:
	switch v {
	case 0:
		goto done
	case 1, 2:
		fallthrough
	default:
		break
	}
done:
	return v
}
`)
	body := ParseFuncBody(file, file.Funcs[0])
	if !body.Ok {
		t.Fatalf("ParseFuncBody failed: err=%d tok=%d", body.Error, body.ErrorTok)
	}
	want := []int{
		StmtBlock,
		StmtLabel,
		StmtSwitch,
		StmtBlock,
		StmtCase,
		StmtGoto,
		StmtCase,
		StmtFallthrough,
		StmtDefault,
		StmtBreak,
		StmtLabel,
		StmtReturn,
	}
	assertStmtKinds(t, body, want)
	if !hasExpr(body, ExprIdent, "v", file) {
		t.Fatalf("switch/return identifier expression not found: %#v", body.Exprs)
	}
	if !hasExpr(body, ExprLiteral, "0", file) {
		t.Fatalf("case literal expression not found: %#v", body.Exprs)
	}
}

func TestParseFuncBodyExpressionKinds(t *testing.T) {
	file := parseOneFuncBodyTestFile(t, `package main

func appMain() int {
	defer cleanup()
	go worker(items[0].Name)
	a := -items[0].Value
	b := []int{1, 2, 3}
	c := obj.Field
	return call(a+b, c)
}
`)
	body := ParseFuncBody(file, file.Funcs[0])
	if !body.Ok {
		t.Fatalf("ParseFuncBody failed: err=%d tok=%d", body.Error, body.ErrorTok)
	}
	want := []int{StmtBlock, StmtDefer, StmtGo, StmtAssign, StmtAssign, StmtAssign, StmtReturn}
	assertStmtKinds(t, body, want)
	checks := []struct {
		kind int
		text string
	}{
		{kind: ExprCall, text: "cleanup()"},
		{kind: ExprCall, text: "worker(items[0].Name)"},
		{kind: ExprUnary, text: "-items[0].Value"},
		{kind: ExprComposite, text: "[]int{1, 2, 3}"},
		{kind: ExprSelector, text: "obj.Field"},
		{kind: ExprCall, text: "call(a+b, c)"},
	}
	for i := 0; i < len(checks); i++ {
		if !hasExpr(body, checks[i].kind, checks[i].text, file) {
			t.Fatalf("expression %q kind %d not found in %#v", checks[i].text, checks[i].kind, body.Exprs)
		}
	}
}

func TestParseFuncBodyErrors(t *testing.T) {
	file := parseOneFuncBodyTestFile(t, `package main

func appMain() int {
	return 0
}
`)
	fn := file.Funcs[0]
	fn.BodyStart = -1
	body := ParseFuncBody(file, fn)
	if body.Ok || body.Error != BodyErrFunc {
		t.Fatalf("invalid function body result = %#v", body)
	}
}

func parseOneFuncBodyTestFile(t *testing.T, src string) File {
	t.Helper()
	file := ParseFile([]byte(src))
	if !file.Ok {
		t.Fatalf("ParseFile failed: err=%d tok=%d", file.Error, file.ErrorTok)
	}
	if len(file.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(file.Funcs))
	}
	return file
}

func assertStmtKinds(t *testing.T, body Body, want []int) {
	t.Helper()
	if len(body.Stmts) != len(want) {
		t.Fatalf("stmt count = %d, want %d: %#v", len(body.Stmts), len(want), body.Stmts)
	}
	for i := 0; i < len(want); i++ {
		if body.Stmts[i].Kind != want[i] {
			t.Fatalf("stmt %d kind = %d, want %d: %#v", i, body.Stmts[i].Kind, want[i], body.Stmts)
		}
	}
}

func hasExpr(body Body, kind int, text string, file File) bool {
	for i := 0; i < len(body.Exprs); i++ {
		if body.Exprs[i].Kind == kind && exprText(file, body.Exprs[i]) == text {
			return true
		}
	}
	return false
}

func exprText(file File, expr Expr) string {
	if expr.StartTok < 0 || expr.EndTok <= expr.StartTok || expr.EndTok > len(file.Tokens) {
		return ""
	}
	start := file.Tokens[expr.StartTok].Start
	end := file.Tokens[expr.EndTok-1].End
	if start < 0 || end < start || end > len(file.Src) {
		return ""
	}
	return string(file.Src[start:end])
}
