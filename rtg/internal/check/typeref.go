package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

const (
	TypeRefUnknown = iota
	TypeRefScope
	TypeRefPackage
	TypeRefImportSelector
	TypeRefBuiltin
)

type TypeRef struct {
	Kind      int
	Name      string
	BaseName  string
	File      int
	OwnerDecl int
	Token     int
	BaseToken int
	DotToken  int
	Ref       NameRef
	Selector  SelectorRef
	Package   int
	Symbol    int
}

func LookupTypeRef(refs []TypeRef, base string, name string, kind int) int {
	for i := 0; i < len(refs); i++ {
		ref := refs[i]
		if ref.BaseName == base && ref.Name == name && ref.Kind == kind {
			return i
		}
	}
	return -1
}

func buildPackageTypeRefs(pkg load.Package, info PackageInfo, checked []PackageInfo) []TypeRef {
	var refs []TypeRef
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.TypeStart < 0 || decl.TypeEnd <= decl.TypeStart {
			continue
		}
		file := pkg.Files[decl.File].File
		if decl.Kind == SymbolType {
			typeIndex := LookupType(info, decl.Name)
			if typeIndex >= 0 {
				refs = appendTypeInfoRefs(refs, pkg, info, checked, info.Types[typeIndex], i)
				continue
			}
		}
		refs = appendDeclTypeSpanRefs(refs, file, decl.File, info, checked, FuncScope{}, i, decl.TypeStart, decl.TypeEnd)
	}
	return refs
}

func appendTypeInfoRefs(refs []TypeRef, pkg load.Package, info PackageInfo, checked []PackageInfo, typ TypeInfo, ownerDecl int) []TypeRef {
	file := pkg.Files[typ.File].File
	if typ.Kind == TypeStruct {
		for i := 0; i < len(typ.Fields); i++ {
			field := typ.Fields[i]
			refs = appendDeclTypeSpanRefs(refs, file, typ.File, info, checked, FuncScope{}, ownerDecl, field.TypeStart, field.TypeEnd)
		}
		return refs
	}
	if typ.Kind == TypeInterface {
		for i := 0; i < len(typ.InterfaceEmbeds); i++ {
			embed := typ.InterfaceEmbeds[i]
			refs = appendDeclTypeSpanRefs(refs, file, typ.File, info, checked, FuncScope{}, ownerDecl, embed.TypeStart, embed.TypeEnd)
		}
		for i := 0; i < len(typ.InterfaceMethods); i++ {
			base := len(refs)
			refs = appendSignatureTypeRefs(refs, file, typ.File, info, checked, FuncScope{}, typ.InterfaceMethods[i].Signature)
			markTypeRefOwnerDecl(refs, base, ownerDecl)
		}
		return refs
	}
	return appendDeclTypeSpanRefs(refs, file, typ.File, info, checked, FuncScope{}, ownerDecl, typ.TypeStart, typ.TypeEnd)
}

func buildFuncTypeRefs(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, signature FuncSignature, locals []LocalDeclInfo, scope FuncScope) []TypeRef {
	var refs []TypeRef
	refs = appendSignatureTypeRefs(refs, file, fileIndex, info, checked, scope, signature)
	for i := 0; i < len(locals); i++ {
		local := locals[i]
		if local.TypeStart >= 0 && local.TypeEnd > local.TypeStart {
			refs = appendTypeSpanRefs(refs, file, fileIndex, info, checked, scope, local.TypeStart, local.TypeEnd)
		}
	}
	return refs
}

func appendSignatureTypeRefs(refs []TypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, signature FuncSignature) []TypeRef {
	for i := 0; i < len(signature.Receiver); i++ {
		field := signature.Receiver[i]
		refs = appendTypeSpanRefs(refs, file, fileIndex, info, checked, scope, field.TypeStart, field.TypeEnd)
	}
	for i := 0; i < len(signature.Params); i++ {
		field := signature.Params[i]
		refs = appendTypeSpanRefs(refs, file, fileIndex, info, checked, scope, field.TypeStart, field.TypeEnd)
	}
	for i := 0; i < len(signature.Results); i++ {
		field := signature.Results[i]
		refs = appendTypeSpanRefs(refs, file, fileIndex, info, checked, scope, field.TypeStart, field.TypeEnd)
	}
	return refs
}

func appendDeclTypeSpanRefs(refs []TypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, ownerDecl int, start int, end int) []TypeRef {
	base := len(refs)
	refs = appendTypeSpanRefs(refs, file, fileIndex, info, checked, scope, start, end)
	markTypeRefOwnerDecl(refs, base, ownerDecl)
	return refs
}

func markTypeRefOwnerDecl(refs []TypeRef, start int, ownerDecl int) {
	for i := start; i < len(refs); i++ {
		refs[i].OwnerDecl = ownerDecl
	}
}

func appendTypeSpanRefs(refs []TypeRef, file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, start int, end int) []TypeRef {
	for i := start; i < end && i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind != syntax.TokenIdent {
			continue
		}
		if i > start && tokenTextIs(file, i-1, ".") {
			continue
		}
		name := tokenString(file, i)
		if name == "_" {
			continue
		}
		if i+2 < end && tokenTextIs(file, i+1, ".") && file.Tokens[i+2].Kind == syntax.TokenIdent {
			refs = append(refs, resolveSelectorTypeRef(fileIndex, info, checked, scope, name, tokenString(file, i+2), i, i+1, i+2))
			i += 2
			continue
		}
		refs = append(refs, resolveDirectTypeRef(fileIndex, info, scope, name, i))
	}
	return refs
}

func resolveDirectTypeRef(fileIndex int, info PackageInfo, scope FuncScope, name string, tok int) TypeRef {
	ref := resolveNameRef(fileIndex, info, scope, name, tok)
	out := TypeRef{
		Kind:      TypeRefUnknown,
		Name:      name,
		File:      fileIndex,
		OwnerDecl: -1,
		Token:     tok,
		BaseToken: -1,
		DotToken:  -1,
		Ref:       ref,
		Package:   -1,
		Symbol:    -1,
	}
	if ref.Kind == RefScope {
		out.Kind = TypeRefScope
	} else if ref.Kind == RefPackage {
		out.Kind = TypeRefPackage
		out.Package = ref.Package
		out.Symbol = ref.Index
	} else if ref.Kind == RefBuiltin {
		out.Kind = TypeRefBuiltin
	}
	return out
}

func resolveSelectorTypeRef(fileIndex int, info PackageInfo, checked []PackageInfo, scope FuncScope, base string, name string, baseTok int, dotTok int, nameTok int) TypeRef {
	selector := resolveSelector(fileIndex, info, checked, scope, base, name, baseTok, dotTok, nameTok)
	out := TypeRef{
		Kind:      TypeRefUnknown,
		Name:      name,
		BaseName:  base,
		File:      fileIndex,
		OwnerDecl: -1,
		Token:     nameTok,
		BaseToken: baseTok,
		DotToken:  dotTok,
		Selector:  selector,
		Package:   -1,
		Symbol:    -1,
	}
	if selector.Kind == SelectorImport {
		out.Kind = TypeRefImportSelector
		out.Package = selector.Package
		out.Symbol = selector.Symbol
	}
	return out
}
