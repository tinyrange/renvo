package check

// Generic checking needs type identity independently of source spelling and
// token lifetime. IDs are local to a checking context; zero is invalid. Aliases
// resolve to their target ID rather than introducing another named identity.
const (
	genericInvalid = iota
	genericBasic
	genericNamed
	genericParameter
	genericPointer
	genericSlice
	genericArray
	genericMap
	genericChan
	genericFunc
	genericStruct
	genericInterface
)

type genericField struct {
	name     string
	pkg      string // declaration package for an unexported field
	typ      int
	tag      string
	embedded bool
}

type genericMethod struct {
	name     string
	pkg      string // declaration package for an unexported method
	typ      int    // receiver-free signature
	pointer  bool
	embedded bool // field entry returned by promotedMembers
}

type genericType struct {
	kind       int
	name       string
	origin     string // declaration identity, including local scope when needed
	elem       int
	key        int
	length     uint64 // target array length; ^uint64(0) means inferred
	direction  int
	variadic   bool
	args       []int
	params     []int
	results    []int
	fields     []genericField
	methods    []genericMethod
	terms      []genericTerm
	restricted bool
	comparable bool
	underlying int // filled after reserving a recursive named type
}

type genericTypes struct {
	items    []genericType
	buckets  []int
	invalid  genericType
	nonbasic bool
}

func (t *genericTypes) intern(v genericType) int {
	if v.kind == genericInterface && (v.restricted || v.comparable) {
		t.nonbasic = true
	}
	if len(t.buckets) == 0 || (len(t.items)+1)*2 > len(t.buckets) {
		t.rehash(len(t.buckets)*2 + 31)
	}
	hash := genericTypeHash(&v)
	i := hash % len(t.buckets)
	for t.buckets[i] != 0 {
		id := t.buckets[i]
		if genericTypeEqual(nil, &t.items[id-1], &v) {
			return id
		}
		i = (i + 1) % len(t.buckets)
	}
	// Own these slices: inference and substitution reuse their scratch buffers.
	v.args = append([]int(nil), v.args...)
	v.params = append([]int(nil), v.params...)
	v.results = append([]int(nil), v.results...)
	v.fields = append([]genericField(nil), v.fields...)
	v.methods = append([]genericMethod(nil), v.methods...)
	v.terms = append([]genericTerm(nil), v.terms...)
	t.items = append(t.items, v)
	id := len(t.items)
	t.buckets[i] = id
	return id
}

func (t *genericTypes) rehash(size int) {
	t.buckets = make([]int, size)
	for j := 0; j < len(t.items); j++ {
		i := genericTypeHash(&t.items[j]) % size
		for t.buckets[i] != 0 {
			i = (i + 1) % size
		}
		t.buckets[i] = j + 1
	}
}

func genericHashWord(hash int, value int) int {
	return (hash*33 + value) & 2147483647
}

func genericHashText(hash int, value string) int {
	for i := 0; i < len(value); i++ {
		hash = genericHashWord(hash, int(value[i]))
	}
	return genericHashWord(hash, len(value))
}

func genericTypeHash(v *genericType) int {
	h := genericHashText(v.kind, v.name)
	h = genericHashText(h, v.origin)
	if v.kind == genericNamed || v.kind == genericParameter {
		for _, id := range v.args {
			h = genericHashWord(h, id)
		}
		return h
	}
	h = genericHashWord(h, v.elem)
	h = genericHashWord(h, v.key)
	h = genericHashWord(h, int(v.length))
	if v.length > 2147483647 {
		h = genericHashWord(h, int(v.length>>31))
	}
	h = genericHashWord(h, v.direction)
	if v.variadic {
		h = genericHashWord(h, 1)
	}
	for _, id := range v.params {
		h = genericHashWord(h, id)
	}
	for _, id := range v.results {
		h = genericHashWord(h, id)
	}
	for _, f := range v.fields {
		h = genericHashText(h, f.name)
		h = genericHashText(h, f.pkg)
		h = genericHashText(h, f.tag)
		h = genericHashWord(h, f.typ)
		if f.embedded {
			h = genericHashWord(h, 1)
		}
	}
	for _, m := range v.methods {
		h = genericHashText(h, m.name)
		h = genericHashText(h, m.pkg)
		h = genericHashWord(h, m.typ)
	}
	if v.restricted {
		h = genericHashWord(h, 1)
	}
	if v.comparable {
		h = genericHashWord(h, 2)
	}
	for _, term := range v.terms {
		h = genericHashWord(h, term.typ)
		if term.tilde {
			h = genericHashWord(h, 3)
		}
	}
	return h
}

func genericIDsEqual(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A non-nil table compares unnamed component types recursively, ignoring
// struct tags as required for explicit conversions. Named identity is retained.
func genericTypeEqual(t *genericTypes, a *genericType, b *genericType) bool {
	if a.kind != b.kind || a.name != b.name || a.origin != b.origin {
		return false
	}
	if a.kind == genericNamed || a.kind == genericParameter {
		return genericIDsEqual(a.args, b.args)
	}
	if !genericConversionTypeEqual(t, a.elem, b.elem) || !genericConversionTypeEqual(t, a.key, b.key) || a.length != b.length || a.direction != b.direction || a.variadic != b.variadic {
		return false
	}
	if !genericConversionIDsEqual(t, a.params, b.params) || !genericConversionIDsEqual(t, a.results, b.results) || len(a.fields) != len(b.fields) || len(a.methods) != len(b.methods) {
		return false
	}
	if a.restricted != b.restricted || a.comparable != b.comparable || len(a.terms) != len(b.terms) {
		return false
	}
	for i := 0; i < len(a.terms); i++ {
		if a.terms[i] != b.terms[i] {
			return false
		}
	}
	for i := 0; i < len(a.fields); i++ {
		x, y := a.fields[i], b.fields[i]
		if x.name != y.name || x.pkg != y.pkg || x.embedded != y.embedded || t == nil && x.tag != y.tag || !genericConversionTypeEqual(t, x.typ, y.typ) {
			return false
		}
	}
	for i := 0; i < len(a.methods); i++ {
		x, y := a.methods[i], b.methods[i]
		if x.name != y.name || x.pkg != y.pkg || x.pointer != y.pointer || x.embedded != y.embedded || !genericConversionTypeEqual(t, x.typ, y.typ) {
			return false
		}
	}
	return true
}

func (t *genericTypes) get(id int) *genericType {
	if id <= 0 || id > len(t.items) {
		return &t.invalid
	}
	return &t.items[id-1]
}

func (t *genericTypes) basic(name string) int {
	if name == "byte" {
		name = "uint8"
	}
	if name == "rune" {
		name = "int32"
	}
	return t.intern(genericType{kind: genericBasic, name: name})
}

func (t *genericTypes) errorType() int {
	sig := t.intern(genericType{kind: genericFunc, results: []int{t.basic("string")}})
	methods := []genericMethod{{name: "Error", typ: sig}}
	underlying := t.intern(genericType{kind: genericInterface, methods: methods})
	return t.intern(genericType{kind: genericNamed, name: "error", origin: "builtin.error", underlying: underlying})
}

func (t *genericTypes) underlying(id int) int {
	// A chain longer than the table has a cycle or an unresolved declaration.
	for n := 0; n <= len(t.items); n++ {
		v := t.get(id)
		if v.kind == genericInvalid {
			return 0
		}
		if v.kind != genericNamed {
			return id
		}
		id = v.underlying
	}
	return 0
}

func (t *genericTypes) comparable(id int, strict bool) bool {
	return t.comparableAt(id, strict, 0)
}

func (t *genericTypes) comparableAt(id int, strict bool, depth int) bool {
	if depth > len(t.items) {
		return false
	}
	v := t.get(t.underlying(id))
	if v.kind == genericBasic || v.kind == genericPointer || v.kind == genericChan {
		return true
	}
	if v.kind == genericInterface {
		return !strict
	}
	if v.kind == genericArray {
		return t.comparableAt(v.elem, strict, depth+1)
	}
	if v.kind == genericStruct {
		for _, f := range v.fields {
			if !t.comparableAt(f.typ, strict, depth+1) {
				return false
			}
		}
		return true
	}
	return false
}

// substitute preserves defined type identity: N[int] and N[string] remain
// distinct even when their underlying representations happen to coincide.
func (t *genericTypes) substitute(id int, parameters []int, arguments []int) int {
	var before []int
	var after []int
	return t.substituteMemo(id, parameters, arguments, &before, &after)
}

func (t *genericTypes) substituteMemo(id int, parameters []int, arguments []int, before *[]int, after *[]int) int {
	for i := 0; i < len(parameters); i++ {
		if id == parameters[i] {
			if i < len(arguments) {
				return arguments[i]
			}
			return 0
		}
	}
	for i := 0; i < len(*before); i++ {
		if id == (*before)[i] {
			return (*after)[i]
		}
	}
	v := *t.get(id)
	if v.kind == genericInvalid || v.kind == genericBasic || v.kind == genericParameter {
		return id
	}
	if v.kind == genericNamed {
		args := make([]int, len(v.args))
		for i := 0; i < len(args); i++ {
			args[i] = t.substituteMemo(v.args[i], parameters, arguments, before, after)
		}
		if genericIDsEqual(args, v.args) {
			return id
		}
		out := t.intern(genericType{kind: genericNamed, name: v.name, origin: v.origin, args: args})
		*before = append(*before, id)
		*after = append(*after, out)
		underlying := t.substituteMemo(v.underlying, parameters, arguments, before, after)
		t.items[out-1].underlying = underlying
		methods := append([]genericMethod(nil), v.methods...)
		for i := 0; i < len(methods); i++ {
			methods[i].typ = t.substituteMemo(methods[i].typ, parameters, arguments, before, after)
		}
		t.items[out-1].methods = methods
		return out
	}
	if v.elem != 0 {
		v.elem = t.substituteMemo(v.elem, parameters, arguments, before, after)
	}
	if v.key != 0 {
		v.key = t.substituteMemo(v.key, parameters, arguments, before, after)
	}
	params := make([]int, len(v.params))
	for i := 0; i < len(params); i++ {
		params[i] = t.substituteMemo(v.params[i], parameters, arguments, before, after)
	}
	v.params = params
	results := make([]int, len(v.results))
	for i := 0; i < len(results); i++ {
		results[i] = t.substituteMemo(v.results[i], parameters, arguments, before, after)
	}
	v.results = results
	fields := append([]genericField(nil), v.fields...)
	for i := 0; i < len(fields); i++ {
		fields[i].typ = t.substituteMemo(fields[i].typ, parameters, arguments, before, after)
	}
	v.fields = fields
	methods := append([]genericMethod(nil), v.methods...)
	for i := 0; i < len(methods); i++ {
		methods[i].typ = t.substituteMemo(methods[i].typ, parameters, arguments, before, after)
	}
	v.methods = methods
	terms := append([]genericTerm(nil), v.terms...)
	for i := 0; i < len(terms); i++ {
		terms[i].typ = t.substituteMemo(terms[i].typ, parameters, arguments, before, after)
	}
	v.terms = terms
	out := t.intern(v)
	*before = append(*before, id)
	*after = append(*after, out)
	return out
}

func genericConversionTypeEqual(t *genericTypes, a int, b int) bool {
	if a == b {
		return true
	}
	if t == nil {
		return false
	}
	return genericTypeEqual(t, t.get(a), t.get(b))
}
func genericConversionIDsEqual(t *genericTypes, a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !genericConversionTypeEqual(t, a[i], b[i]) {
			return false
		}
	}
	return true
}
