package rfe

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

type decodeValue struct {
	n       uint64
	boolean bool
}

func evaluate(e ast.Expr, opcode uint64) (decodeValue, error) {
	bad := func() (decodeValue, error) { return decodeValue{}, fmt.Errorf("invalid decode expression types") }
	switch x := e.(type) {
	case *ast.ParenExpr:
		return evaluate(x.X, opcode)
	case *ast.Ident:
		switch x.Name {
		case "opcode":
			return decodeValue{n: opcode}, nil
		case "true":
			return decodeValue{n: 1, boolean: true}, nil
		case "false":
			return decodeValue{boolean: true}, nil
		}
	case *ast.BasicLit:
		n, err := strconv.ParseUint(x.Value, 0, 64)
		return decodeValue{n: n}, err
	case *ast.UnaryExpr:
		v, err := evaluate(x.X, opcode)
		if err != nil {
			return v, err
		}
		if x.Op == token.XOR && !v.boolean {
			v.n = ^v.n
			return v, nil
		}
		if x.Op == token.NOT && v.boolean {
			v.n ^= 1
			return v, nil
		}
	case *ast.BinaryExpr:
		a, err := evaluate(x.X, opcode)
		if err != nil {
			return a, err
		}
		b, err := evaluate(x.Y, opcode)
		if err != nil {
			return b, err
		}
		truth := func(v bool) (decodeValue, error) {
			if v {
				return decodeValue{n: 1, boolean: true}, nil
			}
			return decodeValue{boolean: true}, nil
		}
		if x.Op == token.LAND || x.Op == token.LOR {
			if !a.boolean || !b.boolean {
				return bad()
			}
			if x.Op == token.LAND {
				return truth(a.n != 0 && b.n != 0)
			}
			return truth(a.n != 0 || b.n != 0)
		}
		if a.boolean || b.boolean {
			return bad()
		}
		switch x.Op {
		case token.ADD:
			a.n += b.n
		case token.SUB:
			a.n -= b.n
		case token.AND:
			a.n &= b.n
		case token.OR:
			a.n |= b.n
		case token.XOR:
			a.n ^= b.n
		case token.AND_NOT:
			a.n &^= b.n
		case token.SHL:
			if b.n >= 64 {
				a.n = 0
			} else {
				a.n <<= b.n
			}
		case token.SHR:
			if b.n >= 64 {
				a.n = 0
			} else {
				a.n >>= b.n
			}
		case token.EQL:
			return truth(a.n == b.n)
		case token.NEQ:
			return truth(a.n != b.n)
		case token.LSS:
			return truth(a.n < b.n)
		case token.LEQ:
			return truth(a.n <= b.n)
		case token.GTR:
			return truth(a.n > b.n)
		case token.GEQ:
			return truth(a.n >= b.n)
		default:
			return bad()
		}
		return a, nil
	}
	return bad()
}

// Enumerate each rule's operand encodings, not its full word space. Requiring
// at most 16 variable bits keeps generation bounded even for 32-bit ISAs.
func validateIndices(r rule, bits, words int) error {
	var positions []uint
	for bit := 0; bit < bits; bit++ {
		if r.mask&(1<<bit) == 0 {
			positions = append(positions, uint(bit))
		}
	}
	if len(positions) > 16 {
		return fmt.Errorf("%s: split rules with more than 16 variable bits", r.name)
	}
	var guard ast.Expr
	var indices []ast.Expr
	if len(r.body.List) > 0 {
		if s, ok := r.body.List[0].(*ast.ExprStmt); ok {
			if c, ok := s.X.(*ast.CallExpr); ok && len(c.Args) == 1 {
				if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "guard" {
					guard = c.Args[0]
				}
			}
		}
	}
	ast.Inspect(r.body, func(n ast.Node) bool {
		if index, ok := n.(*ast.IndexExpr); ok {
			indices = append(indices, index.Index)
		}
		return true
	})
	for combination := 0; combination < 1<<len(positions); combination++ {
		opcode := r.value
		for i, position := range positions {
			if combination&(1<<i) != 0 {
				opcode |= 1 << position
			}
		}
		if guard != nil {
			v, err := evaluate(guard, opcode)
			if err != nil || !v.boolean {
				return fmt.Errorf("%s: guard must be an opcode boolean expression", r.name)
			}
			if v.n == 0 {
				continue
			}
		}
		for _, index := range indices {
			v, err := evaluate(index, opcode)
			if err != nil || v.boolean || v.n >= uint64(words) {
				return fmt.Errorf("%s: state index outside 0..%d at opcode %#x", r.name, words-1, opcode)
			}
		}
	}
	return nil
}
