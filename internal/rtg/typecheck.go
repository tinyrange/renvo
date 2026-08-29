package rtg

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

const embeddedGoPrelude = `package main

type RTGRegister struct {
	Code int
	Valid bool
}

type RTGLabel struct {
	Code int
	Valid bool
}

type RTGAddress struct {
	Target RTGLabel
	Kind int
	Addend int
	Base RTGRegister
	Index RTGRegister
	Displacement int
	Scale int
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

const RTGShiftLeft RTGShiftDirection = 1
const RTGShiftRight RTGShiftDirection = 2
const RTGRelocationAbsoluteData = 0
const RTGRelocationAbsoluteBSS = 1
const RTGRelocationImport = 2
const RTGRelocationAbsoluteBSSEnd = 3
const RTGRuntimeRead = 1
const RTGRuntimeWrite = 2
const RTGRuntimeReadAt = 3
const RTGRuntimeWriteAt = 4
const RTGRuntimeOpen = 5
const RTGRuntimeClose = 6
const RTGRuntimeChmod = 7
const RTGScalarByte = 3
const RTGScalarInt8 = 7
const RTGScalarInt16 = 8
const RTGScalarInt32 = 9
const RTGScalarUint16 = 16
const RTGScalarUint32 = 17
var RTGNoRegister RTGRegister

type RTGEmitter struct{}

func (out *RTGEmitter) Len() int { return 0 }
func (out *RTGEmitter) Byte(value byte) {}
func (out *RTGEmitter) Int8(value int) {}
func (out *RTGEmitter) Uint16(value int) {}
func (out *RTGEmitter) Uint24(value int) {}
func (out *RTGEmitter) Uint32(value uint32) {}
func (out *RTGEmitter) Int32(value int) {}
func (out *RTGEmitter) Uint64(value uint64) {}
func (out *RTGEmitter) Int64(value int) {}
func (out *RTGEmitter) Bytes2(v0 int, v1 int) {}
func (out *RTGEmitter) Bytes3(v0 int, v1 int, v2 int) {}
func (out *RTGEmitter) Bytes4(v0 int, v1 int, v2 int, v3 int) {}
func (out *RTGEmitter) Text(value string) {}
func (out *RTGEmitter) PatchUint32(at int, value int) {}
func (out *RTGEmitter) AbsoluteReloc(at int, offset int, kind int) {}
func (out *RTGEmitter) RelocAt(at int, label int) {}
func (out *RTGEmitter) PrimaryLoad() int { return 0 }
func (out *RTGEmitter) SetPrimaryLoad(value int) {}
func (out *RTGEmitter) OptimizeRuntime() bool { return false }
func (out *RTGEmitter) VMBytecode() bool { return false }
func (out *RTGEmitter) ByteAt(at int) byte { return 0 }
func (out *RTGEmitter) SetByteAt(at int, value byte) {}
func (out *RTGEmitter) AddByteAt(at int, value byte) {}
func (out *RTGEmitter) AppendByte(value int) {}
func (out *RTGEmitter) Truncate(size int) {}
func (out *RTGEmitter) Code() []byte { return nil }
func (out *RTGEmitter) SetCode(value []byte) {}
func (out *RTGEmitter) Data() []byte { return nil }
func (out *RTGEmitter) SetData(value []byte) {}
func (out *RTGEmitter) SetDataOffset(value int) {}
func (out *RTGEmitter) BSSSize() int { return 0 }
func (out *RTGEmitter) RejectImageSize(memory bool, needed int, limit int) {}
func (out *RTGEmitter) WasmLocalSlots() []int32 { return nil }
func (out *RTGEmitter) ReserveBSS(size int, alignment int) int { return 0 }
func (out *RTGEmitter) WindowsSubsystem() int { return 0 }
func (out *RTGEmitter) StaticImportCount() int { return 0 }
func (out *RTGEmitter) StaticImportDLL(index int) string { return "" }
func (out *RTGEmitter) StaticImportName(index int) string { return "" }
func (out *RTGEmitter) StaticCallParameterCount() int { return 0 }
func (out *RTGEmitter) StaticCallParameterKind(index int) int { return 0 }
func (out *RTGEmitter) StaticCallResultFloatRegister() int { return -1 }
func (out *RTGEmitter) DynamicImport(library string, name string) int { return 0 }
func (out *RTGEmitter) DynamicImportCount() int { return 0 }
func (out *RTGEmitter) DynamicImportLibrary(index int) string { return "" }
func (out *RTGEmitter) DynamicImportName(index int) string { return "" }
func (out *RTGEmitter) DynamicImportLabel(index int) int { return -1 }
func (out *RTGEmitter) ExternalImport(name string) int { return -1 }
func (out *RTGEmitter) ExternalImportCount() int { return 0 }
func (out *RTGEmitter) ExternalImportName(index int) string { return "" }
func (out *RTGEmitter) KernelModuleName() string { return "" }
func (out *RTGEmitter) KernelLicense() string { return "" }
func (out *RTGEmitter) KernelRelease() string { return "" }
func (out *RTGEmitter) KernelVersion() string { return "" }
func (out *RTGEmitter) KernelBTF() []byte { return nil }
func (out *RTGEmitter) KernelSymvers() []byte { return nil }
func (out *RTGEmitter) RelocationCount() int { return 0 }
func (out *RTGEmitter) RelocationAt(index int) (int, int) { return 0, 0 }
func (out *RTGEmitter) RelocationOffset(index int) int { return 0 }
func (out *RTGEmitter) RelocationLabel(index int) int { return 0 }
func (out *RTGEmitter) RelocationWordCount() int { return 0 }
func (out *RTGEmitter) RelocationWord(index int) int { return 0 }
func (out *RTGEmitter) AbsoluteRelocationCount() int { return 0 }
func (out *RTGEmitter) AbsoluteRelocationAt(index int) (int, int, int) { return 0, 0, 0 }
func (out *RTGEmitter) AbsoluteRelocationOffset(index int) int { return 0 }
func (out *RTGEmitter) AbsoluteRelocationAddend(index int) int { return 0 }
func (out *RTGEmitter) AbsoluteRelocationKind(index int) int { return 0 }
func (out *RTGEmitter) AbsoluteRelocationWordCount() int { return 0 }
func (out *RTGEmitter) AbsoluteRelocationWord(index int) int { return 0 }
func (out *RTGEmitter) LabelPosition(label int) int { return -1 }
func (out *RTGEmitter) SymbolCount() int { return 0 }
func (out *RTGEmitter) SymbolPC(index int) int { return -1 }
func (out *RTGEmitter) SymbolNameEquals(index int, name string) bool { return false }
func (out *RTGEmitter) SymbolNameLength(index int) int { return 0 }
func (out *RTGEmitter) SymbolNameByte(index int, offset int) byte { return 0 }
func (out *RTGEmitter) Symbol(index int) *RTGSymbol { return nil }
func (out *RTGEmitter) SymbolNameByteAt(index int) byte { return 0 }
func (out *RTGEmitter) Rel32(label RTGLabel) {}
func (out *RTGEmitter) Rel32Addend(label RTGLabel, addend int) {}
func (out *RTGEmitter) Reloc(label RTGLabel) {}
func (out *RTGEmitter) Patch() {}
func (out *RTGEmitter) NewLabel() RTGLabel { return RTGLabel{} }
func (out *RTGEmitter) Mark(label RTGLabel) {}
func RTGByte(out *RTGEmitter, value byte) {}
func RTGUint32(out *RTGEmitter, value int) {}
func RTGUint64(out *RTGEmitter, value uint64) {}
func RTGPatchUint32(out *RTGEmitter, at int, value int) {}
func RTGReloc(out *RTGEmitter, label RTGLabel) {}
func RTGLabelIndex(label RTGLabel) int { return label.Code }
func RTGAddressValid(address RTGAddress) bool { return false }
func RTGAddressRel32Addend(out *RTGEmitter, address RTGAddress) {}
func RTGAddressRelocAt(out *RTGEmitter, address RTGAddress, at int) {}
func RTGDataAddress(offset int) RTGAddress { return RTGAddress{} }
func RTGBSSAddress(offset int) RTGAddress { return RTGAddress{} }
func RTGBSSEndAddress(alignment int) RTGAddress { return RTGAddress{} }
func RTGResolveAbsoluteRelocation(out *RTGEmitter, kind int, addend int, dataAddress int, bssAddress int) (int, bool) {
	return 0, false
}
func RTGInt8Fits(value int) bool { return false }
func RTGPopPrimary(out *RTGEmitter) {}
func RTGArenaMark() int { return 0 }
func RTGArenaReset(mark int) {}
func RTGByteAt(in []byte, at int) byte { return 0 }
func RTGGet32At(in []byte, at int) int { return 0 }
func RTGPut32At(out []byte, at int, value int) {}
func RTGAppend32(out []byte, value int) []byte { return out }
func RTGAlignValue(value int, alignment int) int { return 0 }
func RTGAlign8(value int) int { return 0 }
func RTGMakeIntScratch(capacity int) []int { return nil }
func RTGMakeByteBuffer(length int) []byte { return nil }
func RTGSHA256(data []byte, digest []byte) bool { return false }

func RTGSignedFits(value int64, bits int) bool { return false }
func RTGUnsignedFits(value uint64, bits int) bool { return false }
func RTGLog2(value int) int { return 0 }
`

func validateEmbeddedGoTypes(document Document) (Diagnostic, bool) {
	var source []byte
	source = append(source, embeddedGoPrelude...)
	source = appendGeneratedSymbolPrelude(source, document)
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind == DeclArch {
			source = appendStatefulArchitectureSequences(source, document.Declarations[i])
		}
	}
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo || declaration.Name != "backend" {
			continue
		}
		source = append(source, '\n')
		source = append(source, declaration.GoSource...)
		source = append(source, '\n')
	}
	source = append(source, "\nfunc main() {}\n"...)
	files := []load.SourceFile{
		{Path: "/rtg/go.mod", Src: []byte("module rtg.backend\n")},
		{Path: "/rtg/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/rtg", "/std", ".", files)
	if !workspace.Ok {
		return Diagnostic{
			Filename: document.Filename,
			Span:     sourceSpan(document.Source, 0, 0),
			Code:     "RTG-GO-009",
			Message:  "embedded Go could not be loaded for type checking",
		}, false
	}
	program := check.CheckGraphCore(workspace.Graph)
	if program.Ok {
		return Diagnostic{}, true
	}
	message := "embedded Go does not type-check against the RTG backend API"
	if program.Error == check.CheckErrUndefined {
		message = "embedded Go refers to an undefined name"
	} else if program.Error == check.CheckErrCallArity || program.Error == check.CheckErrCallArgument {
		message = "embedded Go call does not match its declared signature"
	} else if program.Error == check.CheckErrReturnType || program.Error == check.CheckErrReturnCount {
		message = "embedded Go return does not match its declared signature"
	}
	if program.ErrorPackage >= 0 && program.ErrorPackage < len(program.Graph.Packages) {
		pkg := program.Graph.Packages[program.ErrorPackage]
		if program.ErrorFile >= 0 && program.ErrorFile < len(pkg.Files) {
			file := pkg.Files[program.ErrorFile]
			if program.ErrorToken >= 0 && program.ErrorToken < len(file.File.Tokens) {
				token := file.File.Tokens[program.ErrorToken]
				name := string(syntax.TokenText(file.Src, token))
				if name != "" {
					message += ": " + name
				}
				if program.ErrorToken > 0 {
					previous := string(syntax.TokenText(
						file.Src, file.File.Tokens[program.ErrorToken-1]))
					if previous != "" {
						message += " after " + previous
					}
				}
				line := syntax.TokenLine(token) + 1
				message += " at embedded line " + decimalString(line)
				text := sourceLine(file.Src, line)
				if compactSource([]byte(text)) == "" && line > 1 {
					text = sourceLine(file.Src, line-1)
				}
				if text != "" {
					message += ": " + text
				}
			}
		}
	}
	message += " (check error " + decimalString(program.Error) + ")"
	return Diagnostic{
		Filename: document.Filename,
		Span:     sourceSpan(document.Source, 0, 0),
		Code:     "RTG-GO-009",
		Message:  message,
	}, false
}

func sourceLine(source []byte, line int) string {
	if line <= 0 {
		return ""
	}
	start := 0
	current := 1
	for start < len(source) && current < line {
		if source[start] == '\n' {
			current++
		}
		start++
	}
	end := start
	for end < len(source) && source[end] != '\n' && end-start < 160 {
		end++
	}
	return string(source[start:end])
}

func decimalString(value int) string {
	if value == 0 {
		return "0"
	}
	var reversed []byte
	for value > 0 {
		reversed = append(reversed, byte('0'+value%10))
		value /= 10
	}
	var out []byte
	for i := len(reversed) - 1; i >= 0; i-- {
		out = append(out, reversed[i])
	}
	return string(out)
}
