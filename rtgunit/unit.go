package rtgunit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	Magic   = "RTGU"
	Version = uint16(1)
)

const (
	TagUnit       = uint16(1)
	TagPackage    = uint16(2)
	TagImportPath = uint16(3)
	TagText       = uint16(7)
	TagTokens     = uint16(8)
	TagDecls      = uint16(9)
	TagFuncs      = uint16(10)
	TagIndexes    = uint16(11)
	TagComps      = uint16(12)
	TagAssigns    = uint16(13)
	TagReturns    = uint16(14)
	TagCalls      = uint16(15)
	TagRefs       = uint16(16)
	TagSels       = uint16(17)
	TagTypes      = uint16(18)
	TagTypeRefs   = uint16(19)
	TagLocals     = uint16(20)
	TagSigs       = uint16(21)
	TagDeclMeta   = uint16(22)
	TagImports    = uint16(23)
	TagSymbols    = uint16(24)
	TagInitOrder  = uint16(25)
	TagConsts     = uint16(26)
	TagTypeFields = uint16(27)
	TagTypeIfaces = uint16(28)
	TagMethods    = uint16(29)
)

const (
	rtgTokEOF      = 0
	rtgTokIdent    = 1
	rtgTokNumber   = 2
	rtgTokFloat    = 3
	rtgTokString   = 4
	rtgTokChar     = 5
	rtgTokPackage  = 6
	rtgTokConst    = 7
	rtgTokVar      = 8
	rtgTokType     = 9
	rtgTokFunc     = 10
	rtgTokStruct   = 11
	rtgTokReturn   = 12
	rtgTokIf       = 13
	rtgTokElse     = 14
	rtgTokFor      = 15
	rtgTokBreak    = 16
	rtgTokContinue = 17
	rtgTokGoto     = 18
	rtgTokSwitch   = 19
	rtgTokCase     = 20
	rtgTokDefault  = 21
	rtgTokOp       = 22
)

const tokenStride = 8

type Node struct {
	Tag      uint16
	Data     []byte
	Children []Node
}

type Decl struct {
	Kind      int
	NameStart int
	NameEnd   int
	StartTok  int
	EndTok    int
}

type Import struct {
	Name       string
	ImportPath string
	Package    int
	NameTok    int
	PathTok    int
	Dot        bool
	Blank      bool
}

const (
	SymbolConst = iota + 1
	SymbolVar
	SymbolType
	SymbolFunc
	SymbolMethod
)

type Symbol struct {
	Name       string
	Kind       int
	Package    int
	Token      int
	OwnerKind  int
	OwnerIndex int
}

type DeclMeta struct {
	DeclIndex  int
	Symbol     int
	ValueIndex int
	TypeStart  int
	TypeEnd    int
	ValueStart int
	ValueEnd   int
	Values     []ExprSpan
	Alias      bool
}

const (
	ConstInt = iota + 1
	ConstString
	ConstBool
)

type ConstValue struct {
	DeclIndex int
	Kind      int
	Int       int
	String    string
	Bool      bool
}

type Func struct {
	NameStart     int
	NameEnd       int
	StartTok      int
	NameTok       int
	ReceiverStart int
	ReceiverEnd   int
	BodyStart     int
	BodyEnd       int
	EndTok        int
}

type Field struct {
	NameTok   int
	TypeStart int
	TypeEnd   int
	Variadic  bool
}

type FuncSignature struct {
	FuncIndex int
	Receiver  []Field
	Params    []Field
	Results   []Field
}

const (
	TypeOther = iota
	TypeNamed
	TypeStruct
	TypeInterface
	TypeMap
	TypeSlice
	TypeArray
	TypePointer
	TypeFunc
)

type TypeInfo struct {
	NameStart int
	NameEnd   int
	Kind      int
	Decl      int
	Symbol    int
	Alias     bool
	TypeStart int
	TypeEnd   int
	LenStart  int
	LenEnd    int
	KeyStart  int
	KeyEnd    int
	ElemStart int
	ElemEnd   int
}

type TypeFields struct {
	TypeIndex int
	Fields    []Field
}

type InterfaceMethod struct {
	NameTok int
	Params  []Field
	Results []Field
}

type InterfaceEmbed struct {
	TypeStart int
	TypeEnd   int
}

type TypeIface struct {
	TypeIndex int
	Methods   []InterfaceMethod
	Embeds    []InterfaceEmbed
}

type MethodInfo struct {
	NameTok   int
	TypeIndex int
	Symbol    int
	FuncIndex int
	Pointer   bool
}

const (
	OwnerDecl = iota + 1
	OwnerFunc
)

type ExprSpan struct {
	StartTok int
	EndTok   int
}

type IndexExpr struct {
	OwnerKind  int
	OwnerIndex int
	StartTok   int
	EndTok     int
	BaseStart  int
	BaseEnd    int
	OpenTok    int
	CloseTok   int
	IndexStart int
	IndexEnd   int
}

type CompositeExpr struct {
	OwnerKind  int
	OwnerIndex int
	StartTok   int
	EndTok     int
	TypeStart  int
	TypeEnd    int
	OpenTok    int
	CloseTok   int
	Elems      []ExprSpan
}

const (
	AssignUnknown = iota
	AssignSet
	AssignDefine
	AssignAdd
	AssignSub
	AssignMul
	AssignDiv
	AssignMod
	AssignAnd
	AssignOr
	AssignXor
)

type Assignment struct {
	FuncIndex  int
	Kind       int
	StartTok   int
	EndTok     int
	OpTok      int
	LeftStart  int
	LeftEnd    int
	RightStart int
	RightEnd   int
	Targets    []ExprSpan
	Values     []ExprSpan
}

type Return struct {
	FuncIndex int
	StartTok  int
	EndTok    int
	Values    []ExprSpan
}

const (
	CallUnknown = iota
	CallScope
	CallPackage
	CallImportSelector
	CallBuiltin
)

type Call struct {
	OwnerKind  int
	OwnerIndex int
	Kind       int
	CalleeTok  int
	BaseTok    int
	DotTok     int
	ArgsStart  int
	ArgsEnd    int
	Args       []ExprSpan
}

const (
	RefUnknown = iota
	RefScope
	RefPackage
	RefImport
	RefBuiltin
	RefLabel
)

type NameRef struct {
	OwnerKind  int
	OwnerIndex int
	Kind       int
	Token      int
	Index      int
	Package    int
}

const (
	SelectorUnknown = iota
	SelectorImport
)

type Selector struct {
	OwnerKind   int
	OwnerIndex  int
	Kind        int
	BaseTok     int
	DotTok      int
	NameTok     int
	BaseKind    int
	BaseIndex   int
	BasePackage int
	Package     int
	Symbol      int
}

const (
	TypeRefUnknown = iota
	TypeRefScope
	TypeRefPackage
	TypeRefImportSelector
	TypeRefBuiltin
)

type TypeRef struct {
	OwnerKind  int
	OwnerIndex int
	Kind       int
	Token      int
	BaseTok    int
	DotTok     int
	Package    int
	Symbol     int
}

type LocalDecl struct {
	FuncIndex  int
	Kind       int
	NameStart  int
	NameEnd    int
	Token      int
	Scope      int
	ValueIndex int
	TypeStart  int
	TypeEnd    int
	ValueStart int
	ValueEnd   int
	Values     []ExprSpan
	Alias      bool
}

type Program struct {
	Package    string
	ImportPath string
	Text       []byte
	Tokens     []byte
	Imports    []Import
	Symbols    []Symbol
	Decls      []Decl
	DeclMeta   []DeclMeta
	InitOrder  []int
	Consts     []ConstValue
	Funcs      []Func
	Signatures []FuncSignature
	Types      []TypeInfo
	TypeFields []TypeFields
	TypeIfaces []TypeIface
	Methods    []MethodInfo
	TypeRefs   []TypeRef
	Locals     []LocalDecl
	Indexes    []IndexExpr
	Composites []CompositeExpr
	Assigns    []Assignment
	Returns    []Return
	Calls      []Call
	Refs       []NameRef
	Selectors  []Selector
}

type sourceToken struct {
	kind  int
	text  string
	line  int
	start int
	end   int
}

type sourceDecl struct {
	kind     int
	nameTok  int
	startTok int
	endTok   int
}

type sourceFunc struct {
	nameTok       int
	startTok      int
	receiverStart int
	receiverEnd   int
	bodyStart     int
	bodyEnd       int
	endTok        int
}

func NewNode(tag uint16, data []byte) Node {
	if len(data) == 0 {
		return Node{Tag: tag}
	}
	out := make([]byte, len(data))
	copy(out, data)
	return Node{Tag: tag, Data: out}
}

func Parent(tag uint16, children ...Node) Node {
	out := Node{Tag: tag}
	out.Children = append(out.Children, children...)
	return out
}

func Marshal(program Program) ([]byte, error) {
	if len(program.Tokens)%tokenStride != 0 {
		return nil, fmt.Errorf("token data length %d is not a multiple of %d", len(program.Tokens), tokenStride)
	}
	tokens, err := encodeTokens(program.Text, program.Tokens)
	if err != nil {
		return nil, err
	}
	root := Parent(TagUnit,
		NewNode(TagPackage, []byte(program.Package)),
		NewNode(TagImportPath, []byte(program.ImportPath)),
		NewNode(TagText, program.Text),
		NewNode(TagTokens, tokens),
		NewNode(TagImports, encodeImports(program.Imports)),
		NewNode(TagSymbols, encodeSymbols(program.Symbols)),
		NewNode(TagDecls, encodeDecls(program.Decls)),
		NewNode(TagDeclMeta, encodeDeclMeta(program.DeclMeta)),
		NewNode(TagInitOrder, encodeInitOrder(program.InitOrder)),
		NewNode(TagConsts, encodeConsts(program.Consts)),
		NewNode(TagFuncs, encodeFuncs(program.Funcs)),
		NewNode(TagSigs, encodeSignatures(program.Signatures)),
		NewNode(TagTypes, encodeTypes(program.Types)),
		NewNode(TagTypeFields, encodeTypeFields(program.TypeFields)),
		NewNode(TagTypeIfaces, encodeTypeInterfaces(program.TypeIfaces)),
		NewNode(TagMethods, encodeMethods(program.Methods)),
		NewNode(TagTypeRefs, encodeTypeRefs(program.TypeRefs)),
		NewNode(TagLocals, encodeLocals(program.Locals)),
		NewNode(TagIndexes, encodeIndexes(program.Indexes)),
		NewNode(TagComps, encodeComposites(program.Composites)),
		NewNode(TagAssigns, encodeAssignments(program.Assigns)),
		NewNode(TagReturns, encodeReturns(program.Returns)),
		NewNode(TagCalls, encodeCalls(program.Calls)),
		NewNode(TagRefs, encodeRefs(program.Refs)),
		NewNode(TagSels, encodeSelectors(program.Selectors)),
	)

	var out bytes.Buffer
	out.WriteString(Magic)
	var header [4]byte
	binary.LittleEndian.PutUint16(header[0:2], Version)
	binary.LittleEndian.PutUint16(header[2:4], 0)
	out.Write(header[:])
	if err := writeNode(&out, root); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func Unmarshal(data []byte) (Program, error) {
	var program Program
	if len(data) < 14 {
		return program, fmt.Errorf("unit too short")
	}
	if string(data[:4]) != Magic {
		return program, fmt.Errorf("bad unit magic")
	}
	version := binary.LittleEndian.Uint16(data[4:6])
	if version != Version {
		return program, fmt.Errorf("unsupported unit version %d", version)
	}
	rootTag := binary.LittleEndian.Uint16(data[8:10])
	rootLength := int(binary.LittleEndian.Uint32(data[10:14]))
	if rootTag != TagUnit {
		return program, fmt.Errorf("root tag = %d, want %d", rootTag, TagUnit)
	}
	rootStart := 14
	rootEnd := rootStart + rootLength
	if rootEnd < rootStart || rootEnd != len(data) {
		return program, fmt.Errorf("invalid root length")
	}
	var tokenData []byte
	var importData []byte
	var symbolData []byte
	var declMetaData []byte
	var initOrderData []byte
	var constData []byte
	var sigData []byte
	var typeData []byte
	var typeFieldData []byte
	var typeIfaceData []byte
	var methodData []byte
	var typeRefData []byte
	var localData []byte
	var indexData []byte
	var compData []byte
	var assignData []byte
	var returnData []byte
	var callData []byte
	var refData []byte
	var selectorData []byte
	seenImportPath := false
	seenImports := false
	seenSymbols := false
	seenDeclMeta := false
	seenInitOrder := false
	seenConsts := false
	seenSigs := false
	seenTypes := false
	seenTypeFields := false
	seenTypeIfaces := false
	seenMethods := false
	seenTypeRefs := false
	seenLocals := false
	seenIndexes := false
	seenComps := false
	seenAssigns := false
	seenReturns := false
	seenCalls := false
	seenRefs := false
	seenSelectors := false
	pos := rootStart
	for pos < rootEnd {
		if pos+6 > rootEnd {
			return program, fmt.Errorf("truncated child node")
		}
		tag := binary.LittleEndian.Uint16(data[pos : pos+2])
		length := int(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))
		pos += 6
		next := pos + length
		if next < pos || next > rootEnd {
			return program, fmt.Errorf("invalid child length")
		}
		payload := data[pos:next]
		switch tag {
		case TagPackage:
			program.Package = string(payload)
		case TagImportPath:
			if seenImportPath {
				return program, fmt.Errorf("duplicate import path")
			}
			seenImportPath = true
			program.ImportPath = string(payload)
		case TagText:
			program.Text = payload
		case TagTokens:
			tokenData = payload
		case TagImports:
			if seenImports {
				return program, fmt.Errorf("duplicate import table")
			}
			seenImports = true
			importData = payload
		case TagSymbols:
			if seenSymbols {
				return program, fmt.Errorf("duplicate symbol table")
			}
			seenSymbols = true
			symbolData = payload
		case TagDecls:
			decls, err := decodeDecls(payload)
			if err != nil {
				return program, err
			}
			program.Decls = decls
		case TagDeclMeta:
			if seenDeclMeta {
				return program, fmt.Errorf("duplicate decl metadata table")
			}
			seenDeclMeta = true
			declMetaData = payload
		case TagInitOrder:
			if seenInitOrder {
				return program, fmt.Errorf("duplicate init order table")
			}
			seenInitOrder = true
			initOrderData = payload
		case TagConsts:
			if seenConsts {
				return program, fmt.Errorf("duplicate const table")
			}
			seenConsts = true
			constData = payload
		case TagFuncs:
			funcs, err := decodeFuncs(payload)
			if err != nil {
				return program, err
			}
			program.Funcs = funcs
		case TagSigs:
			if seenSigs {
				return program, fmt.Errorf("duplicate signature table")
			}
			seenSigs = true
			sigData = payload
		case TagTypes:
			if seenTypes {
				return program, fmt.Errorf("duplicate type table")
			}
			seenTypes = true
			typeData = payload
		case TagTypeFields:
			if seenTypeFields {
				return program, fmt.Errorf("duplicate type field table")
			}
			seenTypeFields = true
			typeFieldData = payload
		case TagTypeIfaces:
			if seenTypeIfaces {
				return program, fmt.Errorf("duplicate type interface table")
			}
			seenTypeIfaces = true
			typeIfaceData = payload
		case TagMethods:
			if seenMethods {
				return program, fmt.Errorf("duplicate method table")
			}
			seenMethods = true
			methodData = payload
		case TagTypeRefs:
			if seenTypeRefs {
				return program, fmt.Errorf("duplicate type ref table")
			}
			seenTypeRefs = true
			typeRefData = payload
		case TagLocals:
			if seenLocals {
				return program, fmt.Errorf("duplicate local table")
			}
			seenLocals = true
			localData = payload
		case TagIndexes:
			if seenIndexes {
				return program, fmt.Errorf("duplicate index table")
			}
			seenIndexes = true
			indexData = payload
		case TagComps:
			if seenComps {
				return program, fmt.Errorf("duplicate composite table")
			}
			seenComps = true
			compData = payload
		case TagAssigns:
			if seenAssigns {
				return program, fmt.Errorf("duplicate assignment table")
			}
			seenAssigns = true
			assignData = payload
		case TagReturns:
			if seenReturns {
				return program, fmt.Errorf("duplicate return table")
			}
			seenReturns = true
			returnData = payload
		case TagCalls:
			if seenCalls {
				return program, fmt.Errorf("duplicate call table")
			}
			seenCalls = true
			callData = payload
		case TagRefs:
			if seenRefs {
				return program, fmt.Errorf("duplicate ref table")
			}
			seenRefs = true
			refData = payload
		case TagSels:
			if seenSelectors {
				return program, fmt.Errorf("duplicate selector table")
			}
			seenSelectors = true
			selectorData = payload
		default:
			return program, fmt.Errorf("unknown unit child tag %d", tag)
		}
		pos = next
	}
	if program.Package == "" {
		return program, fmt.Errorf("unit missing package")
	}
	if len(program.Text) == 0 {
		return program, fmt.Errorf("unit missing text pool")
	}
	tokens, err := decodeTokens(program.Text, tokenData)
	if err != nil {
		return program, err
	}
	program.Tokens = tokens
	if seenImports {
		imports, err := decodeImports(importData)
		if err != nil {
			return program, err
		}
		program.Imports = imports
	}
	if seenSymbols {
		symbols, err := decodeSymbols(symbolData)
		if err != nil {
			return program, err
		}
		program.Symbols = symbols
	}
	if seenDeclMeta {
		declMeta, err := decodeDeclMeta(declMetaData)
		if err != nil {
			return program, err
		}
		program.DeclMeta = declMeta
	}
	if seenInitOrder {
		initOrder, err := decodeInitOrder(initOrderData)
		if err != nil {
			return program, err
		}
		program.InitOrder = initOrder
	}
	if seenConsts {
		consts, err := decodeConsts(constData)
		if err != nil {
			return program, err
		}
		program.Consts = consts
	}
	if seenSigs {
		sigs, err := decodeSignatures(sigData)
		if err != nil {
			return program, err
		}
		program.Signatures = sigs
	}
	if seenTypes {
		types, err := decodeTypes(typeData)
		if err != nil {
			return program, err
		}
		program.Types = types
	}
	if seenTypeFields {
		typeFields, err := decodeTypeFields(typeFieldData)
		if err != nil {
			return program, err
		}
		program.TypeFields = typeFields
	}
	if seenTypeIfaces {
		typeIfaces, err := decodeTypeInterfaces(typeIfaceData)
		if err != nil {
			return program, err
		}
		program.TypeIfaces = typeIfaces
	}
	if seenMethods {
		methods, err := decodeMethods(methodData)
		if err != nil {
			return program, err
		}
		program.Methods = methods
	}
	if seenTypeRefs {
		typeRefs, err := decodeTypeRefs(typeRefData)
		if err != nil {
			return program, err
		}
		program.TypeRefs = typeRefs
	}
	if seenLocals {
		locals, err := decodeLocals(localData)
		if err != nil {
			return program, err
		}
		program.Locals = locals
	}
	if seenIndexes {
		indexes, err := decodeIndexes(indexData)
		if err != nil {
			return program, err
		}
		program.Indexes = indexes
	}
	if seenComps {
		composites, err := decodeComposites(compData)
		if err != nil {
			return program, err
		}
		program.Composites = composites
	}
	if seenAssigns {
		assigns, err := decodeAssignments(assignData)
		if err != nil {
			return program, err
		}
		program.Assigns = assigns
	}
	if seenReturns {
		returns, err := decodeReturns(returnData)
		if err != nil {
			return program, err
		}
		program.Returns = returns
	}
	if seenCalls {
		calls, err := decodeCalls(callData)
		if err != nil {
			return program, err
		}
		program.Calls = calls
	}
	if seenRefs {
		refs, err := decodeRefs(refData)
		if err != nil {
			return program, err
		}
		program.Refs = refs
	}
	if seenSelectors {
		selectors, err := decodeSelectors(selectorData)
		if err != nil {
			return program, err
		}
		program.Selectors = selectors
	}
	if len(program.Tokens) == 0 {
		return program, fmt.Errorf("unit missing token table")
	}
	return program, nil
}

func Source(program Program) []byte {
	out := make([]byte, len(program.Text))
	copy(out, program.Text)
	return out
}

func ConvertFiles(paths []string) (Program, error) {
	var builder programBuilder
	if len(paths) == 0 {
		return Program{}, fmt.Errorf("no input files")
	}
	for _, path := range paths {
		if err := builder.addFile(path); err != nil {
			return Program{}, err
		}
	}
	return builder.finish(), nil
}

func WriteFile(path string, program Program) error {
	data, err := Marshal(program)
	if err != nil {
		return err
	}
	if path == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ReadFile(path string) (Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Program{}, err
	}
	return Unmarshal(data)
}

type programBuilder struct {
	pkg           string
	text          []byte
	tokens        []byte
	decls         []Decl
	funcs         []Func
	lineOffset    int
	writtenLine   int
	wroteOnLine   bool
	prevTokenText string
	tokenStart    []int
	tokenEnd      []int
	linkDirective map[int]string
}

func (b *programBuilder) addFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tokens, lineCount := scanSource(data)
	pkg, decls, funcs, err := parseTopLevel(tokens)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if b.pkg == "" {
		b.pkg = pkg
	} else if b.pkg != pkg {
		return fmt.Errorf("%s uses package %s, want %s", path, pkg, b.pkg)
	}

	oldToNew := make([]int, len(tokens))
	oldToNewEOF := len(b.tokens) / tokenStride
	if len(tokens) > 0 {
		oldToNewEOF += len(tokens) - 1
	}
	b.tokenStart = make([]int, len(tokens))
	b.tokenEnd = make([]int, len(tokens))
	b.linkDirective = findLinkStaticDirectives(data, funcs, tokens)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].kind == rtgTokEOF {
			oldToNew[i] = oldToNewEOF
			continue
		}
		oldToNew[i] = len(b.tokens) / tokenStride
		start, end := b.appendToken(tokens[i])
		b.tokenStart[i] = start
		b.tokenEnd[i] = end
	}
	for _, decl := range decls {
		nameTok := decl.nameTok
		b.decls = append(b.decls, Decl{
			Kind:      decl.kind,
			NameStart: b.tokenStart[nameTok],
			NameEnd:   b.tokenEnd[nameTok],
			StartTok:  mapTok(oldToNew, decl.startTok, oldToNewEOF),
			EndTok:    mapTok(oldToNew, decl.endTok, oldToNewEOF),
		})
	}
	for _, fn := range funcs {
		nameTok := fn.nameTok
		b.funcs = append(b.funcs, Func{
			NameStart:     b.tokenStart[nameTok],
			NameEnd:       b.tokenEnd[nameTok],
			StartTok:      mapTok(oldToNew, fn.startTok, oldToNewEOF),
			NameTok:       mapTok(oldToNew, fn.nameTok, oldToNewEOF),
			ReceiverStart: mapTok(oldToNew, fn.receiverStart, oldToNewEOF),
			ReceiverEnd:   mapTok(oldToNew, fn.receiverEnd, oldToNewEOF),
			BodyStart:     mapTok(oldToNew, fn.bodyStart, oldToNewEOF),
			BodyEnd:       mapTok(oldToNew, fn.bodyEnd, oldToNewEOF),
			EndTok:        mapTok(oldToNew, fn.endTok, oldToNewEOF),
		})
	}
	b.lineOffset += lineCount + 1
	return nil
}

func (b *programBuilder) finish() Program {
	line := b.lineOffset + 1
	if line <= 0 {
		line = 1
	}
	start := len(b.text)
	b.appendTokenData(rtgTokEOF, start, 0, line)
	return Program{Package: b.pkg, Text: b.text, Tokens: b.tokens, Decls: b.decls, Funcs: b.funcs}
}

func (b *programBuilder) appendToken(tok sourceToken) (int, int) {
	line := b.lineOffset + tok.line
	if directive, ok := b.linkDirective[tok.start]; ok {
		b.appendDirectiveBefore(line, directive)
	}
	for b.writtenLine < line {
		b.text = append(b.text, '\n')
		b.writtenLine++
		b.wroteOnLine = false
	}
	if b.wroteOnLine && unitNeedsSpace(b.prevTokenText, tok.text) {
		b.text = append(b.text, ' ')
	}
	start := len(b.text)
	b.text = append(b.text, tok.text...)
	end := len(b.text)
	b.appendTokenData(tok.kind, start, end-start, line)
	b.wroteOnLine = true
	b.prevTokenText = tok.text
	return start, end
}

func (b *programBuilder) appendDirectiveBefore(line int, directive string) {
	target := line - 1
	if target < 1 {
		target = 1
	}
	for b.writtenLine < target {
		b.text = append(b.text, '\n')
		b.writtenLine++
		b.wroteOnLine = false
	}
	if b.wroteOnLine {
		b.text = append(b.text, '\n')
		b.writtenLine++
		b.wroteOnLine = false
	}
	b.text = append(b.text, directive...)
	b.text = append(b.text, '\n')
	b.writtenLine++
	b.wroteOnLine = false
	b.prevTokenText = ""
}

func unitNeedsSpace(prev string, next string) bool {
	if len(prev) == 0 || len(next) == 0 {
		return false
	}
	a := prev[len(prev)-1]
	b := next[0]
	if isIdentPart(a) && isIdentPart(b) {
		return true
	}
	if isSpaceSensitiveOpPair(a, b) {
		return true
	}
	return false
}

func isSpaceSensitiveOpPair(a byte, b byte) bool {
	if b == '=' {
		return a == ':' || a == '=' || a == '!' || a == '<' || a == '>' || a == '+' || a == '-' || a == '*' || a == '/' || a == '%'
	}
	if a == '&' && (b == '&' || b == '^') {
		return true
	}
	if a == '|' && b == '|' {
		return true
	}
	if a == '<' && b == '<' {
		return true
	}
	if a == '>' && b == '>' {
		return true
	}
	if a == '+' && b == '+' {
		return true
	}
	if a == '-' && b == '-' {
		return true
	}
	return false
}

func (b *programBuilder) appendTokenData(kind int, start int, size int, line int) {
	var rec [tokenStride]byte
	rec[0] = byte(kind)
	rec[1] = byte(start)
	rec[2] = byte(start >> 8)
	rec[3] = byte(start >> 16)
	rec[4] = byte(size)
	if kind == rtgTokOp {
		rec[5] = b.text[start]
	} else {
		rec[5] = byte(size >> 8)
	}
	rec[6] = byte(line)
	rec[7] = byte(line >> 8)
	b.tokens = append(b.tokens, rec[:]...)
}

func mapTok(oldToNew []int, tok int, eof int) int {
	if tok < 0 || tok >= len(oldToNew) {
		return eof
	}
	return oldToNew[tok]
}

func findLinkStaticDirectives(data []byte, funcs []sourceFunc, tokens []sourceToken) map[int]string {
	out := make(map[int]string)
	for _, fn := range funcs {
		if fn.startTok < 0 || fn.startTok >= len(tokens) {
			continue
		}
		if directive, ok := linkStaticDirectiveBefore(data, tokens[fn.startTok].start); ok {
			out[tokens[fn.startTok].start] = directive
		}
	}
	return out
}

func linkStaticDirectiveBefore(data []byte, pos int) (string, bool) {
	if pos < 0 || pos > len(data) {
		return "", false
	}
	lineStart := pos
	for lineStart > 0 {
		prev := lineStart - 1
		if data[prev] == '\n' {
			break
		}
		lineStart--
	}
	end := lineStart
	for end > 0 {
		prev := end - 1
		c := data[prev]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		end--
	}
	if end <= 0 {
		return "", false
	}
	start := end
	for start > 0 {
		prev := start - 1
		if data[prev] == '\n' {
			break
		}
		start--
	}
	for start < end && (data[start] == ' ' || data[start] == '\t') {
		start++
	}
	prefix := []byte("// rtg:linkstatic ")
	if end-start < len(prefix) || !bytes.Equal(data[start:start+len(prefix)], prefix) {
		return "", false
	}
	return string(data[start:end]), true
}

func scanSource(src []byte) ([]sourceToken, int) {
	var toks []sourceToken
	i := 0
	line := 1
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if isIdentStart(c) {
			start := i
			i++
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			toks = append(toks, sourceToken{kind: keywordKind(src, start, i), text: string(src[start:i]), line: line, start: start, end: i})
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			kind := rtgTokNumber
			if c == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B') {
				hex := src[i+1] == 'x' || src[i+1] == 'X'
				i += 2
				for i < len(src) {
					cc := src[i]
					if cc == '.' && hex {
						kind = rtgTokFloat
						i++
						continue
					}
					if hex && (cc == 'p' || cc == 'P') {
						kind = rtgTokFloat
						i++
						if i < len(src) && (src[i] == '+' || src[i] == '-') {
							i++
						}
						for i < len(src) && ((src[i] >= '0' && src[i] <= '9') || src[i] == '_') {
							i++
						}
						break
					}
					if !(isIdentPart(cc)) {
						break
					}
					i++
				}
			} else {
				i++
				for i < len(src) && src[i] >= '0' && src[i] <= '9' {
					i++
				}
				if i < len(src) && src[i] == '.' {
					kind = rtgTokFloat
					i++
					for i < len(src) && src[i] >= '0' && src[i] <= '9' {
						i++
					}
				}
			}
			toks = append(toks, sourceToken{kind: kind, text: string(src[start:i]), line: line, start: start, end: i})
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < len(src) && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					if src[i] == '\n' {
						line++
					}
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, sourceToken{kind: rtgTokString, text: string(src[start:i]), line: line, start: start, end: i})
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < len(src) && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, sourceToken{kind: rtgTokChar, text: string(src[start:i]), line: line, start: start, end: i})
			continue
		}
		start := i
		i++
		if i < len(src) && isTwoByteOp(c, src[i]) {
			i++
		}
		toks = append(toks, sourceToken{kind: rtgTokOp, text: string(src[start:i]), line: line, start: start, end: i})
	}
	toks = append(toks, sourceToken{kind: rtgTokEOF, line: line, start: len(src), end: len(src)})
	return toks, line
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isTwoByteOp(c0 byte, c1 byte) bool {
	if c1 == '=' {
		return c0 == ':' || c0 == '=' || c0 == '!' || c0 == '<' || c0 == '>' || c0 == '+' || c0 == '-' || c0 == '*' || c0 == '/' || c0 == '%'
	}
	if c0 == '&' && (c1 == '&' || c1 == '^') {
		return true
	}
	if c0 == '|' && c1 == '|' {
		return true
	}
	if c0 == '<' && c1 == '<' {
		return true
	}
	if c0 == '>' && c1 == '>' {
		return true
	}
	if c0 == '+' && c1 == '+' {
		return true
	}
	if c0 == '-' && c1 == '-' {
		return true
	}
	return false
}

func keywordKind(src []byte, start int, end int) int {
	n := end - start
	if n > 8 {
		return rtgTokIdent
	}
	h := 0
	for i := start; i < end; i++ {
		h = h*5 + int(src[i])
	}
	if n == 2 && h == 627 {
		return rtgTokIf
	}
	if n == 3 {
		if h == 3549 {
			return rtgTokVar
		}
		if h == 3219 {
			return rtgTokFor
		}
	}
	if n == 4 {
		if h == 18186 {
			return rtgTokType
		}
		if h == 16324 {
			return rtgTokFunc
		}
		if h == 16001 {
			return rtgTokElse
		}
		if h == 16341 {
			return rtgTokGoto
		}
		if h == 15476 {
			return rtgTokCase
		}
	}
	if n == 5 {
		if h == 79191 {
			return rtgTokConst
		}
		if h == 78617 {
			return rtgTokBreak
		}
	}
	if n == 6 {
		if h == 449661 {
			return rtgTokStruct
		}
		if h == 437480 {
			return rtgTokReturn
		}
		if h == 450374 {
			return rtgTokSwitch
		}
	}
	if n == 7 {
		if h == 2131416 {
			return rtgTokPackage
		}
		if h == 1957581 {
			return rtgTokDefault
		}
	}
	if n == 8 && h == 9901561 {
		return rtgTokContinue
	}
	return rtgTokIdent
}

func parseTopLevel(toks []sourceToken) (string, []sourceDecl, []sourceFunc, error) {
	if len(toks) < 3 || toks[0].kind != rtgTokPackage || toks[1].kind != rtgTokIdent {
		return "", nil, nil, fmt.Errorf("missing package declaration")
	}
	pkg := toks[1].text
	var decls []sourceDecl
	var funcs []sourceFunc
	i := 0
	for i < len(toks) && toks[i].kind != rtgTokEOF {
		if toks[i].kind == rtgTokPackage {
			if i+1 >= len(toks) || toks[i+1].kind != rtgTokIdent {
				return "", nil, nil, fmt.Errorf("missing package name")
			}
			i += 2
			continue
		}
		if toks[i].kind == rtgTokConst || toks[i].kind == rtgTokVar || toks[i].kind == rtgTokType {
			start := i
			kind := toks[i].kind
			i++
			if tokCharIs(toks, i, '(') {
				end := skipBalanced(toks, i, '(', ')')
				if end <= i {
					return "", nil, nil, fmt.Errorf("invalid grouped declaration")
				}
				decls = append(decls, sourceDecl{kind: kind, nameTok: start, startTok: start, endTok: end})
				i = end
				continue
			}
			if i >= len(toks) || toks[i].kind != rtgTokIdent {
				return "", nil, nil, fmt.Errorf("invalid top-level declaration")
			}
			nameTok := i
			i++
			end := skipTopLevelLine(toks, i)
			decls = append(decls, sourceDecl{kind: kind, nameTok: nameTok, startTok: start, endTok: end})
			i = end
			continue
		}
		if toks[i].kind == rtgTokFunc {
			fn, ok := parseFuncDecl(toks, i)
			if !ok || fn.endTok <= i {
				return "", nil, nil, fmt.Errorf("invalid function declaration")
			}
			funcs = append(funcs, fn)
			i = fn.endTok
			continue
		}
		i++
	}
	return pkg, decls, funcs, nil
}

func parseFuncDecl(toks []sourceToken, start int) (sourceFunc, bool) {
	fn := sourceFunc{startTok: start}
	i := start + 1
	if i >= len(toks) {
		return fn, false
	}
	if toks[i].kind != rtgTokIdent {
		receiverEnd := i + 1
		for receiverEnd < len(toks) && !tokCharIs(toks, receiverEnd, ')') {
			receiverEnd++
		}
		if receiverEnd <= i {
			return fn, false
		}
		fn.receiverStart = i + 1
		fn.receiverEnd = receiverEnd
		i = receiverEnd + 1
	}
	if i >= len(toks) || toks[i].kind != rtgTokIdent {
		return fn, false
	}
	fn.nameTok = i
	i++
	for i < len(toks) && !tokCharIs(toks, i, '{') && toks[i].kind != rtgTokEOF {
		i++
	}
	if !tokCharIs(toks, i, '{') {
		return fn, false
	}
	fn.bodyStart = i
	depth := 1
	i++
	for i < len(toks) && depth > 0 {
		if tokCharIs(toks, i, '{') {
			depth++
		} else if tokCharIs(toks, i, '}') {
			depth--
		}
		i++
	}
	if depth != 0 {
		return fn, false
	}
	fn.bodyEnd = i - 1
	fn.endTok = i
	return fn, true
}

func skipBalanced(toks []sourceToken, start int, open byte, close byte) int {
	if !tokCharIs(toks, start, open) {
		return start
	}
	depth := 1
	i := start + 1
	for i < len(toks) && depth > 0 {
		if tokCharIs(toks, i, open) {
			depth++
		} else if tokCharIs(toks, i, close) {
			depth--
		}
		i++
	}
	if depth != 0 {
		return start
	}
	return i
}

func skipTopLevelLine(toks []sourceToken, start int) int {
	if start >= len(toks) {
		return start
	}
	line := toks[start-1].line
	i := start
	depth := 0
	for i < len(toks) {
		if toks[i].kind == rtgTokEOF {
			return i
		}
		if toks[i].line != line && depth == 0 {
			return i
		}
		if tokCharIs(toks, i, '{') || tokCharIs(toks, i, '(') {
			depth++
		} else if tokCharIs(toks, i, '}') || tokCharIs(toks, i, ')') {
			depth--
		}
		i++
	}
	return i
}

func tokCharIs(toks []sourceToken, i int, c byte) bool {
	if i < 0 || i >= len(toks) {
		return false
	}
	return toks[i].kind == rtgTokOp && len(toks[i].text) == 1 && toks[i].text[0] == c
}

func encodeTokens(text []byte, tokens []byte) ([]byte, error) {
	var out []byte
	out = appendVarint(out, len(tokens)/tokenStride)
	prevStart := 0
	prevLine := 0
	for pos := 0; pos < len(tokens); pos += tokenStride {
		kind := int(tokens[pos])
		start := readTokenStart(tokens, pos)
		size := int(tokens[pos+4])
		if kind != rtgTokOp {
			size = size | int(tokens[pos+5])<<8
		}
		line := int(tokens[pos+6]) | int(tokens[pos+7])<<8
		if start < prevStart {
			return nil, fmt.Errorf("token start moved backwards at token %d", pos/tokenStride)
		}
		if line < prevLine {
			return nil, fmt.Errorf("token line moved backwards at token %d", pos/tokenStride)
		}
		if start+size > len(text) {
			return nil, fmt.Errorf("token %d range %d:%d exceeds text size %d", pos/tokenStride, start, start+size, len(text))
		}
		out = appendVarint(out, kind)
		out = appendVarint(out, start-prevStart)
		out = appendVarint(out, size)
		out = appendVarint(out, line-prevLine)
		prevStart = start
		prevLine = line
	}
	return out, nil
}

func decodeTokens(text []byte, data []byte) ([]byte, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid token count")
	}
	pos = next
	if count < 0 {
		return nil, fmt.Errorf("invalid token count %d", count)
	}
	out := make([]byte, 0, count*tokenStride)
	start := 0
	line := 0
	for i := 0; i < count; i++ {
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid token %d kind", i)
		}
		pos = n
		startDelta, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid token %d start", i)
		}
		pos = n
		size, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid token %d size", i)
		}
		pos = n
		lineDelta, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid token %d line", i)
		}
		pos = n
		start += startDelta
		line += lineDelta
		if kind < 0 || kind > 255 || start < 0 || start > 0xffffff || size < 0 || line < 0 || line > 0xffff || start+size > len(text) {
			return nil, fmt.Errorf("invalid token %d", i)
		}
		if kind == rtgTokOp && size > 255 {
			return nil, fmt.Errorf("operator token %d too large", i)
		}
		if kind != rtgTokOp && size > 0xffff {
			return nil, fmt.Errorf("token %d too large", i)
		}
		var rec [tokenStride]byte
		rec[0] = byte(kind)
		rec[1] = byte(start)
		rec[2] = byte(start >> 8)
		rec[3] = byte(start >> 16)
		rec[4] = byte(size)
		if kind == rtgTokOp {
			if size > 0 {
				rec[5] = text[start]
			}
		} else {
			rec[5] = byte(size >> 8)
		}
		rec[6] = byte(line)
		rec[7] = byte(line >> 8)
		out = append(out, rec[:]...)
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing token data")
	}
	return out, nil
}

func readTokenStart(tokens []byte, pos int) int {
	return int(tokens[pos+1]) | int(tokens[pos+2])<<8 | int(tokens[pos+3])<<16
}

func encodeDecls(decls []Decl) []byte {
	var out []byte
	out = appendVarint(out, len(decls))
	for _, decl := range decls {
		out = appendVarint(out, decl.Kind)
		out = appendVarint(out, decl.NameStart)
		out = appendVarint(out, decl.NameEnd-decl.NameStart)
		out = appendVarint(out, decl.StartTok)
		out = appendVarint(out, decl.EndTok-decl.StartTok)
	}
	return out
}

func encodeImports(imports []Import) []byte {
	var out []byte
	out = appendVarint(out, len(imports))
	for _, imp := range imports {
		out = appendString(out, imp.Name)
		out = appendString(out, imp.ImportPath)
		out = appendNullable(out, imp.Package)
		out = appendNullable(out, imp.NameTok)
		out = appendVarint(out, imp.PathTok)
		flags := 0
		if imp.Dot {
			flags |= 1
		}
		if imp.Blank {
			flags |= 2
		}
		out = appendVarint(out, flags)
	}
	return out
}

func encodeSymbols(symbols []Symbol) []byte {
	var out []byte
	out = appendVarint(out, len(symbols))
	for _, symbol := range symbols {
		out = appendString(out, symbol.Name)
		out = appendVarint(out, symbol.Kind)
		out = appendNullable(out, symbol.Package)
		out = appendVarint(out, symbol.Token)
		out = appendVarint(out, symbol.OwnerKind)
		out = appendVarint(out, symbol.OwnerIndex)
	}
	return out
}

func encodeDeclMeta(metas []DeclMeta) []byte {
	var out []byte
	out = appendVarint(out, len(metas))
	for _, meta := range metas {
		out = appendVarint(out, meta.DeclIndex)
		out = appendNullable(out, meta.Symbol)
		out = appendVarint(out, meta.ValueIndex)
		if meta.Alias {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
		out = appendNullableSpan(out, meta.TypeStart, meta.TypeEnd)
		out = appendNullableSpan(out, meta.ValueStart, meta.ValueEnd)
		out = appendSpanList(out, meta.Values)
	}
	return out
}

func encodeInitOrder(order []int) []byte {
	var out []byte
	out = appendVarint(out, len(order))
	for _, decl := range order {
		out = appendVarint(out, decl)
	}
	return out
}

func encodeConsts(values []ConstValue) []byte {
	var out []byte
	out = appendVarint(out, len(values))
	for _, value := range values {
		out = appendVarint(out, value.DeclIndex)
		out = appendVarint(out, value.Kind)
		if value.Kind == ConstInt {
			out = appendSigned(out, value.Int)
		} else if value.Kind == ConstString {
			out = appendString(out, value.String)
		} else {
			if value.Bool {
				out = appendVarint(out, 1)
			} else {
				out = appendVarint(out, 0)
			}
		}
	}
	return out
}

func encodeFuncs(funcs []Func) []byte {
	var out []byte
	out = appendVarint(out, len(funcs))
	for _, fn := range funcs {
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

func encodeSignatures(signatures []FuncSignature) []byte {
	var out []byte
	out = appendVarint(out, len(signatures))
	for _, sig := range signatures {
		out = appendVarint(out, sig.FuncIndex)
		out = appendFieldList(out, sig.Receiver)
		out = appendFieldList(out, sig.Params)
		out = appendFieldList(out, sig.Results)
	}
	return out
}

func encodeTypes(types []TypeInfo) []byte {
	var out []byte
	out = appendVarint(out, len(types))
	for _, typ := range types {
		out = appendVarint(out, typ.Kind)
		out = appendVarint(out, typ.NameStart)
		out = appendVarint(out, typ.NameEnd-typ.NameStart)
		out = appendVarint(out, typ.Decl)
		out = appendNullable(out, typ.Symbol)
		if typ.Alias {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
		out = appendNullableSpan(out, typ.TypeStart, typ.TypeEnd)
		out = appendNullableSpan(out, typ.LenStart, typ.LenEnd)
		out = appendNullableSpan(out, typ.KeyStart, typ.KeyEnd)
		out = appendNullableSpan(out, typ.ElemStart, typ.ElemEnd)
	}
	return out
}

func encodeTypeFields(rows []TypeFields) []byte {
	var out []byte
	out = appendVarint(out, len(rows))
	for _, row := range rows {
		out = appendVarint(out, row.TypeIndex)
		out = appendFieldList(out, row.Fields)
	}
	return out
}

func encodeTypeInterfaces(rows []TypeIface) []byte {
	var out []byte
	out = appendVarint(out, len(rows))
	for _, row := range rows {
		out = appendVarint(out, row.TypeIndex)
		out = appendVarint(out, len(row.Embeds))
		for _, embed := range row.Embeds {
			out = appendVarint(out, embed.TypeStart)
			out = appendVarint(out, embed.TypeEnd-embed.TypeStart)
		}
		out = appendVarint(out, len(row.Methods))
		for _, method := range row.Methods {
			out = appendVarint(out, method.NameTok)
			out = appendFieldList(out, method.Params)
			out = appendFieldList(out, method.Results)
		}
	}
	return out
}

func encodeMethods(methods []MethodInfo) []byte {
	var out []byte
	out = appendVarint(out, len(methods))
	for _, method := range methods {
		out = appendVarint(out, method.NameTok)
		out = appendVarint(out, method.TypeIndex)
		out = appendVarint(out, method.Symbol)
		out = appendVarint(out, method.FuncIndex)
		if method.Pointer {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
	}
	return out
}

func encodeTypeRefs(refs []TypeRef) []byte {
	var out []byte
	out = appendVarint(out, len(refs))
	for _, ref := range refs {
		out = appendVarint(out, ref.OwnerKind)
		out = appendVarint(out, ref.OwnerIndex)
		out = appendVarint(out, ref.Kind)
		out = appendVarint(out, ref.Token)
		out = appendVarint(out, ref.BaseTok)
		out = appendVarint(out, ref.DotTok)
		out = appendNullable(out, ref.Package)
		out = appendNullable(out, ref.Symbol)
	}
	return out
}

func encodeLocals(locals []LocalDecl) []byte {
	var out []byte
	out = appendVarint(out, len(locals))
	for _, local := range locals {
		out = appendVarint(out, local.FuncIndex)
		out = appendVarint(out, local.Kind)
		out = appendVarint(out, local.NameStart)
		out = appendVarint(out, local.NameEnd-local.NameStart)
		out = appendVarint(out, local.Token)
		out = appendNullable(out, local.Scope)
		out = appendVarint(out, local.ValueIndex)
		if local.Alias {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
		out = appendNullableSpan(out, local.TypeStart, local.TypeEnd)
		out = appendNullableSpan(out, local.ValueStart, local.ValueEnd)
		out = appendSpanList(out, local.Values)
	}
	return out
}

func encodeIndexes(indexes []IndexExpr) []byte {
	var out []byte
	out = appendVarint(out, len(indexes))
	for _, index := range indexes {
		out = appendVarint(out, index.OwnerKind)
		out = appendVarint(out, index.OwnerIndex)
		out = appendVarint(out, index.StartTok)
		out = appendVarint(out, index.EndTok-index.StartTok)
		out = appendVarint(out, index.BaseStart)
		out = appendVarint(out, index.BaseEnd-index.BaseStart)
		out = appendVarint(out, index.OpenTok)
		out = appendVarint(out, index.CloseTok)
		out = appendVarint(out, index.IndexStart)
		out = appendVarint(out, index.IndexEnd-index.IndexStart)
	}
	return out
}

func encodeComposites(composites []CompositeExpr) []byte {
	var out []byte
	out = appendVarint(out, len(composites))
	for _, composite := range composites {
		out = appendVarint(out, composite.OwnerKind)
		out = appendVarint(out, composite.OwnerIndex)
		out = appendVarint(out, composite.StartTok)
		out = appendVarint(out, composite.EndTok-composite.StartTok)
		out = appendVarint(out, composite.TypeStart)
		out = appendVarint(out, composite.TypeEnd-composite.TypeStart)
		out = appendVarint(out, composite.OpenTok)
		out = appendVarint(out, composite.CloseTok)
		out = appendVarint(out, len(composite.Elems))
		for _, elem := range composite.Elems {
			out = appendVarint(out, elem.StartTok)
			out = appendVarint(out, elem.EndTok-elem.StartTok)
		}
	}
	return out
}

func encodeAssignments(assigns []Assignment) []byte {
	var out []byte
	out = appendVarint(out, len(assigns))
	for _, assign := range assigns {
		out = appendVarint(out, assign.FuncIndex)
		out = appendVarint(out, assign.Kind)
		out = appendVarint(out, assign.StartTok)
		out = appendVarint(out, assign.EndTok-assign.StartTok)
		out = appendVarint(out, assign.OpTok)
		out = appendVarint(out, assign.LeftStart)
		out = appendVarint(out, assign.LeftEnd-assign.LeftStart)
		out = appendVarint(out, assign.RightStart)
		out = appendVarint(out, assign.RightEnd-assign.RightStart)
		out = appendSpanList(out, assign.Targets)
		out = appendSpanList(out, assign.Values)
	}
	return out
}

func encodeReturns(returns []Return) []byte {
	var out []byte
	out = appendVarint(out, len(returns))
	for _, ret := range returns {
		out = appendVarint(out, ret.FuncIndex)
		out = appendVarint(out, ret.StartTok)
		out = appendVarint(out, ret.EndTok-ret.StartTok)
		out = appendSpanList(out, ret.Values)
	}
	return out
}

func encodeCalls(calls []Call) []byte {
	var out []byte
	out = appendVarint(out, len(calls))
	for _, call := range calls {
		out = appendVarint(out, call.OwnerKind)
		out = appendVarint(out, call.OwnerIndex)
		out = appendVarint(out, call.Kind)
		out = appendVarint(out, call.CalleeTok)
		out = appendVarint(out, call.BaseTok)
		out = appendVarint(out, call.DotTok)
		out = appendVarint(out, call.ArgsStart)
		out = appendVarint(out, call.ArgsEnd-call.ArgsStart)
		out = appendSpanList(out, call.Args)
	}
	return out
}

func encodeRefs(refs []NameRef) []byte {
	var out []byte
	out = appendVarint(out, len(refs))
	for _, ref := range refs {
		out = appendVarint(out, ref.OwnerKind)
		out = appendVarint(out, ref.OwnerIndex)
		out = appendVarint(out, ref.Kind)
		out = appendVarint(out, ref.Token)
		out = appendNullable(out, ref.Index)
		out = appendNullable(out, ref.Package)
	}
	return out
}

func encodeSelectors(selectors []Selector) []byte {
	var out []byte
	out = appendVarint(out, len(selectors))
	for _, selector := range selectors {
		out = appendVarint(out, selector.OwnerKind)
		out = appendVarint(out, selector.OwnerIndex)
		out = appendVarint(out, selector.Kind)
		out = appendVarint(out, selector.BaseTok)
		out = appendVarint(out, selector.DotTok)
		out = appendVarint(out, selector.NameTok)
		out = appendVarint(out, selector.BaseKind)
		out = appendNullable(out, selector.BaseIndex)
		out = appendNullable(out, selector.BasePackage)
		out = appendNullable(out, selector.Package)
		out = appendNullable(out, selector.Symbol)
	}
	return out
}

func appendSpanList(out []byte, spans []ExprSpan) []byte {
	out = appendVarint(out, len(spans))
	for _, span := range spans {
		out = appendVarint(out, span.StartTok)
		out = appendVarint(out, span.EndTok-span.StartTok)
	}
	return out
}

func appendFieldList(out []byte, fields []Field) []byte {
	out = appendVarint(out, len(fields))
	for _, field := range fields {
		out = appendNullable(out, field.NameTok)
		out = appendVarint(out, field.TypeStart)
		out = appendVarint(out, field.TypeEnd-field.TypeStart)
		if field.Variadic {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
	}
	return out
}

func appendNullableSpan(out []byte, start int, end int) []byte {
	if start < 0 && end < 0 {
		return appendVarint(out, 0)
	}
	out = appendVarint(out, start+1)
	out = appendVarint(out, end-start)
	return out
}

func decodeDecls(data []byte) ([]Decl, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid decl count")
	}
	pos = next
	decls := make([]Decl, 0, count)
	for i := 0; i < count; i++ {
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl %d kind", i)
		}
		pos = n
		nameStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl %d name", i)
		}
		pos = n
		nameSize, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl %d name size", i)
		}
		pos = n
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl %d start", i)
		}
		pos = n
		tokCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl %d end", i)
		}
		pos = n
		decls = append(decls, Decl{Kind: kind, NameStart: nameStart, NameEnd: nameStart + nameSize, StartTok: startTok, EndTok: startTok + tokCount})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing decl data")
	}
	return decls, nil
}

func decodeImports(data []byte) ([]Import, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid import count")
	}
	pos = next
	imports := make([]Import, 0, count)
	for i := 0; i < count; i++ {
		name, n, ok := readString(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid import %d name", i)
		}
		pos = n
		importPath, n, ok := readString(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid import %d path", i)
		}
		pos = n
		pkg, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid import %d package", i)
		}
		pos = n
		nameTok, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid import %d name token", i)
		}
		pos = n
		pathTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid import %d path token", i)
		}
		pos = n
		flags, n, ok := readVarint(data, pos)
		if !ok || flags > 3 {
			return nil, fmt.Errorf("invalid import %d flags", i)
		}
		pos = n
		imp := Import{
			Name:       name,
			ImportPath: importPath,
			Package:    pkg,
			NameTok:    nameTok,
			PathTok:    pathTok,
			Dot:        flags&1 != 0,
			Blank:      flags&2 != 0,
		}
		if imp.Dot && imp.Blank {
			return nil, fmt.Errorf("invalid import %d dot blank flags", i)
		}
		imports = append(imports, imp)
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing import data")
	}
	return imports, nil
}

func decodeSymbols(data []byte) ([]Symbol, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid symbol count")
	}
	pos = next
	symbols := make([]Symbol, 0, count)
	for i := 0; i < count; i++ {
		name, n, ok := readString(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid symbol %d name", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid symbol %d kind", i)
		}
		pos = n
		pkg, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid symbol %d package", i)
		}
		pos = n
		token, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid symbol %d token", i)
		}
		pos = n
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid symbol %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid symbol %d owner index", i)
		}
		pos = n
		symbols = append(symbols, Symbol{
			Name:       name,
			Kind:       kind,
			Package:    pkg,
			Token:      token,
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing symbol data")
	}
	return symbols, nil
}

func decodeDeclMeta(data []byte) ([]DeclMeta, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid decl metadata count")
	}
	pos = next
	metas := make([]DeclMeta, 0, count)
	for i := 0; i < count; i++ {
		declIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl metadata %d decl", i)
		}
		pos = n
		symbol, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl metadata %d symbol", i)
		}
		pos = n
		valueIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl metadata %d value index", i)
		}
		pos = n
		aliasValue, n, ok := readVarint(data, pos)
		if !ok || aliasValue > 1 {
			return nil, fmt.Errorf("invalid decl metadata %d alias", i)
		}
		pos = n
		typeStart, typeEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl metadata %d type span", i)
		}
		pos = n
		valueStart, valueEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl metadata %d value span", i)
		}
		pos = n
		values, n, ok := readSpanList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid decl metadata %d values", i)
		}
		pos = n
		metas = append(metas, DeclMeta{
			DeclIndex:  declIndex,
			Symbol:     symbol,
			ValueIndex: valueIndex,
			TypeStart:  typeStart,
			TypeEnd:    typeEnd,
			ValueStart: valueStart,
			ValueEnd:   valueEnd,
			Values:     values,
			Alias:      aliasValue == 1,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing decl metadata")
	}
	return metas, nil
}

func decodeInitOrder(data []byte) ([]int, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid init order count")
	}
	pos = next
	order := make([]int, 0, count)
	for i := 0; i < count; i++ {
		decl, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid init order %d", i)
		}
		pos = n
		order = append(order, decl)
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing init order data")
	}
	return order, nil
}

func decodeConsts(data []byte) ([]ConstValue, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid const count")
	}
	pos = next
	values := make([]ConstValue, 0, count)
	for i := 0; i < count; i++ {
		decl, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid const %d decl", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid const %d kind", i)
		}
		pos = n
		value := ConstValue{DeclIndex: decl, Kind: kind}
		if value.Kind == ConstInt {
			value.Int, n, ok = readSigned(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid const %d int", i)
			}
			pos = n
		} else if value.Kind == ConstString {
			value.String, n, ok = readString(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid const %d string", i)
			}
			pos = n
		} else if value.Kind == ConstBool {
			boolValue, n, ok := readVarint(data, pos)
			if !ok || boolValue > 1 {
				return nil, fmt.Errorf("invalid const %d bool", i)
			}
			pos = n
			value.Bool = boolValue == 1
		} else {
			return nil, fmt.Errorf("invalid const %d kind %d", i, value.Kind)
		}
		values = append(values, value)
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing const data")
	}
	return values, nil
}

func decodeFuncs(data []byte) ([]Func, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid func count")
	}
	pos = next
	funcs := make([]Func, 0, count)
	for i := 0; i < count; i++ {
		nameStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d name", i)
		}
		pos = n
		nameSize, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d name size", i)
		}
		pos = n
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d start", i)
		}
		pos = n
		nameTokDelta, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d name token", i)
		}
		pos = n
		receiverStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d receiver", i)
		}
		pos = n
		receiverCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d receiver end", i)
		}
		pos = n
		bodyStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d body", i)
		}
		pos = n
		bodyCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d body end", i)
		}
		pos = n
		endCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid func %d end", i)
		}
		pos = n
		funcs = append(funcs, Func{
			NameStart:     nameStart,
			NameEnd:       nameStart + nameSize,
			StartTok:      startTok,
			NameTok:       startTok + nameTokDelta,
			ReceiverStart: receiverStart,
			ReceiverEnd:   receiverStart + receiverCount,
			BodyStart:     bodyStart,
			BodyEnd:       bodyStart + bodyCount,
			EndTok:        bodyStart + bodyCount + endCount,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing func data")
	}
	return funcs, nil
}

func decodeSignatures(data []byte) ([]FuncSignature, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid signature count")
	}
	pos = next
	signatures := make([]FuncSignature, 0, count)
	for i := 0; i < count; i++ {
		funcIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid signature %d func", i)
		}
		pos = n
		receiver, n, ok := readFieldList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid signature %d receiver", i)
		}
		pos = n
		params, n, ok := readFieldList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid signature %d params", i)
		}
		pos = n
		results, n, ok := readFieldList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid signature %d results", i)
		}
		pos = n
		signatures = append(signatures, FuncSignature{
			FuncIndex: funcIndex,
			Receiver:  receiver,
			Params:    params,
			Results:   results,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing signature data")
	}
	return signatures, nil
}

func decodeTypes(data []byte) ([]TypeInfo, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid type count")
	}
	pos = next
	types := make([]TypeInfo, 0, count)
	for i := 0; i < count; i++ {
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d kind", i)
		}
		pos = n
		nameStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d name", i)
		}
		pos = n
		nameSize, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d name size", i)
		}
		pos = n
		decl, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d decl", i)
		}
		pos = n
		symbol, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d symbol", i)
		}
		pos = n
		aliasValue, n, ok := readVarint(data, pos)
		if !ok || aliasValue > 1 {
			return nil, fmt.Errorf("invalid type %d alias", i)
		}
		pos = n
		typeStart, typeEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d type span", i)
		}
		pos = n
		lenStart, lenEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d length span", i)
		}
		pos = n
		keyStart, keyEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d key span", i)
		}
		pos = n
		elemStart, elemEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type %d element span", i)
		}
		pos = n
		types = append(types, TypeInfo{
			NameStart: nameStart,
			NameEnd:   nameStart + nameSize,
			Kind:      kind,
			Decl:      decl,
			Symbol:    symbol,
			Alias:     aliasValue == 1,
			TypeStart: typeStart,
			TypeEnd:   typeEnd,
			LenStart:  lenStart,
			LenEnd:    lenEnd,
			KeyStart:  keyStart,
			KeyEnd:    keyEnd,
			ElemStart: elemStart,
			ElemEnd:   elemEnd,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing type data")
	}
	return types, nil
}

func decodeTypeFields(data []byte) ([]TypeFields, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid type field count")
	}
	pos = next
	rows := make([]TypeFields, 0, count)
	for i := 0; i < count; i++ {
		typeIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type field %d type", i)
		}
		pos = n
		fields, n, ok := readFieldList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type field %d fields", i)
		}
		pos = n
		rows = append(rows, TypeFields{TypeIndex: typeIndex, Fields: fields})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing type field data")
	}
	return rows, nil
}

func decodeTypeInterfaces(data []byte) ([]TypeIface, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid type interface count")
	}
	pos = next
	rows := make([]TypeIface, 0, count)
	for i := 0; i < count; i++ {
		typeIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type interface %d type", i)
		}
		pos = n
		embedCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type interface %d embed count", i)
		}
		pos = n
		embeds := make([]InterfaceEmbed, 0, embedCount)
		for j := 0; j < embedCount; j++ {
			typeStart, n, ok := readVarint(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid type interface %d embed %d start", i, j)
			}
			pos = n
			typeCount, n, ok := readVarint(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid type interface %d embed %d size", i, j)
			}
			pos = n
			embeds = append(embeds, InterfaceEmbed{TypeStart: typeStart, TypeEnd: typeStart + typeCount})
		}
		methodCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type interface %d method count", i)
		}
		pos = n
		methods := make([]InterfaceMethod, 0, methodCount)
		for j := 0; j < methodCount; j++ {
			nameTok, n, ok := readVarint(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid type interface %d method %d name", i, j)
			}
			pos = n
			params, n, ok := readFieldList(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid type interface %d method %d params", i, j)
			}
			pos = n
			results, n, ok := readFieldList(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid type interface %d method %d results", i, j)
			}
			pos = n
			methods = append(methods, InterfaceMethod{NameTok: nameTok, Params: params, Results: results})
		}
		rows = append(rows, TypeIface{TypeIndex: typeIndex, Methods: methods, Embeds: embeds})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing type interface data")
	}
	return rows, nil
}

func decodeMethods(data []byte) ([]MethodInfo, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid method count")
	}
	pos = next
	methods := make([]MethodInfo, 0, count)
	for i := 0; i < count; i++ {
		nameTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid method %d name", i)
		}
		pos = n
		typeIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid method %d type", i)
		}
		pos = n
		symbol, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid method %d symbol", i)
		}
		pos = n
		funcIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid method %d func", i)
		}
		pos = n
		pointerValue, n, ok := readVarint(data, pos)
		if !ok || pointerValue > 1 {
			return nil, fmt.Errorf("invalid method %d pointer", i)
		}
		pos = n
		methods = append(methods, MethodInfo{
			NameTok:   nameTok,
			TypeIndex: typeIndex,
			Symbol:    symbol,
			FuncIndex: funcIndex,
			Pointer:   pointerValue == 1,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing method data")
	}
	return methods, nil
}

func decodeTypeRefs(data []byte) ([]TypeRef, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid type ref count")
	}
	pos = next
	refs := make([]TypeRef, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d owner index", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d kind", i)
		}
		pos = n
		token, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d token", i)
		}
		pos = n
		baseTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d base", i)
		}
		pos = n
		dotTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d dot", i)
		}
		pos = n
		pkg, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d package", i)
		}
		pos = n
		symbol, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid type ref %d symbol", i)
		}
		pos = n
		refs = append(refs, TypeRef{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			Kind:       kind,
			Token:      token,
			BaseTok:    baseTok,
			DotTok:     dotTok,
			Package:    pkg,
			Symbol:     symbol,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing type ref data")
	}
	return refs, nil
}

func decodeLocals(data []byte) ([]LocalDecl, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid local count")
	}
	pos = next
	locals := make([]LocalDecl, 0, count)
	for i := 0; i < count; i++ {
		funcIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d func", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d kind", i)
		}
		pos = n
		nameStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d name", i)
		}
		pos = n
		nameSize, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d name size", i)
		}
		pos = n
		token, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d token", i)
		}
		pos = n
		scope, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d scope", i)
		}
		pos = n
		valueIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d value index", i)
		}
		pos = n
		aliasValue, n, ok := readVarint(data, pos)
		if !ok || aliasValue > 1 {
			return nil, fmt.Errorf("invalid local %d alias", i)
		}
		pos = n
		typeStart, typeEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d type span", i)
		}
		pos = n
		valueStart, valueEnd, n, ok := readNullableSpan(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d value span", i)
		}
		pos = n
		values, n, ok := readSpanList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid local %d values", i)
		}
		pos = n
		locals = append(locals, LocalDecl{
			FuncIndex:  funcIndex,
			Kind:       kind,
			NameStart:  nameStart,
			NameEnd:    nameStart + nameSize,
			Token:      token,
			Scope:      scope,
			ValueIndex: valueIndex,
			TypeStart:  typeStart,
			TypeEnd:    typeEnd,
			ValueStart: valueStart,
			ValueEnd:   valueEnd,
			Values:     values,
			Alias:      aliasValue == 1,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing local data")
	}
	return locals, nil
}

func decodeIndexes(data []byte) ([]IndexExpr, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid index count")
	}
	pos = next
	indexes := make([]IndexExpr, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d owner index", i)
		}
		pos = n
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d start", i)
		}
		pos = n
		tokCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d end", i)
		}
		pos = n
		baseStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d base start", i)
		}
		pos = n
		baseCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d base end", i)
		}
		pos = n
		openTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d open", i)
		}
		pos = n
		closeTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d close", i)
		}
		pos = n
		indexStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d expr start", i)
		}
		pos = n
		indexCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid index %d expr end", i)
		}
		pos = n
		indexes = append(indexes, IndexExpr{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			StartTok:   startTok,
			EndTok:     startTok + tokCount,
			BaseStart:  baseStart,
			BaseEnd:    baseStart + baseCount,
			OpenTok:    openTok,
			CloseTok:   closeTok,
			IndexStart: indexStart,
			IndexEnd:   indexStart + indexCount,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing index data")
	}
	return indexes, nil
}

func decodeComposites(data []byte) ([]CompositeExpr, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid composite count")
	}
	pos = next
	composites := make([]CompositeExpr, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d owner index", i)
		}
		pos = n
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d start", i)
		}
		pos = n
		tokCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d end", i)
		}
		pos = n
		typeStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d type start", i)
		}
		pos = n
		typeCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d type end", i)
		}
		pos = n
		openTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d open", i)
		}
		pos = n
		closeTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d close", i)
		}
		pos = n
		elemCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid composite %d elem count", i)
		}
		pos = n
		composite := CompositeExpr{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			StartTok:   startTok,
			EndTok:     startTok + tokCount,
			TypeStart:  typeStart,
			TypeEnd:    typeStart + typeCount,
			OpenTok:    openTok,
			CloseTok:   closeTok,
			Elems:      make([]ExprSpan, 0, elemCount),
		}
		for j := 0; j < elemCount; j++ {
			elemStart, n, ok := readVarint(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid composite %d elem %d start", i, j)
			}
			pos = n
			elemCount, n, ok := readVarint(data, pos)
			if !ok {
				return nil, fmt.Errorf("invalid composite %d elem %d end", i, j)
			}
			pos = n
			composite.Elems = append(composite.Elems, ExprSpan{StartTok: elemStart, EndTok: elemStart + elemCount})
		}
		composites = append(composites, composite)
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing composite data")
	}
	return composites, nil
}

func decodeAssignments(data []byte) ([]Assignment, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid assignment count")
	}
	pos = next
	assigns := make([]Assignment, 0, count)
	for i := 0; i < count; i++ {
		funcIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d func", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d kind", i)
		}
		pos = n
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d start", i)
		}
		pos = n
		tokCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d end", i)
		}
		pos = n
		opTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d op", i)
		}
		pos = n
		leftStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d left start", i)
		}
		pos = n
		leftCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d left end", i)
		}
		pos = n
		rightStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d right start", i)
		}
		pos = n
		rightCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d right end", i)
		}
		pos = n
		targets, n, ok := readSpanList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d targets", i)
		}
		pos = n
		values, n, ok := readSpanList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid assignment %d values", i)
		}
		pos = n
		assigns = append(assigns, Assignment{
			FuncIndex:  funcIndex,
			Kind:       kind,
			StartTok:   startTok,
			EndTok:     startTok + tokCount,
			OpTok:      opTok,
			LeftStart:  leftStart,
			LeftEnd:    leftStart + leftCount,
			RightStart: rightStart,
			RightEnd:   rightStart + rightCount,
			Targets:    targets,
			Values:     values,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing assignment data")
	}
	return assigns, nil
}

func decodeReturns(data []byte) ([]Return, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid return count")
	}
	pos = next
	returns := make([]Return, 0, count)
	for i := 0; i < count; i++ {
		funcIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid return %d func", i)
		}
		pos = n
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid return %d start", i)
		}
		pos = n
		tokCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid return %d end", i)
		}
		pos = n
		values, n, ok := readSpanList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid return %d values", i)
		}
		pos = n
		returns = append(returns, Return{
			FuncIndex: funcIndex,
			StartTok:  startTok,
			EndTok:    startTok + tokCount,
			Values:    values,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing return data")
	}
	return returns, nil
}

func decodeCalls(data []byte) ([]Call, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid call count")
	}
	pos = next
	calls := make([]Call, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d owner index", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d kind", i)
		}
		pos = n
		calleeTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d callee", i)
		}
		pos = n
		baseTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d base", i)
		}
		pos = n
		dotTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d dot", i)
		}
		pos = n
		argsStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d args start", i)
		}
		pos = n
		argsCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d args end", i)
		}
		pos = n
		args, n, ok := readSpanList(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid call %d args", i)
		}
		pos = n
		calls = append(calls, Call{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			Kind:       kind,
			CalleeTok:  calleeTok,
			BaseTok:    baseTok,
			DotTok:     dotTok,
			ArgsStart:  argsStart,
			ArgsEnd:    argsStart + argsCount,
			Args:       args,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing call data")
	}
	return calls, nil
}

func decodeRefs(data []byte) ([]NameRef, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid ref count")
	}
	pos = next
	refs := make([]NameRef, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid ref %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid ref %d owner index", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid ref %d kind", i)
		}
		pos = n
		token, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid ref %d token", i)
		}
		pos = n
		index, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid ref %d index", i)
		}
		pos = n
		pkg, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid ref %d package", i)
		}
		pos = n
		refs = append(refs, NameRef{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			Kind:       kind,
			Token:      token,
			Index:      index,
			Package:    pkg,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing ref data")
	}
	return refs, nil
}

func decodeSelectors(data []byte) ([]Selector, error) {
	pos := 0
	count, next, ok := readVarint(data, pos)
	if !ok {
		return nil, fmt.Errorf("invalid selector count")
	}
	pos = next
	selectors := make([]Selector, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d owner kind", i)
		}
		pos = n
		ownerIndex, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d owner index", i)
		}
		pos = n
		kind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d kind", i)
		}
		pos = n
		baseTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d base", i)
		}
		pos = n
		dotTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d dot", i)
		}
		pos = n
		nameTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d name", i)
		}
		pos = n
		baseKind, n, ok := readVarint(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d base kind", i)
		}
		pos = n
		baseIndex, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d base index", i)
		}
		pos = n
		basePackage, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d base package", i)
		}
		pos = n
		pkg, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d package", i)
		}
		pos = n
		symbol, n, ok := readNullable(data, pos)
		if !ok {
			return nil, fmt.Errorf("invalid selector %d symbol", i)
		}
		pos = n
		selectors = append(selectors, Selector{
			OwnerKind:   ownerKind,
			OwnerIndex:  ownerIndex,
			Kind:        kind,
			BaseTok:     baseTok,
			DotTok:      dotTok,
			NameTok:     nameTok,
			BaseKind:    baseKind,
			BaseIndex:   baseIndex,
			BasePackage: basePackage,
			Package:     pkg,
			Symbol:      symbol,
		})
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing selector data")
	}
	return selectors, nil
}

func readSpanList(data []byte, pos int) ([]ExprSpan, int, bool) {
	count, n, ok := readVarint(data, pos)
	if !ok {
		return nil, pos, false
	}
	pos = n
	spans := make([]ExprSpan, 0, count)
	for i := 0; i < count; i++ {
		startTok, n, ok := readVarint(data, pos)
		if !ok {
			return nil, pos, false
		}
		pos = n
		tokCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, pos, false
		}
		pos = n
		spans = append(spans, ExprSpan{StartTok: startTok, EndTok: startTok + tokCount})
	}
	return spans, pos, true
}

func readFieldList(data []byte, pos int) ([]Field, int, bool) {
	count, n, ok := readVarint(data, pos)
	if !ok {
		return nil, pos, false
	}
	pos = n
	fields := make([]Field, 0, count)
	for i := 0; i < count; i++ {
		nameTok, n, ok := readNullable(data, pos)
		if !ok {
			return nil, pos, false
		}
		pos = n
		typeStart, n, ok := readVarint(data, pos)
		if !ok {
			return nil, pos, false
		}
		pos = n
		typeCount, n, ok := readVarint(data, pos)
		if !ok {
			return nil, pos, false
		}
		pos = n
		variadicValue, n, ok := readVarint(data, pos)
		if !ok || variadicValue > 1 {
			return nil, pos, false
		}
		pos = n
		fields = append(fields, Field{
			NameTok:   nameTok,
			TypeStart: typeStart,
			TypeEnd:   typeStart + typeCount,
			Variadic:  variadicValue == 1,
		})
	}
	return fields, pos, true
}

func readNullableSpan(data []byte, pos int) (int, int, int, bool) {
	encodedStart, next, ok := readVarint(data, pos)
	if !ok {
		return 0, 0, pos, false
	}
	pos = next
	if encodedStart == 0 {
		return -1, -1, pos, true
	}
	count, next, ok := readVarint(data, pos)
	if !ok {
		return 0, 0, pos, false
	}
	start := encodedStart - 1
	return start, start + count, next, true
}

func appendNullable(out []byte, v int) []byte {
	return appendVarint(out, v+1)
}

func readNullable(data []byte, pos int) (int, int, bool) {
	value, next, ok := readVarint(data, pos)
	if !ok {
		return 0, pos, false
	}
	return value - 1, next, true
}

func appendString(out []byte, text string) []byte {
	out = appendVarint(out, len(text))
	return append(out, text...)
}

func readString(data []byte, pos int) (string, int, bool) {
	size, next, ok := readVarint(data, pos)
	if !ok {
		return "", pos, false
	}
	pos = next
	end := pos + size
	if end < pos || end > len(data) {
		return "", pos, false
	}
	return string(data[pos:end]), end, true
}

func appendSigned(out []byte, v int) []byte {
	if v < 0 {
		return appendVarint(out, -v*2-1)
	}
	return appendVarint(out, v*2)
}

func readSigned(data []byte, pos int) (int, int, bool) {
	value, next, ok := readVarint(data, pos)
	if !ok {
		return 0, pos, false
	}
	if value&1 == 1 {
		return -(value / 2) - 1, next, true
	}
	return value / 2, next, true
}

func appendVarint(out []byte, v int) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v = v >> 7
	}
	return append(out, byte(v))
}

func readVarint(data []byte, pos int) (int, int, bool) {
	value := 0
	shift := 0
	for pos < len(data) && shift <= 28 {
		b := data[pos]
		pos++
		value = value | int(b&0x7f)<<shift
		if b < 0x80 {
			return value, pos, true
		}
		shift += 7
	}
	return 0, pos, false
}

func writeNode(out *bytes.Buffer, node Node) error {
	payload := node.Data
	if len(node.Children) > 0 {
		var nested bytes.Buffer
		for _, child := range node.Children {
			if err := writeNode(&nested, child); err != nil {
				return err
			}
		}
		payload = nested.Bytes()
	}
	if len(payload) > int(^uint32(0)) {
		return fmt.Errorf("node %d payload too large", node.Tag)
	}
	var header [6]byte
	binary.LittleEndian.PutUint16(header[0:2], node.Tag)
	binary.LittleEndian.PutUint32(header[2:6], uint32(len(payload)))
	out.Write(header[:])
	out.Write(payload)
	return nil
}

func readNode(data []byte, pos int) (Node, int, error) {
	var node Node
	if pos+6 > len(data) {
		return node, pos, fmt.Errorf("truncated node header")
	}
	node.Tag = binary.LittleEndian.Uint16(data[pos : pos+2])
	length := int(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))
	pos += 6
	end := pos + length
	if length < 0 || end < pos || end > len(data) {
		return node, pos, fmt.Errorf("invalid node length")
	}
	payload := data[pos:end]
	if node.Tag == TagUnit {
		childPos := 0
		for childPos < len(payload) {
			child, next, err := readNode(payload, childPos)
			if err != nil {
				return node, pos, err
			}
			node.Children = append(node.Children, child)
			childPos = next
		}
	} else {
		node.Data = append(node.Data[:0], payload...)
	}
	return node, end, nil
}
