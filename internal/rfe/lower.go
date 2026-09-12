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

type assignment struct{ target, value *expression }
type rule struct {
	name        string
	mask, value uint64
	guard       *expression
	assignments []assignment
	indices     []*expression
}

func parseRule(fn *ast.FuncDecl, bits, words int) (rule, error) {
	r := rule{name: fn.Name.Name}
	if fn.Recv != nil || fn.Doc == nil || fn.Body == nil || fn.Type.Results != nil || fn.Type.TypeParams != nil || len(fn.Type.Params.List) != 1 {
		return r, fmt.Errorf("expected annotated func name(opcode uint64)")
	}
	p := fn.Type.Params.List[0]
	typ, ok := p.Type.(*ast.Ident)
	if !ok || typ.Name != "uint64" || len(p.Names) != 1 || p.Names[0].Name != "opcode" {
		return r, fmt.Errorf("expected opcode uint64")
	}
	var fields []string
	for _, c := range fn.Doc.List {
		if strings.HasPrefix(c.Text, "//rfe:instruction ") {
			if fields != nil {
				return r, fmt.Errorf("duplicate instruction annotation")
			}
			fields = strings.Fields(strings.TrimPrefix(c.Text, "//rfe:instruction "))
		}
	}
	if len(fields) != 2 {
		return r, fmt.Errorf("missing instruction mask/value")
	}
	mask, e1 := strconv.ParseUint(fields[0], 0, 64)
	value, e2 := strconv.ParseUint(fields[1], 0, 64)
	if e1 != nil || e2 != nil || mask>>bits != 0 || value&^mask != 0 {
		return r, fmt.Errorf("invalid instruction mask/value")
	}
	r.mask, r.value = mask, value
	locals := map[string]*expression{}
	var indices func(*expression)
	indices = func(e *expression) {
		if e.op == token.LBRACK {
			r.indices = append(r.indices, e.args[0])
		}
		if e.op != token.IDENT {
			for _, a := range e.args {
				indices(a)
			}
		}
	}
	for i, stmt := range fn.Body.List {
		if s, ok := stmt.(*ast.ExprStmt); ok {
			c, ok := s.X.(*ast.CallExpr)
			if !ok || i != 0 || len(c.Args) != 1 {
				return r, fmt.Errorf("only an initial guard is allowed")
			}
			id, ok := c.Fun.(*ast.Ident)
			if !ok || id.Name != "guard" {
				return r, fmt.Errorf("expected guard")
			}
			g, err := parseExpression(c.Args[0], locals)
			if err != nil {
				return r, err
			}
			if !g.boolean || g.dynamic {
				return r, fmt.Errorf("guard must be an opcode boolean")
			}
			r.guard = g
			continue
		}
		s, ok := stmt.(*ast.AssignStmt)
		if !ok || len(s.Lhs) != 1 || len(s.Rhs) != 1 {
			return r, fmt.Errorf("expected single assignment")
		}
		v, err := parseExpression(s.Rhs[0], locals)
		if err != nil {
			return r, err
		}
		indices(v)
		var target *expression
		if id, ok := s.Lhs[0].(*ast.Ident); ok {
			if s.Tok != token.DEFINE || locals[id.Name] != nil || id.Name == "opcode" || id.Name == "state" || id.Name == "true" || id.Name == "false" || id.Name == "_" {
				return r, fmt.Errorf("invalid temporary %s", id.Name)
			}
			target = &expression{op: token.IDENT, name: fmt.Sprintf("rfeLocal%d", len(locals)), args: []*expression{v}, boolean: v.boolean, dynamic: v.dynamic}
			locals[id.Name] = target
		} else {
			target, err = parseExpression(s.Lhs[0], locals)
			if err != nil {
				return r, err
			}
			if s.Tok != token.ASSIGN || target.op != token.LBRACK || v.boolean {
				return r, fmt.Errorf("expected integer state assignment")
			}
			indices(target)
		}
		r.assignments = append(r.assignments, assignment{target, v})
	}
	var positions []uint
	for bit := 0; bit < bits; bit++ {
		if mask&(1<<bit) == 0 {
			positions = append(positions, uint(bit))
		}
	}
	if len(positions) > 16 {
		return r, fmt.Errorf("split rules with more than 16 variable bits")
	}
	for combination := 0; combination < 1<<len(positions); combination++ {
		opcode := value
		for i, bit := range positions {
			if combination&(1<<i) != 0 {
				opcode |= 1 << bit
			}
		}
		if r.guard != nil && r.guard.evaluate(opcode) == 0 {
			continue
		}
		for _, idx := range r.indices {
			if idx.evaluate(opcode) >= uint64(words) {
				return r, fmt.Errorf("state index out of bounds at opcode %#x", opcode)
			}
		}
	}
	return r, nil
}

// GenerateLowering emits ordinary Go execution and IR construction using one
// decoder and typed semantics. User Go expressions are never copied verbatim.
func GenerateLowering(source []byte) ([]byte, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "instructions.lower", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	annotations := map[string]int{}
	for _, g := range file.Comments {
		for _, c := range g.List {
			for _, key := range []string{"word", "state"} {
				prefix := "//rfe:" + key + " "
				if strings.HasPrefix(c.Text, prefix) {
					if _, ok := annotations[key]; ok {
						return nil, fmt.Errorf("duplicate %s annotation", key)
					}
					n, err := strconv.Atoi(strings.TrimPrefix(c.Text, prefix))
					if err != nil {
						return nil, err
					}
					annotations[key] = n
				}
			}
		}
	}
	bits, words := annotations["word"], annotations["state"]
	if bits < 1 || bits > 32 || words < 1 || words > 256 {
		return nil, fmt.Errorf("lowering requires word 1..32 and state 1..256")
	}
	var rules []rule
	names := map[string]bool{}
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			return nil, fmt.Errorf("expected instruction function")
		}
		r, err := parseRule(fn, bits, words)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fn.Name.Name, err)
		}
		if names[r.name] {
			return nil, fmt.Errorf("duplicate instruction %s", r.name)
		}
		names[r.name] = true
		for _, old := range rules {
			if (old.value^r.value)&(old.mask&r.mask) == 0 {
				return nil, fmt.Errorf("ambiguous encodings: %s and %s", old.name, r.name)
			}
		}
		rules = append(rules, r)
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("lowering has no instructions")
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated from instructions.lower by renvoemu; DO NOT EDIT.\npackage %s\nimport emu %q\nconst LoweringStateWords = %d\n", file.Name.Name, "renvo.dev/internal/rfe/runtime", words)
	out.WriteString("func rfeBool(v bool) uint64 {if v{return 1};return 0}\nfunc rfeChoose(c bool,t,f uint64)uint64{if c{return t};return f}\nfunc rfeChooseBool(c,t,f bool)bool{if c{return t};return f}\n")
	// Emit the same validated dispatch directly into each entry point. This
	// avoids decoding to a rule number followed by a second switch in Execute.
	// Equal masks form value switches; overlap validation makes grouping safe.
	dispatch := func(body func(rule)) {
		fmt.Fprintf(&out, "if opcode >> %d != 0 {return false};\n", bits)
		grouped := map[uint64]bool{}
		for _, group := range rules {
			if grouped[group.mask] {
				continue
			}
			grouped[group.mask] = true
			fmt.Fprintf(&out, "switch opcode & %#x {\n", group.mask)
			for _, r := range rules {
				if r.mask != group.mask {
					continue
				}
				fmt.Fprintf(&out, "case %#x:\n", r.value)
				if r.guard != nil {
					fmt.Fprintf(&out, "if !(%s){return false}\n", r.guard.emit(false))
				}
				body(r)
				out.WriteString("return true\n")
			}
			out.WriteString("}\n")
		}
		out.WriteString("return false}\n")
	}
	out.WriteString("func Decodable(opcode uint64)bool{\n")
	dispatch(func(rule) {})
	for _, ir := range []bool{false, true} {
		if ir {
			out.WriteString("func Lower(opcode uint64,b *emu.Builder)bool{\n")
		} else {
			fmt.Fprintf(&out, "func Execute(opcode uint64,state *[%d]uint64)bool{\n", words)
		}
		dispatch(func(r rule) {
			for _, s := range r.assignments {
				v := s.value.emit(ir && s.value.dynamic)
				if s.target.op == token.IDENT {
					fmt.Fprintf(&out, "%s:=%s;_=%s\n", s.target.name, v, s.target.name)
				} else if ir {
					fmt.Fprintf(&out, "b.Store(int(%s),%s)\n", s.target.args[0].emit(false), s.value.emit(true))
				} else {
					fmt.Fprintf(&out, "%s=%s\n", s.target.emit(false), v)
				}
			}
		})
	}
	return format.Source(out.Bytes())
}
