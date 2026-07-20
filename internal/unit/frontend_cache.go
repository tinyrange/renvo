package unit

const frontendCacheMagic = "RVFC1"

// MarshalFrontendCache preserves the linker-only semantic tables omitted from
// the backend unit format. It is used for in-process package caching; it is not
// a public or stable object format.
func MarshalFrontendCache(program Program) ([]byte, bool) {
	out := make([]byte, 0, len(program.Text)+len(program.Tokens)*8)
	out = append(out, frontendCacheMagic...)
	out = appendFrontendCacheString(out, program.Package)
	out = appendFrontendCacheString(out, program.ImportPath)
	out = appendFrontendCacheBytes(out, program.Text)
	out = appendFrontendCacheInt(out, len(program.Tokens))
	for i := 0; i < len(program.Tokens); i++ {
		item := program.Tokens[i]
		out = appendFrontendCacheInt(out, item.KindLine&255)
		out = appendFrontendCacheInt(out, item.Start)
		out = appendFrontendCacheInt(out, item.Size)
		out = appendFrontendCacheInt(out, item.KindLine>>8)
	}
	out = appendFrontendCacheInt(out, len(program.Imports))
	for i := 0; i < len(program.Imports); i++ {
		item := program.Imports[i]
		out = appendFrontendCacheInt(out, item.NameTok)
		out = appendFrontendCacheInt(out, item.PathTok)
	}
	out = appendFrontendCacheInt(out, len(program.Symbols))
	for i := 0; i < len(program.Symbols); i++ {
		item := program.Symbols[i]
		out = appendFrontendCacheString(out, item.Name)
		out = appendFrontendCacheInt(out, item.Package)
		out = appendFrontendCacheInt(out, item.Token)
	}
	out = appendFrontendCacheInt(out, len(program.Decls))
	for i := 0; i < len(program.Decls); i++ {
		item := program.Decls[i]
		out = appendFrontendCacheInt(out, item.Kind)
		out = appendFrontendCacheInt(out, item.NameStart)
		out = appendFrontendCacheInt(out, item.NameEnd)
		out = appendFrontendCacheInt(out, item.StartTok)
		out = appendFrontendCacheInt(out, item.EndTok)
	}
	out = appendFrontendCacheInt(out, len(program.Funcs))
	for i := 0; i < len(program.Funcs); i++ {
		item := program.Funcs[i]
		out = appendFrontendCacheInt(out, item.NameStart)
		out = appendFrontendCacheInt(out, item.NameEnd)
		out = appendFrontendCacheInt(out, item.StartTok)
		out = appendFrontendCacheInt(out, item.NameTok)
		out = appendFrontendCacheInt(out, item.ReceiverStart)
		out = appendFrontendCacheInt(out, item.ReceiverEnd)
		out = appendFrontendCacheInt(out, item.BodyStart)
		out = appendFrontendCacheInt(out, item.BodyEnd)
		out = appendFrontendCacheInt(out, item.EndTok)
	}
	out = appendFrontendCacheInt(out, len(program.TypeRefs))
	for i := 0; i < len(program.TypeRefs); i++ {
		item := program.TypeRefs[i]
		out = appendFrontendCacheInt(out, item.Kind)
		out = appendFrontendCacheInt(out, item.Token)
		out = appendFrontendCacheInt(out, item.BaseTok)
		out = appendFrontendCacheInt(out, item.DotTok)
		out = appendFrontendCacheInt(out, item.Package)
		out = appendFrontendCacheInt(out, item.Symbol)
	}
	out = appendFrontendCacheInt(out, len(program.Calls))
	for i := 0; i < len(program.Calls); i++ {
		item := program.Calls[i]
		out = appendFrontendCacheInt(out, item.Kind)
		out = appendFrontendCacheInt(out, item.CalleeTok)
		out = appendFrontendCacheInt(out, item.BaseTok)
		out = appendFrontendCacheInt(out, item.DotTok)
	}
	out = appendFrontendCacheInt(out, len(program.Refs))
	for i := 0; i < len(program.Refs); i++ {
		item := program.Refs[i]
		out = appendFrontendCacheInt(out, item.Kind)
		out = appendFrontendCacheInt(out, item.Token)
		out = appendFrontendCacheInt(out, item.Index)
		out = appendFrontendCacheInt(out, item.Package)
	}
	out = appendFrontendCacheInt(out, len(program.Selectors))
	for i := 0; i < len(program.Selectors); i++ {
		item := program.Selectors[i]
		out = appendFrontendCacheInt(out, item.BaseTok)
		out = appendFrontendCacheInt(out, item.DotTok)
		out = appendFrontendCacheInt(out, item.NameTok)
		out = appendFrontendCacheInt(out, item.BaseKind)
		out = appendFrontendCacheInt(out, item.BaseIndex)
		out = appendFrontendCacheInt(out, item.BasePackage)
		out = appendFrontendCacheInt(out, item.Package)
		out = appendFrontendCacheInt(out, item.Symbol)
	}
	return out, true
}

type frontendCacheReader struct {
	data []byte
	pos  int
	ok   bool
}

func UnmarshalFrontendCache(data []byte) (Program, bool) {
	var out Program
	if len(data) < len(frontendCacheMagic) || string(data[:len(frontendCacheMagic)]) != frontendCacheMagic {
		return out, false
	}
	r := frontendCacheReader{data: data, pos: len(frontendCacheMagic), ok: true}
	out.Package = r.stringValue()
	out.ImportPath = r.stringValue()
	out.Text = r.byteValue()
	out.Tokens = make([]Token, r.count())
	for i := 0; i < len(out.Tokens); i++ {
		kind := r.intValue()
		start := r.intValue()
		size := r.intValue()
		line := r.intValue()
		out.Tokens[i] = MakeToken(kind, start, size, line)
	}
	out.Imports = make([]Import, r.count())
	for i := 0; i < len(out.Imports); i++ {
		out.Imports[i] = Import{NameTok: r.intValue(), PathTok: r.intValue()}
	}
	out.Symbols = make([]Symbol, r.count())
	for i := 0; i < len(out.Symbols); i++ {
		out.Symbols[i] = Symbol{Name: r.stringValue(), Package: r.intValue(), Token: r.intValue()}
	}
	out.Decls = make([]Decl, r.count())
	for i := 0; i < len(out.Decls); i++ {
		out.Decls[i] = Decl{Kind: r.intValue(), NameStart: r.intValue(), NameEnd: r.intValue(), StartTok: r.intValue(), EndTok: r.intValue()}
	}
	out.Funcs = make([]Func, r.count())
	for i := 0; i < len(out.Funcs); i++ {
		out.Funcs[i] = Func{
			NameStart: r.intValue(), NameEnd: r.intValue(), StartTok: r.intValue(), NameTok: r.intValue(),
			ReceiverStart: r.intValue(), ReceiverEnd: r.intValue(), BodyStart: r.intValue(), BodyEnd: r.intValue(), EndTok: r.intValue(),
		}
	}
	out.TypeRefs = make([]TypeRef, r.count())
	for i := 0; i < len(out.TypeRefs); i++ {
		out.TypeRefs[i] = TypeRef{Kind: r.intValue(), Token: r.intValue(), BaseTok: r.intValue(), DotTok: r.intValue(), Package: r.intValue(), Symbol: r.intValue()}
	}
	out.Calls = make([]Call, r.count())
	for i := 0; i < len(out.Calls); i++ {
		out.Calls[i] = Call{Kind: r.intValue(), CalleeTok: r.intValue(), BaseTok: r.intValue(), DotTok: r.intValue()}
	}
	out.Refs = make([]NameRef, r.count())
	for i := 0; i < len(out.Refs); i++ {
		out.Refs[i] = NameRef{Kind: r.intValue(), Token: r.intValue(), Index: r.intValue(), Package: r.intValue()}
	}
	out.Selectors = make([]Selector, r.count())
	for i := 0; i < len(out.Selectors); i++ {
		out.Selectors[i] = Selector{
			BaseTok: r.intValue(), DotTok: r.intValue(), NameTok: r.intValue(), BaseKind: r.intValue(),
			BaseIndex: r.intValue(), BasePackage: r.intValue(), Package: r.intValue(), Symbol: r.intValue(),
		}
	}
	return out, r.ok && r.pos == len(r.data)
}

func appendFrontendCacheInt(out []byte, value int) []byte {
	return appendVarint(out, value+1)
}

func appendFrontendCacheBytes(out []byte, value []byte) []byte {
	out = appendVarint(out, len(value))
	return append(out, value...)
}

func appendFrontendCacheString(out []byte, value string) []byte {
	out = appendVarint(out, len(value))
	return append(out, value...)
}

func (r *frontendCacheReader) rawVarint() int {
	if !r.ok {
		return 0
	}
	value := 0
	shift := 0
	for r.pos < len(r.data) && shift < 63 {
		part := int(r.data[r.pos])
		r.pos++
		value |= (part & 0x7f) << shift
		if part < 0x80 {
			return value
		}
		shift += 7
	}
	r.ok = false
	return 0
}

func (r *frontendCacheReader) intValue() int { return r.rawVarint() - 1 }

func (r *frontendCacheReader) count() int {
	value := r.intValue()
	if value < 0 || value > 1<<24 {
		r.ok = false
		return 0
	}
	return value
}

func (r *frontendCacheReader) byteValue() []byte {
	size := r.rawVarint()
	if !r.ok || size < 0 || size > len(r.data)-r.pos {
		r.ok = false
		return nil
	}
	value := make([]byte, size)
	copy(value, r.data[r.pos:r.pos+size])
	r.pos += size
	return value
}

func (r *frontendCacheReader) stringValue() string {
	return string(r.byteValue())
}
