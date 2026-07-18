package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

type MethodInfo struct {
	Name      string
	Receiver  string
	Pointer   bool
	Type      int
	Symbol    int
	Body      int
	File      int
	Token     int
	Func      int
	Signature FuncSignature
}

func LookupMethod(info PackageInfo, receiver string, name string) int {
	for i := 0; i < len(info.Methods); i++ {
		method := info.Methods[i]
		if method.Receiver == receiver && method.Name == name {
			return i
		}
	}
	return -1
}

func buildMethodSets(info *PackageInfo, pkg load.Package) {
	for i := 0; i < len(info.Bodies); i++ {
		body := info.Bodies[i]
		if body.Kind != SymbolMethod {
			continue
		}
		method := buildMethodInfo(*info, pkg, body, i)
		index := len(info.Methods)
		info.Methods = append(info.Methods, method)
		if method.Type >= 0 && method.Type < len(info.Types) {
			info.Types[method.Type].Methods = append(info.Types[method.Type].Methods, index)
		}
	}
	sortMethods(info.Methods)
	remapTypeMethods(info)
}

func buildMethodSetsCore(info *PackageInfo, pkg load.Package) {
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for funcIndex := 0; funcIndex < len(file.Funcs); funcIndex++ {
			fn := file.Funcs[funcIndex]
			if fn.ReceiverStart < 0 {
				continue
			}
			signature := buildFuncSignature(file, fn)
			if len(signature.Receiver) == 0 {
				continue
			}
			receiver := receiverBaseName(pkg, fileIndex, signature.Receiver[0])
			name := tokenString(&file, fn.NameTok)
			method := MethodInfo{
				Name:      name,
				Receiver:  receiver,
				Pointer:   receiverIsPointer(pkg, fileIndex, signature.Receiver[0]),
				Type:      LookupType(*info, receiver),
				Symbol:    LookupPackageSymbol(*info, receiver+"."+name),
				Body:      -1,
				File:      fileIndex,
				Token:     fn.NameTok,
				Func:      funcIndex,
				Signature: signature,
			}
			info.Methods = append(info.Methods, method)
		}
	}
	sortMethods(info.Methods)
	remapTypeMethods(info)
}

func buildMethodInfo(info PackageInfo, pkg load.Package, body FuncBody, bodyIndex int) MethodInfo {
	name := methodName(body.Name)
	receiver := ""
	pointer := false
	if len(body.Signature.Receiver) > 0 {
		field := body.Signature.Receiver[0]
		pointer = receiverIsPointer(pkg, body.File, field)
		receiver = receiverBaseName(pkg, body.File, field)
	}
	symbol := LookupPackageSymbol(info, body.Name)
	return MethodInfo{
		Name:      name,
		Receiver:  receiver,
		Pointer:   pointer,
		Type:      LookupType(info, receiver),
		Symbol:    symbol,
		Body:      bodyIndex,
		File:      body.File,
		Token:     bodyToken(pkg, body),
		Func:      body.Func,
		Signature: body.Signature,
	}
}

func receiverIsPointer(pkg load.Package, fileIndex int, field Field) bool {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) {
		return false
	}
	file := pkg.Files[fileIndex].File
	return tokCharIs(&file, field.TypeStart, '*')
}

func receiverBaseName(pkg load.Package, fileIndex int, field Field) string {
	if fileIndex < 0 || fileIndex >= len(pkg.Files) {
		return ""
	}
	file := pkg.Files[fileIndex].File
	for i := field.TypeEnd - 1; i >= field.TypeStart; i-- {
		if file.Tokens[i].Kind == syntax.TokenIdent {
			return tokenString(&file, i)
		}
	}
	return ""
}

func bodyToken(pkg load.Package, body FuncBody) int {
	if body.File < 0 || body.File >= len(pkg.Files) {
		return -1
	}
	file := pkg.Files[body.File].File
	if body.Func < 0 || body.Func >= len(file.Funcs) {
		return -1
	}
	return file.Funcs[body.Func].NameTok
}

func methodName(symbol string) string {
	for i := len(symbol) - 1; i >= 0; i-- {
		if symbol[i] == '.' {
			return symbol[i+1:]
		}
	}
	return symbol
}

func sortMethods(methods []MethodInfo) {
	for i := 1; i < len(methods); i++ {
		item := methods[i]
		j := i - 1
		for j >= 0 && methodAfter(methods[j], item) {
			methods[j+1] = methods[j]
			j--
		}
		methods[j+1] = item
	}
}

func methodAfter(left MethodInfo, right MethodInfo) bool {
	if left.Receiver != right.Receiver {
		return checkStringAfter(left.Receiver, right.Receiver)
	}
	if left.Name != right.Name {
		return checkStringAfter(left.Name, right.Name)
	}
	if left.Pointer != right.Pointer {
		return left.Pointer && !right.Pointer
	}
	if left.File != right.File {
		return left.File > right.File
	}
	return left.Token > right.Token
}

func remapTypeMethods(info *PackageInfo) {
	for i := 0; i < len(info.Types); i++ {
		info.Types[i].Methods = nil
	}
	for i := 0; i < len(info.Methods); i++ {
		typeIndex := info.Methods[i].Type
		if typeIndex >= 0 && typeIndex < len(info.Types) {
			info.Types[typeIndex].Methods = append(info.Types[typeIndex].Methods, i)
		}
	}
}
