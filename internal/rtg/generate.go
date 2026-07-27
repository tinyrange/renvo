package rtg

import "renvo.dev/internal/syntax"

const GeneratorVersion = 1

type GenerateResult struct {
	Source      []byte
	Descriptor  TargetDescriptor
	Manifest    []string
	Diagnostics []Diagnostic
	Ok          bool
}

func GenerateFixedBackend(resolved ResolveResult, targetName string) GenerateResult {
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-001",
			Message:  "definition does not export target " + targetName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeader(manifest, target.Descriptor.Name)
	source = appendDescriptorSource(source, target.Descriptor)
	source = appendBackendAPI(source)
	source = appendArchitectureFacts(source, resolved.Document, target.Arch, true)
	source = appendTargetEmbeddedGo(source, resolved.Document, target, true, false)
	source = appendArchitectureBindings(source, resolved.Document, target.Arch, true, false)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

// GenerateArchitectureBackend emits the architecture-owned algorithms and
// declarative instruction bindings without a target descriptor. Built-in
// compilers use this form for checked-in architecture source shared by several
// ABI/runtime/format compositions.
func GenerateArchitectureBackend(resolved ResolveResult, archName string, packageName string) GenerateResult {
	return generateArchitectureBackend(resolved, archName, packageName, true)
}

func GenerateStatefulArchitectureBackend(resolved ResolveResult, archName string, packageName string) GenerateResult {
	return generateArchitectureBackend(resolved, archName, packageName, false)
}

func generateArchitectureBackend(resolved ResolveResult, archName string, packageName string, nativeEmitter bool) GenerateResult {
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	arch, ok := resolved.Document.Declaration(DeclArch, archName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-004",
			Message:  "definition does not declare architecture " + archName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(manifest, "arch/"+archName, packageName)
	source = appendArchitectureFacts(source, resolved.Document, arch, true)
	source = appendArchitectureEmbeddedGo(source, resolved.Document, arch, nativeEmitter)
	source = appendArchitectureBindings(source, resolved.Document, arch, true, nativeEmitter)
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

func GenerateArchitectureKernel(packageName string) GenerateResult {
	source := generateHeaderPackage(nil, "architecture-kernel", packageName)
	source = appendArchitectureBackendAPI(source)
	return GenerateResult{Source: source, Ok: true}
}

// GeneratePreparedBackend emits a closed definition as one package-main source
// file for the host-bootstrap preparation pipeline. Prepared and checked-in
// projections deliberately use identical unit mangling, so the same generated
// backend contract is exercised in both release topologies.
func GeneratePreparedBackend(resolved ResolveResult, targetName string) GenerateResult {
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-001",
			Message:  "definition does not export target " + targetName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(manifest, target.Descriptor.Name, "main")
	source = appendDescriptorSource(source, target.Descriptor)
	source = appendArchitectureBackendAPI(source)
	source = appendArchitectureFacts(source, resolved.Document, target.Arch, true)
	source = appendTargetEmbeddedGo(source, resolved.Document, target, true, true)
	source = appendArchitectureBindings(source, resolved.Document, target.Arch, true, true)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

func GenerateUniversalBackend(definitions []ResolveResult) GenerateResult {
	var diagnostics []Diagnostic
	var targets []ResolvedTarget
	var manifest []string
	for i := 0; i < len(definitions); i++ {
		if !definitions[i].Ok {
			diagnostics = append(diagnostics, definitions[i].Diagnostics...)
			continue
		}
		manifest = append(manifest, definitions[i].Document.Unit+" "+HashText(definitions[i].Document.Hash))
		for j := 0; j < len(definitions[i].Targets); j++ {
			for k := 0; k < len(targets); k++ {
				if targetNameCollides(definitions[i].Targets[j].Descriptor, targets[k].Descriptor) {
					diagnostics = append(diagnostics, resolveDiagnostic(
						definitions[i].Document,
						definitions[i].Targets[j].Declaration,
						"RTG-GENERATE-002",
						"duplicate canonical target or alias "+definitions[i].Targets[j].Descriptor.Name,
					))
				}
			}
			targets = append(targets, definitions[i].Targets[j])
		}
	}
	if len(diagnostics) != 0 {
		return GenerateResult{Diagnostics: diagnostics}
	}
	sortStrings(manifest)
	sortResolvedTargets(targets)
	source := generateHeader(manifest, "universal")
	source = append(source, "type RTGTargetDescriptor struct { Name string; OS string; ISA string; WordBits int; PointerBits int; Endian string; ABI string; Runtime string; Executable string; Object string; Definition string; Version int }\n"...)
	source = append(source, "func RTGLookupTarget(name string) (RTGTargetDescriptor, bool) {\n"...)
	source = append(source, "switch name {\n"...)
	for i := 0; i < len(targets); i++ {
		source = append(source, "case "...)
		source = appendQuoted(source, targets[i].Descriptor.Name)
		for j := 0; j < len(targets[i].Descriptor.Aliases); j++ {
			source = append(source, ", "...)
			source = appendQuoted(source, targets[i].Descriptor.Aliases[j])
		}
		source = append(source, ":\nreturn "...)
		source = appendDescriptorLiteral(source, targets[i].Descriptor)
		source = append(source, ", true\n"...)
	}
	source = append(source, "}\nreturn RTGTargetDescriptor{}, false\n}\n"...)
	source = appendBackendAPI(source)
	for i := 0; i < len(definitions); i++ {
		for j := 0; j < len(definitions[i].Document.Declarations); j++ {
			declaration := definitions[i].Document.Declarations[j]
			if declaration.Kind == DeclArch {
				source = appendArchitectureFacts(source, definitions[i].Document, declaration, true)
			}
		}
		source = appendMangledEmbeddedGo(source, definitions[i].Document, false)
		for j := 0; j < len(definitions[i].Document.Declarations); j++ {
			declaration := definitions[i].Document.Declarations[j]
			if declaration.Kind == DeclArch {
				source = appendArchitectureBindings(source, definitions[i].Document, declaration, true, false)
			}
		}
	}
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

// appendArchitectureBackendAPI specializes the emitter onto the assembler
// already owned by a checked-in built-in backend. Embedded algorithms retain
// their authored *RTGEmitter signatures, while Go sees the receiver as the
// existing *renvoAsm and introduces no temporary emitter state in hot paths.
func appendArchitectureBackendAPI(source []byte) []byte {
	source = appendBackendAPI(source)
	return append(source, `
type renvoRTGAddress struct {
	Target       int
	TargetValid  bool
	Addend       int
	Base         RTGRegister
	Index        RTGRegister
	Displacement int
	Scale        int
}

func (out *renvoAsm) Byte(value byte) {
	renvoAsmEmit8(out, int(value))
}

func (out *renvoAsm) Uint32(value uint32) {
	renvoAsmEmit32(out, int(value))
}

func (out *renvoAsm) Uint64(value uint64) {
	renvoAsmEmit32(out, int(value))
	renvoAsmEmit32(out, int(value >> 32))
}

func (out *renvoAsm) PatchUint32(at int, value int) {
	renvoPut32At(out.code, at, value)
}

func (out *renvoAsm) NewLabel() int {
	return renvoAsmNewLabel(out)
}

func (out *renvoAsm) Mark(label int) {
	if label >= 0 {
		renvoAsmMarkLabel(out, label)
	}
}

func (out *renvoAsm) Rel32(label int) {
	out.Rel32Addend(label, 0)
}

func (out *renvoAsm) Rel32Addend(label int, addend int) {
	at := len(out.code)
	renvoAsmEmit32(out, addend)
	if label >= 0 {
		renvoAsmAddReloc(out, at, label)
	}
}

func (out *renvoAsm) Reloc(label int) {
	if label >= 0 && len(out.code) >= 4 {
		renvoAsmAddReloc(out, len(out.code)-4, label)
	}
}

func (out *renvoAsm) Patch() {
	renvoAsmPatch(out)
}

func (out *renvoAsm) Code() []byte {
	return out.code
}

func (out *renvoAsm) SetCode(value []byte) {
	out.code = value
}

func (out *renvoAsm) Data() []byte {
	return out.data
}

func (out *renvoAsm) BSSSize() int {
	return out.bssSize
}

func (out *renvoAsm) RelocationCount() int {
	return len(out.relocs) / 2
}

func (out *renvoAsm) RelocationAt(index int) (int, int) {
	at := index * 2
	return int(out.relocs[at]), int(out.relocs[at+1])
}

func (out *renvoAsm) RelocationOffset(index int) int {
	return int(out.relocs[index*2])
}

func (out *renvoAsm) RelocationLabel(index int) int {
	return int(out.relocs[index*2+1])
}

func (out *renvoAsm) RelocationWordCount() int {
	return len(out.relocs)
}

func (out *renvoAsm) RelocationWord(index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.relocs, index))
}

func (out *renvoAsm) AbsoluteRelocationCount() int {
	return len(out.absRelocs) / 3
}

func (out *renvoAsm) AbsoluteRelocationAt(index int) (int, int, int) {
	at := index * 3
	return int(out.absRelocs[at]), int(out.absRelocs[at+1]), int(out.absRelocs[at+2])
}

func (out *renvoAsm) AbsoluteRelocationOffset(index int) int {
	return int(out.absRelocs[index*3])
}

func (out *renvoAsm) AbsoluteRelocationAddend(index int) int {
	return int(out.absRelocs[index*3+1])
}

func (out *renvoAsm) AbsoluteRelocationKind(index int) int {
	return int(out.absRelocs[index*3+2])
}

func (out *renvoAsm) AbsoluteRelocationWordCount() int {
	return len(out.absRelocs)
}

func (out *renvoAsm) AbsoluteRelocationWord(index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.absRelocs, index))
}

func (out *renvoAsm) LabelPosition(label int) int {
	return renvoAsmLabelPosition(out, label)
}

func (out *renvoAsm) SymbolCount() int {
	return len(out.symbols)
}

func (out *renvoAsm) SymbolPC(index int) int {
	return renvoAsmLabelPosition(out, out.symbols[index].label)
}

func (out *renvoAsm) SymbolNameEquals(index int, name string) bool {
	symbol := &out.symbols[index]
	if symbol.nameEnd-symbol.nameStart != len(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if out.symbolName[symbol.nameStart+i] != name[i] {
			return false
		}
	}
	return true
}

func (out *renvoAsm) SymbolNameLength(index int) int {
	symbol := &out.symbols[index]
	return symbol.nameEnd - symbol.nameStart
}

func (out *renvoAsm) SymbolNameByte(index int, offset int) byte {
	symbol := &out.symbols[index]
	return out.symbolName[symbol.nameStart+offset]
}

func (out *renvoAsm) Symbol(index int) *RTGSymbol {
	symbol := &out.symbols[index]
	return &RTGSymbol{nameStart: symbol.nameStart, nameEnd: symbol.nameEnd, label: symbol.label}
}

func (out *renvoAsm) SymbolNameByteAt(index int) byte {
	return out.symbolName[index]
}

func renvoRTGRelocationAt(out *renvoAsm, index int) (int, int) {
	at := index * 2
	return int(out.relocs[at]), int(out.relocs[at+1])
}

func renvoRTGAbsoluteRelocationAt(out *renvoAsm, index int) (int, int, int) {
	at := index * 3
	return int(out.absRelocs[at]), int(out.absRelocs[at+1]), int(out.absRelocs[at+2])
}

func renvoRTGSymbolPC(out *renvoAsm, index int) int {
	return renvoAsmLabelPosition(out, out.symbols[index].label)
}

func renvoRTGSymbolNameEquals(out *renvoAsm, index int, name string) bool {
	symbol := &out.symbols[index]
	if symbol.nameEnd-symbol.nameStart != len(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if out.symbolName[symbol.nameStart+i] != name[i] {
			return false
		}
	}
	return true
}

func renvoRTGAddressValid(address renvoRTGAddress) bool {
	return address.TargetValid
}

func renvoRTGAddressRel32Addend(out *renvoAsm, address renvoRTGAddress) {
	out.Rel32Addend(address.Target, address.Addend)
}

func renvoRTGByte(out *renvoAsm, value byte) {
	renvoAsmEmit8(out, int(value))
}

func renvoRTGUint32(out *renvoAsm, value int) {
	renvoAsmEmit32(out, value)
}

func renvoRTGUint64(out *renvoAsm, value uint64) {
	renvoAsmEmit32(out, int(value))
	renvoAsmEmit32(out, int(value >> 32))
}

func renvoRTGPatchUint32(out *renvoAsm, at int, value int) {
	renvoPut32At(out.code, at, value)
}

func renvoRTGNewLabel(out *renvoAsm) int {
	return renvoAsmNewLabel(out)
}

func renvoRTGMark(out *renvoAsm, label int) {
	if label >= 0 {
		renvoAsmMarkLabel(out, label)
	}
}

func renvoRTGReloc(out *renvoAsm, label int) {
	if label >= 0 && len(out.code) >= 4 {
		renvoAsmAddReloc(out, len(out.code)-4, label)
	}
}
`...)
}

// appendBackendAPI emits the small target-neutral surface used by generated
// encoding algorithms. Keeping it in generated source makes a checked-in
// backend an ordinary, self-contained Go source file without increasing the
// size of legacy fixed-target compilers which do not use RTG definitions.
func appendBackendAPI(source []byte) []byte {
	return append(source, `
type RTGRegister struct {
	Code int
	Valid bool
}

type RTGLabel struct {
	Code int
	Valid bool
}

type RTGAddress struct {
	Target       RTGLabel
	Addend       int
	Base         RTGRegister
	Index        RTGRegister
	Displacement int
	Scale        int
}

type RTGCondition struct {
	Code       int
	SetOpcode  byte
	JumpOpcode byte
}
type RTGSymbol struct {
	nameStart int
	nameEnd   int
	label     int
}
type RTGShiftDirection int

const (
	RTGShiftLeft RTGShiftDirection = 1
	RTGShiftRight RTGShiftDirection = 2
)

const RTGRelocationAbsoluteData = 0
const RTGRelocationAbsoluteBSS = 1

const RTGScalarByte = 3
const RTGScalarInt8 = 7
const RTGScalarInt16 = 8
const RTGScalarInt32 = 9
const RTGScalarUint16 = 16
const RTGScalarUint32 = 17

var RTGNoRegister = RTGRegister{}

type RTGEmitter struct {
	asm    *renvoAsm
	relocs []int
}

func renvoRTGEmitter(asm *renvoAsm) RTGEmitter {
	return RTGEmitter{asm: asm}
}

func renvoRTGLabel(code int) RTGLabel {
	return RTGLabel{Code: code, Valid: code >= 0}
}

func renvoRTGLabelCode(code int) int {
	return code
}

func RTGLabelFromCode(code int) RTGLabel {
	return renvoRTGLabel(code)
}

func (out *RTGEmitter) Len() int {
	return len(out.asm.code)
}

func (out *RTGEmitter) Byte(value byte) {
	out.asm.code = append(out.asm.code, value)
}

func (out *RTGEmitter) Int8(value int) {
	out.Byte(byte(value))
}

func (out *RTGEmitter) Uint16(value int) {
	out.asm.code = append(out.asm.code, byte(value), byte(value >> 8))
}

func (out *RTGEmitter) Uint24(value int) {
	out.Uint16(value)
	out.Byte(byte(value >> 16))
}

func RTGByte(out *RTGEmitter, value byte) {
	out.Byte(value)
}

func (out *RTGEmitter) Uint32(value uint32) {
	out.asm.code = append(out.asm.code, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24))
}

func (out *RTGEmitter) Int32(value int) {
	out.Uint32(uint32(value))
}

func RTGUint32(out *RTGEmitter, value int) {
	out.Uint32(uint32(value))
}

func (out *RTGEmitter) Uint64(value uint64) {
	out.Uint32(uint32(value))
	out.Uint32(uint32(value >> 32))
}

func (out *RTGEmitter) Int64(value int) {
	out.Uint64(uint64(value))
}

func (out *RTGEmitter) Bytes2(v0 int, v1 int) {
	out.Byte(byte(v0))
	out.Byte(byte(v1))
}

func (out *RTGEmitter) Bytes3(v0 int, v1 int, v2 int) {
	out.Bytes2(v0, v1)
	out.Byte(byte(v2))
}

func (out *RTGEmitter) Bytes4(v0 int, v1 int, v2 int, v3 int) {
	out.Bytes3(v0, v1, v2)
	out.Byte(byte(v3))
}

func (out *RTGEmitter) Text(value string) {
	for i := 0; i < len(value); i++ {
		out.Byte(value[i])
	}
}

func RTGUint64(out *RTGEmitter, value uint64) {
	out.Uint64(value)
}

func (out *RTGEmitter) PatchUint32(at int, value int) {
	renvoPut32At(out.asm.code, at, value)
}

func (out *RTGEmitter) AbsoluteReloc(at int, offset int, kind int) {
	renvoAsmAddAbsReloc(out.asm, at, offset, kind)
}

func (out *RTGEmitter) RelocAt(at int, label int) {
	if label >= 0 {
		renvoAsmAddReloc(out.asm, at, label)
	}
}

func (out *RTGEmitter) PrimaryLoad() int {
	return out.asm.lastPrimaryLoad
}

func (out *RTGEmitter) SetPrimaryLoad(value int) {
	out.asm.lastPrimaryLoad = value
}

func (out *RTGEmitter) ByteAt(at int) byte {
	return out.asm.code[at]
}

func (out *RTGEmitter) SetByteAt(at int, value byte) {
	out.asm.code[at] = value
}

func (out *RTGEmitter) AddByteAt(at int, value byte) {
	out.asm.code[at] += value
}

func (out *RTGEmitter) AppendByte(value int) {
	out.asm.code = append(out.asm.code, byte(value))
}

func (out *RTGEmitter) Truncate(size int) {
	out.asm.code = out.asm.code[:size]
}

func (out *RTGEmitter) Code() []byte {
	return out.asm.code
}

func (out *RTGEmitter) SetCode(value []byte) {
	out.asm.code = value
}

func (out *RTGEmitter) Data() []byte {
	return out.asm.data
}

func (out *RTGEmitter) BSSSize() int {
	return out.asm.bssSize
}

func (out *RTGEmitter) RelocationCount() int {
	return len(out.asm.relocs) / 2
}

func (out *RTGEmitter) RelocationAt(index int) (int, int) {
	at := index * 2
	return int(out.asm.relocs[at]), int(out.asm.relocs[at+1])
}

func (out *RTGEmitter) RelocationOffset(index int) int {
	return int(out.asm.relocs[index*2])
}

func (out *RTGEmitter) RelocationLabel(index int) int {
	return int(out.asm.relocs[index*2+1])
}

func (out *RTGEmitter) RelocationWordCount() int {
	return len(out.asm.relocs)
}

func (out *RTGEmitter) RelocationWord(index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.asm.relocs, index))
}

func (out *RTGEmitter) AbsoluteRelocationCount() int {
	return len(out.asm.absRelocs) / 3
}

func (out *RTGEmitter) AbsoluteRelocationAt(index int) (int, int, int) {
	at := index * 3
	return int(out.asm.absRelocs[at]), int(out.asm.absRelocs[at+1]), int(out.asm.absRelocs[at+2])
}

func (out *RTGEmitter) AbsoluteRelocationOffset(index int) int {
	return int(out.asm.absRelocs[index*3])
}

func (out *RTGEmitter) AbsoluteRelocationAddend(index int) int {
	return int(out.asm.absRelocs[index*3+1])
}

func (out *RTGEmitter) AbsoluteRelocationKind(index int) int {
	return int(out.asm.absRelocs[index*3+2])
}

func (out *RTGEmitter) AbsoluteRelocationWordCount() int {
	return len(out.asm.absRelocs)
}

func (out *RTGEmitter) AbsoluteRelocationWord(index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.asm.absRelocs, index))
}

func (out *RTGEmitter) LabelPosition(label int) int {
	return renvoAsmLabelPosition(out.asm, label)
}

func (out *RTGEmitter) SymbolCount() int {
	return len(out.asm.symbols)
}

func (out *RTGEmitter) SymbolPC(index int) int {
	return renvoAsmLabelPosition(out.asm, out.asm.symbols[index].label)
}

func (out *RTGEmitter) SymbolNameEquals(index int, name string) bool {
	symbol := &out.asm.symbols[index]
	if symbol.nameEnd-symbol.nameStart != len(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if out.asm.symbolName[symbol.nameStart+i] != name[i] {
			return false
		}
	}
	return true
}

func (out *RTGEmitter) SymbolNameLength(index int) int {
	symbol := &out.asm.symbols[index]
	return symbol.nameEnd - symbol.nameStart
}

func (out *RTGEmitter) SymbolNameByte(index int, offset int) byte {
	symbol := &out.asm.symbols[index]
	return out.asm.symbolName[symbol.nameStart+offset]
}

func (out *RTGEmitter) Symbol(index int) *RTGSymbol {
	symbol := &out.asm.symbols[index]
	return &RTGSymbol{nameStart: symbol.nameStart, nameEnd: symbol.nameEnd, label: symbol.label}
}

func (out *RTGEmitter) SymbolNameByteAt(index int) byte {
	return out.asm.symbolName[index]
}

func RTGPatchUint32(out *RTGEmitter, at int, value int) {
	out.PatchUint32(at, value)
}

func (out *RTGEmitter) NewLabel() RTGLabel {
	return RTGLabel{Code: renvoAsmNewLabel(out.asm), Valid: true}
}

func (out *RTGEmitter) Mark(label RTGLabel) {
	if label.Valid {
		renvoAsmMarkLabel(out.asm, label.Code)
	}
}

func (out *RTGEmitter) Rel32(label RTGLabel) {
	out.Rel32Addend(label, 0)
}

func (out *RTGEmitter) Rel32Addend(label RTGLabel, addend int) {
	at := len(out.asm.code)
	out.Uint32(uint32(0))
	if label.Valid {
		out.relocs = append(out.relocs, at, label.Code, addend)
	}
}

func (out *RTGEmitter) Reloc(label RTGLabel) {
	if label.Valid && len(out.asm.code) >= 4 {
		renvoAsmAddReloc(out.asm, len(out.asm.code)-4, label.Code)
	}
}

func RTGReloc(out *RTGEmitter, label RTGLabel) {
	out.Reloc(label)
}

func RTGAddressValid(address RTGAddress) bool {
	return address.Target.Valid
}

func RTGAddressRel32Addend(out *RTGEmitter, address RTGAddress) {
	out.Rel32Addend(address.Target, address.Addend)
}

func RTGInt8Fits(value int) bool {
	return value >= -128 && value <= 127
}

func RTGPopPrimary(out *RTGEmitter) {
	out.Byte(0x58)
}

func RTGArenaMark() int {
	return renvo_runtime_ArenaMark()
}

func RTGArenaReset(mark int) {
	renvo_runtime_ArenaReset(mark)
}

func RTGByteAt(in []byte, at int) byte {
	return in[at]
}

func RTGGet32At(in []byte, at int) int {
	return int(in[at]) | int(in[at+1])<<8 | int(in[at+2])<<16 | int(in[at+3])<<24
}

func RTGPut32At(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
	out[at+2] = byte(value >> 16)
	out[at+3] = byte(value >> 24)
}

func RTGAppend32(out []byte, value int) []byte {
	out = append(out, byte(value))
	out = append(out, byte(value>>8))
	out = append(out, byte(value>>16))
	out = append(out, byte(value>>24))
	return out
}

func RTGAlignValue(value int, alignment int) int {
	return (value + alignment - 1) & -alignment
}

func RTGAlign8(value int) int {
	return (value + 7) & -8
}

func RTGMakeIntScratch(capacity int) []int {
	return make([]int, 0, capacity)
}

func RTGMakeByteBuffer(length int) []byte {
	return make([]byte, length)
}

func (out *RTGEmitter) Patch() {
	for i := 0; i+2 < len(out.relocs); i += 3 {
		at := out.relocs[i]
		label := out.relocs[i+1]
		addend := out.relocs[i+2]
		target := renvoAsmLabelPosition(out.asm, label)
		if at < 0 || at+4 > len(out.asm.code) || target < 0 {
			continue
		}
		renvoPut32At(out.asm.code, at, target+addend-(at+4))
	}
}

func RTGSignedFits(value int64, bits int) bool {
	if bits <= 0 {
		return false
	}
	if bits >= 63 {
		return true
	}
	limit := int64(1) << (bits - 1)
	return value >= -limit && value < limit
}

func RTGUnsignedFits(value uint64, bits int) bool {
	if bits <= 0 {
		return false
	}
	if bits >= 63 {
		return true
	}
	return value < uint64(1) << bits
}

func RTGLog2(value int) int {
	result := 0
	for value > 1 {
		value >>= 1
		result++
	}
	return result
}
`...)
}

func generateHeader(manifest []string, target string) []byte {
	return generateHeaderPackage(manifest, target, "backend")
}

func generateHeaderPackage(manifest []string, target string, packageName string) []byte {
	var source []byte
	source = append(source, "// Code generated by Renvo RTG; DO NOT EDIT.\n// generator: "...)
	source = appendDecimalFrame(source, GeneratorVersion)
	source = append(source, "\n// target: "...)
	source = append(source, target...)
	source = append(source, '\n')
	for i := 0; i < len(manifest); i++ {
		source = append(source, "// unit: "...)
		source = append(source, manifest[i]...)
		source = append(source, '\n')
	}
	source = append(source, "package "...)
	source = append(source, packageName...)
	source = append(source, "\n\n"...)
	return source
}

func trimBlockNewlines(source []byte) []byte {
	start := 0
	for start < len(source) && (source[start] == '\n' || source[start] == '\r') {
		start++
	}
	end := len(source)
	for end > start && (source[end-1] == '\n' || source[end-1] == '\r') {
		end--
	}
	return source[start:end]
}

func appendDescriptorSource(source []byte, descriptor TargetDescriptor) []byte {
	source = append(source, "type RTGTargetDescriptor struct { Name string; OS string; ISA string; WordBits int; PointerBits int; Endian string; ABI string; Runtime string; Executable string; Object string; Definition string; Version int }\n"...)
	source = append(source, "var RTGTarget = "...)
	source = appendDescriptorLiteral(source, descriptor)
	source = append(source, '\n')
	return source
}

func appendDescriptorLiteral(source []byte, descriptor TargetDescriptor) []byte {
	source = append(source, "RTGTargetDescriptor{"...)
	values := []string{
		descriptor.Name, descriptor.OS, descriptor.ISA, descriptor.Endian,
		descriptor.ABI, descriptor.Runtime, descriptor.Executable, descriptor.Object,
		HashText(descriptor.Definition),
	}
	names := []string{"Name:", "OS:", "ISA:", "Endian:", "ABI:", "Runtime:", "Executable:", "Object:", "Definition:"}
	for i := 0; i < len(values); i++ {
		source = append(source, names[i]...)
		source = appendQuoted(source, values[i])
		source = append(source, ',')
	}
	source = append(source, "WordBits:"...)
	source = appendDecimalFrame(source, descriptor.WordBits)
	source = append(source, ",PointerBits:"...)
	source = appendDecimalFrame(source, descriptor.PointerBits)
	source = append(source, ",Version:"...)
	source = appendDecimalFrame(source, descriptor.Version)
	source = append(source, '}')
	return source
}

func appendEmbeddedGo(source []byte, document Document) []byte {
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		source = append(source, '\n')
		source = appendRewrittenGo(source, dedentGoSource(declaration.GoSource), names, prefix)
		source = append(source, '\n')
	}
	return source
}

func appendMangledEmbeddedGo(source []byte, document Document, nativeEmitter bool) []byte {
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		source = append(source, '\n')
		source = appendRewrittenGoMode(source, dedentGoSource(declaration.GoSource), names, prefix, nativeEmitter, &document)
		source = append(source, '\n')
	}
	return source
}

// appendArchitectureEmbeddedGo emits only the embedded declarations reachable
// from an architecture's declarative hooks. A closed definition can also own
// ABI, runtime, and format algorithms; those must not leak into a checked-in
// architecture-only projection or prevent fixed-target pruning.
func appendArchitectureEmbeddedGo(source []byte, document Document, arch Declaration, nativeEmitter bool) []byte {
	roots := architectureGoRoots(arch)
	return appendReachableEmbeddedGo(source, document, roots, nativeEmitter, architectureExports(arch))
}

func appendTargetEmbeddedGo(source []byte, document Document, target ResolvedTarget, mangle bool, nativeEmitter bool) []byte {
	roots := architectureGoRoots(target.Arch)
	roots = appendDeclarationGoRoots(roots, target.ABI)
	roots = appendDeclarationGoRoots(roots, target.Runtime)
	roots = appendDeclarationGoRoots(roots, target.Executable)
	roots = appendDeclarationGoRoots(roots, target.Object)
	roots = appendDeclarationGoRoots(roots, target.Declaration)
	if len(roots) == 0 {
		// Definitions predating declarative hook references treated every
		// embedded declaration as an exported target algorithm.
		roots = embeddedGoNames(document)
	}
	sortStrings(roots)
	if !mangle {
		body := reachableEmbeddedGo(document, roots)
		if len(body) != 0 {
			source = append(source, '\n')
			source = append(source, body...)
			source = append(source, '\n')
		}
		return source
	}
	return appendReachableEmbeddedGo(source, document, roots, nativeEmitter, architectureExports(target.Arch))
}

func appendReachableEmbeddedGo(source []byte, document Document, roots []string, nativeEmitter bool, exports []embeddedExport) []byte {
	body := reachableEmbeddedGo(document, roots)
	if len(body) == 0 {
		return source
	}
	body = trimBlockNewlines(body)
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	source = append(source, '\n')
	source = appendRewrittenGoModeExports(source, body, names, prefix, nativeEmitter, &document, exports)
	source = append(source, '\n')
	return source
}

func architectureGoRoots(arch Declaration) []string {
	var roots []string
	roots = appendDeclarationGoRoots(roots, arch)
	sortStrings(roots)
	return roots
}

type embeddedExport struct {
	External string
	Local    string
}

func architectureExports(arch Declaration) []embeddedExport {
	block, ok := declarationBlock(arch, "exports")
	if !ok {
		return nil
	}
	var exports []embeddedExport
	for i := 0; i < len(block.Children); i++ {
		left, right, assignment := statementAssignment(block.Children[i])
		if !assignment || len(left) != 1 || len(right) != 2 || right[0] != "go" {
			continue
		}
		exports = append(exports, embeddedExport{External: left[0], Local: right[1]})
	}
	return exports
}

func embeddedExternalName(exports []embeddedExport, local string) (string, bool) {
	for i := 0; i < len(exports); i++ {
		if exports[i].Local == local {
			return exports[i].External, true
		}
	}
	return "", false
}

func appendDeclarationGoRoots(roots []string, declaration Declaration) []string {
	var visit func([]Statement)
	visit = func(statements []Statement) {
		for i := 0; i < len(statements); i++ {
			tokens := statements[i].Tokens
			for j := 0; j+1 < len(tokens); j++ {
				if tokens[j] == "go" && stringIndex(roots, tokens[j+1]) < 0 {
					roots = append(roots, tokens[j+1])
				}
			}
			visit(statements[i].Children)
		}
	}
	visit(declaration.Statements)
	return roots
}

type embeddedGoPart struct {
	name   string
	source []byte
	refs   []string
	isFunc bool
}

func reachableEmbeddedGo(document Document, roots []string) []byte {
	var combined []byte
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		combined = append(combined, '\n')
		combined = append(combined, dedentGoSource(declaration.GoSource)...)
		combined = append(combined, '\n')
	}
	if len(combined) == 0 {
		return nil
	}
	wrapped := make([]byte, 0, len(combined)+16)
	wrapped = append(wrapped, "package backend\n"...)
	wrapped = append(wrapped, combined...)
	file := syntax.ParseFile(wrapped)
	if !file.Ok {
		return nil
	}
	var parts []embeddedGoPart
	lastDeclarationStart := -1
	lastDeclarationEnd := -1
	for i := 0; i < len(file.Decls); i++ {
		declaration := file.Decls[i]
		if declaration.StartTok == lastDeclarationStart && declaration.EndTok == lastDeclarationEnd {
			continue
		}
		lastDeclarationStart = declaration.StartTok
		lastDeclarationEnd = declaration.EndTok
		var declarationSource []byte
		if declaration.Kind == syntax.TokenType {
			declarationSource = append(declarationSource, "type "...)
		} else if declaration.Kind == syntax.TokenConst {
			declarationSource = append(declarationSource, "const "...)
		} else if declaration.Kind == syntax.TokenVar {
			declarationSource = append(declarationSource, "var "...)
		}
		declarationSource = append(declarationSource,
			embeddedSourceRange(wrapped, file.Tokens, declaration.StartTok, declaration.EndTok)...)
		parts = append(parts, embeddedGoPart{
			name:   string(syntax.TokenText(wrapped, file.Tokens[declaration.NameTok])),
			source: declarationSource,
		})
	}
	for i := 0; i < len(file.Funcs); i++ {
		function := file.Funcs[i]
		parts = append(parts, embeddedGoPart{
			name:   string(syntax.TokenText(wrapped, file.Tokens[function.NameTok])),
			source: embeddedSourceRange(wrapped, file.Tokens, function.StartTok, function.EndTok),
			isFunc: true,
		})
	}
	names := make([]string, len(parts))
	for i := 0; i < len(parts); i++ {
		names[i] = parts[i].name
	}
	for i := 0; i < len(parts); i++ {
		tokens, diagnostics := scan(parts[i].source, "")
		if len(diagnostics) != 0 {
			continue
		}
		for j := 0; j < len(tokens); j++ {
			if tokens[j].Kind != TokenIdent {
				continue
			}
			name := tokenText(parts[i].source, tokens[j])
			if name != parts[i].name && stringIndex(names, name) >= 0 &&
				stringIndex(parts[i].refs, name) < 0 {
				parts[i].refs = append(parts[i].refs, name)
			}
		}
	}
	reachable := make([]string, len(roots))
	copy(reachable, roots)
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(parts); i++ {
			if stringIndex(reachable, parts[i].name) < 0 {
				continue
			}
			for j := 0; j < len(parts[i].refs); j++ {
				if stringIndex(reachable, parts[i].refs[j]) < 0 {
					reachable = append(reachable, parts[i].refs[j])
					changed = true
				}
			}
		}
	}
	var out []byte
	for i := 0; i < len(parts); i++ {
		if !parts[i].isFunc || stringIndex(reachable, parts[i].name) >= 0 {
			out = append(out, parts[i].source...)
			out = append(out, '\n')
		}
	}
	return out
}

func embeddedSourceRange(source []byte, tokens []syntax.Token, start int, end int) []byte {
	if start < 0 || end <= start || end > len(tokens) {
		return nil
	}
	sourceStart := tokens[start].Start
	sourceEnd := tokens[end-1].End
	if sourceStart < 0 || sourceEnd < sourceStart || sourceEnd > len(source) {
		return nil
	}
	out := make([]byte, sourceEnd-sourceStart)
	copy(out, source[sourceStart:sourceEnd])
	return out
}

func dedentGoSource(source []byte) []byte {
	source = trimBlockNewlines(source)
	indent := -1
	lineStart := 0
	for lineStart < len(source) {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		at := lineStart
		for at < lineEnd && (source[at] == ' ' || source[at] == '\t') {
			at++
		}
		if at < lineEnd && (indent < 0 || at-lineStart < indent) {
			indent = at - lineStart
		}
		lineStart = lineEnd + 1
	}
	if indent <= 0 {
		return source
	}
	out := make([]byte, 0, len(source))
	lineStart = 0
	for lineStart < len(source) {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		remove := indent
		for remove > 0 && lineStart < lineEnd &&
			(source[lineStart] == ' ' || source[lineStart] == '\t') {
			lineStart++
			remove--
		}
		out = append(out, source[lineStart:lineEnd]...)
		if lineEnd < len(source) {
			out = append(out, '\n')
		}
		lineStart = lineEnd + 1
	}
	return out
}

func embeddedGoNames(document Document) []string {
	var names []string
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		wrapped := append([]byte("package backend\n"), declaration.GoSource...)
		file := syntax.ParseFile(wrapped)
		if !file.Ok {
			continue
		}
		for j := 0; j < len(file.Decls); j++ {
			names = append(names, string(syntax.TokenText(wrapped, file.Tokens[file.Decls[j].NameTok])))
		}
		for j := 0; j < len(file.Funcs); j++ {
			names = append(names, string(syntax.TokenText(wrapped, file.Tokens[file.Funcs[j].NameTok])))
		}
	}
	sortStrings(names)
	return names
}

func appendRewrittenGo(out []byte, source []byte, names []string, prefix string) []byte {
	return appendRewrittenGoMode(out, source, names, prefix, false, nil)
}

func appendRewrittenGoMode(out []byte, source []byte, names []string, prefix string, nativeEmitter bool, document *Document) []byte {
	return appendRewrittenGoModeExports(out, source, names, prefix, nativeEmitter, document, nil)
}

func appendRewrittenGoModeExports(out []byte, source []byte, names []string, prefix string, nativeEmitter bool, document *Document, exports []embeddedExport) []byte {
	tokens, diagnostics := scan(source, "")
	if len(diagnostics) != 0 {
		return out
	}
	last := 0
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.Kind == TokenEOF {
			break
		}
		out = append(out, source[last:token.Start]...)
		text := tokenText(source, token)
		if nativeEmitter && token.Kind == TokenIdent && i+3 < len(tokens) &&
			tokenText(source, tokens[i+1]) == "." && tokenText(source, tokens[i+3]) == "(" {
			method := tokenText(source, tokens[i+2])
			if replacement, end, found := nativeEmitterStateMethod(source, tokens, i, text, method,
				names, prefix, document, exports); found {
				out = append(out, replacement...)
				last = tokens[end].End
				i = end
				continue
			}
			if method == "Len" && i+4 < len(tokens) && tokenText(source, tokens[i+4]) == ")" {
				out = append(out, "len("...)
				out = append(out, text...)
				out = append(out, ".code)"...)
				last = tokens[i+4].End
				i += 4
				continue
			}
			replacement := ""
			if method == "PatchUint32" {
				replacement = "renvoPut32At"
			} else if method == "AbsoluteReloc" {
				replacement = "renvoAsmAddAbsReloc"
			} else if method == "RelocAt" {
				replacement = "renvoAsmAddReloc"
			} else if method == "Byte" {
				replacement = "renvoAsmEmit8"
			} else if method == "Int8" {
				replacement = "renvoAsmEmit8"
			} else if method == "Uint16" {
				replacement = "renvoAsmEmit16"
			} else if method == "Uint24" {
				replacement = "renvoAsmEmit24"
			} else if method == "Uint32" {
				replacement = "renvoAsmEmit32"
			} else if method == "Int32" {
				replacement = "renvoAsmEmit32"
			} else if method == "Int64" {
				replacement = "renvoAsmEmit64"
			} else if method == "Bytes2" {
				replacement = "renvoAsmEmit2"
			} else if method == "Bytes3" {
				replacement = "renvoAsmEmit3"
			} else if method == "Bytes4" {
				replacement = "renvoAsmEmit4"
			} else if method == "Text" {
				replacement = "renvoAsmEmitText"
			}
			if replacement != "" {
				out = append(out, replacement...)
				out = append(out, '(')
				out = append(out, text...)
				if method == "PatchUint32" {
					out = append(out, ".code"...)
				}
				out = append(out, ',')
				last = tokens[i+3].End
				i += 3
				continue
			}
		}
		if nativeEmitter && document != nil && token.Kind == TokenIdent && i+2 < len(tokens) &&
			tokenText(source, tokens[i+1]) == "." && tokenText(source, tokens[i+2]) == "Code" {
			if code, found := generatedArchitectureCode(*document, text); found {
				out = appendDecimalFrame(out, code)
				last = tokens[i+2].End
				i += 2
				continue
			}
		}
		if nativeEmitter && token.Kind == TokenIdent && nativeBuiltinLiteral(text) != "" {
			out = append(out, nativeBuiltinLiteral(text)...)
		} else if nativeEmitter && token.Kind == TokenIdent && text == "RTGEmitter" {
			out = append(out, "renvoAsm"...)
		} else if nativeEmitter && token.Kind == TokenIdent && text == "RTGLabel" {
			out = append(out, "int"...)
		} else if nativeEmitter && token.Kind == TokenIdent && text == "RTGAddress" {
			out = append(out, "renvoRTGAddress"...)
		} else if nativeEmitter && token.Kind == TokenIdent && text == "RTGSymbol" {
			out = append(out, "renvoAsmSymbol"...)
		} else if nativeEmitter && token.Kind == TokenIdent && nativeEmitterFunction(text) != "" {
			out = append(out, nativeEmitterFunction(text)...)
		} else if external, found := embeddedExternalName(exports, text); found && token.Kind == TokenIdent {
			out = append(out, external...)
		} else if document != nil && token.Kind == TokenIdent {
			replacement, found := generatedArchitectureOutput(*document, text)
			if found {
				out = append(out, replacement...)
			} else if stringIndex(names, text) >= 0 {
				out = append(out, prefix...)
				out = append(out, exportedName(text)...)
			} else {
				out = append(out, source[token.Start:token.End]...)
			}
		} else if token.Kind == TokenIdent && stringIndex(names, text) >= 0 {
			out = append(out, prefix...)
			out = append(out, exportedName(text)...)
		} else {
			out = append(out, source[token.Start:token.End]...)
		}
		last = token.End
	}
	out = append(out, source[last:]...)
	return out
}

func nativeEmitterStateMethod(source []byte, tokens []Token, start int, receiver string, method string,
	names []string, prefix string, document *Document, exports []embeddedExport) ([]byte, int, bool) {
	if method != "PrimaryLoad" && method != "SetPrimaryLoad" && method != "ByteAt" &&
		method != "SetByteAt" && method != "AddByteAt" && method != "AppendByte" &&
		method != "Truncate" && method != "Code" && method != "SetCode" &&
		method != "Data" && method != "BSSSize" && method != "RelocationCount" &&
		method != "RelocationAt" && method != "RelocationOffset" &&
		method != "RelocationLabel" && method != "RelocationWordCount" &&
		method != "RelocationWord" && method != "AbsoluteRelocationCount" &&
		method != "AbsoluteRelocationAt" && method != "AbsoluteRelocationOffset" &&
		method != "AbsoluteRelocationAddend" && method != "AbsoluteRelocationKind" &&
		method != "AbsoluteRelocationWordCount" && method != "AbsoluteRelocationWord" &&
		method != "LabelPosition" &&
		method != "SymbolCount" && method != "SymbolPC" && method != "SymbolNameEquals" &&
		method != "SymbolNameLength" && method != "SymbolNameByte" &&
		method != "Symbol" && method != "SymbolNameByteAt" {
		return nil, start, false
	}
	end := start + 3
	depth := 0
	for end < len(tokens) {
		text := tokenText(source, tokens[end])
		if text == "(" {
			depth++
		} else if text == ")" {
			depth--
			if depth == 0 {
				break
			}
		}
		end++
	}
	if end >= len(tokens) {
		return nil, start, false
	}
	var arguments [][]byte
	argStart := start + 4
	nesting := 0
	for i := argStart; i < end; i++ {
		text := tokenText(source, tokens[i])
		if text == "(" || text == "[" || text == "{" {
			nesting++
		} else if text == ")" || text == "]" || text == "}" {
			nesting--
		} else if text == "," && nesting == 0 {
			arguments = append(arguments, source[tokens[argStart].Start:tokens[i].Start])
			argStart = i + 1
		}
	}
	if argStart < end {
		arguments = append(arguments, source[tokens[argStart].Start:tokens[end].Start])
	}
	for i := 0; i < len(arguments); i++ {
		arguments[i] = appendRewrittenGoModeExports(nil, arguments[i], names, prefix, true, document, exports)
	}
	var replacement []byte
	if method == "PrimaryLoad" && len(arguments) == 0 {
		replacement = append(replacement, receiver...)
		return append(replacement, ".lastPrimaryLoad"...), end, true
	}
	if method == "Code" && len(arguments) == 0 {
		replacement = append(replacement, receiver...)
		return append(replacement, ".code"...), end, true
	}
	if method == "SetCode" && len(arguments) == 1 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".code = "...)
		return append(replacement, arguments[0]...), end, true
	}
	if method == "Data" && len(arguments) == 0 {
		replacement = append(replacement, receiver...)
		return append(replacement, ".data"...), end, true
	}
	if method == "BSSSize" && len(arguments) == 0 {
		replacement = append(replacement, receiver...)
		return append(replacement, ".bssSize"...), end, true
	}
	if method == "RelocationCount" && len(arguments) == 0 {
		replacement = append(replacement, "len("...)
		replacement = append(replacement, receiver...)
		return append(replacement, ".relocs) / 2"...), end, true
	}
	if method == "AbsoluteRelocationCount" && len(arguments) == 0 {
		replacement = append(replacement, "len("...)
		replacement = append(replacement, receiver...)
		return append(replacement, ".absRelocs) / 3"...), end, true
	}
	if method == "RelocationWordCount" && len(arguments) == 0 {
		replacement = append(replacement, "len("...)
		replacement = append(replacement, receiver...)
		return append(replacement, ".relocs)"...), end, true
	}
	if method == "AbsoluteRelocationWordCount" && len(arguments) == 0 {
		replacement = append(replacement, "len("...)
		replacement = append(replacement, receiver...)
		return append(replacement, ".absRelocs)"...), end, true
	}
	if method == "SymbolCount" && len(arguments) == 0 {
		replacement = append(replacement, "len("...)
		replacement = append(replacement, receiver...)
		return append(replacement, ".symbols)"...), end, true
	}
	field := ""
	scale := 0
	addend := 0
	if method == "RelocationOffset" && len(arguments) == 1 {
		field = "relocs"
		scale = 2
	} else if method == "RelocationLabel" && len(arguments) == 1 {
		field = "relocs"
		scale = 2
		addend = 1
	} else if method == "AbsoluteRelocationOffset" && len(arguments) == 1 {
		field = "absRelocs"
		scale = 3
	} else if method == "AbsoluteRelocationAddend" && len(arguments) == 1 {
		field = "absRelocs"
		scale = 3
		addend = 1
	} else if method == "AbsoluteRelocationKind" && len(arguments) == 1 {
		field = "absRelocs"
		scale = 3
		addend = 2
	}
	if field != "" {
		replacement = append(replacement, "int("...)
		replacement = append(replacement, receiver...)
		replacement = append(replacement, '.')
		replacement = append(replacement, field...)
		replacement = append(replacement, '[')
		replacement = append(replacement, arguments[0]...)
		replacement = append(replacement, '*')
		replacement = appendDecimalFrame(replacement, scale)
		if addend != 0 {
			replacement = append(replacement, '+')
			replacement = appendDecimalFrame(replacement, addend)
		}
		return append(replacement, ']', ')'), end, true
	}
	if (method == "RelocationWord" || method == "AbsoluteRelocationWord") && len(arguments) == 1 {
		replacement = append(replacement, "int(renvo_runtime_UnsafeInt32At("...)
		replacement = append(replacement, receiver...)
		if method == "RelocationWord" {
			replacement = append(replacement, ".relocs,"...)
		} else {
			replacement = append(replacement, ".absRelocs,"...)
		}
		replacement = append(replacement, arguments[0]...)
		return append(replacement, ')', ')'), end, true
	}
	if method == "Symbol" && len(arguments) == 1 {
		replacement = append(replacement, '&')
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbols["...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, ']'), end, true
	}
	if method == "SymbolNameByteAt" && len(arguments) == 1 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbolName["...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, ']'), end, true
	}
	if method == "SymbolPC" && len(arguments) == 1 {
		replacement = append(replacement, "renvoAsmLabelPosition("...)
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ',')
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbols["...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, "].label)"...), end, true
	}
	if method == "SymbolNameLength" && len(arguments) == 1 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbols["...)
		replacement = append(replacement, arguments[0]...)
		replacement = append(replacement, "].nameEnd - "...)
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbols["...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, "].nameStart"...), end, true
	}
	if method == "SymbolNameByte" && len(arguments) == 2 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbolName["...)
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".symbols["...)
		replacement = append(replacement, arguments[0]...)
		replacement = append(replacement, "].nameStart+"...)
		replacement = append(replacement, arguments[1]...)
		return append(replacement, ']'), end, true
	}
	nativeCall := ""
	if method == "RelocationAt" && len(arguments) == 1 {
		nativeCall = "renvoRTGRelocationAt"
	} else if method == "AbsoluteRelocationAt" && len(arguments) == 1 {
		nativeCall = "renvoRTGAbsoluteRelocationAt"
	} else if method == "LabelPosition" && len(arguments) == 1 {
		nativeCall = "renvoAsmLabelPosition"
	} else if method == "SymbolNameEquals" && len(arguments) == 2 {
		nativeCall = "renvoRTGSymbolNameEquals"
	}
	if nativeCall != "" {
		replacement = append(replacement, nativeCall...)
		replacement = append(replacement, '(')
		replacement = append(replacement, receiver...)
		for i := 0; i < len(arguments); i++ {
			replacement = append(replacement, ',')
			replacement = append(replacement, arguments[i]...)
		}
		return append(replacement, ')'), end, true
	}
	if method == "SetPrimaryLoad" && len(arguments) == 1 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".lastPrimaryLoad = "...)
		return append(replacement, arguments[0]...), end, true
	}
	if method == "ByteAt" && len(arguments) == 1 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".code["...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, ']'), end, true
	}
	if (method == "SetByteAt" || method == "AddByteAt") && len(arguments) == 2 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".code["...)
		replacement = append(replacement, arguments[0]...)
		if method == "SetByteAt" {
			replacement = append(replacement, "] = "...)
		} else {
			replacement = append(replacement, "] += "...)
		}
		return append(replacement, arguments[1]...), end, true
	}
	if method == "AppendByte" && len(arguments) == 1 {
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".code = append("...)
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".code, byte("...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, ')', ')'), end, true
	}
	if method == "Truncate" && len(arguments) == 1 {
		replacement = append(replacement, "renvoTruncBytes(&"...)
		replacement = append(replacement, receiver...)
		replacement = append(replacement, ".code, "...)
		replacement = append(replacement, arguments[0]...)
		return append(replacement, ')'), end, true
	}
	return nil, start, false
}

func nativeBuiltinLiteral(name string) string {
	if name == "RTGRelocationAbsoluteData" {
		return "0"
	}
	if name == "RTGRelocationAbsoluteBSS" {
		return "1"
	}
	if name == "RTGScalarByte" {
		return "3"
	}
	if name == "RTGScalarInt8" {
		return "7"
	}
	if name == "RTGScalarInt16" {
		return "8"
	}
	if name == "RTGScalarInt32" {
		return "9"
	}
	if name == "RTGScalarUint16" {
		return "16"
	}
	if name == "RTGScalarUint32" {
		return "17"
	}
	return ""
}

func nativeEmitterFunction(name string) string {
	if name == "RTGByte" {
		return "renvoAsmEmit8"
	}
	if name == "RTGUint32" {
		return "renvoAsmEmit32"
	}
	if name == "RTGUint64" {
		return "renvoRTGUint64"
	}
	if name == "RTGPatchUint32" {
		return "renvoRTGPatchUint32"
	}
	if name == "RTGReloc" {
		return "renvoRTGReloc"
	}
	if name == "RTGAddressValid" {
		return "renvoRTGAddressValid"
	}
	if name == "RTGAddressRel32Addend" {
		return "renvoRTGAddressRel32Addend"
	}
	if name == "RTGInt8Fits" {
		return "renvoAsmImmFits8Signed"
	}
	if name == "RTGPopPrimary" {
		return "renvoAsmPopPrimary"
	}
	if name == "RTGArenaMark" {
		return "renvo_runtime_ArenaMark"
	}
	if name == "RTGArenaReset" {
		return "renvo_runtime_ArenaReset"
	}
	if name == "RTGByteAt" {
		return "renvo_runtime_UnsafeByteAt"
	}
	if name == "RTGGet32At" {
		return "renvoGet32At"
	}
	if name == "RTGPut32At" {
		return "renvoPut32At"
	}
	if name == "RTGAppend32" {
		return "renvoAppend32"
	}
	if name == "RTGAlignValue" {
		return "renvoAlignValue"
	}
	if name == "RTGAlign8" {
		return "renvoAlignTo8"
	}
	if name == "RTGMakeIntScratch" {
		return "renvoMakeIntScratch"
	}
	if name == "RTGMakeByteBuffer" {
		return "renvoMakeByteBuffer"
	}
	if name == "RTGNewLabel" {
		return "renvoRTGNewLabel"
	}
	if name == "RTGMark" {
		return "renvoRTGMark"
	}
	if name == "RTGLabelFromCode" {
		return "renvoRTGLabelCode"
	}
	return ""
}

func appendQuoted(out []byte, value string) []byte {
	out = append(out, '"')
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' || value[i] == '"' {
			out = append(out, '\\')
		}
		out = append(out, value[i])
	}
	return append(out, '"')
}

func exportedName(name string) string {
	var out []byte
	upper := true
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '_' || ch == '-' || ch == '/' || ch == '.' {
			upper = true
			continue
		}
		if ch >= 'a' && ch <= 'z' && upper {
			ch -= 'a' - 'A'
		}
		if isIdentPart(ch) {
			out = append(out, ch)
			upper = false
		} else {
			upper = true
		}
	}
	if len(out) == 0 {
		return "Unit"
	}
	return string(out)
}

func lookupResolvedTarget(resolved ResolveResult, name string) (ResolvedTarget, bool) {
	for i := 0; i < len(resolved.Targets); i++ {
		if resolved.Targets[i].Descriptor.Name == name || stringIndex(resolved.Targets[i].Descriptor.Aliases, name) >= 0 {
			return resolved.Targets[i], true
		}
	}
	return ResolvedTarget{}, false
}

func sortResolvedTargets(targets []ResolvedTarget) {
	for i := 1; i < len(targets); i++ {
		value := targets[i]
		at := i
		for at > 0 && stringLess(value.Descriptor.Name, targets[at-1].Descriptor.Name) {
			targets[at] = targets[at-1]
			at--
		}
		targets[at] = value
	}
}

func stringIndex(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return i
		}
	}
	return -1
}
