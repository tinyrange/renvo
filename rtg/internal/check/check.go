package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

const (
	CheckOK = iota
	CheckErrGraph
	CheckErrDuplicate
	CheckErrImport
	CheckErrMethod
	CheckErrBody
	CheckErrScope
	CheckErrReturnCount
	CheckErrType
	CheckErrExcluded
	CheckErrUnusedImport
	CheckErrCall
	CheckErrAssignTarget
	CheckErrAssignCount
	CheckErrBreak
	CheckErrContinue
	CheckErrCallArgument
	CheckErrGoroutine
	CheckErrChannel
	CheckErrSelect
	CheckErrUnusedLocal
	CheckErrMissingMain
	CheckErrMainSignature
	CheckErrMainMethod
	CheckErrSliceOperand
)

const (
	SymbolConst = iota + 1
	SymbolVar
	SymbolType
	SymbolFunc
	SymbolMethod
)

const (
	ConstInvalid = iota
	ConstInt
	ConstString
	ConstBool
)

type Program struct {
	Graph        load.Graph
	Packages     []PackageInfo
	Ok           bool
	Error        int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
}

type PackageInfo struct {
	Name           string
	Symbols        []Symbol
	CoreSymbolHash []int
	Imports        []Import
	Decls          []DeclInfo
	DeclOrder      []int
	InitOrder      []int
	Types          []TypeInfo
	TypeRefs       []TypeRef
	CoreTypeRefs   []CoreTypeRef
	CoreArenaStart int
	CoreArenaEnd   int
	Methods        []MethodInfo
	Bodies         []FuncBody
	CoreBodies     []CoreFuncBody
}

type Symbol struct {
	Name    string
	Kind    int
	Package int
	File    int
	Token   int
}

type Import struct {
	Name       string
	ImportPath string
	Package    int
	File       int
	Token      int
	Dot        bool
	Blank      bool
	Used       bool
}

type DeclInfo struct {
	Name          string
	Kind          int
	File          int
	Token         int
	Symbol        int
	ValueIndex    int
	TypeStart     int
	TypeEnd       int
	ValueStart    int
	ValueEnd      int
	Values        []ExprSpan
	Refs          []NameRef
	CoreRefs      []CoreNameRef
	Selectors     []SelectorRef
	CoreSelectors []CoreSelectorRef
	Calls         []CallRef
	Indexes       []IndexExpr
	Composites    []CompositeExpr
	Deps          []int
	Const         ConstValue
	Alias         bool
}

type LocalDeclInfo struct {
	Name       string
	Kind       int
	File       int
	Token      int
	Scope      int
	ValueIndex int
	TypeStart  int
	TypeEnd    int
	ValueStart int
	ValueEnd   int
	Values     []ExprSpan
	Refs       []NameRef
	Selectors  []SelectorRef
	Calls      []CallRef
	Indexes    []IndexExpr
	Composites []CompositeExpr
	Const      ConstValue
	Alias      bool
}

type ConstValue struct {
	Kind   int
	Int    int
	String string
	Bool   bool
	Ok     bool
}

type FuncBody struct {
	Name          string
	Kind          int
	File          int
	Func          int
	Signature     FuncSignature
	Body          syntax.Body
	Scope         FuncScope
	Refs          []NameRef
	CoreRefs      []CoreNameRef
	Selectors     []SelectorRef
	CoreSelectors []CoreSelectorRef
	Calls         []CallRef
	Indexes       []IndexExpr
	Composites    []CompositeExpr
	Locals        []LocalDeclInfo
	TypeRefs      []TypeRef
	CoreTypeRefs  []CoreTypeRef
	Assigns       []AssignInfo
	Returns       []ReturnInfo
}

type CoreNameRef struct {
	Token   int
	Index   int
	Package int
}

type CoreSelectorRef struct {
	BaseTok     int
	DotTok      int
	NameTok     int
	BaseIndex   int
	BasePackage int
	Symbol      int
}

type CoreTypeRef struct {
	Kind      int
	File      int
	OwnerDecl int
	Token     int
	BaseTok   int
	DotTok    int
	Package   int
	Symbol    int
}

type CoreFuncBody struct {
	Kind          int
	File          int
	Func          int
	ErrorToken    int
	CoreRefs      []CoreNameRef
	CoreSelectors []CoreSelectorRef
	CoreTypeRefs  []CoreTypeRef
}

func CheckGraph(graph load.Graph) Program {
	return CheckGraphCore(graph)
}

func LookupPackageSymbol(info PackageInfo, name string) int {
	for i := 0; i < len(info.Symbols); i++ {
		if info.Symbols[i].Name == name {
			return i
		}
	}
	return -1
}

func LookupImport(info PackageInfo, file int, name string) int {
	for i := 0; i < len(info.Imports); i++ {
		imp := info.Imports[i]
		if imp.File == file && imp.Name == name {
			return i
		}
	}
	return -1
}

func checkPackageHeader(graph load.Graph, pkgIndex int) (PackageInfo, bool, int, int, int) {
	pkg := graph.Packages[pkgIndex]
	symbolCapacity := 0
	importCapacity := 0
	for i := 0; i < len(pkg.Files); i++ {
		symbolCapacity += len(pkg.Files[i].File.Decls) + len(pkg.Files[i].File.Funcs)
		importCapacity += len(pkg.Files[i].File.Imports)
	}
	info := PackageInfo{
		Name:    cloneCheckString(pkg.Name),
		Symbols: make([]Symbol, 0, symbolCapacity),
		Imports: make([]Import, 0, importCapacity),
	}
	var symbolHash []int
	if symbolCapacity > 0 {
		symbolHash = make([]int, symbolCapacity*2+1)
	}
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		if excludedErr, excludedTok := excludedFileFeature(file); excludedErr != CheckOK {
			return info, false, excludedErr, fileIndex, excludedTok
		}
		for i := 0; i < len(file.Decls); i++ {
			decl := file.Decls[i]
			name := tokenString(&file, decl.NameTok)
			kind := declSymbolKind(decl.Kind)
			if findSymbolHashed(info.Symbols, symbolHash, name, kind) >= 0 {
				return info, false, CheckErrDuplicate, fileIndex, decl.NameTok
			}
			info.Symbols = append(info.Symbols, Symbol{Name: name, Kind: kind, Package: pkgIndex, File: fileIndex, Token: decl.NameTok})
			insertSymbolHash(info.Symbols, symbolHash, len(info.Symbols)-1)
		}
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			name := tokenString(&file, fn.NameTok)
			kind := SymbolFunc
			if fn.ReceiverStart >= 0 {
				receiver := receiverTypeName(file, fn)
				if receiver == "" {
					return info, false, CheckErrMethod, fileIndex, fn.NameTok
				}
				name = receiver + "." + name
				kind = SymbolMethod
			}
			duplicate := findSymbolHashed(info.Symbols, symbolHash, name, kind)
			if duplicate >= 0 && (name != "init" || info.Symbols[duplicate].Kind != SymbolFunc) {
				return info, false, CheckErrDuplicate, fileIndex, fn.NameTok
			}
			info.Symbols = append(info.Symbols, Symbol{Name: name, Kind: kind, Package: pkgIndex, File: fileIndex, Token: fn.NameTok})
			insertSymbolHash(info.Symbols, symbolHash, len(info.Symbols)-1)
		}
	}
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Imports); i++ {
			imp, ok := buildImport(graph, pkgIndex, fileIndex, file, i)
			if !ok {
				return info, false, CheckErrImport, fileIndex, file.Imports[i].PathTok
			}
			if !imp.Blank && !imp.Dot && LookupImport(info, fileIndex, imp.Name) >= 0 {
				return info, false, CheckErrDuplicate, fileIndex, imp.Token
			}
			info.Imports = append(info.Imports, imp)
		}
	}
	sortSymbols(info.Symbols)
	sortImports(info.Imports)
	return info, true, CheckOK, -1, -1
}

func CheckRootMain(pkg load.Package) (int, int, int) {
	methodFile, methodTok := -1, -1
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			name := tokenString(&file, fn.NameTok)
			if fn.ReceiverStart >= 0 {
				if name == "main" {
					methodFile, methodTok = fileIndex, fn.NameTok
				}
				continue
			}
			if name == "appMain" {
				return CheckOK, -1, -1
			}
			if name == "main" {
				if fn.ParamsEnd != fn.ParamsStart+2 || fn.ResultEnd != fn.ResultStart {
					return CheckErrMainSignature, fileIndex, fn.NameTok
				}
				return CheckOK, -1, -1
			}
		}
	}
	if methodTok >= 0 {
		return CheckErrMainMethod, methodFile, methodTok
	}
	return CheckErrMissingMain, 0, pkg.Files[0].File.PackageName
}

func findSymbolHashed(symbols []Symbol, buckets []int, name string, kind int) int {
	if len(buckets) == 0 {
		return findSymbol(symbols, name, kind)
	}
	bucket := hashCheckString(name) % len(buckets)
	for probes := 0; probes < len(buckets); probes++ {
		entry := buckets[bucket]
		if entry == 0 {
			return -1
		}
		index := entry - 1
		if index >= 0 && index < len(symbols) && symbols[index].Name == name {
			if kind != SymbolMethod && symbols[index].Kind != SymbolMethod || symbols[index].Kind == kind {
				return index
			}
		}
		bucket++
		if bucket == len(buckets) {
			bucket = 0
		}
	}
	return -1
}

func insertSymbolHash(symbols []Symbol, buckets []int, index int) {
	if len(buckets) == 0 || index < 0 || index >= len(symbols) {
		return
	}
	bucket := hashCheckString(symbols[index].Name) % len(buckets)
	for probes := 0; probes < len(buckets); probes++ {
		if buckets[bucket] == 0 {
			buckets[bucket] = index + 1
			return
		}
		bucket++
		if bucket == len(buckets) {
			bucket = 0
		}
	}
}

func hashCheckString(value string) int {
	hash := 5381
	for i := 0; i < len(value); i++ {
		hash = ((hash << 5) + hash + int(value[i])) & 2147483647
	}
	return hash
}

func buildImport(graph load.Graph, pkgIndex int, fileIndex int, file syntax.File, importIndex int) (Import, bool) {
	decl := file.Imports[importIndex]
	path, ok := syntax.StringLiteralValue(file.Src, file.Tokens[decl.PathTok])
	if !ok {
		return Import{}, false
	}
	target := findGraphPackage(graph, path)
	if target < 0 {
		return Import{}, false
	}
	name := cloneCheckString(graph.Packages[target].Name)
	dot := false
	blank := false
	tok := decl.PathTok
	if decl.NameTok >= 0 {
		tok = decl.NameTok
		explicit := tokenString(&file, decl.NameTok)
		if explicit == "." {
			dot = true
			name = "."
		} else if explicit == "_" {
			blank = true
			name = "_"
		} else {
			name = explicit
		}
	}
	return Import{Name: name, ImportPath: path, Package: target, File: fileIndex, Token: tok, Dot: dot, Blank: blank}, true
}

func cloneCheckString(value string) string {
	data := make([]byte, len(value))
	copy(data, []byte(value))
	return string(data)
}

func findGraphPackage(graph load.Graph, importPath string) int {
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == importPath {
			return i
		}
	}
	return -1
}

func declSymbolKind(kind int) int {
	if kind == syntax.TokenConst {
		return SymbolConst
	}
	if kind == syntax.TokenVar {
		return SymbolVar
	}
	return SymbolType
}

func findSymbol(symbols []Symbol, name string, kind int) int {
	for i := 0; i < len(symbols); i++ {
		if symbols[i].Name == name {
			if kind == SymbolMethod || symbols[i].Kind == SymbolMethod {
				if symbols[i].Kind == kind {
					return i
				}
				continue
			}
			return i
		}
	}
	return -1
}

func receiverTypeName(file syntax.File, fn syntax.FuncDecl) string {
	end := fn.ReceiverEnd
	if end > len(file.Tokens) {
		end = len(file.Tokens)
	}
	for i := end - 1; i >= fn.ReceiverStart; i-- {
		if file.Tokens[i].Kind == syntax.TokenIdent {
			return tokenString(&file, i)
		}
	}
	return ""
}

func tokenString(file *syntax.File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	return string(syntax.TokenText(file.Src, file.Tokens[tok]))
}

func sortSymbols(symbols []Symbol) {
	for root := len(symbols)/2 - 1; root >= 0; root-- {
		siftDownSymbols(symbols, root, len(symbols))
	}
	for end := len(symbols) - 1; end > 0; end-- {
		symbols[0], symbols[end] = symbols[end], symbols[0]
		siftDownSymbols(symbols, 0, end)
	}
}

func siftDownSymbols(symbols []Symbol, root int, end int) {
	for {
		child := root*2 + 1
		if child >= end {
			return
		}
		if child+1 < end && symbolAfter(symbols[child+1], symbols[child]) {
			child++
		}
		if !symbolAfter(symbols[child], symbols[root]) {
			return
		}
		symbols[root], symbols[child] = symbols[child], symbols[root]
		root = child
	}
}

func symbolAfter(left Symbol, right Symbol) bool {
	if left.Name != right.Name {
		return checkStringAfter(left.Name, right.Name)
	}
	if left.Kind != right.Kind {
		return left.Kind > right.Kind
	}
	if left.File != right.File {
		return left.File > right.File
	}
	return left.Token > right.Token
}

func sortImports(imports []Import) {
	for i := 1; i < len(imports); i++ {
		item := imports[i]
		j := i - 1
		for j >= 0 && importAfter(imports[j], item) {
			imports[j+1] = imports[j]
			j--
		}
		imports[j+1] = item
	}
}

func importAfter(left Import, right Import) bool {
	if left.File != right.File {
		return left.File > right.File
	}
	if left.Name != right.Name {
		return checkStringAfter(left.Name, right.Name)
	}
	return checkStringAfter(left.ImportPath, right.ImportPath)
}

func checkFail(prog Program, err int, pkg int, file int, tok int) Program {
	prog.Ok = false
	prog.Error = err
	prog.ErrorPackage = pkg
	prog.ErrorFile = file
	prog.ErrorToken = tok
	return prog
}
