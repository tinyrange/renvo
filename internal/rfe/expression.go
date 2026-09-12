package rfe

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// The typed tree is shared by decode validation, Go execution and IR emission.
// dynamic expressions depend on state; other expressions specialize at decode.
type expression struct {
	op               token.Token
	args             []*expression
	n                uint64
	name             string
	boolean, dynamic bool
}

func number(n uint64) *expression { return &expression{op: token.INT, n: n} }
func parseExpression(node ast.Expr, locals map[string]*expression) (*expression, error) {
	e := &expression{}
	var nodes []ast.Expr
	switch n := node.(type) {
	case *ast.ParenExpr:
		return parseExpression(n.X, locals)
	case *ast.BasicLit:
		if n.Kind != token.INT {
			return nil, fmt.Errorf("expected uint64 literal")
		}
		v, err := strconv.ParseUint(n.Value, 0, 64)
		return number(v), err
	case *ast.Ident:
		if n.Name == "opcode" {
			return &expression{op: token.IDENT, name: "opcode"}, nil
		}
		if n.Name == "true" || n.Name == "false" {
			e = number(0)
			e.boolean = true
			if n.Name == "true" {
				e.n = 1
			}
			return e, nil
		}
		if local := locals[n.Name]; local != nil {
			return local, nil
		}
		return nil, fmt.Errorf("unknown expression name %s", n.Name)
	case *ast.IndexExpr:
		id, ok := n.X.(*ast.Ident)
		if !ok || id.Name != "state" {
			return nil, fmt.Errorf("only state indexing is allowed")
		}
		e.op = token.LBRACK
		nodes = []ast.Expr{n.Index}
	case *ast.UnaryExpr:
		e.op = n.Op
		nodes = []ast.Expr{n.X}
	case *ast.BinaryExpr:
		e.op = n.Op
		nodes = []ast.Expr{n.X, n.Y}
	case *ast.CallExpr:
		if n.Ellipsis.IsValid() {
			return nil, fmt.Errorf("variadic DSL calls are not supported")
		}
		id, ok := n.Fun.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("invalid DSL call")
		}
		if id.Name == "choose" && len(n.Args) == 3 {
			e.op = token.FUNC
			nodes = n.Args
		} else {
			width := map[string]uint{"u8": 8, "u16": 16, "u32": 32}[id.Name]
			if width == 0 || len(n.Args) != 1 {
				return nil, fmt.Errorf("unknown DSL call %s", id.Name)
			}
			v, err := parseExpression(n.Args[0], locals)
			if err != nil {
				return nil, err
			}
			if v.boolean {
				return nil, fmt.Errorf("truncation requires an integer")
			}
			return &expression{op: token.AND, args: []*expression{v, number((1 << width) - 1)}, dynamic: v.dynamic}, nil
		}
	default:
		return nil, fmt.Errorf("unsupported DSL expression")
	}
	for _, n := range nodes {
		a, err := parseExpression(n, locals)
		if err != nil {
			return nil, err
		}
		e.args = append(e.args, a)
		e.dynamic = e.dynamic || a.dynamic
	}
	a := e.args[0]
	valid := false
	if e.op == token.LBRACK {
		valid = !a.boolean && !a.dynamic
		e.dynamic = true
	} else if e.op == token.FUNC {
		if !a.boolean {
			e.args[0] = &expression{op: token.NEQ, args: []*expression{a, number(0)}, boolean: true, dynamic: a.dynamic}
		}
		valid = e.args[1].boolean == e.args[2].boolean
		e.boolean = e.args[1].boolean
	} else if len(e.args) == 1 {
		valid = (e.op == token.NOT && a.boolean) || ((e.op == token.XOR || e.op == token.SUB) && !a.boolean)
		e.boolean = a.boolean
	} else {
		b := e.args[1]
		switch e.op {
		case token.EQL, token.NEQ:
			valid = a.boolean == b.boolean
			e.boolean = true
		case token.LSS, token.LEQ, token.GTR, token.GEQ:
			valid = !a.boolean && !b.boolean
			e.boolean = true
		case token.LAND, token.LOR:
			valid = a.boolean && b.boolean
			e.boolean = true
		case token.SHL, token.SHR:
			valid = !a.boolean && !b.boolean && !b.dynamic
		case token.ADD, token.SUB, token.AND, token.OR, token.XOR, token.AND_NOT:
			valid = !a.boolean && !b.boolean
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid types or operands for %s", e.op)
	}
	constant := e.op != token.LBRACK
	for _, a := range e.args {
		constant = constant && a.op == token.INT
	}
	if constant {
		e.n = e.evaluate(0)
		e.op = token.INT
		e.args = nil
	}
	return e, nil
}
func truth(v bool) uint64 {
	if v {
		return 1
	}
	return 0
}
func (e *expression) evaluate(opcode uint64) uint64 {
	if e.op == token.INT {
		return e.n
	}
	if e.op == token.IDENT {
		if e.name == "opcode" {
			return opcode
		}
		return e.args[0].evaluate(opcode)
	}
	a := e.args[0].evaluate(opcode)
	if len(e.args) == 1 {
		switch e.op {
		case token.NOT:
			return truth(a == 0)
		case token.XOR:
			return ^a
		case token.SUB:
			return -a
		}
	}
	b := e.args[1].evaluate(opcode)
	switch e.op {
	case token.ADD:
		return a + b
	case token.SUB:
		return a - b
	case token.AND:
		return a & b
	case token.OR:
		return a | b
	case token.XOR:
		return a ^ b
	case token.AND_NOT:
		return a &^ b
	case token.SHL:
		return a << b
	case token.SHR:
		return a >> b
	case token.EQL:
		return truth(a == b)
	case token.NEQ:
		return truth(a != b)
	case token.LSS:
		return truth(a < b)
	case token.LEQ:
		return truth(a <= b)
	case token.GTR:
		return truth(a > b)
	case token.GEQ:
		return truth(a >= b)
	case token.LAND:
		return truth(a != 0 && b != 0)
	case token.LOR:
		return truth(a != 0 || b != 0)
	case token.FUNC:
		if a != 0 {
			return b
		}
		return e.args[2].evaluate(opcode)
	}
	panic("state-dependent expression in decoder")
}
func (e *expression) emit(ir bool) string {
	if ir && !e.dynamic {
		x := e.emit(false)
		if e.boolean {
			x = "rfeBool(" + x + ")"
		}
		return "b.Constant(" + x + ")"
	}
	if e.op == token.INT {
		if e.boolean {
			return strconv.FormatBool(e.n != 0)
		}
		return fmt.Sprintf("uint64(%d)", e.n)
	}
	if e.op == token.IDENT {
		return e.name
	}
	if e.op == token.LBRACK {
		idx := e.args[0].emit(false)
		if ir {
			return "b.Load(int(" + idx + "))"
		}
		return "state[int(" + idx + ")]"
	}
	a := e.args[0].emit(ir)
	if !ir {
		if len(e.args) == 1 {
			return "(" + e.op.String() + a + ")"
		}
		b := e.args[1].emit(false)
		if e.op == token.FUNC {
			name := "rfeChoose"
			if e.boolean {
				name = "rfeChooseBool"
			}
			return name + "(" + a + "," + b + "," + e.args[2].emit(false) + ")"
		}
		return "(" + a + e.op.String() + b + ")"
	}
	bin := func(op, a, b string) string { return "b.Binary(emu." + op + "," + a + "," + b + ")" }
	zero := "b.Constant(0)"
	not := func(v string) string { return bin("Equal", v, zero) }
	if len(e.args) == 1 {
		switch e.op {
		case token.NOT:
			return not(a)
		case token.SUB:
			return bin("Sub", zero, a)
		case token.XOR:
			return bin("Xor", a, "b.Constant(^uint64(0))")
		}
	}
	b := e.args[1].emit(true)
	if e.op == token.FUNC {
		return "b.Choose(" + a + "," + b + "," + e.args[2].emit(true) + ")"
	}
	if e.op == token.SHL || e.op == token.SHR {
		op := "Shl"
		if e.op == token.SHR {
			op = "Shr"
		}
		return "b.Shift(emu." + op + "," + a + "," + e.args[1].emit(false) + ")"
	}
	if op := map[token.Token]string{token.ADD: "Add", token.SUB: "Sub", token.AND: "And", token.OR: "Or", token.XOR: "Xor", token.EQL: "Equal", token.LSS: "Less", token.LAND: "And", token.LOR: "Or"}[e.op]; op != "" {
		return bin(op, a, b)
	}
	switch e.op {
	case token.AND_NOT:
		return bin("And", a, bin("Xor", b, "b.Constant(^uint64(0))"))
	case token.NEQ:
		return not(bin("Equal", a, b))
	case token.GTR:
		return bin("Less", b, a)
	case token.LEQ:
		return not(bin("Less", b, a))
	case token.GEQ:
		return not(bin("Less", a, b))
	}
	panic("unvalidated DSL expression")
}
