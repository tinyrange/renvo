// Package rfe implements source emulator archives and instruction lowering.
package rfe

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

type rule struct {
	name        string
	mask, value uint64
	body        *ast.BlockStmt
}
type lowering struct {
	fset   *token.FileSet
	locals map[string]bool
}

// GenerateLowering accepts a Go-shaped expression DSL. It deliberately has no
// loops, arbitrary calls, or implicit host-width integer operations. A rule is
// a pure state transformation; instructions with bus effects use Go execution.
func GenerateLowering(source []byte) ([]byte, error) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "instructions.lower", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	bits, words := 0, 0
	for _, g := range file.Comments {
		for _, c := range g.List {
			if strings.HasPrefix(c.Text, "//rfe:word ") {
				bits, err = strconv.Atoi(strings.TrimPrefix(c.Text, "//rfe:word "))
			}
			if strings.HasPrefix(c.Text, "//rfe:state ") {
				words, err = strconv.Atoi(strings.TrimPrefix(c.Text, "//rfe:state "))
			}
		}
	}
	if err != nil || bits < 1 || bits > 32 || words < 1 || words > 256 {
		return nil, fmt.Errorf("lowering requires word 1..32 and state 1..256")
	}
	var rules []rule
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil || fn.Body == nil || fn.Recv != nil {
			return nil, fmt.Errorf("lowering declarations must be annotated instruction functions")
		}
		if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) != 1 || fn.Type.Params.List[0].Names[0].Name != "opcode" || fn.Type.Results != nil {
			return nil, fmt.Errorf("instruction signature must be func name(opcode uint64)")
		}
		param, ok := fn.Type.Params.List[0].Type.(*ast.Ident)
		if !ok || param.Name != "uint64" {
			return nil, fmt.Errorf("instruction opcode must be uint64")
		}
		var fields []string
		for _, c := range fn.Doc.List {
			if strings.HasPrefix(c.Text, "//rfe:instruction ") {
				fields = strings.Fields(strings.TrimPrefix(c.Text, "//rfe:instruction "))
			}
		}
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s requires instruction mask and value", fn.Name.Name)
		}
		mask, e1 := strconv.ParseUint(fields[0], 0, 64)
		value, e2 := strconv.ParseUint(fields[1], 0, 64)
		if e1 != nil || e2 != nil || mask>>bits != 0 || value&^mask != 0 {
			return nil, fmt.Errorf("invalid instruction mask/value for %s", fn.Name.Name)
		}
		for _, r := range rules {
			if (r.value^value)&(r.mask&mask) == 0 {
				return nil, fmt.Errorf("ambiguous encodings: %s and %s", r.name, fn.Name.Name)
			}
		}
		rules = append(rules, rule{fn.Name.Name, mask, value, fn.Body})
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("lowering has no instructions")
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated from instructions.lower by renvoemu; DO NOT EDIT.\npackage %s\nimport emu %q\nconst LoweringStateWords = %d\nfunc Lower(opcode uint64,b *emu.Builder) bool {\nif opcode >> %d != 0 {return false}\nswitch {\n", file.Name.Name, "renvo.dev/internal/rfe/runtime", words, bits)
	for _, r := range rules {
		if err := validateIndices(r, bits, words); err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, "case opcode & %#x == %#x: {\n", r.mask, r.value)
		l := lowering{fset: fs, locals: map[string]bool{}}
		for i, s := range r.body.List {
			if callStmt, ok := s.(*ast.ExprStmt); ok {
				call, ok := callStmt.X.(*ast.CallExpr)
				if !ok || i != 0 || len(call.Args) != 1 {
					return nil, fmt.Errorf("only an initial guard expression is allowed")
				}
				id, ok := call.Fun.(*ast.Ident)
				if !ok || id.Name != "guard" {
					return nil, fmt.Errorf("unknown lowering statement")
				}
				if err := decodeExpr(call.Args[0]); err != nil {
					return nil, err
				}
				fmt.Fprintf(&out, "if !(%s) {return false}\n", l.text(call.Args[0]))
				continue
			}
			a, ok := s.(*ast.AssignStmt)
			if !ok || len(a.Lhs) != 1 || len(a.Rhs) != 1 {
				return nil, fmt.Errorf("instruction %s requires single assignments", r.name)
			}
			v, err := l.expr(a.Rhs[0])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", r.name, err)
			}
			if id, ok := a.Lhs[0].(*ast.Ident); ok {
				if a.Tok != token.DEFINE || l.locals[id.Name] || id.Name == "opcode" || id.Name == "b" || id.Name == "emu" || id.Name == "state" || id.Name == "_" {
					return nil, fmt.Errorf("invalid lowering temporary %s", id.Name)
				}
				l.locals[id.Name] = true
				fmt.Fprintf(&out, "%s := %s; _ = %s\n", id.Name, v, id.Name)
			} else if index, ok := a.Lhs[0].(*ast.IndexExpr); ok && a.Tok == token.ASSIGN {
				idx, err := l.stateIndex(index)
				if err != nil {
					return nil, err
				}
				fmt.Fprintf(&out, "b.Store(%s,%s)\n", idx, v)
			} else {
				return nil, fmt.Errorf("invalid lowering assignment")
			}
		}
		out.WriteString("return true\n}\n")
	}
	out.WriteString("}\nreturn false\n}\n")
	fmt.Fprintf(&out, "func Decodable(opcode uint64) bool { if opcode >> %d != 0 {return false}; switch {\n", bits)
	for _, r := range rules {
		fmt.Fprintf(&out, "case opcode & %#x == %#x:\n", r.mask, r.value)
		if len(r.body.List) > 0 {
			if s, ok := r.body.List[0].(*ast.ExprStmt); ok {
				call := s.X.(*ast.CallExpr)
				l := lowering{fset: fs}
				fmt.Fprintf(&out, "return %s\n", l.text(call.Args[0]))
				continue
			}
		}
		out.WriteString("return true\n")
	}
	out.WriteString("};return false}\n")
	generated, err := format.Source(out.Bytes())
	if err != nil {
		return nil, err
	}
	return generated, nil
}
func (l *lowering) text(e ast.Expr) string {
	var out bytes.Buffer
	_ = format.Node(&out, l.fset, e)
	return out.String()
}
func decodeExpr(e ast.Expr) error {
	switch n := e.(type) {
	case *ast.ParenExpr:
		return decodeExpr(n.X)
	case *ast.Ident:
		if n.Name == "opcode" || n.Name == "true" || n.Name == "false" {
			return nil
		}
	case *ast.BasicLit:
		if n.Kind == token.INT {
			if _, err := strconv.ParseUint(n.Value, 0, 64); err == nil {
				return nil
			}
		}
	case *ast.UnaryExpr:
		if n.Op == token.NOT || n.Op == token.XOR {
			return decodeExpr(n.X)
		}
	case *ast.BinaryExpr:
		switch n.Op {
		case token.ADD, token.SUB, token.AND, token.OR, token.XOR, token.AND_NOT, token.SHL, token.SHR, token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
			if err := decodeExpr(n.X); err != nil {
				return err
			}
			return decodeExpr(n.Y)
		}
	}
	return fmt.Errorf("invalid opcode extraction or guard expression")
}
func (l *lowering) stateIndex(n *ast.IndexExpr) (string, error) {
	id, ok := n.X.(*ast.Ident)
	if !ok || id.Name != "state" {
		return "", fmt.Errorf("only state[opcode expression] indexing is allowed")
	}
	if err := decodeExpr(n.Index); err != nil {
		return "", err
	}
	return "int(" + l.text(n.Index) + ")", nil
}
func (l *lowering) expr(e ast.Expr) (string, error) {
	binary := func(op, a, b string) string { return "b.Binary(emu." + op + "," + a + "," + b + ")" }
	switch n := e.(type) {
	case *ast.ParenExpr:
		return l.expr(n.X)
	case *ast.BasicLit:
		if n.Kind == token.INT {
			v, err := strconv.ParseUint(n.Value, 0, 64)
			if err == nil {
				return fmt.Sprintf("b.Constant(%d)", v), nil
			}
		}
	case *ast.Ident:
		if l.locals[n.Name] {
			return n.Name, nil
		}
		if n.Name == "opcode" {
			return "b.Constant(opcode)", nil
		}
	case *ast.IndexExpr:
		idx, err := l.stateIndex(n)
		if err != nil {
			return "", err
		}
		return "b.Load(" + idx + ")", nil
	case *ast.UnaryExpr:
		a, err := l.expr(n.X)
		if err != nil {
			return "", err
		}
		switch n.Op {
		case token.XOR:
			return binary("Xor", a, "b.Constant(^uint64(0))"), nil
		case token.SUB:
			return binary("Sub", "b.Constant(0)", a), nil
		case token.NOT:
			return binary("Equal", a, "b.Constant(0)"), nil
		}
	case *ast.BinaryExpr:
		a, err := l.expr(n.X)
		if err != nil {
			return "", err
		}
		if n.Op == token.SHL || n.Op == token.SHR {
			if err := decodeExpr(n.Y); err != nil {
				return "", err
			}
			op := "Shl"
			if n.Op == token.SHR {
				op = "Shr"
			}
			return "b.Shift(emu." + op + "," + a + ",uint64(" + l.text(n.Y) + "))", nil
		}
		b, err := l.expr(n.Y)
		if err != nil {
			return "", err
		}
		ops := map[token.Token]string{token.ADD: "Add", token.SUB: "Sub", token.AND: "And", token.OR: "Or", token.XOR: "Xor", token.EQL: "Equal", token.LSS: "Less"}
		if op, ok := ops[n.Op]; ok {
			return binary(op, a, b), nil
		}
		switch n.Op {
		case token.AND_NOT:
			return binary("And", a, binary("Xor", b, "b.Constant(^uint64(0))")), nil
		case token.NEQ:
			return binary("Equal", binary("Equal", a, b), "b.Constant(0)"), nil
		case token.GTR:
			return binary("Less", b, a), nil
		}
	case *ast.CallExpr:
		id, ok := n.Fun.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("invalid lowering call")
		}
		if id.Name == "choose" && len(n.Args) == 3 {
			values := make([]string, 3)
			for i, e := range n.Args {
				v, err := l.expr(e)
				if err != nil {
					return "", err
				}
				values[i] = v
			}
			return "b.Choose(" + strings.Join(values, ",") + ")", nil
		}
		if len(n.Args) == 1 && (id.Name == "u8" || id.Name == "u16" || id.Name == "u32") {
			v, err := l.expr(n.Args[0])
			if err != nil {
				return "", err
			}
			mask := "255"
			if id.Name == "u16" {
				mask = "65535"
			}
			if id.Name == "u32" {
				mask = "4294967295"
			}
			return binary("And", v, "b.Constant("+mask+")"), nil
		}
	}
	return "", fmt.Errorf("unsupported lowering expression %s", l.text(e))
}
