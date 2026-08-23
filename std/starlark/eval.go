package starlark

import (
	"errors"
	"strconv"
	"strings"
)

// FileOptions is reserved for source-compatibility as the runtime grows. All
// supported language features are always enabled.
type FileOptions struct {
	Set, While, TopLevelControl, Recursion, GlobalReassign bool
}

type env struct {
	vars     StringDict
	parent   *env
	assigned map[string]bool
}

func (e *env) get(name string) (Value, bool) {
	for p := e; p != nil; p = p.parent {
		v, ok := p.vars[name]
		if ok {
			return v, true
		}
	}
	return nil, false
}
func (e *env) set(name string, v Value) {
	e.vars[name] = v
	if e.assigned != nil {
		e.assigned[name] = true
	}
}

func ExecFileOptions(_ *FileOptions, thread *Thread, filename string, src any, predeclared StringDict) (StringDict, error) {
	if thread == nil {
		thread = &Thread{}
	}
	var data []byte
	switch x := src.(type) {
	case string:
		data = []byte(x)
	case []byte:
		data = x
	default:
		return nil, errorf("unsupported source type")
	}
	program, err := parse(filename, data)
	if err != nil {
		return nil, err
	}
	root := &env{vars: StringDict{}, assigned: map[string]bool{}}
	addCoreBuiltins(root.vars)
	for k, v := range predeclared {
		root.vars[k] = v
	}
	sig := execBlock(thread, root, program)
	if sig.err != nil {
		return nil, sig.err
	}
	if sig.kind != "" {
		return nil, errorf("%s: %s outside function or loop", filename, sig.kind)
	}
	out := StringDict{}
	for n := range root.assigned {
		out[n] = root.vars[n]
	}
	return out, nil
}

// ExecFile parses and executes a Starlark file using the default options.
func ExecFile(thread *Thread, filename string, src any, predeclared StringDict) (StringDict, error) {
	return ExecFileOptions(&FileOptions{}, thread, filename, src, predeclared)
}

type signal struct {
	kind  string
	value Value
	err   error
}

func execBlock(t *Thread, e *env, body []stmt) signal {
	for _, s := range body {
		if err := t.step(); err != nil {
			return signal{err: err}
		}
		r := execStmt(t, e, s)
		if r.kind != "" || r.err != nil {
			return r
		}
	}
	return signal{}
}
func execStmt(t *Thread, e *env, s stmt) signal {
	switch x := s.(type) {
	case exprStmt:
		_, err := evalExpr(t, e, x.x)
		return signal{err: err}
	case assignStmt:
		v, err := evalExpr(t, e, x.value)
		if err != nil {
			return signal{err: err}
		}
		if x.op != tAssign {
			old, err := evalExpr(t, e, x.target)
			if err != nil {
				return signal{err: err}
			}
			op := map[tokenKind]string{tPlusEq: "+", tMinusEq: "-", tStarEq: "*", tSlashEq: "/", tPercentEq: "%"}[x.op]
			v, err = binary(op, old, v)
			if err != nil {
				return signal{err: err}
			}
		}
		return signal{err: assign(e, x.target, v)}
	case defStmt:
		params := make([]parameter, len(x.params))
		for i, p := range x.params {
			params[i].name = p.name
			if p.hasDefault {
				v, err := evalExpr(t, e, p.def)
				if err != nil {
					return signal{err: err}
				}
				params[i].def = v
				params[i].hasDefault = true
			}
		}
		function := &Function{}
		function.name = x.name
		function.doc = x.doc
		function.params = params
		function.body = x.body
		function.closure = e
		var value Value
		value = function
		e.set(x.name, value)
		return signal{}
	case ifStmt:
		v, err := evalExpr(t, e, x.cond)
		if err != nil {
			return signal{err: err}
		}
		if truth(v) {
			return execBlock(t, e, x.then)
		}
		return execBlock(t, e, x.otherwise)
	case forStmt:
		v, err := evalExpr(t, e, x.iter)
		if err != nil {
			return signal{err: err}
		}
		it, ok := v.(Iterable)
		if !ok {
			return signal{err: errorf("value of type %s is not iterable", v.Type())}
		}
		iter := it.Iterate()
		defer iter.Done()
		var item Value
		for iter.Next(&item) {
			if err := assign(e, x.target, item); err != nil {
				return signal{err: err}
			}
			r := execBlock(t, e, x.body)
			if r.err != nil || r.kind == "return" {
				return r
			}
			if r.kind == "break" {
				break
			}
			if err := t.step(); err != nil {
				return signal{err: err}
			}
		}
		return signal{}
	case whileStmt:
		for {
			v, err := evalExpr(t, e, x.cond)
			if err != nil {
				return signal{err: err}
			}
			if !truth(v) {
				break
			}
			r := execBlock(t, e, x.body)
			if r.err != nil || r.kind == "return" {
				return r
			}
			if r.kind == "break" {
				break
			}
			if err := t.step(); err != nil {
				return signal{err: err}
			}
		}
		return signal{}
	case returnStmt:
		if x.x == nil {
			return signal{kind: "return", value: None}
		}
		v, err := evalExpr(t, e, x.x)
		return signal{kind: "return", value: v, err: err}
	case flowStmt:
		if x.kind == "pass" {
			return signal{}
		}
		return signal{kind: x.kind}
	}
	return signal{err: errorf("unknown statement")}
}
func assign(e *env, target expr, v Value) error {
	switch x := target.(type) {
	case nameExpr:
		e.set(x.name, v)
		return nil
	case indexExpr:
		container, err := evalExpr(nil, e, x.x)
		if err != nil {
			return err
		}
		idx, err := evalExpr(nil, e, x.index)
		if err != nil {
			return err
		}
		switch c := container.(type) {
		case *List:
			i, ok := idx.(Int)
			if !ok {
				return errorf("list index must be int")
			}
			n := normalizeIndex(int(i), c.Len())
			if n < 0 || n >= c.Len() {
				return errorf("list index out of range")
			}
			c.elems[n] = v
			return nil
		case *Dict:
			return c.SetKey(idx, v)
		}
	case tupleExpr:
		values, err := valuesOf(v)
		if err != nil {
			return err
		}
		if len(values) != len(x.items) {
			return errorf("cannot unpack %d values into %d targets", len(values), len(x.items))
		}
		for i, target := range x.items {
			if err := assign(e, target, values[i]); err != nil {
				return err
			}
		}
		return nil
	}
	return errorf("cannot assign to expression")
}

func evalExpr(t *Thread, e *env, x expr) (Value, error) {
	if t != nil {
		if err := t.step(); err != nil {
			return nil, err
		}
	}
	switch q := x.(type) {
	case literalExpr:
		return q.v, nil
	case nameExpr:
		v, ok := e.get(q.name)
		if !ok {
			return nil, errorf("name %q is not defined", q.name)
		}
		return v, nil
	case listExpr:
		items := make([]Value, len(q.items))
		for i, z := range q.items {
			v, err := evalExpr(t, e, z)
			if err != nil {
				return nil, err
			}
			items[i] = v
		}
		return NewList(items), nil
	case tupleExpr:
		items := make(Tuple, len(q.items))
		for i, z := range q.items {
			v, err := evalExpr(t, e, z)
			if err != nil {
				return nil, err
			}
			items[i] = v
		}
		return items, nil
	case dictExpr:
		d := NewDict()
		for i, kx := range q.keys {
			k, err := evalExpr(t, e, kx)
			if err != nil {
				return nil, err
			}
			v, err := evalExpr(t, e, q.values[i])
			if err != nil {
				return nil, err
			}
			if err := d.SetKey(k, v); err != nil {
				return nil, err
			}
		}
		return d, nil
	case comprehensionExpr:
		iterable, err := evalExpr(t, e, q.iter)
		if err != nil {
			return nil, err
		}
		sequence, ok := iterable.(Iterable)
		if !ok {
			return nil, errorf("%s is not iterable", iterable.Type())
		}
		it := sequence.Iterate()
		defer it.Done()
		var item Value
		var out []Value
		for it.Next(&item) {
			if err := assign(e, q.target, item); err != nil {
				return nil, err
			}
			keep := true
			for _, cond := range q.conds {
				v, err := evalExpr(t, e, cond)
				if err != nil {
					return nil, err
				}
				if !truth(v) {
					keep = false
					break
				}
			}
			if keep {
				v, err := evalExpr(t, e, q.item)
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			}
		}
		return NewList(out), nil
	case dictComprehensionExpr:
		iterable, err := evalExpr(t, e, q.iter)
		if err != nil {
			return nil, err
		}
		sequence, ok := iterable.(Iterable)
		if !ok {
			return nil, errorf("%s is not iterable", iterable.Type())
		}
		it := sequence.Iterate()
		defer it.Done()
		out := NewDict()
		var item Value
		for it.Next(&item) {
			if err := assign(e, q.target, item); err != nil {
				return nil, err
			}
			keep := true
			for _, cond := range q.conds {
				v, err := evalExpr(t, e, cond)
				if err != nil {
					return nil, err
				}
				if !truth(v) {
					keep = false
					break
				}
			}
			if keep {
				key, err := evalExpr(t, e, q.key)
				if err != nil {
					return nil, err
				}
				value, err := evalExpr(t, e, q.value)
				if err != nil {
					return nil, err
				}
				if err := out.SetKey(key, value); err != nil {
					return nil, err
				}
			}
		}
		return out, nil
	case unaryExpr:
		v, err := evalExpr(t, e, q.x)
		if err != nil {
			return nil, err
		}
		switch q.op {
		case "not":
			return Bool(!truth(v)), nil
		case "+":
			if _, ok := v.(Int); ok {
				return v, nil
			}
			if _, ok := v.(Float); ok {
				return v, nil
			}
		case "-":
			switch n := v.(type) {
			case Int:
				return -n, nil
			case Float:
				return -n, nil
			}
		case "~":
			if n, ok := v.(Int); ok {
				return ^n, nil
			}
		}
		return nil, errorf("unsupported unary %s for %s", q.op, v.Type())
	case binaryExpr:
		if q.op == "and" || q.op == "or" {
			a, err := evalExpr(t, e, q.left)
			if err != nil {
				return nil, err
			}
			if (q.op == "and" && !truth(a)) || (q.op == "or" && truth(a)) {
				return a, nil
			}
			return evalExpr(t, e, q.right)
		}
		a, err := evalExpr(t, e, q.left)
		if err != nil {
			return nil, err
		}
		b, err := evalExpr(t, e, q.right)
		if err != nil {
			return nil, err
		}
		return binary(q.op, a, b)
	case conditionalExpr:
		c, err := evalExpr(t, e, q.cond)
		if err != nil {
			return nil, err
		}
		if truth(c) {
			return evalExpr(t, e, q.yes)
		}
		return evalExpr(t, e, q.no)
	case attrExpr:
		v, err := evalExpr(t, e, q.x)
		if err != nil {
			return nil, err
		}
		return getAttr(v, q.name)
	case indexExpr:
		v, err := evalExpr(t, e, q.x)
		if err != nil {
			return nil, err
		}
		if q.isSlice {
			return sliceValue(t, e, v, q.index, q.end)
		}
		idx, err := evalExpr(t, e, q.index)
		if err != nil {
			return nil, err
		}
		return indexValue(v, idx)
	case callExpr:
		fn, err := evalExpr(t, e, q.fn)
		if err != nil {
			return nil, err
		}
		args := make(Tuple, len(q.args))
		for i, a := range q.args {
			args[i], err = evalExpr(t, e, a)
			if err != nil {
				return nil, err
			}
		}
		kwargs := make([]Tuple, len(q.kwargs))
		for i, k := range q.kwargs {
			v, er := evalExpr(t, e, k.x)
			if er != nil {
				return nil, er
			}
			kwargs[i] = Tuple{String(k.name), v}
		}
		return call(t, fn, args, kwargs)
	}
	return nil, errorf("unknown expression")
}

func call(t *Thread, v Value, args Tuple, kwargs []Tuple) (Value, error) {
	switch fn := v.(type) {
	case *Builtin:
		return fn.fn(t, fn, args, kwargs)
	case *Function:
		return callFunction(t, fn, args, kwargs)
	default:
		return nil, errorf("%s value is not callable", v.Type())
	}
}

// Call invokes a Starlark function or host built-in.
func Call(thread *Thread, callable Value, args Tuple, kwargs []Tuple) (Value, error) {
	if thread == nil {
		thread = &Thread{}
	}
	return call(thread, callable, args, kwargs)
}
func callFunction(t *Thread, f *Function, args Tuple, kwargs []Tuple) (Value, error) {
	if len(args) > len(f.params) {
		return nil, errorf("%s: got too many arguments", f.name)
	}
	local := &env{vars: StringDict{}, parent: f.closure}
	used := map[string]bool{}
	for i, v := range args {
		local.vars[f.params[i].name] = v
		used[f.params[i].name] = true
	}
	for _, kw := range kwargs {
		name, _ := AsString(kw[0])
		found := -1
		for i, p := range f.params {
			if p.name == name {
				found = i
				break
			}
		}
		if found < 0 {
			return nil, errorf("%s: unexpected keyword %s", f.name, name)
		}
		if used[name] {
			return nil, errorf("%s: multiple values for %s", f.name, name)
		}
		local.vars[name] = kw[1]
		used[name] = true
	}
	for _, p := range f.params {
		if !used[p.name] {
			if !p.hasDefault {
				return nil, errorf("%s: missing argument %s", f.name, p.name)
			}
			local.vars[p.name] = p.def
		}
	}
	r := execBlock(t, local, f.body)
	if r.err != nil {
		return nil, r.err
	}
	if r.kind == "return" {
		return r.value, nil
	}
	if r.kind != "" {
		return nil, errorf("%s outside loop", r.kind)
	}
	return None, nil
}

func binary(op string, a, b Value) (Value, error) {
	if op == "==" {
		return Bool(equal(a, b)), nil
	}
	if op == "!=" {
		return Bool(!equal(a, b)), nil
	}
	if op == "in" {
		ok, err := contains(b, a)
		return Bool(ok), err
	}
	if ai, ok := a.(Int); ok {
		if bi, ok := b.(Int); ok {
			switch op {
			case "+":
				return ai + bi, nil
			case "-":
				return ai - bi, nil
			case "*":
				return ai * bi, nil
			case "/":
				if bi == 0 {
					return nil, errorf("division by zero")
				}
				return Float(ai) / Float(bi), nil
			case "//":
				if bi == 0 {
					return nil, errorf("division by zero")
				}
				return Int(floorDiv(int64(ai), int64(bi))), nil
			case "%":
				if bi == 0 {
					return nil, errorf("division by zero")
				}
				return Int(int64(ai) - floorDiv(int64(ai), int64(bi))*int64(bi)), nil
			case "**":
				if bi >= 0 {
					return Int(integerPower(int64(ai), int64(bi))), nil
				}
				value, err := numericPower(float64(ai), float64(bi))
				return Float(value), err
			case "|":
				return ai | bi, nil
			case "&":
				return ai & bi, nil
			case "^":
				return ai ^ bi, nil
			case "<":
				return Bool(ai < bi), nil
			case "<=":
				return Bool(ai <= bi), nil
			case ">":
				return Bool(ai > bi), nil
			case ">=":
				return Bool(ai >= bi), nil
			}
		}
	}
	if af, aok := numericFloat(a); aok {
		if bf, bok := numericFloat(b); bok {
			_, aFloat := a.(Float)
			_, bFloat := b.(Float)
			if aFloat || bFloat {
				switch op {
				case "+":
					return Float(af + bf), nil
				case "-":
					return Float(af - bf), nil
				case "*":
					return Float(af * bf), nil
				case "/":
					if bf == 0 {
						return nil, errorf("division by zero")
					}
					return Float(af / bf), nil
				case "//":
					if bf == 0 {
						return nil, errorf("division by zero")
					}
					return Float(floatFloor(af / bf)), nil
				case "%":
					if bf == 0 {
						return nil, errorf("division by zero")
					}
					return Float(af - floatFloor(af/bf)*bf), nil
				case "**":
					value, err := numericPower(af, bf)
					return Float(value), err
				case "<":
					return Bool(af < bf), nil
				case "<=":
					return Bool(af <= bf), nil
				case ">":
					return Bool(af > bf), nil
				case ">=":
					return Bool(af >= bf), nil
				}
			}
		}
	}
	if as, ok := a.(String); ok {
		switch op {
		case "+":
			if bs, ok := b.(String); ok {
				return as + bs, nil
			}
		case "*":
			if n, ok := b.(Int); ok {
				return String(strings.Repeat(string(as), max(0, int(n)))), nil
			}
		case "%":
			return stringFormat(string(as), b)
		case "<", "<=", ">", ">=":
			if bs, ok := b.(String); ok {
				return compareStrings(op, string(as), string(bs)), nil
			}
		}
	}
	if al, ok := a.(*List); ok {
		switch op {
		case "+":
			if bl, ok := b.(*List); ok {
				out := make([]Value, 0, len(al.elems)+len(bl.elems))
				out = append(out, al.elems...)
				out = append(out, bl.elems...)
				return NewList(out), nil
			}
		case "*":
			if n, ok := b.(Int); ok {
				var out []Value
				count := max(0, int(n))
				for i := 0; i < count; i++ {
					out = append(out, al.elems...)
				}
				return NewList(out), nil
			}
		}
	}
	if at, ok := a.(Tuple); ok {
		switch op {
		case "+":
			if bt, ok := b.(Tuple); ok {
				var out Tuple
				out = append(out, at...)
				out = append(out, bt...)
				return out, nil
			}
		case "*":
			if n, ok := b.(Int); ok {
				var out Tuple
				for i := 0; i < max(0, int(n)); i++ {
					out = append(out, at...)
				}
				return out, nil
			}
		}
	}
	if n, ok := a.(Int); ok && op == "*" {
		if s, ok := b.(String); ok {
			return String(strings.Repeat(string(s), max(0, int(n)))), nil
		}
		if l, ok := b.(*List); ok {
			var out []Value
			count := max(0, int(n))
			for i := 0; i < count; i++ {
				out = append(out, l.elems...)
			}
			return NewList(out), nil
		}
		if tuple, ok := b.(Tuple); ok {
			var out Tuple
			for i := 0; i < max(0, int(n)); i++ {
				out = append(out, tuple...)
			}
			return out, nil
		}
	}
	return nil, errorf("unsupported binary operation: %s %s %s", a.Type(), op, b.Type())
}

func numericFloat(value Value) (float64, bool) {
	switch number := value.(type) {
	case Int:
		return float64(number), true
	case Float:
		return float64(number), true
	}
	return 0, false
}

func integerPower(base, exponent int64) int64 {
	result := int64(1)
	for exponent > 0 {
		if exponent&1 != 0 {
			result *= base
		}
		base *= base
		exponent >>= 1
	}
	return result
}

func numericPower(base, exponent float64) (float64, error) {
	integerExponent := int64(exponent)
	if float64(integerExponent) != exponent {
		return 0, errorf("fractional exponent is unsupported")
	}
	if base == 0 && integerExponent < 0 {
		return 0, errorf("division by zero")
	}
	negative := integerExponent < 0
	if negative {
		integerExponent = -integerExponent
	}
	result := float64(1)
	for integerExponent > 0 {
		if integerExponent&1 != 0 {
			result *= base
		}
		base *= base
		integerExponent >>= 1
	}
	if negative {
		result = 1 / result
	}
	return result, nil
}

func floatFloor(value float64) float64 {
	integer := int64(value)
	result := float64(integer)
	if result > value {
		result--
	}
	return result
}
func floorDiv(a, b int64) int64 {
	q := a / b
	r := a % b
	if r != 0 && ((r < 0) != (b < 0)) {
		q--
	}
	return q
}
func compareStrings(op, a, b string) Bool {
	switch op {
	case "<":
		return Bool(a < b)
	case "<=":
		return Bool(a <= b)
	case ">":
		return Bool(a > b)
	default:
		return Bool(a >= b)
	}
}
func contains(container, needle Value) (bool, error) {
	switch x := container.(type) {
	case String:
		n, ok := AsString(needle)
		return ok && strings.Contains(string(x), n), nil
	case *List:
		for _, v := range x.elems {
			if equal(v, needle) {
				return true, nil
			}
		}
		return false, nil
	case Tuple:
		for _, v := range x {
			if equal(v, needle) {
				return true, nil
			}
		}
		return false, nil
	case *Dict:
		_, ok, _ := x.Get(needle)
		return ok, nil
	}
	return false, errorf("%s is not iterable", container.Type())
}

func indexValue(v, idx Value) (Value, error) {
	switch x := v.(type) {
	case *Dict:
		y, ok, _ := x.Get(idx)
		if !ok {
			return nil, errorf("key %s not found", idx.String())
		}
		return y, nil
	case *List:
		i, ok := idx.(Int)
		if !ok {
			return nil, errorf("list index must be int")
		}
		n := normalizeIndex(int(i), x.Len())
		if n < 0 || n >= x.Len() {
			return nil, errorf("list index out of range")
		}
		return x.Index(n), nil
	case Tuple:
		i, ok := idx.(Int)
		if !ok {
			return nil, errorf("tuple index must be int")
		}
		n := normalizeIndex(int(i), len(x))
		if n < 0 || n >= len(x) {
			return nil, errorf("tuple index out of range")
		}
		return x[n], nil
	case String:
		i, ok := idx.(Int)
		if !ok {
			return nil, errorf("string index must be int")
		}
		r := []rune(x)
		n := normalizeIndex(int(i), len(r))
		if n < 0 || n >= len(r) {
			return nil, errorf("string index out of range")
		}
		return stringFromRune(r[n]), nil
	}
	return nil, errorf("%s is not indexable", v.Type())
}
func normalizeIndex(i, n int) int {
	if i < 0 {
		return n + i
	}
	return i
}
func sliceValue(t *Thread, e *env, v Value, startX, endX expr) (Value, error) {
	start, end := 0, -1
	var err error
	if startX != nil {
		z, er := evalExpr(t, e, startX)
		if er != nil {
			return nil, er
		}
		n, ok := z.(Int)
		if !ok {
			return nil, errorf("slice index must be int")
		}
		start = int(n)
	}
	switch x := v.(type) {
	case *List:
		end = x.Len()
		if endX != nil {
			end, err = evalIndex(t, e, endX)
			if err != nil {
				return nil, err
			}
		}
		start, end = clampSlice(start, end, x.Len())
		return NewList(x.elems[start:end]), nil
	case Tuple:
		end = len(x)
		if endX != nil {
			end, err = evalIndex(t, e, endX)
			if err != nil {
				return nil, err
			}
		}
		start, end = clampSlice(start, end, len(x))
		var tuple Tuple
		tuple = append(tuple, x[start:end]...)
		var result Value
		result = tuple
		return result, nil
	case String:
		r := []rune(x)
		end = len(r)
		if endX != nil {
			end, err = evalIndex(t, e, endX)
			if err != nil {
				return nil, err
			}
		}
		start, end = clampSlice(start, end, len(r))
		return stringFromRunes(r[start:end]), nil
	}
	return nil, errorf("%s cannot be sliced", v.Type())
}
func evalIndex(t *Thread, e *env, x expr) (int, error) {
	v, err := evalExpr(t, e, x)
	if err != nil {
		return 0, err
	}
	n, ok := v.(Int)
	if !ok {
		return 0, errorf("slice index must be int")
	}
	return int(n), nil
}
func clampSlice(a, b, n int) (int, int) {
	if a < 0 {
		a = n + a
	}
	if b < 0 {
		b = n + b
	}
	a = max(0, min(a, n))
	b = max(0, min(b, n))
	if b < a {
		b = a
	}
	return a, b
}

func getAttr(v Value, name string) (Value, error) {
	if h, ok := v.(HasAttrs); ok {
		x, err := h.Attr(name)
		if err != nil {
			return nil, err
		}
		if x == nil {
			return nil, errorf("%s has no attribute %s", v.Type(), name)
		}
		return x, nil
	}
	if b := method(v, name); b != nil {
		return b, nil
	}
	return nil, errorf("%s has no attribute %s", v.Type(), name)
}
func method(receiver Value, name string) *Builtin {
	return NewBuiltin(receiver.Type()+"."+name, func(t *Thread, b *Builtin, args Tuple, kwargs []Tuple) (Value, error) {
		if len(kwargs) > 0 {
			return nil, errorf("%s: keyword arguments not supported", b.Name())
		}
		switch x := receiver.(type) {
		case String:
			s := string(x)
			switch name {
			case "lower":
				if len(args) == 0 {
					return String(lowerString(s)), nil
				}
			case "upper":
				if len(args) == 0 {
					return String(upperString(s)), nil
				}
			case "strip":
				if len(args) == 0 {
					return String(strings.TrimSpace(s)), nil
				}
			case "split":
				if len(args) <= 1 {
					sep := ""
					if len(args) == 1 {
						var ok bool
						sep, ok = AsString(args[0])
						if !ok {
							return nil, errorf("separator must be string")
						}
					}
					var p []string
					if sep == "" {
						p = strings.Fields(s)
					} else {
						p = strings.Split(s, sep)
					}
					v := make([]Value, len(p))
					for i, z := range p {
						v[i] = String(z)
					}
					return NewList(v), nil
				}
			case "rsplit":
				if len(args) <= 2 {
					sep := ""
					whitespace := true
					if len(args) >= 1 {
						if _, none := args[0].(noneType); !none {
							var ok bool
							sep, ok = AsString(args[0])
							if !ok {
								return nil, errorf("separator must be string or None")
							}
							if sep == "" {
								return nil, errorf("split: empty separator")
							}
							whitespace = false
						}
					}
					maxSplit := -1
					if len(args) == 2 {
						n, ok := args[1].(Int)
						if !ok {
							return nil, errorf("maxsplit must be int")
						}
						maxSplit = int(n)
					}
					var parts []string
					if whitespace {
						parts = splitWhitespaceFromRight(s, maxSplit)
					} else {
						parts = splitFromRight(s, sep, maxSplit)
					}
					values := make([]Value, len(parts))
					for i, part := range parts {
						values[i] = String(part)
					}
					return NewList(values), nil
				}
			case "replace":
				if len(args) == 2 || len(args) == 3 {
					a, aok := AsString(args[0])
					z, zok := AsString(args[1])
					if !aok || !zok {
						return nil, errorf("replace arguments must be strings")
					}
					n := -1
					if len(args) == 3 {
						q, ok := args[2].(Int)
						if !ok {
							return nil, errorf("count must be int")
						}
						n = int(q)
					}
					return String(strings.Replace(s, a, z, n)), nil
				}
			case "startswith", "endswith":
				if len(args) == 1 {
					z, ok := AsString(args[0])
					if !ok {
						return nil, errorf("argument must be string")
					}
					if name == "startswith" {
						return Bool(strings.HasPrefix(s, z)), nil
					}
					return Bool(strings.HasSuffix(s, z)), nil
				}
			case "join":
				if len(args) == 1 {
					it, ok := args[0].(Iterable)
					if !ok {
						return nil, errorf("join argument must be iterable")
					}
					iter := it.Iterate()
					defer iter.Done()
					var vals []string
					var z Value
					for iter.Next(&z) {
						q, ok := AsString(z)
						if !ok {
							return nil, errorf("join items must be strings")
						}
						vals = append(vals, q)
					}
					return String(strings.Join(vals, s)), nil
				}
			}
		case *List:
			switch name {
			case "append":
				if len(args) == 1 {
					x.elems = append(x.elems, args[0])
					return None, nil
				}
			case "extend":
				if len(args) == 1 {
					it, ok := args[0].(Iterable)
					if !ok {
						return nil, errorf("extend argument must be iterable")
					}
					q := it.Iterate()
					defer q.Done()
					var z Value
					for q.Next(&z) {
						x.elems = append(x.elems, z)
					}
					return None, nil
				}
			case "index":
				if len(args) == 1 {
					for i, z := range x.elems {
						if equal(z, args[0]) {
							return Int(i), nil
						}
					}
					return nil, errorf("item not found")
				}
			case "pop":
				i := len(x.elems) - 1
				if len(args) == 1 {
					n, ok := args[0].(Int)
					if !ok {
						return nil, errorf("index must be int")
					}
					i = normalizeIndex(int(n), len(x.elems))
				}
				if i < 0 || i >= len(x.elems) {
					return nil, errorf("pop index out of range")
				}
				z := x.elems[i]
				x.elems = append(x.elems[:i], x.elems[i+1:]...)
				return z, nil
			}
		case *Dict:
			switch name {
			case "get":
				if len(args) == 1 || len(args) == 2 {
					z, ok, _ := x.Get(args[0])
					if ok {
						return z, nil
					}
					if len(args) == 2 {
						return args[1], nil
					}
					return None, nil
				}
			case "keys":
				if len(args) == 0 {
					var z []Value
					for _, e := range x.entries {
						z = append(z, e.key)
					}
					return NewList(z), nil
				}
			case "values":
				if len(args) == 0 {
					var z []Value
					for _, e := range x.entries {
						z = append(z, e.value)
					}
					return NewList(z), nil
				}
			case "items":
				if len(args) == 0 {
					var z []Value
					for _, e := range x.entries {
						z = append(z, Tuple{e.key, e.value})
					}
					return NewList(z), nil
				}
			}
		}
		return nil, errorf("%s: invalid arguments", b.Name())
	})
}

func splitFromRight(s, sep string, maxSplit int) []string {
	if maxSplit == 0 {
		return []string{s}
	}
	var reversed []string
	rest := s
	for maxSplit < 0 || len(reversed) < maxSplit {
		index := strings.LastIndex(rest, sep)
		if index < 0 {
			break
		}
		reversed = append(reversed, rest[index+len(sep):])
		rest = rest[:index]
	}
	parts := make([]string, len(reversed)+1)
	parts[0] = rest
	for i := range reversed {
		parts[len(reversed)-i] = reversed[i]
	}
	return parts
}

func splitWhitespaceFromRight(s string, maxSplit int) []string {
	runes := []rune(s)
	end := len(runes)
	for end > 0 && isSpace(runes[end-1]) {
		end--
	}
	if end == 0 {
		return []string{}
	}
	if maxSplit == 0 {
		return []string{string(runes[:end])}
	}
	var reversed []string
	for end > 0 && (maxSplit < 0 || len(reversed) < maxSplit) {
		start := end
		for start > 0 && !isSpace(runes[start-1]) {
			start--
		}
		reversed = append(reversed, string(runes[start:end]))
		end = start
		for end > 0 && isSpace(runes[end-1]) {
			end--
		}
	}
	if end > 0 {
		reversed = append(reversed, string(runes[:end]))
	}
	parts := make([]string, len(reversed))
	for i := range reversed {
		parts[len(reversed)-1-i] = reversed[i]
	}
	return parts
}

func addCoreBuiltins(dict StringDict) {
	var fn BuiltinFunc
	fn = builtinPrint
	dict["print"] = newBuiltinValue("print", fn)
	fn = builtinFail
	dict["fail"] = newBuiltinValue("fail", fn)
	fn = builtinLen
	dict["len"] = newBuiltinValue("len", fn)
	fn = builtinRange
	dict["range"] = newBuiltinValue("range", fn)
	fn = builtinStr
	dict["str"] = newBuiltinValue("str", fn)
	fn = builtinRepr
	dict["repr"] = newBuiltinValue("repr", fn)
	fn = builtinBool
	dict["bool"] = newBuiltinValue("bool", fn)
	fn = builtinInt
	dict["int"] = newBuiltinValue("int", fn)
	fn = builtinList
	dict["list"] = newBuiltinValue("list", fn)
	fn = builtinTuple
	dict["tuple"] = newBuiltinValue("tuple", fn)
	fn = builtinSorted
	dict["sorted"] = newBuiltinValue("sorted", fn)
	fn = builtinEnumerate
	dict["enumerate"] = newBuiltinValue("enumerate", fn)
	fn = builtinType
	dict["type"] = newBuiltinValue("type", fn)
}

func newBuiltinValue(name string, fn BuiltinFunc) Value {
	var value Value
	value = NewBuiltin(name, fn)
	return value
}

func builtinPrint(t *Thread, _ *Builtin, args Tuple, _ []Tuple) (Value, error) {
	parts := make([]string, len(args))
	for i, v := range args {
		if s, ok := AsString(v); ok {
			parts[i] = s
		} else {
			parts[i] = v.String()
		}
	}
	if t.Print != nil {
		t.Print(t, strings.Join(parts, " "))
	}
	return None, nil
}

func builtinFail(_ *Thread, _ *Builtin, args Tuple, kwargs []Tuple) (Value, error) {
	if len(args) != 1 || len(kwargs) > 0 {
		return nil, errorf("fail: want one argument")
	}
	message, ok := AsString(args[0])
	if !ok {
		return nil, errorf("fail: argument must be string")
	}
	return nil, errors.New(message)
}

func builtinLen(_ *Thread, _ *Builtin, args Tuple, kw []Tuple) (Value, error) {
	if len(args) != 1 || len(kw) > 0 {
		return nil, errorf("len: want one argument")
	}
	switch x := args[0].(type) {
	case String:
		return Int(len([]rune(x))), nil
	case *List:
		return Int(x.Len()), nil
	case Tuple:
		return Int(len(x)), nil
	case *Dict:
		return Int(x.Len()), nil
	}
	return nil, errorf("len: %s has no length", args[0].Type())
}

func builtinBool(_ *Thread, _ *Builtin, args Tuple, _ []Tuple) (Value, error) {
	if len(args) > 1 {
		return nil, errorf("bool: want at most one argument")
	}
	if len(args) == 0 {
		return False, nil
	}
	return Bool(truth(args[0])), nil
}

func builtinType(_ *Thread, _ *Builtin, args Tuple, _ []Tuple) (Value, error) {
	if len(args) != 1 {
		return nil, errorf("type: want one argument")
	}
	return String(args[0].Type()), nil
}

func builtinRange(_ *Thread, _ *Builtin, a Tuple, kw []Tuple) (Value, error) {
	if len(kw) > 0 || len(a) < 1 || len(a) > 3 {
		return nil, errorf("range: want 1 to 3 integers")
	}
	nums := make([]int, len(a))
	for i, v := range a {
		n, ok := v.(Int)
		if !ok {
			return nil, errorf("range: arguments must be int")
		}
		nums[i] = int(n)
	}
	start, end, step := 0, nums[0], 1
	if len(nums) >= 2 {
		start, end = nums[0], nums[1]
	}
	if len(nums) == 3 {
		step = nums[2]
	}
	if step == 0 {
		return nil, errorf("range: step cannot be zero")
	}
	var out []Value
	for i := start; (step > 0 && i < end) || (step < 0 && i > end); i += step {
		out = append(out, Int(i))
	}
	return NewList(out), nil
}
func builtinStr(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) != 1 {
		return nil, errorf("str: want one argument")
	}
	if s, ok := AsString(a[0]); ok {
		return String(s), nil
	}
	return String(a[0].String()), nil
}
func builtinRepr(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) != 1 {
		return nil, errorf("repr: want one argument")
	}
	return String(a[0].String()), nil
}
func builtinInt(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) != 1 {
		return nil, errorf("int: want one argument")
	}
	switch x := a[0].(type) {
	case Int:
		return x, nil
	case Bool:
		if x {
			return Int(1), nil
		}
		return Int(0), nil
	case String:
		n, e := strconv.ParseInt(string(x), 10, 64)
		return Int(n), e
	case Float:
		return Int(x), nil
	}
	return nil, errorf("int: cannot convert %s", a[0].Type())
}
func valuesOf(v Value) ([]Value, error) {
	it, ok := v.(Iterable)
	if !ok {
		return nil, errorf("%s is not iterable", v.Type())
	}
	q := it.Iterate()
	defer q.Done()
	var out []Value
	var z Value
	for q.Next(&z) {
		out = append(out, z)
	}
	return out, nil
}
func builtinList(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) == 0 {
		return NewList(nil), nil
	}
	if len(a) != 1 {
		return nil, errorf("list: want at most one argument")
	}
	v, e := valuesOf(a[0])
	return NewList(v), e
}
func builtinTuple(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) == 0 {
		return Tuple{}, nil
	}
	if len(a) != 1 {
		return nil, errorf("tuple: want at most one argument")
	}
	v, e := valuesOf(a[0])
	var tuple Tuple
	tuple = append(tuple, v...)
	return tuple, e
}
func builtinSorted(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) != 1 {
		return nil, errorf("sorted: want one argument")
	}
	v, e := valuesOf(a[0])
	if e != nil {
		return nil, e
	}
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && valueLess(v[j], v[j-1]); j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
	return NewList(v), nil
}

func valueLess(a, b Value) bool {
	switch x := a.(type) {
	case Int:
		if y, ok := b.(Int); ok {
			return x < y
		}
	case Float:
		if y, ok := b.(Float); ok {
			return x < y
		}
	case String:
		if y, ok := b.(String); ok {
			return x < y
		}
	case Bool:
		if y, ok := b.(Bool); ok {
			return !bool(x) && bool(y)
		}
	}
	return a.String() < b.String()
}
func builtinEnumerate(_ *Thread, _ *Builtin, a Tuple, _ []Tuple) (Value, error) {
	if len(a) < 1 || len(a) > 2 {
		return nil, errorf("enumerate: want one or two arguments")
	}
	v, e := valuesOf(a[0])
	if e != nil {
		return nil, e
	}
	start := Int(0)
	if len(a) == 2 {
		var ok bool
		start, ok = a[1].(Int)
		if !ok {
			return nil, errorf("enumerate start must be int")
		}
	}
	out := make([]Value, len(v))
	for i, z := range v {
		out[i] = Tuple{start + Int(i), z}
	}
	return NewList(out), nil
}
func stringFormat(format string, v Value) (Value, error) {
	var args []Value
	if t, ok := v.(Tuple); ok {
		args = []Value(t)
	} else {
		args = []Value{v}
	}
	var b stringBuilder
	ai := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			b.WriteByte(format[i])
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			b.WriteByte('%')
			i++
			continue
		}
		if i+1 >= len(format) || ai >= len(args) {
			return nil, errorf("not enough arguments for format string")
		}
		i++
		x := args[ai]
		ai++
		switch format[i] {
		case 's':
			if s, ok := AsString(x); ok {
				b.WriteString(s)
			} else {
				b.WriteString(x.String())
			}
		case 'r':
			b.WriteString(x.String())
		case 'd':
			n, ok := x.(Int)
			if !ok {
				return nil, errorf("%%d requires int")
			}
			b.WriteString(n.String())
		default:
			return nil, errorf("unsupported format character")
		}
	}
	return String(b.String()), nil
}
