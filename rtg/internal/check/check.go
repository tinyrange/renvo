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
	prog := Program{
		Graph:        graph,
		Ok:           true,
		Error:        CheckOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	if !graph.Ok {
		return checkFail(prog, CheckErrGraph, graph.ErrorPackage, -1, -1)
	}
	for i := 0; i < len(graph.Packages); i++ {
		info, ok, err, file, tok := checkPackageHeader(graph, i)
		prog.Packages = append(prog.Packages, info)
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	for i := 0; i < len(graph.Packages); i++ {
		info, ok, err, file, tok := checkPackageBody(graph, i, prog.Packages[i], prog.Packages)
		prog.Packages[i] = info
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	return prog
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

func LookupFuncBody(info PackageInfo, name string) int {
	for i := 0; i < len(info.Bodies); i++ {
		if info.Bodies[i].Name == name {
			return i
		}
	}
	return -1
}

func checkPackageHeader(graph load.Graph, pkgIndex int) (PackageInfo, bool, int, int, int) {
	pkg := graph.Packages[pkgIndex]
	info := PackageInfo{Name: cloneCheckString(pkg.Name)}
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Decls); i++ {
			decl := file.Decls[i]
			name := tokenString(file, decl.NameTok)
			kind := declSymbolKind(decl.Kind)
			if findSymbol(info.Symbols, name, kind) >= 0 {
				return info, false, CheckErrDuplicate, fileIndex, decl.NameTok
			}
			info.Symbols = append(info.Symbols, Symbol{Name: name, Kind: kind, Package: pkgIndex, File: fileIndex, Token: decl.NameTok})
		}
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			if excludedTok := excludedFeatureToken(file, fn); excludedTok >= 0 {
				return info, false, CheckErrExcluded, fileIndex, excludedTok
			}
			name := tokenString(file, fn.NameTok)
			kind := SymbolFunc
			if fn.ReceiverStart >= 0 {
				receiver := receiverTypeName(file, fn)
				if receiver == "" {
					return info, false, CheckErrMethod, fileIndex, fn.NameTok
				}
				name = receiver + "." + name
				kind = SymbolMethod
			}
			if findSymbol(info.Symbols, name, kind) >= 0 {
				return info, false, CheckErrDuplicate, fileIndex, fn.NameTok
			}
			info.Symbols = append(info.Symbols, Symbol{Name: name, Kind: kind, Package: pkgIndex, File: fileIndex, Token: fn.NameTok})
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

func checkPackageBody(graph load.Graph, pkgIndex int, info PackageInfo, checked []PackageInfo) (PackageInfo, bool, int, int, int) {
	pkg := graph.Packages[pkgIndex]
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Decls); i++ {
			info.Decls = append(info.Decls, buildDeclInfo(file, fileIndex, info, checked, file.Decls[i]))
		}
	}
	sortDecls(info.Decls)
	info.DeclOrder = buildDeclOrder(info.Decls)
	buildDeclDeps(&info)
	info.InitOrder = buildInitOrder(info.Decls, info.DeclOrder)
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Kind == SymbolType {
			file := pkg.Files[decl.File].File
			info.Types = append(info.Types, buildTypeInfo(file, decl, i))
		}
	}
	sortTypes(info.Types)
	info.TypeRefs = buildPackageTypeRefs(pkg, info, checked)
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			name := tokenString(file, fn.NameTok)
			kind := SymbolFunc
			if fn.ReceiverStart >= 0 {
				name = receiverTypeName(file, fn) + "." + name
				kind = SymbolMethod
			}
			signature := buildFuncSignature(file, fn)
			body := syntax.ParseFuncBody(file, fn)
			if !body.Ok {
				return info, false, CheckErrBody, fileIndex, body.ErrorTok
			}
			if returnTok := invalidReturnCount(file, fn, signature); returnTok >= 0 {
				return info, false, CheckErrReturnCount, fileIndex, returnTok
			}
			if typeTok := invalidDefiniteAssignmentType(file, fn); typeTok >= 0 {
				return info, false, CheckErrType, fileIndex, typeTok
			}
			scope, ok, scopeTok := buildFuncScope(file, fn, body)
			if !ok {
				return info, false, CheckErrScope, fileIndex, scopeTok
			}
			refs := buildFuncRefs(file, fileIndex, info, body, scope)
			selectors := buildFuncSelectors(file, fileIndex, info, checked, body, scope)
			calls := buildFuncCalls(file, fileIndex, info, checked, body, scope)
			indexes := buildFuncIndexExprs(file, body)
			composites := buildFuncCompositeExprs(file, body)
			locals := buildFuncLocalDecls(file, fileIndex, info, checked, body, scope)
			typeRefs := buildFuncTypeRefs(file, fileIndex, info, checked, signature, locals, scope)
			assigns := buildFuncAssignments(file, fileIndex, info, body, scope)
			returns := buildFuncReturns(file, body)
			info.Bodies = append(info.Bodies, FuncBody{Name: name, Kind: kind, File: fileIndex, Func: i, Signature: signature, Body: body, Scope: scope, Refs: refs, Selectors: selectors, Calls: calls, Indexes: indexes, Composites: composites, Locals: locals, TypeRefs: typeRefs, Assigns: assigns, Returns: returns})
		}
	}
	buildMethodSets(&info, pkg)
	return info, true, CheckOK, -1, -1
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
		explicit := tokenString(file, decl.NameTok)
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
			return tokenString(file, i)
		}
	}
	return ""
}

func tokenString(file syntax.File, tok int) string {
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
