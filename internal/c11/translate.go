package c11

const (
	TranslateOK = iota
	TranslateErrScan
	TranslateErrDeclaration
	TranslateErrStatement
	TranslateErrUnsupported
)

const (
	storageNone = iota
	storageOther
	storageExtern
	storageTypedef
)

// Result is the C11 source adapter result. Source is ordinary Renvo Go input,
// so every later frontend phase is shared with Go packages.
type Result struct {
	Source  []byte
	Ok      bool
	Error   int
	ErrorAt int
}

type declarator struct {
	name        token
	typeID      int
	params      []parameter
	function    bool
	initializer []token
}

type parameter struct {
	name   token
	typeID int
}

const (
	declaratorPointer = iota + 1
	declaratorArray
	declaratorFunction
)

type declaratorOp struct {
	kind   int
	count  int
	params []parameter
}

type translator struct {
	src            []byte
	tokens         []token
	pos            int
	packageName    string
	out            []byte
	object         bool
	ok             bool
	err            int
	errorAt        int
	types          []cTypeInfo
	fields         []cField
	typedefs       []cTypeName
	tags           []cTypeName
	opaqueTypes    []cTypeName
	values         []cValueName
	functionParams []int
	typeSerial     int
	dataModel      int
	pointerSize    int
}

// Translate lowers one preprocessed C translation unit into the shared Go
// frontend syntax. It deliberately performs no backend-specific lowering.
func Translate(packageName string, src []byte) Result {
	return translate(packageName, src, nil, false, DataModelLP64)
}

// TranslateObject lowers a hosted C translation unit. Prelude contains
// declarations recovered from the translation unit's included headers; it is
// kept separate so diagnostics in src retain their original byte offsets.
func TranslateObject(packageName string, src []byte, prelude []byte) Result {
	return TranslateObjectForDataModel(packageName, src, prelude, DataModelLP64)
}

func TranslateObjectForDataModel(packageName string, src []byte, prelude []byte, dataModel int) Result {
	return translate(packageName, src, prelude, true, dataModel)
}

func translate(packageName string, src []byte, prelude []byte, object bool, dataModel int) Result {
	t := translator{
		packageName: packageName,
		out:         make([]byte, 0, len(src)+len(src)/4+len(prelude)+64),
		object:      object,
		ok:          true,
		errorAt:     -1,
	}
	t.initTypes(dataModel)
	t.out = append(t.out, "package "...)
	t.out = append(t.out, packageName...)
	t.out = append(t.out, '\n')
	if len(prelude) > 0 && !t.translateSource(prelude) {
		return Result{Ok: false, Error: t.err, ErrorAt: -1}
	}
	if !t.translateSource(src) {
		return Result{Ok: false, Error: t.err, ErrorAt: t.errorAt}
	}
	return Result{Source: t.out, Ok: true, Error: TranslateOK, ErrorAt: -1}
}

func (t *translator) translateSource(src []byte) bool {
	scanned := scan(src)
	if !scanned.ok {
		t.ok = false
		t.err = TranslateErrScan
		t.errorAt = scanned.errorAt
		return false
	}
	t.src = src
	t.tokens = scanned.tokens
	t.pos = 0
	for t.ok && t.kind() != tokenEOF {
		t.skipDirectives()
		if t.kind() == tokenEOF {
			break
		}
		t.externalDeclaration()
	}
	return t.ok
}

func (t *translator) kind() int {
	if t.pos < 0 || t.pos >= len(t.tokens) {
		return tokenEOF
	}
	return tokenKind(t.tokens[t.pos])
}

func (t *translator) currentIs(text string) bool {
	return t.pos >= 0 && t.pos < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos], text)
}

func (t *translator) take(text string) bool {
	if !t.currentIs(text) {
		return false
	}
	t.pos++
	return true
}

func (t *translator) fail(err int) {
	if !t.ok {
		return
	}
	t.ok = false
	t.err = err
	if t.pos >= 0 && t.pos < len(t.tokens) {
		t.errorAt = t.tokens[t.pos].start
	} else {
		t.errorAt = len(t.src)
	}
}

func (t *translator) skipDirectives() {
	for t.kind() != tokenEOF && t.currentIs("#") && (t.pos == 0 || tokenLine(t.tokens[t.pos-1]) < tokenLine(t.tokens[t.pos])) {
		line := tokenLine(t.tokens[t.pos])
		if t.pos+1 >= len(t.tokens) ||
			!tokenIs(t.src, t.tokens[t.pos+1], "include") && !tokenIs(t.src, t.tokens[t.pos+1], "pragma") {
			t.fail(TranslateErrUnsupported)
			return
		}
		for t.kind() != tokenEOF && tokenLine(t.tokens[t.pos]) == line {
			t.pos++
		}
	}
}

func (t *translator) externalDeclaration() {
	base, storage, ok := t.parseType()
	if !ok {
		t.fail(TranslateErrDeclaration)
		return
	}
	if t.take(";") {
		return
	}
	decl, ok := t.parseDeclarator(base, true)
	if !ok {
		t.fail(TranslateErrDeclaration)
		return
	}
	if decl.function {
		if t.take(";") {
			if t.object {
				t.emitForeignFunction(decl)
			}
			return // Otherwise a C or Go definition in the package satisfies it.
		}
		if !t.currentIs("{") {
			t.fail(TranslateErrDeclaration)
			return
		}
		t.emitFunction(decl, storage)
		return
	}
	decl.initializer = t.takeInitializer()
	decls := []declarator{decl}
	for t.take(",") {
		next, valid := t.parseDeclarator(base, false)
		if !valid {
			t.fail(TranslateErrDeclaration)
			return
		}
		next.initializer = t.takeInitializer()
		decls = append(decls, next)
	}
	if !t.take(";") {
		t.fail(TranslateErrDeclaration)
		return
	}
	if storage == storageTypedef {
		for i := 0; i < len(decls); i++ {
			t.rememberTypedef(string(tokenText(t.src, decls[i].name)), decls[i].typeID)
		}
		return
	}
	if storage == storageExtern {
		return
	}
	for i := 0; i < len(decls); i++ {
		t.emitVariable(decls[i])
	}
}

func (t *translator) parseAggregateType(union bool) (int, bool) {
	kind := cTypeStruct
	prefix := "__c_struct_"
	if union {
		kind = cTypeUnion
		prefix = "__c_union_"
	}
	t.pos++
	name := ""
	if t.kind() == tokenIdent {
		name = string(tokenText(t.src, t.tokens[t.pos]))
		t.pos++
	}
	if !t.currentIs("{") {
		if name == "" {
			return cTypeVoidID, false
		}
		if typeID, ok := t.lookupTag([]byte(name)); ok {
			return typeID, true
		}
		t.typeSerial++
		t.types = append(t.types, cTypeInfo{kind: kind, size: 0, align: 1, goName: prefix + name})
		typeID := len(t.types) - 1
		t.tags = append(t.tags, cTypeName{name: name, typeID: typeID})
		return typeID, true
	}
	t.typeSerial++
	goName := prefix + name
	if name == "" {
		goName = prefix + "anon_" + decimalString(t.typeSerial)
	}
	t.types = append(t.types, cTypeInfo{kind: kind, align: 1, goName: goName})
	typeID := len(t.types) - 1
	if name != "" {
		if prior, ok := t.lookupTag([]byte(name)); ok && t.types[prior].size == 0 {
			t.types = t.types[:len(t.types)-1]
			typeID = prior
			t.types[typeID].goName = goName
			t.types[typeID].kind = kind
		} else {
			t.tags = append(t.tags, cTypeName{name: name, typeID: typeID})
		}
	}
	t.pos++
	var aggregateFields []cField
	offset := 0
	maxAlign := 1
	maxSize := 0
	for !t.currentIs("}") && t.kind() != tokenEOF {
		fieldBase, storage, ok := t.parseType()
		if !ok || storage != storageNone && storage != storageOther {
			return cTypeVoidID, false
		}
		if t.take(";") {
			fieldInfo := t.typeInfo(fieldBase)
			if fieldInfo.kind != cTypeStruct && fieldInfo.kind != cTypeUnion {
				return cTypeVoidID, false
			}
			alignment := t.typeAlign(fieldBase)
			fieldOffset := 0
			if union {
				if fieldInfo.size > maxSize {
					maxSize = fieldInfo.size
				}
			} else {
				offset = alignC(offset, alignment)
				fieldOffset = offset
				offset += fieldInfo.size
			}
			if alignment > maxAlign {
				maxAlign = alignment
			}
			t.typeSerial++
			aggregateFields = append(aggregateFields, cField{name: "__c_anon_" + decimalString(t.typeSerial), typeID: fieldBase, offset: fieldOffset, emit: true})
			for i := 0; i < fieldInfo.fieldCount; i++ {
				promoted := t.fields[fieldInfo.fieldStart+i]
				promoted.offset += fieldOffset
				promoted.emit = false
				aggregateFields = append(aggregateFields, promoted)
			}
			continue
		}
		for {
			field, valid := t.parseDeclarator(fieldBase, false)
			if !valid {
				return cTypeVoidID, false
			}
			fieldInfo := t.typeInfo(field.typeID)
			alignment := t.typeAlign(field.typeID)
			fieldOffset := 0
			if union {
				if fieldInfo.size > maxSize {
					maxSize = fieldInfo.size
				}
			} else {
				offset = alignC(offset, alignment)
				fieldOffset = offset
				offset += fieldInfo.size
			}
			if alignment > maxAlign {
				maxAlign = alignment
			}
			aggregateFields = append(aggregateFields, cField{name: string(tokenText(t.src, field.name)), typeID: field.typeID, offset: fieldOffset, emit: true})
			if !t.take(",") {
				break
			}
		}
		if !t.take(";") {
			return cTypeVoidID, false
		}
	}
	if !t.take("}") {
		return cTypeVoidID, false
	}
	size := offset
	if union {
		size = maxSize
	}
	size = alignC(size, maxAlign)
	fieldStart := len(t.fields)
	t.fields = append(t.fields, aggregateFields...)
	t.types[typeID].size = size
	t.types[typeID].align = maxAlign
	t.types[typeID].fieldStart = fieldStart
	t.types[typeID].fieldCount = len(t.fields) - fieldStart
	t.emitAggregateDeclaration(typeID)
	return typeID, true
}

func (t *translator) parseEnumType() (int, bool) {
	t.pos++
	name := ""
	if t.kind() == tokenIdent {
		name = string(tokenText(t.src, t.tokens[t.pos]))
		t.pos++
	}
	if !t.take("{") {
		if name == "" {
			return cTypeVoidID, false
		}
		if typeID, ok := t.lookupTag([]byte(name)); ok {
			return typeID, true
		}
		t.tags = append(t.tags, cTypeName{name: name, typeID: cTypeInt32ID})
		return cTypeInt32ID, true
	}
	value := 0
	for !t.currentIs("}") && t.kind() != tokenEOF {
		if t.kind() != tokenIdent {
			return cTypeVoidID, false
		}
		enumerator := string(tokenText(t.src, t.tokens[t.pos]))
		t.pos++
		if t.take("=") {
			start := t.pos
			paren := 0
			for t.kind() != tokenEOF {
				if paren == 0 && (t.currentIs(",") || t.currentIs("}")) {
					break
				}
				if t.currentIs("(") {
					paren++
				} else if t.currentIs(")") {
					paren--
				}
				t.pos++
			}
			var ok bool
			value, ok = t.constantExpression(t.tokens[start:t.pos])
			if !ok {
				return cTypeVoidID, false
			}
		}
		t.rememberValue(enumerator, value)
		value++
		if !t.take(",") {
			break
		}
	}
	if !t.take("}") {
		return cTypeVoidID, false
	}
	if name != "" {
		for i := len(t.tags) - 1; i >= 0; i-- {
			if t.tags[i].name == name {
				t.tags[i].typeID = cTypeInt32ID
				return cTypeInt32ID, true
			}
		}
		t.tags = append(t.tags, cTypeName{name: name, typeID: cTypeInt32ID})
	}
	return cTypeInt32ID, true
}

func (t *translator) rememberValue(name string, value int) {
	for i := len(t.values) - 1; i >= 0; i-- {
		if t.values[i].name == name {
			t.values[i].value = value
			return
		}
	}
	t.values = append(t.values, cValueName{name: name, value: value})
}

func (t *translator) lookupValue(name []byte) (int, bool) {
	for i := len(t.values) - 1; i >= 0; i-- {
		if textEquals(name, t.values[i].name) {
			return t.values[i].value, true
		}
	}
	return 0, false
}

func (t *translator) emitAggregateDeclaration(typeID int) {
	info := t.typeInfo(typeID)
	t.out = append(t.out, "type "...)
	t.out = append(t.out, info.goName...)
	if info.kind == cTypeUnion {
		t.out = append(t.out, " struct{__c_align "...)
		switch info.align {
		case 8:
			t.out = append(t.out, "uint64"...)
		case 4:
			t.out = append(t.out, "uint32"...)
		case 2:
			t.out = append(t.out, "uint16"...)
		default:
			t.out = append(t.out, "uint8"...)
		}
		if tail := info.size - info.align; tail > 0 {
			t.out = append(t.out, ";__c_tail ["...)
			t.appendDecimal(tail)
			t.out = append(t.out, "]byte"...)
		}
		t.out = append(t.out, "};\n"...)
		return
	}
	t.out = append(t.out, " struct{"...)
	for i := 0; i < info.fieldCount; i++ {
		field := t.fields[info.fieldStart+i]
		if !field.emit {
			continue
		}
		t.out = append(t.out, field.name...)
		t.out = append(t.out, ' ')
		t.emitType(field.typeID)
		t.out = append(t.out, ';')
	}
	t.out = append(t.out, '}', ';', '\n')
}

func decimalString(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	count := 0
	for value > 0 {
		digits[count] = byte('0' + value%10)
		count++
		value /= 10
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = digits[count-i-1]
	}
	return string(out)
}

func (t *translator) emitForeignFunction(decl declarator) {
	name := tokenText(t.src, decl.name)
	t.out = append(t.out, "// renvo:linkstatic libc,"...)
	t.out = append(t.out, name...)
	t.out = append(t.out, '\n', 'f', 'u', 'n', 'c', ' ')
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	for i := 0; i < len(decl.params); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, 'p')
		t.appendDecimal(i)
		t.out = append(t.out, ' ')
		t.emitForeignType(decl.params[i].typeID)
	}
	t.out = append(t.out, ')')
	if decl.typeID != cTypeVoidID {
		t.out = append(t.out, ' ')
		t.emitType(decl.typeID)
	}
	t.out = append(t.out, '{')
	if t.typeInfo(decl.typeID).kind == cTypePointer {
		t.out = append(t.out, "return nil"...)
	} else if decl.typeID != cTypeVoidID {
		t.out = append(t.out, "return 0"...)
	}
	t.out = append(t.out, '}', '\n')
}

func (t *translator) appendDecimal(value int) {
	if value == 0 {
		t.out = append(t.out, '0')
		return
	}
	var digits [20]byte
	count := 0
	for value > 0 {
		digits[count] = byte('0' + value%10)
		count++
		value /= 10
	}
	for count > 0 {
		count--
		t.out = append(t.out, digits[count])
	}
}

func (t *translator) appendDecimalSigned(value int) {
	if value < 0 {
		t.out = append(t.out, '-')
		value = -value
	}
	t.appendDecimal(value)
}

func (t *translator) emitForeignType(typeID int) {
	// A Go string is used only as the shared frontend carrier for a C string
	// pointer. Object-mode call lowering emits its data word alone, so the ELF
	// boundary is the exact one-register System V `char *` ABI.
	info := t.typeInfo(typeID)
	if info.kind == cTypePointer && info.base == cTypeInt8ID {
		t.out = append(t.out, "string"...)
		return
	}
	t.emitType(typeID)
}

func (t *translator) parseType() (int, int, bool) {
	storage := storageNone
	unsigned := false
	longCount := 0
	base := ""
	typeID := cTypeVoidID
	hasTypeID := false
	seen := false
	for t.kind() == tokenIdent {
		switch {
		case t.currentIs("__extension__"):
			t.pos++
		case t.currentIs("static") || t.currentIs("register") || t.currentIs("auto") || t.currentIs("inline"):
			if storage == storageNone {
				storage = storageOther
			}
			t.pos++
		case t.currentIs("extern"):
			storage = storageExtern
			t.pos++
		case t.currentIs("typedef"):
			storage = storageTypedef
			t.pos++
		case t.currentIs("const") || t.currentIs("volatile") || t.currentIs("restrict") || t.currentIs("_Atomic"):
			t.pos++
		case t.currentIs("unsigned"):
			unsigned, seen = true, true
			t.pos++
		case t.currentIs("signed") || t.currentIs("__signed__"):
			seen = true
			t.pos++
		case t.currentIs("long"):
			longCount++
			seen = true
			t.pos++
		case t.currentIs("short"):
			base, seen = "short", true
			t.pos++
		case t.currentIs("char"):
			base, seen = "char", true
			t.pos++
		case t.currentIs("int"):
			if base == "" {
				base = "int"
			}
			seen = true
			t.pos++
		case t.currentIs("void"):
			base, seen = "void", true
			t.pos++
		case t.currentIs("float"):
			base, seen = "float", true
			t.pos++
		case t.currentIs("double"):
			base, seen = "double", true
			t.pos++
		case t.currentIs("_Bool"):
			base, seen = "bool", true
			t.pos++
		case t.currentIs("struct") || t.currentIs("union"):
			if seen {
				goto done
			}
			typeID, hasTypeID = t.parseAggregateType(t.currentIs("union"))
			if !hasTypeID {
				return cTypeVoidID, storage, false
			}
			seen = true
		case t.currentIs("enum"):
			if seen {
				goto done
			}
			typeID, hasTypeID = t.parseEnumType()
			if !hasTypeID {
				return cTypeVoidID, storage, false
			}
			seen = true
		default:
			if !seen {
				typeName := tokenText(t.src, t.tokens[t.pos])
				var known bool
				typeID, known = t.lookupTypedef(typeName)
				if !known {
					typeID, known = builtinCType(typeName)
				}
				if !known {
					for i := 0; i < len(t.opaqueTypes); i++ {
						if textEquals(typeName, t.opaqueTypes[i].name) {
							typeID = t.opaqueTypes[i].typeID
							known = true
							break
						}
					}
				}
				if !known {
					typeID = t.opaqueType(string(typeName))
				}
				hasTypeID = true
				seen = true
				t.pos++
				continue
			}
			goto done
		}
	}
done:
	if !seen {
		return cTypeVoidID, storage, false
	}
	if hasTypeID {
		return typeID, storage, true
	}
	if base == "" {
		base = "int"
	}
	typeID = cTypeInt32ID
	switch base {
	case "void":
		typeID = cTypeVoidID
	case "bool":
		typeID = cTypeBoolID
	case "char":
		if unsigned {
			typeID = cTypeUint8ID
		} else {
			typeID = cTypeInt8ID
		}
	case "short":
		if unsigned {
			typeID = cTypeUint16ID
		} else {
			typeID = cTypeInt16ID
		}
	case "float":
		typeID = cTypeFloat32ID
	case "double":
		typeID = cTypeFloat64ID
	default:
		if longCount > 0 {
			if unsigned {
				typeID = cTypeUint64ID
			} else {
				typeID = cTypeInt64ID
			}
		} else if unsigned {
			typeID = cTypeUint32ID
		}
	}
	return typeID, storage, true
}

func (t *translator) parseDeclarator(base int, allowFunction bool) (declarator, bool) {
	name, ops, ok := t.parseDeclaratorOps(false)
	result := declarator{name: name, typeID: base}
	if !ok {
		return result, false
	}
	for i := len(ops) - 1; i >= 0; i-- {
		switch ops[i].kind {
		case declaratorPointer:
			result.typeID = t.pointerType(result.typeID)
		case declaratorArray:
			result.typeID = t.arrayType(result.typeID, ops[i].count)
		case declaratorFunction:
			if t.typeInfo(result.typeID).kind == cTypeArray || t.typeInfo(result.typeID).kind == cTypeFunction {
				return result, false
			}
			result.typeID = t.functionType(result.typeID, ops[i].params)
		}
	}
	info := t.typeInfo(result.typeID)
	if info.kind == cTypeFunction {
		if !allowFunction || len(ops) == 0 || ops[0].kind != declaratorFunction {
			return result, false
		}
		result.function = true
		result.params = ops[0].params
		result.typeID = info.base
	} else if info.kind == cTypeOpaque {
		// The ABI size of an unknown scalar typedef cannot be inferred safely
		// from an unpreprocessed header. Opaque typedefs are exact when used
		// behind a pointer, which is the normal system-library handle shape.
		return result, false
	}
	return result, true
}

func (t *translator) parseDeclaratorOps(abstract bool) (token, []declaratorOp, bool) {
	pointers := 0
	for t.take("*") {
		pointers++
		for t.currentIs("const") || t.currentIs("volatile") || t.currentIs("restrict") {
			t.pos++
		}
	}
	var name token
	var ops []declaratorOp
	if t.kind() == tokenIdent {
		name = t.tokens[t.pos]
		t.pos++
	} else if t.take("(") {
		var ok bool
		name, ops, ok = t.parseDeclaratorOps(abstract)
		if !ok || !t.take(")") {
			return name, ops, false
		}
	} else if !abstract {
		return name, ops, false
	}
	for {
		if t.take("[") {
			end := t.findClosing("]")
			if end <= t.pos {
				return name, ops, false
			}
			count, valid := t.constantExpression(t.tokens[t.pos:end])
			if !valid || count < 1 {
				return name, ops, false
			}
			ops = append(ops, declaratorOp{kind: declaratorArray, count: count})
			t.pos = end + 1
			continue
		}
		if t.take("(") {
			params, valid := t.parseParameterList()
			if !valid {
				return name, ops, false
			}
			ops = append(ops, declaratorOp{kind: declaratorFunction, params: params})
			continue
		}
		break
	}
	for pointers > 0 {
		ops = append(ops, declaratorOp{kind: declaratorPointer})
		pointers--
	}
	return name, ops, true
}

func (t *translator) parseParameterList() ([]parameter, bool) {
	if t.take(")") {
		return nil, true
	}
	if t.currentIs("void") && t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], ")") {
		t.pos += 2
		return nil, true
	}
	var params []parameter
	for {
		paramBase, _, ok := t.parseType()
		if !ok {
			return params, false
		}
		name, ops, ok := t.parseDeclaratorOps(true)
		if !ok {
			return params, false
		}
		paramType := paramBase
		for i := len(ops) - 1; i >= 0; i-- {
			switch ops[i].kind {
			case declaratorPointer:
				paramType = t.pointerType(paramType)
			case declaratorArray:
				paramType = t.arrayType(paramType, ops[i].count)
			case declaratorFunction:
				paramType = t.functionType(paramType, ops[i].params)
			}
		}
		paramInfo := t.typeInfo(paramType)
		if paramInfo.kind == cTypeArray {
			paramType = t.pointerType(paramInfo.base)
		} else if paramInfo.kind == cTypeFunction {
			paramType = t.pointerType(paramType)
		}
		params = append(params, parameter{name: name, typeID: paramType})
		if t.take(")") {
			return params, true
		}
		if !t.take(",") || t.currentIs("...") {
			return params, false
		}
	}
}

func integerTokenValue(src []byte, tokens []token) (int, bool) {
	if len(tokens) != 1 || tokenKind(tokens[0]) != tokenNumber {
		return 0, false
	}
	text := trimIntegerSuffix(tokenText(src, tokens[0]))
	base := 10
	at := 0
	if len(text) > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X') {
		base = 16
		at = 2
	} else if len(text) > 1 && text[0] == '0' {
		base = 8
		at = 1
	}
	value := 0
	if at == len(text) {
		return 0, true
	}
	for ; at < len(text); at++ {
		digit := -1
		if text[at] >= '0' && text[at] <= '9' {
			digit = int(text[at] - '0')
		} else if text[at] >= 'a' && text[at] <= 'f' {
			digit = int(text[at]-'a') + 10
		} else if text[at] >= 'A' && text[at] <= 'F' {
			digit = int(text[at]-'A') + 10
		}
		if digit < 0 || digit >= base {
			return 0, false
		}
		value = value*base + digit
	}
	return value, true
}

func (t *translator) takeInitializer() []token {
	if !t.take("=") {
		return nil
	}
	start := t.pos
	paren, bracket, brace := 0, 0, 0
	for t.kind() != tokenEOF {
		if paren == 0 && bracket == 0 && brace == 0 && (t.currentIs(",") || t.currentIs(";")) {
			break
		}
		switch {
		case t.currentIs("("):
			paren++
		case t.currentIs(")"):
			paren--
		case t.currentIs("["):
			bracket++
		case t.currentIs("]"):
			bracket--
		case t.currentIs("{"):
			brace++
		case t.currentIs("}"):
			brace--
		}
		t.pos++
	}
	return t.tokens[start:t.pos]
}

func (t *translator) emitFunction(decl declarator, storage int) {
	if tokenIs(t.src, decl.name, "main") && t.packageName == "main" &&
		(len(decl.params) != 0 || decl.typeID != cTypeInt32ID) {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.object && storage != storageOther {
		t.out = append(t.out, "//export "...)
		t.out = append(t.out, tokenText(t.src, decl.name)...)
		t.out = append(t.out, '\n')
	}
	t.out = append(t.out, "func "...)
	name := tokenText(t.src, decl.name)
	if tokenIs(t.src, decl.name, "main") && t.packageName == "main" {
		name = []byte("appMain")
	}
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	for i := 0; i < len(decl.params); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, tokenText(t.src, decl.params[i].name)...)
		t.out = append(t.out, ' ')
		t.emitType(decl.params[i].typeID)
	}
	t.out = append(t.out, ')')
	if decl.typeID != cTypeVoidID {
		t.out = append(t.out, ' ')
		t.emitType(decl.typeID)
	}
	t.out = append(t.out, ' ')
	t.block()
	t.out = append(t.out, '\n')
}

func (t *translator) emitVariable(decl declarator) {
	t.out = append(t.out, "var "...)
	t.out = append(t.out, tokenText(t.src, decl.name)...)
	t.out = append(t.out, ' ')
	t.emitType(decl.typeID)
	if len(decl.initializer) > 0 {
		t.out = append(t.out, '=')
		kind := t.typeInfo(decl.typeID).kind
		if (kind == cTypeArray || kind == cTypeStruct) && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitType(decl.typeID)
		}
		t.expression(decl.initializer)
	}
	t.out = append(t.out, ';', '\n')
}

func (t *translator) block() {
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '{')
	for t.ok && t.kind() != tokenEOF && !t.currentIs("}") {
		t.skipDirectives()
		if t.currentIs("}") {
			break
		}
		t.statement()
		if t.ok {
			// A newline is a Go statement terminator after a completed block.
			// Without it, valid C99 such as `if (...) {} int value;` becomes
			// the invalid shared-syntax token sequence `}var`.
			t.out = append(t.out, '\n')
		}
	}
	if !t.take("}") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '}')
}

func (t *translator) statement() {
	if t.currentIs("{") {
		t.block()
		return
	}
	if t.isTypeStart() {
		t.localDeclaration()
		return
	}
	if t.take("if") {
		t.out = append(t.out, "if "...)
		t.parenthesizedCondition()
		t.out = append(t.out, ' ')
		t.controlledStatement()
		if t.take("else") {
			t.out = append(t.out, " else "...)
			t.controlledStatement()
		}
		return
	}
	if t.take("while") {
		t.out = append(t.out, "for "...)
		t.parenthesizedCondition()
		t.out = append(t.out, ' ')
		t.controlledStatement()
		return
	}
	if t.take("for") {
		t.forStatement()
		return
	}
	if t.take("switch") {
		t.switchStatement()
		return
	}
	if t.take("do") {
		t.doStatement()
		return
	}
	if t.take("goto") {
		if t.kind() != tokenIdent {
			t.fail(TranslateErrStatement)
			return
		}
		t.out = append(t.out, "goto "...)
		t.out = append(t.out, tokenText(t.src, t.tokens[t.pos])...)
		t.pos++
		if !t.take(";") {
			t.fail(TranslateErrStatement)
			return
		}
		t.out = append(t.out, ';')
		return
	}
	if t.currentIs("case") || t.currentIs("default") || t.currentIs("asm") ||
		t.currentIs("__asm") || t.currentIs("__asm__") {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.take("return") {
		t.out = append(t.out, "return"...)
		start := t.pos
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		if end > start {
			t.out = append(t.out, ' ')
			t.expression(t.tokens[start:end])
		}
		t.out = append(t.out, ';')
		t.pos = end + 1
		return
	}
	if t.currentIs("break") || t.currentIs("continue") {
		start := t.pos
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		t.expression(t.tokens[start:end])
		t.out = append(t.out, ';')
		t.pos = end + 1
		return
	}
	if t.take(";") {
		t.out = append(t.out, ';')
		return
	}
	if t.kind() == tokenIdent && t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], ":") {
		t.out = append(t.out, tokenText(t.src, t.tokens[t.pos])...)
		t.out = append(t.out, ':')
		t.pos += 2
		return
	}
	start := t.pos
	end := t.findAtDepth(";")
	if end < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.expression(t.tokens[start:end])
	t.out = append(t.out, ';')
	t.pos = end + 1
}

func (t *translator) controlledStatement() {
	if t.currentIs("{") {
		t.block()
		return
	}
	t.out = append(t.out, '{')
	t.statement()
	t.out = append(t.out, '}')
}

func (t *translator) localDeclaration() {
	base, storage, ok := t.parseType()
	if !ok || storage == storageExtern {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.take(";") {
		return
	}
	for {
		decl, valid := t.parseDeclarator(base, false)
		if !valid {
			t.fail(TranslateErrDeclaration)
			return
		}
		decl.initializer = t.takeInitializer()
		if storage == storageTypedef {
			t.rememberTypedef(string(tokenText(t.src, decl.name)), decl.typeID)
		} else {
			t.emitVariable(decl)
		}
		if !t.take(",") {
			break
		}
	}
	if !t.take(";") {
		t.fail(TranslateErrDeclaration)
	}
}

func (t *translator) forStatement() {
	if !t.take("(") {
		t.fail(TranslateErrStatement)
		return
	}
	declaration := t.isTypeStart()
	if declaration {
		t.out = append(t.out, '{')
		t.localDeclaration()
		if !t.ok {
			return
		}
		t.out = append(t.out, "for ;"...)
	} else {
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		t.out = append(t.out, "for "...)
		t.expression(t.tokens[t.pos:end])
		t.out = append(t.out, ';')
		t.pos = end + 1
	}
	end := t.findAtDepth(";")
	if end < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.condition(t.tokens[t.pos:end])
	t.out = append(t.out, ';')
	t.pos = end + 1
	close := t.findClosing(")")
	if close < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.expression(t.tokens[t.pos:close])
	t.pos = close + 1
	t.out = append(t.out, ' ')
	t.controlledStatement()
	if declaration {
		t.out = append(t.out, '}')
	}
}

func (t *translator) switchStatement() {
	if !t.take("(") {
		t.fail(TranslateErrStatement)
		return
	}
	close := t.findClosing(")")
	if close < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, "switch "...)
	t.expression(t.tokens[t.pos:close])
	t.pos = close + 1
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '{')
	haveCase := false
	terminal := false
	for t.ok && t.kind() != tokenEOF && !t.currentIs("}") {
		t.skipDirectives()
		if t.currentIs("case") || t.currentIs("default") {
			if haveCase && !terminal {
				t.out = append(t.out, "fallthrough;"...)
			}
			terminal = false
			haveCase = true
			if t.take("case") {
				end := t.findAtDepth(":")
				if end < 0 || end == t.pos {
					t.fail(TranslateErrStatement)
					return
				}
				t.out = append(t.out, "case "...)
				t.expression(t.tokens[t.pos:end])
				t.out = append(t.out, ':')
				t.pos = end + 1
			} else {
				t.pos++
				if !t.take(":") {
					t.fail(TranslateErrStatement)
					return
				}
				t.out = append(t.out, "default:"...)
			}
			continue
		}
		if !haveCase {
			t.fail(TranslateErrUnsupported)
			return
		}
		if t.currentIs("break") || t.currentIs("continue") || t.currentIs("return") || t.currentIs("goto") {
			terminal = true
		}
		t.statement()
		if t.ok {
			t.out = append(t.out, '\n')
		}
	}
	if !t.take("}") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '}')
}

func (t *translator) doStatement() {
	outer := t.out
	t.out = nil
	t.controlledStatement()
	body := t.out
	t.out = outer
	if !t.ok || len(body) == 0 || body[0] != '{' || !t.take("while") || !t.take("(") {
		t.fail(TranslateErrStatement)
		return
	}
	close := t.findClosing(")")
	if close < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.typeSerial++
	name := "__c_do_first_" + decimalString(t.typeSerial)
	t.out = append(t.out, "{var "...)
	t.out = append(t.out, name...)
	t.out = append(t.out, " bool=true;for "...)
	t.out = append(t.out, name...)
	t.out = append(t.out, "||"...)
	t.condition(t.tokens[t.pos:close])
	t.out = append(t.out, " {"...)
	t.out = append(t.out, name...)
	t.out = append(t.out, "=false;"...)
	t.out = append(t.out, body[1:]...)
	t.out = append(t.out, '}')
	t.pos = close + 1
	if !t.take(";") {
		t.fail(TranslateErrStatement)
	}
}

func (t *translator) parenthesizedCondition() {
	if !t.take("(") {
		t.fail(TranslateErrStatement)
		return
	}
	end := t.findClosing(")")
	if end < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.condition(t.tokens[t.pos:end])
	t.pos = end + 1
}

func (t *translator) condition(tokens []token) {
	if len(tokens) == 0 {
		t.out = append(t.out, "true"...)
		return
	}
	boolean := false
	for i := 0; i < len(tokens); i++ {
		if tokenIs(t.src, tokens[i], "==") || tokenIs(t.src, tokens[i], "!=") ||
			tokenIs(t.src, tokens[i], "<") || tokenIs(t.src, tokens[i], ">") ||
			tokenIs(t.src, tokens[i], "<=") || tokenIs(t.src, tokens[i], ">=") ||
			tokenIs(t.src, tokens[i], "&&") || tokenIs(t.src, tokens[i], "||") ||
			tokenIs(t.src, tokens[i], "true") || tokenIs(t.src, tokens[i], "false") {
			boolean = true
			break
		}
	}
	if boolean {
		t.expression(tokens)
		return
	}
	t.out = append(t.out, '(')
	t.expression(tokens)
	t.out = append(t.out, ")!=0"...)
}

func (t *translator) expression(tokens []token) {
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tokenIs(t.src, tok, "sizeof") || tokenIs(t.src, tok, "_Alignof") {
			end := matchingToken(t.src, tokens, i+1, "(", ")")
			if end > i+2 {
				if typeID, ok := t.typeFromTokens(tokens[i+2 : end]); ok {
					if tokenIs(t.src, tok, "sizeof") {
						t.appendDecimal(t.typeSize(typeID))
					} else {
						t.appendDecimal(t.typeAlign(typeID))
					}
					i = end
					continue
				}
			}
		}
		if tokenIs(t.src, tok, "__builtin_offsetof") {
			end := matchingToken(t.src, tokens, i+1, "(", ")")
			if end > i+3 {
				parts := splitTopLevel(t.src, tokens[i+2:end], ",")
				if len(parts) == 2 {
					if typeID, ok := t.typeFromTokens(parts[0]); ok {
						if offset, found := t.fieldOffset(typeID, parts[1]); found {
							t.appendDecimal(offset)
							i = end
							continue
						}
					}
				}
			}
		}
		if consumed := t.nullPointerPrefix(tokens[i:]); consumed > 0 {
			t.out = append(t.out, "nil"...)
			i += consumed - 1
			continue
		}
		text := tokenText(t.src, tok)
		if t.object && tokenKind(tok) == tokenString {
			t.emitCStringValue(text)
			continue
		} else if tokenKind(tok) == tokenNumber {
			text = trimIntegerSuffix(text)
		} else if tokenKind(tok) == tokenIdent {
			if value, ok := t.lookupValue(text); ok {
				t.appendDecimalSigned(value)
				continue
			}
		} else if tokenIs(t.src, tok, "NULL") {
			text = []byte("nil")
		} else if tokenIs(t.src, tok, "->") {
			text = []byte(".")
		} else if tokenIs(t.src, tok, "~") {
			text = []byte("^")
		} else if (tokenKind(tok) == tokenString || tokenKind(tok) == tokenChar) && len(text) > 1 && text[0] != '"' && text[0] != '\'' {
			for len(text) > 0 && text[0] != '"' && text[0] != '\'' {
				text = text[1:]
			}
		}
		t.out = append(t.out, text...)
		if i+1 < len(tokens) && expressionNeedsSpace(text, tokenText(t.src, tokens[i+1])) {
			t.out = append(t.out, ' ')
		}
	}
}

func matchingToken(src []byte, tokens []token, at int, open string, close string) int {
	if at < 0 || at >= len(tokens) || !tokenIs(src, tokens[at], open) {
		return -1
	}
	depth := 0
	for i := at; i < len(tokens); i++ {
		if tokenIs(src, tokens[i], open) {
			depth++
		} else if tokenIs(src, tokens[i], close) {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func (t *translator) typeFromTokens(tokens []token) (int, bool) {
	start, end := 0, len(tokens)
	for start < end && (tokenIs(t.src, tokens[start], "const") || tokenIs(t.src, tokens[start], "volatile") || tokenIs(t.src, tokens[start], "restrict")) {
		start++
	}
	for end > start && (tokenIs(t.src, tokens[end-1], "const") || tokenIs(t.src, tokens[end-1], "volatile") || tokenIs(t.src, tokens[end-1], "restrict")) {
		end--
	}
	tokens = tokens[start:end]
	if len(tokens) == 0 {
		return cTypeVoidID, false
	}
	pointers := 0
	for len(tokens) > 0 && tokenIs(t.src, tokens[len(tokens)-1], "*") {
		pointers++
		tokens = tokens[:len(tokens)-1]
	}
	typeID := cTypeVoidID
	ok := false
	if len(tokens) == 1 {
		name := tokenText(t.src, tokens[0])
		typeID, ok = t.lookupTypedef(name)
		if !ok {
			typeID, ok = builtinCType(name)
		}
		if !ok {
			switch {
			case textEquals(name, "long"):
				typeID, ok = scalarCType("int", false, 1)
			case textEquals(name, "unsigned"):
				typeID, ok = scalarCType("int", true, 0)
			case textEquals(name, "signed") || textEquals(name, "__signed__"):
				typeID, ok = scalarCType("int", false, 0)
			default:
				typeID, ok = scalarCType(string(name), false, 0)
			}
		}
	} else if len(tokens) == 2 && (tokenIs(t.src, tokens[0], "struct") || tokenIs(t.src, tokens[0], "union")) {
		typeID, ok = t.lookupTag(tokenText(t.src, tokens[1]))
	} else {
		unsigned := false
		longCount := 0
		base := ""
		valid := true
		for i := 0; i < len(tokens); i++ {
			switch {
			case tokenIs(t.src, tokens[i], "unsigned"):
				unsigned = true
			case tokenIs(t.src, tokens[i], "signed") || tokenIs(t.src, tokens[i], "__signed__") || tokenIs(t.src, tokens[i], "int"):
			case tokenIs(t.src, tokens[i], "long"):
				longCount++
			case tokenIs(t.src, tokens[i], "short"):
				base = "short"
			case tokenIs(t.src, tokens[i], "char"):
				base = "char"
			case tokenIs(t.src, tokens[i], "void"):
				base = "void"
			case tokenIs(t.src, tokens[i], "float"):
				base = "float"
			case tokenIs(t.src, tokens[i], "double"):
				base = "double"
			case tokenIs(t.src, tokens[i], "_Bool"):
				base = "bool"
			default:
				valid = false
			}
		}
		if valid {
			typeID, ok = scalarCType(base, unsigned, longCount)
		}
	}
	if !ok {
		return cTypeVoidID, false
	}
	for i := 0; i < pointers; i++ {
		typeID = t.pointerType(typeID)
	}
	return typeID, true
}

func scalarCType(base string, unsigned bool, longCount int) (int, bool) {
	if base == "" {
		base = "int"
	}
	switch base {
	case "void":
		return cTypeVoidID, true
	case "bool":
		return cTypeBoolID, true
	case "char":
		if unsigned {
			return cTypeUint8ID, true
		}
		return cTypeInt8ID, true
	case "short":
		if unsigned {
			return cTypeUint16ID, true
		}
		return cTypeInt16ID, true
	case "float":
		return cTypeFloat32ID, true
	case "double":
		return cTypeFloat64ID, true
	case "int":
		if longCount > 0 {
			if unsigned {
				return cTypeUint64ID, true
			}
			return cTypeInt64ID, true
		}
		if unsigned {
			return cTypeUint32ID, true
		}
		return cTypeInt32ID, true
	}
	return cTypeVoidID, false
}

func (t *translator) fieldOffset(typeID int, tokens []token) (int, bool) {
	info := t.typeInfo(typeID)
	if info.kind != cTypeStruct && info.kind != cTypeUnion || len(tokens) != 1 || tokenKind(tokens[0]) != tokenIdent {
		return 0, false
	}
	name := tokenText(t.src, tokens[0])
	for i := 0; i < info.fieldCount; i++ {
		field := t.fields[info.fieldStart+i]
		if textEquals(name, field.name) {
			return field.offset, true
		}
	}
	return 0, false
}

func (t *translator) nullPointerPrefix(tokens []token) int {
	if len(tokens) < 5 || !tokenIs(t.src, tokens[0], "(") {
		return 0
	}
	outer := matchingToken(t.src, tokens, 0, "(", ")")
	if outer < 0 {
		return 0
	}
	inside := tokens[1:outer]
	for len(inside) >= 2 && tokenIs(t.src, inside[0], "(") {
		close := matchingToken(t.src, inside, 0, "(", ")")
		if close <= 1 || close+1 >= len(inside) {
			break
		}
		if typeID, ok := t.typeFromTokens(inside[1:close]); ok && t.typeInfo(typeID).kind == cTypePointer {
			if value, valid := integerTokenValue(t.src, inside[close+1:]); valid && value == 0 {
				return outer + 1
			}
		}
		break
	}
	return 0
}

func (t *translator) emitCStringValue(text []byte) {
	value, ok := decodeCString(text)
	if !ok {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.out = append(t.out, '"')
	hex := "0123456789abcdef"
	for i := 0; i < len(value); i++ {
		t.out = append(t.out, '\\', 'x', hex[value[i]>>4], hex[value[i]&15])
	}
	t.out = append(t.out, '\\', 'x', '0', '0', '"')
}

func decodeCString(text []byte) ([]byte, bool) {
	start := 0
	for start < len(text) && text[start] != '"' {
		start++
	}
	if start >= len(text) || len(text) < start+2 || text[len(text)-1] != '"' {
		return nil, false
	}
	out := make([]byte, 0, len(text)-start-2)
	for i := start + 1; i < len(text)-1; i++ {
		ch := text[i]
		if ch != '\\' {
			out = append(out, ch)
			continue
		}
		i++
		if i >= len(text)-1 {
			return nil, false
		}
		ch = text[i]
		switch ch {
		case 'a':
			out = append(out, 7)
		case 'b':
			out = append(out, 8)
		case 'f':
			out = append(out, 12)
		case 'n':
			out = append(out, 10)
		case 'r':
			out = append(out, 13)
		case 't':
			out = append(out, 9)
		case 'v':
			out = append(out, 11)
		case '\\', '\'', '"', '?':
			out = append(out, ch)
		default:
			if ch < '0' || ch > '7' {
				return nil, false
			}
			value := int(ch - '0')
			for count := 1; count < 3 && i+1 < len(text)-1 && text[i+1] >= '0' && text[i+1] <= '7'; count++ {
				i++
				value = value*8 + int(text[i]-'0')
			}
			out = append(out, byte(value))
		}
	}
	return out, true
}

func expressionNeedsSpace(left []byte, right []byte) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	a, b := left[len(left)-1], right[0]
	return isIdentPart(a) && isIdentPart(b)
}

func trimIntegerSuffix(text []byte) []byte {
	end := len(text)
	for end > 0 {
		c := text[end-1]
		if c != 'u' && c != 'U' && c != 'l' && c != 'L' {
			break
		}
		end--
	}
	return text[:end]
}

func (t *translator) findClosing(close string) int {
	depth := 0
	open := "("
	if close == "]" {
		open = "["
	}
	for i := t.pos; i < len(t.tokens); i++ {
		if tokenIs(t.src, t.tokens[i], open) {
			depth++
		} else if tokenIs(t.src, t.tokens[i], close) {
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func (t *translator) findAtDepth(text string) int {
	paren, bracket, brace := 0, 0, 0
	for i := t.pos; i < len(t.tokens); i++ {
		tok := t.tokens[i]
		if paren == 0 && bracket == 0 && brace == 0 && tokenIs(t.src, tok, text) {
			return i
		}
		switch {
		case tokenIs(t.src, tok, "("):
			paren++
		case tokenIs(t.src, tok, ")"):
			paren--
		case tokenIs(t.src, tok, "["):
			bracket++
		case tokenIs(t.src, tok, "]"):
			bracket--
		case tokenIs(t.src, tok, "{"):
			brace++
		case tokenIs(t.src, tok, "}"):
			if brace == 0 {
				return -1
			}
			brace--
		}
	}
	return -1
}

func (t *translator) isTypeStart() bool {
	if t.kind() != tokenIdent {
		return false
	}
	tok := t.tokens[t.pos]
	if isTypeToken(t.src, tok) {
		return true
	}
	name := tokenText(t.src, tok)
	if _, known := builtinCType(name); known {
		return true
	}
	if _, known := t.lookupTypedef(name); known {
		return true
	}
	for i := 0; i < len(t.opaqueTypes); i++ {
		if textEquals(name, t.opaqueTypes[i].name) {
			return true
		}
	}
	return false
}

func isTypeToken(src []byte, tok token) bool {
	return tokenIs(src, tok, "auto") || tokenIs(src, tok, "register") || tokenIs(src, tok, "static") ||
		tokenIs(src, tok, "extern") || tokenIs(src, tok, "typedef") || tokenIs(src, tok, "const") ||
		tokenIs(src, tok, "volatile") || tokenIs(src, tok, "restrict") || tokenIs(src, tok, "inline") ||
		tokenIs(src, tok, "void") || tokenIs(src, tok, "char") || tokenIs(src, tok, "short") ||
		tokenIs(src, tok, "int") || tokenIs(src, tok, "long") || tokenIs(src, tok, "float") ||
		tokenIs(src, tok, "double") || tokenIs(src, tok, "signed") || tokenIs(src, tok, "unsigned") ||
		tokenIs(src, tok, "_Bool") || tokenIs(src, tok, "_Atomic") || tokenIs(src, tok, "struct") ||
		tokenIs(src, tok, "union") || tokenIs(src, tok, "enum") || tokenIs(src, tok, "__extension__") || tokenIs(src, tok, "__signed__")
}

func splitTopLevel(src []byte, tokens []token, separator string) [][]token {
	var out [][]token
	start := 0
	paren, bracket, brace := 0, 0, 0
	for i := 0; i < len(tokens); i++ {
		if paren == 0 && bracket == 0 && brace == 0 && tokenIs(src, tokens[i], separator) {
			out = append(out, tokens[start:i])
			start = i + 1
			continue
		}
		switch {
		case tokenIs(src, tokens[i], "("):
			paren++
		case tokenIs(src, tokens[i], ")"):
			paren--
		case tokenIs(src, tokens[i], "["):
			bracket++
		case tokenIs(src, tokens[i], "]"):
			bracket--
		case tokenIs(src, tokens[i], "{"):
			brace++
		case tokenIs(src, tokens[i], "}"):
			brace--
		}
	}
	out = append(out, tokens[start:])
	return out
}
