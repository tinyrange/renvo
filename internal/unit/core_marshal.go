package unit

import "renvo.dev/internal/arena"

const transientMarshalChunk = 8192

func renvo_runtime_ArenaDiscardUnitTokens(tokens []Token) {}

// MarshalCore encodes the compact unit contract consumed by every backend.
// Frontend-only semantic tables deliberately remain outside this boundary.
func MarshalCore(program CoreProgram) ([]byte, bool) {
	return marshalCore(program, false)
}

// MarshalCoreTransient encodes a linked program whose text and token storage
// will never be used again. It releases completed arena pages while encoding
// so the compact output does not overlap the full linked representation at the
// frontend's peak resident set.
func MarshalCoreTransient(program CoreProgram) ([]byte, bool) {
	return marshalCore(program, true)
}

func marshalCore(program CoreProgram, transient bool) ([]byte, bool) {
	assemblyCapacity := len(program.RTGAssemblyFuncs) * 4
	for i := 0; i < len(program.RTGAssembly); i++ {
		assemblyCapacity += len(program.RTGAssembly[i].Path) + len(program.RTGAssembly[i].Source) + 4
	}
	capacity := 82 + unboundTargetBindingReserve + len(program.Package) + len(program.ImportPath) + len(program.Text) + len(program.Tokens)*5 + len(program.Decls)*8 + len(program.Funcs)*12 + len(program.Packages)*48 + assemblyCapacity
	out := make([]byte, 0, capacity)
	for i := 0; i < len(Magic); i++ {
		out = append(out, Magic[i])
	}
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendUint16(out, TagUnit)
	rootLength := len(out)
	out = appendUint32(out, 0)
	out = appendStringNodeCore(out, TagPackage, program.Package)
	out = appendStringNodeCore(out, TagImportPath, program.ImportPath)
	if transient {
		out = appendTransientTextNodeCore(out, program.Text)
	} else {
		out = appendNode(out, TagText, program.Text)
	}
	tokenHeader := len(out)
	out = appendNodeHeader(out, TagTokens, 0)
	tokenStart := len(out)
	out = appendEncodedTokensCore(out, program.Tokens, transient)
	patchUint32Core(out, tokenHeader+2, len(out)-tokenStart)
	declHeader := len(out)
	out = appendNodeHeader(out, TagDecls, 0)
	declStart := len(out)
	out = appendEncodedDeclsCore(out, program.Decls)
	patchUint32Core(out, declHeader+2, len(out)-declStart)
	funcHeader := len(out)
	out = appendNodeHeader(out, TagFuncs, 0)
	funcStart := len(out)
	out = appendEncodedFuncsCore(out, program.Funcs)
	patchUint32Core(out, funcHeader+2, len(out)-funcStart)
	if len(program.Packages) > 0 {
		packageHeader := len(out)
		out = appendNodeHeader(out, TagPackages, 0)
		packageStart := len(out)
		out = appendEncodedPackagesCore(out, program.Packages)
		patchUint32Core(out, packageHeader+2, len(out)-packageStart)
	}
	if len(program.RTGAssembly) > 0 || len(program.RTGAssemblyFuncs) > 0 {
		out = appendNode(out, TagRTGAssembly, encodeRTGAssemblyCore(program.RTGAssembly, program.RTGAssemblyFuncs))
	}
	if program.Entrypoint > 0 {
		payload := appendVarint(nil, program.Entrypoint-1)
		out = appendNode(out, TagEntrypoint, payload)
	}
	if len(program.ForeignPrograms) > 0 {
		out = appendNode(out, TagForeignPrograms, encodeForeignProgramsCore(program.ForeignPrograms))
	}
	patchUint32Core(out, rootLength, len(out)-14)
	return out, true
}

func encodeForeignProgramsCore(programs []ForeignProgram) []byte {
	out := appendVarint(nil, len(programs))
	for i := 0; i < len(programs); i++ {
		program := programs[i]
		out = appendVarint(out, len(program.Name))
		out = appendCoreStringBytes(out, program.Name)
		out = appendVarint(out, program.Kind)
		out = appendVarint(out, len(program.Target))
		out = appendCoreStringBytes(out, program.Target)
		if program.InPlace {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
		out = appendVarint(out, len(program.Unit))
		out = append(out, program.Unit...)
		out = appendVarint(out, len(program.Artifact))
		out = append(out, program.Artifact...)
		out = appendVarint(out, program.EntryOffset)
	}
	return out
}

func encodeRTGAssemblyCore(sources []RTGAssemblySource, bindings []RTGAssemblyBinding) []byte {
	out := make([]byte, 0)
	out = appendVarint(out, len(sources))
	for i := 0; i < len(sources); i++ {
		out = appendVarint(out, len(sources[i].Path))
		out = appendCoreStringBytes(out, sources[i].Path)
		out = appendVarint(out, len(sources[i].Source))
		out = append(out, sources[i].Source...)
	}
	out = appendVarint(out, len(bindings))
	for i := 0; i < len(bindings); i++ {
		out = appendVarint(out, bindings[i].Func)
		out = appendVarint(out, bindings[i].Source)
		out = appendVarint(out, bindings[i].Entry)
		out = appendVarint(out, len(bindings[i].Code))
		out = append(out, bindings[i].Code...)
	}
	return out
}

func appendEncodedPackagesCore(out []byte, packages []PackageInfo) []byte {
	out = appendVarint(out, len(packages))
	for i := 0; i < len(packages); i++ {
		item := packages[i]
		out = appendVarint(out, len(item.Name))
		out = appendCoreStringBytes(out, item.Name)
		out = appendVarint(out, len(item.ImportPath))
		out = appendCoreStringBytes(out, item.ImportPath)
		out = appendUint32(out, item.GraphKeyA)
		out = appendUint32(out, item.GraphKeyB)
		out = appendUint32(out, item.SourceKeyA)
		out = appendUint32(out, item.SourceKeyB)
		out = appendVarint(out, item.TextStart)
		out = appendVarint(out, item.TextEnd-item.TextStart)
		out = appendVarint(out, item.TokenStart)
		out = appendVarint(out, item.TokenEnd-item.TokenStart)
		out = appendVarint(out, item.DeclStart)
		out = appendVarint(out, item.DeclEnd-item.DeclStart)
		out = appendVarint(out, item.FuncStart)
		out = appendVarint(out, item.FuncEnd-item.FuncStart)
	}
	return out
}

func appendCoreStringBytes(out []byte, value string) []byte {
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return out
}

// Marshal is the canonical unit encoder used by both host-built and
// self-hosted frontends.
func Marshal(program Program) ([]byte, bool) {
	return MarshalCore(CoreProgramFrom(program))
}

func appendEncodedTokensCore(out []byte, tokens []Token, transient bool) []byte {
	out = appendVarint(out, len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		out = append(out, byte(tok.KindLine))
		value := tok.Start - prevStart
		if value < 128 {
			out = append(out, byte(value))
		} else {
			out = appendVarint(out, value)
		}
		if tok.Size < 128 {
			out = append(out, byte(tok.Size))
		} else {
			out = appendVarint(out, tok.Size)
		}
		line := tok.KindLine >> 8
		value = line - prevLine
		if value < 128 {
			out = append(out, byte(value))
		} else {
			out = appendVarint(out, value)
		}
		prevStart = tok.Start
		prevLine = line
		if transient && (i+1)%transientMarshalChunk == 0 {
			renvo_runtime_ArenaDiscardUnitTokens(tokens[i+1-transientMarshalChunk : i+1])
		}
	}
	if transient && len(tokens)%transientMarshalChunk != 0 {
		start := len(tokens) - len(tokens)%transientMarshalChunk
		renvo_runtime_ArenaDiscardUnitTokens(tokens[start:])
	}
	return out
}

func appendTransientTextNodeCore(out []byte, text []byte) []byte {
	out = appendNodeHeader(out, TagText, len(text))
	for start := 0; start < len(text); start += transientMarshalChunk {
		end := start + transientMarshalChunk
		if end > len(text) {
			end = len(text)
		}
		out = append(out, text[start:end]...)
		arena.DiscardBytes(text[start:end])
	}
	return out
}

func appendEncodedDeclsCore(out []byte, decls []Decl) []byte {
	out = appendVarint(out, len(decls))
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		out = appendVarint(out, decl.Kind)
		out = appendVarint(out, decl.NameStart)
		out = appendVarint(out, decl.NameEnd-decl.NameStart)
		out = appendVarint(out, decl.StartTok)
		out = appendVarint(out, decl.EndTok-decl.StartTok)
	}
	return out
}

func appendEncodedFuncsCore(out []byte, funcs []Func) []byte {
	out = appendVarint(out, len(funcs))
	for i := 0; i < len(funcs); i++ {
		fn := funcs[i]
		out = appendVarint(out, fn.NameStart)
		out = appendVarint(out, fn.NameEnd-fn.NameStart)
		out = appendVarint(out, fn.StartTok)
		out = appendVarint(out, fn.NameTok-fn.StartTok)
		out = appendVarint(out, fn.ReceiverStart)
		out = appendVarint(out, fn.ReceiverEnd-fn.ReceiverStart)
		out = appendVarint(out, fn.BodyStart)
		out = appendVarint(out, fn.BodyEnd-fn.BodyStart)
		out = appendVarint(out, fn.EndTok-fn.BodyEnd)
	}
	return out
}

func appendStringNodeCore(out []byte, tag int, payload string) []byte {
	out = appendNodeHeader(out, tag, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func patchUint32Core(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
	out[at+2] = byte(value >> 16)
	out[at+3] = byte(value >> 24)
}
