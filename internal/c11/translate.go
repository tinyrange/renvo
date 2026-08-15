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
	storageStatic
	storageThread
	storageExtern
	storageTypedef
	storageInvalid
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
	variadic    bool
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
	kind       int
	count      int
	params     []parameter
	variadic   bool
	qualifiers int
}

type initializerPath struct {
	start  int
	count  int
	typeID int
}

type initializerStep struct {
	field int
	index int
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
	objects        []cObjectName
	tentatives     []cObjectName
	definitions    []cObjectName
	functions      []cFunctionName
	variadicCalls  []cVariadicCall
	functionParams []int
	typeSerial     int
	dataModel      int
	pointerSize    int
	packageHeader  int
	usesUnsafe     bool
	staticOut      []byte
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
	t.packageHeader = len(t.out)
	if len(prelude) > 0 && !t.translateSource(prelude) {
		return Result{Ok: false, Error: t.err, ErrorAt: -1}
	}
	if !t.translateSource(src) {
		return Result{Ok: false, Error: t.err, ErrorAt: t.errorAt}
	}
	t.emitPendingDeclarations()
	if t.usesUnsafe {
		declaration := []byte("import __c_unsafe \"unsafe\"\n")
		at := t.packageHeader
		t.out = append(t.out, make([]byte, len(declaration))...)
		copy(t.out[at+len(declaration):], t.out[at:len(t.out)-len(declaration)])
		copy(t.out[at:], declaration)
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
	if t.typeInfo(base).qualifiers&cQualifierRestrict != 0 && t.typeInfo(base).kind != cTypePointer {
		t.fail(TranslateErrDeclaration)
		return
	}
	if storage == storageThread {
		t.fail(TranslateErrUnsupported)
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
			if t.object && storage != storageStatic {
				t.rememberFunction(decl, false)
			}
			return // Otherwise a C or Go definition in the package satisfies it.
		}
		if !t.currentIs("{") {
			t.fail(TranslateErrDeclaration)
			return
		}
		if decl.variadic {
			t.fail(TranslateErrUnsupported)
			return
		}
		if !t.rememberFunction(decl, true) {
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
		for i := 0; i < len(decls); i++ {
			t.rememberObject(tokenText(t.src, decls[i].name), decls[i].typeID)
		}
		return
	}
	if storage == storageOther {
		t.fail(TranslateErrDeclaration)
		return
	}
	for i := 0; i < len(decls); i++ {
		if !t.fileVariable(decls[i]) {
			t.fail(TranslateErrDeclaration)
			return
		}
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
	bitActive := false
	bitSize := 0
	bitAlign := 0
	bitOffset := 0
	bitUsed := 0
	bitCarrier := ""
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
			field := declarator{typeID: fieldBase}
			valid := true
			if !t.currentIs(":") {
				field, valid = t.parseDeclarator(fieldBase, false)
			}
			if !valid {
				return cTypeVoidID, false
			}
			fieldInfo := t.typeInfo(field.typeID)
			alignment := t.typeAlign(field.typeID)
			if t.take(":") {
				if fieldInfo.kind != cTypeInt && fieldInfo.kind != cTypeUint && fieldInfo.kind != cTypeBool {
					return cTypeVoidID, false
				}
				end := t.findFieldSeparator()
				if end <= t.pos {
					return cTypeVoidID, false
				}
				width, widthOK := t.constantExpression(t.tokens[t.pos:end])
				if !widthOK || width < 0 || width > fieldInfo.size*8 || width == 0 && len(tokenText(t.src, field.name)) != 0 {
					return cTypeVoidID, false
				}
				t.pos = end
				if alignment > maxAlign {
					maxAlign = alignment
				}
				if union {
					bitActive = false
					if width > 0 && fieldInfo.size > maxSize {
						maxSize = fieldInfo.size
					}
					if len(tokenText(t.src, field.name)) > 0 {
						aggregateFields = append(aggregateFields, cField{
							name: string(tokenText(t.src, field.name)), typeID: field.typeID,
							offset: 0, bitWidth: width,
						})
					}
				} else if width == 0 {
					bitActive = false
					offset = alignC(offset, alignment)
				} else {
					storageType := t.unsignedStorageType(fieldInfo.size)
					if !bitActive || bitSize != fieldInfo.size || bitAlign != alignment || width > bitSize*8-bitUsed {
						offset = alignC(offset, alignment)
						bitOffset = offset
						offset += fieldInfo.size
						bitSize = fieldInfo.size
						bitAlign = alignment
						bitUsed = 0
						bitActive = true
						t.typeSerial++
						bitCarrier = "__c_bits_" + decimalString(t.typeSerial)
						aggregateFields = append(aggregateFields, cField{
							name: bitCarrier, typeID: storageType, offset: bitOffset, emit: true, synthetic: true,
						})
					}
					if len(tokenText(t.src, field.name)) > 0 {
						aggregateFields = append(aggregateFields, cField{
							name: string(tokenText(t.src, field.name)), typeID: field.typeID,
							offset: bitOffset, bitOffset: bitUsed, bitWidth: width, carrier: bitCarrier,
						})
					}
					bitUsed += width
					if bitUsed == bitSize*8 {
						bitActive = false
					}
				}
				if !t.take(",") {
					break
				}
				continue
			}
			bitActive = false
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
	t.emitAggregateAccessors(typeID)
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

func (t *translator) rememberObject(name []byte, typeID int) {
	t.rememberAliasedObject(name, string(name), typeID)
}

func (t *translator) rememberAliasedObject(name []byte, goName string, typeID int) {
	if len(name) == 0 {
		return
	}
	t.objects = append(t.objects, cObjectName{name: string(name), goName: goName, typeID: typeID})
}

func (t *translator) rememberFunction(decl declarator, definition bool) bool {
	name := string(tokenText(t.src, decl.name))
	for i := 0; i < len(t.functions); i++ {
		fn := &t.functions[i]
		if fn.name != name {
			continue
		}
		if !t.compatibleParameterType(fn.resultType, decl.typeID) || fn.paramCount != len(decl.params) || fn.variadic != decl.variadic || definition && fn.defined {
			return false
		}
		for j := 0; j < fn.paramCount; j++ {
			if !t.compatibleParameterType(t.functionParams[fn.paramStart+j], decl.params[j].typeID) {
				return false
			}
		}
		if definition {
			fn.defined = true
		}
		return true
	}
	start := len(t.functionParams)
	for i := 0; i < len(decl.params); i++ {
		t.functionParams = append(t.functionParams, decl.params[i].typeID)
	}
	t.functions = append(t.functions, cFunctionName{
		name: name, resultType: decl.typeID, paramStart: start,
		paramCount: len(decl.params), variadic: decl.variadic, defined: definition,
	})
	return true
}

func (t *translator) compatibleParameterType(left int, right int) bool {
	if left == right {
		return true
	}
	a, b := t.typeInfo(left), t.typeInfo(right)
	return a.kind == b.kind && a.base == b.base && a.count == b.count &&
		a.fieldStart == b.fieldStart && a.paramStart == b.paramStart &&
		a.paramCount == b.paramCount && a.variadic == b.variadic
}

func (t *translator) fileVariable(decl declarator) bool {
	name := string(tokenText(t.src, decl.name))
	if !t.completeInitializerArray(&decl) {
		return false
	}
	t.rememberObject([]byte(name), decl.typeID)
	for i := 0; i < len(t.definitions); i++ {
		if t.definitions[i].name != name {
			continue
		}
		return t.compatibleType(t.definitions[i].typeID, decl.typeID) && len(decl.initializer) == 0
	}
	tentative := -1
	for i := 0; i < len(t.tentatives); i++ {
		if t.tentatives[i].name == name {
			if !t.compatibleType(t.tentatives[i].typeID, decl.typeID) {
				return false
			}
			if t.typeInfo(t.tentatives[i].typeID).kind == cTypeArray && t.typeInfo(t.tentatives[i].typeID).count == 0 {
				t.tentatives[i].typeID = decl.typeID
			}
			tentative = i
			break
		}
	}
	if len(decl.initializer) == 0 {
		if tentative < 0 {
			t.tentatives = append(t.tentatives, cObjectName{name: name, typeID: decl.typeID})
		}
		return true
	}
	if tentative >= 0 {
		copy(t.tentatives[tentative:], t.tentatives[tentative+1:])
		t.tentatives = t.tentatives[:len(t.tentatives)-1]
	}
	t.definitions = append(t.definitions, cObjectName{name: name, typeID: decl.typeID})
	t.emitVariable(decl)
	return true
}

func (t *translator) completeInitializerArray(decl *declarator) bool {
	info := t.typeInfo(decl.typeID)
	if info.kind != cTypeArray || info.count != 0 {
		return true
	}
	if len(decl.initializer) < 2 || !tokenIs(t.src, decl.initializer[0], "{") || !tokenIs(t.src, decl.initializer[len(decl.initializer)-1], "}") {
		return len(decl.initializer) == 0
	}
	items := splitTopLevel(t.src, decl.initializer[1:len(decl.initializer)-1], ",")
	next, count := 0, 0
	for i := 0; i < len(items); i++ {
		if len(items[i]) == 0 {
			continue
		}
		if tokenIs(t.src, items[i][0], "[") {
			close := matchingToken(t.src, items[i], 0, "[", "]")
			if close <= 1 {
				return false
			}
			index, ok := t.constantExpression(items[i][1:close])
			if !ok || index < 0 {
				return false
			}
			next = index
		}
		next++
		if next > count {
			count = next
		}
	}
	if count == 0 {
		return false
	}
	decl.typeID = t.arrayType(info.base, count)
	return true
}

func (t *translator) emitPendingDeclarations() {
	t.out = append(t.out, t.staticOut...)
	for i := 0; i < len(t.functions); i++ {
		if !t.functions[i].defined && !t.functions[i].variadic {
			t.emitForeignFunctionName(t.functions[i])
		}
	}
	for i := 0; i < len(t.variadicCalls); i++ {
		t.emitVariadicForeignFunction(t.variadicCalls[i])
	}
	for i := 0; i < len(t.tentatives); i++ {
		typeID := t.tentatives[i].typeID
		info := t.typeInfo(typeID)
		if info.kind == cTypeArray && info.count == 0 {
			typeID = t.arrayType(info.base, 1)
		}
		t.out = append(t.out, "var "...)
		t.out = append(t.out, t.tentatives[i].name...)
		t.out = append(t.out, ' ')
		t.emitType(typeID)
		t.out = append(t.out, ';', '\n')
	}
}

func (t *translator) compatibleType(left int, right int) bool {
	if left == right {
		return true
	}
	a, b := t.typeInfo(left), t.typeInfo(right)
	return a.kind == cTypeArray && b.kind == cTypeArray && a.base == b.base && (a.count == 0 || b.count == 0)
}

func (t *translator) lookupObject(name []byte) (cObjectName, bool) {
	for i := len(t.objects) - 1; i >= 0; i-- {
		if textEquals(name, t.objects[i].name) {
			return t.objects[i], true
		}
	}
	return cObjectName{}, false
}

func (t *translator) lookupFunction(name []byte) (int, bool) {
	for i := len(t.functions) - 1; i >= 0; i-- {
		if textEquals(name, t.functions[i].name) {
			return i, true
		}
	}
	return 0, false
}

func (t *translator) lookupField(typeID int, name []byte) (cField, bool) {
	info := t.typeInfo(typeID)
	if info.kind != cTypeStruct && info.kind != cTypeUnion {
		return cField{}, false
	}
	for i := 0; i < info.fieldCount; i++ {
		field := t.fields[info.fieldStart+i]
		if textEquals(name, field.name) {
			return field, true
		}
	}
	return cField{}, false
}

func (t *translator) unsignedStorageType(size int) int {
	switch size {
	case 1:
		return cTypeUint8ID
	case 2:
		return cTypeUint16ID
	case 4:
		return cTypeUint32ID
	default:
		return cTypeUint64ID
	}
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

func (t *translator) emitAggregateAccessors(typeID int) {
	info := t.typeInfo(typeID)
	for i := 0; i < info.fieldCount; i++ {
		field := t.fields[info.fieldStart+i]
		if field.name == "" || field.emit && info.kind != cTypeUnion {
			continue
		}
		if field.bitWidth == 0 {
			if info.kind != cTypeUnion {
				continue
			}
			t.usesUnsafe = true
			t.out = append(t.out, "func(p *"...)
			t.out = append(t.out, info.goName...)
			t.out = append(t.out, ")__c_ptr_"...)
			t.out = append(t.out, field.name...)
			t.out = append(t.out, "()*"...)
			t.emitType(field.typeID)
			t.out = append(t.out, "{return (*"...)
			t.emitType(field.typeID)
			t.out = append(t.out, ")(__c_unsafe.Pointer(p))}\n"...)
			continue
		}
		t.emitBitfieldAccessors(typeID, field)
	}
}

func (t *translator) emitBitfieldAccessors(typeID int, field cField) {
	info := t.typeInfo(typeID)
	fieldInfo := t.typeInfo(field.typeID)
	storageType := t.unsignedStorageType(fieldInfo.size)
	bits := fieldInfo.size * 8
	t.out = append(t.out, "func(p *"...)
	t.out = append(t.out, info.goName...)
	t.out = append(t.out, ")__c_get_"...)
	t.out = append(t.out, field.name...)
	t.out = append(t.out, "() "...)
	if fieldInfo.kind == cTypeBool {
		t.out = append(t.out, "uint8"...)
	} else {
		t.emitType(field.typeID)
	}
	t.out = append(t.out, "{return "...)
	if fieldInfo.kind == cTypeInt {
		t.emitType(field.typeID)
		t.out = append(t.out, '(')
		t.emitBitfieldCarrier(info, field, storageType)
		t.out = append(t.out, "<<"...)
		t.appendDecimal(bits - field.bitOffset - field.bitWidth)
		t.out = append(t.out, ")>>"...)
		t.appendDecimal(bits - field.bitWidth)
	} else {
		t.out = append(t.out, '(')
		t.emitBitfieldCarrier(info, field, storageType)
		if field.bitOffset > 0 {
			t.out = append(t.out, ">>"...)
			t.appendDecimal(field.bitOffset)
		}
		t.out = append(t.out, ")&"...)
		t.emitBitfieldMask(storageType, field.bitWidth, bits)
	}
	t.out = append(t.out, "}\nfunc(p *"...)
	t.out = append(t.out, info.goName...)
	t.out = append(t.out, ")__c_set_"...)
	t.out = append(t.out, field.name...)
	t.out = append(t.out, "(v "...)
	if fieldInfo.kind == cTypeBool {
		t.out = append(t.out, "uint8"...)
	} else {
		t.emitType(field.typeID)
	}
	t.out = append(t.out, "){"...)
	if info.kind == cTypeUnion {
		t.usesUnsafe = true
		t.out = append(t.out, "q:=(*"...)
		t.emitType(storageType)
		t.out = append(t.out, ")(__c_unsafe.Pointer(p));*q="...)
		t.emitBitfieldUpdated("*q", storageType, field, bits)
	} else {
		t.out = append(t.out, "p."...)
		t.out = append(t.out, field.carrier...)
		t.out = append(t.out, '=')
		t.emitBitfieldUpdated("p."+field.carrier, storageType, field, bits)
	}
	t.out = append(t.out, "}\n"...)
}

func (t *translator) emitBitfieldCarrier(info cTypeInfo, field cField, storageType int) {
	if info.kind == cTypeUnion {
		t.usesUnsafe = true
		t.out = append(t.out, "*(*"...)
		t.emitType(storageType)
		t.out = append(t.out, ")(__c_unsafe.Pointer(p))"...)
		return
	}
	t.out = append(t.out, "p."...)
	t.out = append(t.out, field.carrier...)
}

func (t *translator) emitBitfieldMask(storageType int, width int, bits int) {
	if width == bits {
		t.out = append(t.out, '^')
		t.emitType(storageType)
		t.out = append(t.out, "(0)"...)
		return
	}
	t.out = append(t.out, "(("...)
	t.emitType(storageType)
	t.out = append(t.out, "(1)<<"...)
	t.appendDecimal(width)
	t.out = append(t.out, ")-1)"...)
}

func (t *translator) emitBitfieldUpdated(carrier string, storageType int, field cField, bits int) {
	t.out = append(t.out, '(')
	t.out = append(t.out, carrier...)
	t.out = append(t.out, "&^("...)
	t.emitBitfieldMask(storageType, field.bitWidth, bits)
	if field.bitOffset > 0 {
		t.out = append(t.out, "<<"...)
		t.appendDecimal(field.bitOffset)
	}
	t.out = append(t.out, "))|(("...)
	t.emitType(storageType)
	t.out = append(t.out, "(v)&"...)
	t.emitBitfieldMask(storageType, field.bitWidth, bits)
	t.out = append(t.out, ')')
	if field.bitOffset > 0 {
		t.out = append(t.out, "<<"...)
		t.appendDecimal(field.bitOffset)
	}
	t.out = append(t.out, ')')
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

func (t *translator) emitForeignFunctionName(fn cFunctionName) {
	t.emitForeignFunction(fn.name, fn.name, fn.resultType, fn.paramStart, fn.paramCount)
}

func (t *translator) emitForeignFunction(symbol string, goName string, resultType int, paramStart int, paramCount int) {
	t.out = append(t.out, "// renvo:linkstatic libc,"...)
	t.out = append(t.out, symbol...)
	t.out = append(t.out, '\n', 'f', 'u', 'n', 'c', ' ')
	t.out = append(t.out, goName...)
	t.out = append(t.out, '(')
	for i := 0; i < paramCount; i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, 'p')
		t.appendDecimal(i)
		t.out = append(t.out, ' ')
		t.emitForeignType(t.functionParams[paramStart+i])
	}
	t.out = append(t.out, ')')
	if resultType != cTypeVoidID {
		t.out = append(t.out, ' ')
		t.emitType(resultType)
	}
	t.out = append(t.out, '{')
	if t.typeInfo(resultType).kind == cTypePointer {
		t.out = append(t.out, "return nil"...)
	} else if resultType != cTypeVoidID {
		t.out = append(t.out, "return 0"...)
	}
	t.out = append(t.out, '}', '\n')
}

func (t *translator) emitVariadicForeignFunction(call cVariadicCall) {
	fn := t.functions[call.function]
	t.emitForeignFunction(fn.name, call.goName, fn.resultType, call.paramStart, call.paramCount)
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
	base := t.typeInfo(info.base)
	if info.kind == cTypePointer && base.kind == cTypeInt && base.size == 1 {
		t.out = append(t.out, "string"...)
		return
	}
	t.emitType(typeID)
}

func (t *translator) parseType() (int, int, bool) {
	storage := storageNone
	qualifiers := 0
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
		case t.currentIs("static"):
			if storage == storageNone {
				storage = storageStatic
			} else if storage != storageThread {
				storage = storageInvalid
			}
			t.pos++
		case t.currentIs("_Thread_local"):
			if storage == storageNone || storage == storageStatic || storage == storageExtern {
				storage = storageThread
			} else {
				storage = storageInvalid
			}
			t.pos++
		case t.currentIs("register") || t.currentIs("auto"):
			if storage == storageNone {
				storage = storageOther
			} else {
				storage = storageInvalid
			}
			t.pos++
		case t.currentIs("inline"):
			t.pos++
		case t.currentIs("extern"):
			if storage == storageNone {
				storage = storageExtern
			} else if storage != storageThread {
				storage = storageInvalid
			}
			t.pos++
		case t.currentIs("typedef"):
			if storage == storageNone {
				storage = storageTypedef
			} else {
				storage = storageInvalid
			}
			t.pos++
		case t.currentIs("const"):
			qualifiers |= cQualifierConst
			t.pos++
		case t.currentIs("volatile"):
			qualifiers |= cQualifierVolatile
			t.pos++
		case t.currentIs("restrict"):
			qualifiers |= cQualifierRestrict
			t.pos++
		case t.currentIs("_Atomic"):
			qualifiers |= cQualifierAtomic
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
	if !seen || storage == storageInvalid {
		return cTypeVoidID, storage, false
	}
	if hasTypeID {
		return t.qualifiedType(typeID, qualifiers), storage, true
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
	return t.qualifiedType(typeID, qualifiers), storage, true
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
			result.typeID = t.qualifiedType(t.pointerType(result.typeID), ops[i].qualifiers)
		case declaratorArray:
			result.typeID = t.arrayType(result.typeID, ops[i].count)
		case declaratorFunction:
			if t.typeInfo(result.typeID).kind == cTypeArray || t.typeInfo(result.typeID).kind == cTypeFunction {
				return result, false
			}
			result.typeID = t.functionType(result.typeID, ops[i].params, ops[i].variadic)
		}
	}
	info := t.typeInfo(result.typeID)
	if info.kind == cTypeFunction {
		if !allowFunction || len(ops) == 0 || ops[0].kind != declaratorFunction {
			return result, false
		}
		result.function = true
		result.params = ops[0].params
		result.variadic = ops[0].variadic
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
	var pointerQualifiers []int
	for t.take("*") {
		qualifiers := 0
		for t.currentIs("const") || t.currentIs("volatile") || t.currentIs("restrict") {
			if t.currentIs("const") {
				qualifiers |= cQualifierConst
			} else if t.currentIs("volatile") {
				qualifiers |= cQualifierVolatile
			} else {
				qualifiers |= cQualifierRestrict
			}
			t.pos++
		}
		pointerQualifiers = append(pointerQualifiers, qualifiers)
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
			if end < t.pos {
				return name, ops, false
			}
			count := 0
			if end > t.pos {
				var valid bool
				count, valid = t.constantExpression(t.tokens[t.pos:end])
				if !valid || count < 1 {
					return name, ops, false
				}
			}
			ops = append(ops, declaratorOp{kind: declaratorArray, count: count})
			t.pos = end + 1
			continue
		}
		if t.take("(") {
			params, variadic, valid := t.parseParameterList()
			if !valid {
				return name, ops, false
			}
			ops = append(ops, declaratorOp{kind: declaratorFunction, params: params, variadic: variadic})
			continue
		}
		break
	}
	for i := len(pointerQualifiers) - 1; i >= 0; i-- {
		ops = append(ops, declaratorOp{kind: declaratorPointer, qualifiers: pointerQualifiers[i]})
	}
	return name, ops, true
}

func (t *translator) parseParameterList() ([]parameter, bool, bool) {
	if t.take(")") {
		return nil, false, true
	}
	if t.currentIs("void") && t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], ")") {
		t.pos += 2
		return nil, false, true
	}
	var params []parameter
	for {
		paramBase, _, ok := t.parseType()
		if !ok {
			return params, false, false
		}
		if t.typeInfo(paramBase).qualifiers&cQualifierRestrict != 0 && t.typeInfo(paramBase).kind != cTypePointer {
			return params, false, false
		}
		name, ops, ok := t.parseDeclaratorOps(true)
		if !ok {
			return params, false, false
		}
		paramType := paramBase
		for i := len(ops) - 1; i >= 0; i-- {
			switch ops[i].kind {
			case declaratorPointer:
				paramType = t.qualifiedType(t.pointerType(paramType), ops[i].qualifiers)
			case declaratorArray:
				paramType = t.arrayType(paramType, ops[i].count)
			case declaratorFunction:
				paramType = t.functionType(paramType, ops[i].params, ops[i].variadic)
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
			return params, false, true
		}
		if !t.take(",") {
			return params, false, false
		}
		if t.take("...") {
			if !t.take(")") {
				return params, false, false
			}
			return params, true, true
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
	if t.object && storage != storageStatic {
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
	objectMark := len(t.objects)
	for i := 0; i < len(decl.params); i++ {
		t.rememberObject(tokenText(t.src, decl.params[i].name), decl.params[i].typeID)
	}
	t.block()
	t.objects = t.objects[:objectMark]
	t.out = append(t.out, '\n')
}

func (t *translator) emitVariable(decl declarator) {
	t.rememberObject(tokenText(t.src, decl.name), decl.typeID)
	t.emitVariableName(string(tokenText(t.src, decl.name)), decl)
}

func (t *translator) emitVariableName(name string, decl declarator) {
	t.out = append(t.out, "var "...)
	t.out = append(t.out, name...)
	t.out = append(t.out, ' ')
	t.emitType(decl.typeID)
	if len(decl.initializer) > 0 {
		t.out = append(t.out, '=')
		kind := t.typeInfo(decl.typeID).kind
		if (kind == cTypeArray || kind == cTypeStruct) && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitInitializerValue(decl.typeID, decl.initializer)
		} else if kind == cTypeUnion && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitUnionInitializer(decl.typeID, decl.initializer)
		} else {
			t.expression(decl.initializer)
		}
	}
	t.out = append(t.out, ';', '\n')
}

func (t *translator) emitStaticLocal(decl declarator) {
	t.typeSerial++
	name := "__c_static_" + decimalString(t.typeSerial) + "_" + string(tokenText(t.src, decl.name))
	t.rememberAliasedObject(tokenText(t.src, decl.name), name, decl.typeID)
	outer := t.out
	t.out = t.staticOut
	t.emitVariableName(name, decl)
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) block() {
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	objectMark := len(t.objects)
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
	t.objects = t.objects[:objectMark]
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
	if !ok {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.typeInfo(base).qualifiers&cQualifierRestrict != 0 && t.typeInfo(base).kind != cTypePointer {
		t.fail(TranslateErrDeclaration)
		return
	}
	if storage == storageThread {
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
		if storage != storageExtern && (!t.completeInitializerArray(&decl) || t.typeInfo(decl.typeID).kind == cTypeArray && t.typeInfo(decl.typeID).count == 0) {
			t.fail(TranslateErrDeclaration)
			return
		}
		if storage == storageTypedef {
			t.rememberTypedef(string(tokenText(t.src, decl.name)), decl.typeID)
		} else if storage == storageExtern {
			if len(decl.initializer) != 0 {
				t.fail(TranslateErrDeclaration)
				return
			}
			t.rememberObject(tokenText(t.src, decl.name), decl.typeID)
		} else if storage == storageStatic {
			t.emitStaticLocal(decl)
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
	objectMark := len(t.objects)
	t.forStatementBody()
	t.objects = t.objects[:objectMark]
}

func (t *translator) forStatementBody() {
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
	objectMark := len(t.objects)
	t.switchStatementBody()
	t.objects = t.objects[:objectMark]
}

func (t *translator) switchStatementBody() {
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
	if t.rejectConstMutation(tokens) {
		return
	}
	if t.emitBitfieldMutation(tokens) {
		return
	}
	t.emitExpression(tokens)
}

func (t *translator) rejectConstMutation(tokens []token) bool {
	depth := 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "["):
			depth++
		case tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]"):
			depth--
		}
		if depth == 0 && isAssignmentToken(t.src, tokens[i]) {
			if t.typeInfo(t.lvalueType(tokens[:i])).qualifiers&cQualifierConst != 0 {
				t.fail(TranslateErrUnsupported)
				return true
			}
			return false
		}
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[0], "++") || tokenIs(t.src, tokens[0], "--")) {
		if t.typeInfo(t.lvalueType(tokens[1:])).qualifiers&cQualifierConst != 0 {
			t.fail(TranslateErrUnsupported)
			return true
		}
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[len(tokens)-1], "++") || tokenIs(t.src, tokens[len(tokens)-1], "--")) {
		if t.typeInfo(t.lvalueType(tokens[:len(tokens)-1])).qualifiers&cQualifierConst != 0 {
			t.fail(TranslateErrUnsupported)
			return true
		}
	}
	return false
}

func (t *translator) lvalueType(tokens []token) int {
	if len(tokens) == 1 && tokenKind(tokens[0]) == tokenIdent {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			return object.typeID
		}
	}
	if access, ok := t.memberAccess(tokens, 0); ok && access.end == len(tokens) {
		return access.typeID
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "*") {
		pointer := t.typeInfo(t.expressionType(tokens[1:]))
		if pointer.kind == cTypePointer {
			return pointer.base
		}
	}
	if len(tokens) > 3 && tokenKind(tokens[0]) == tokenIdent && tokenIs(t.src, tokens[1], "[") && tokenIs(t.src, tokens[len(tokens)-1], "]") {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			info := t.typeInfo(object.typeID)
			if info.kind == cTypeArray || info.kind == cTypePointer {
				return info.base
			}
		}
	}
	return cTypeVoidID
}

func (t *translator) emitExpression(tokens []token) {
	if len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close < len(tokens)-1 && !tokenIs(t.src, tokens[close+1], "{") {
			if typeID, ok := t.typeFromTokens(tokens[1:close]); ok {
				t.emitType(typeID)
				t.out = append(t.out, '(')
				t.emitExpression(tokens[close+1:])
				t.out = append(t.out, ')')
				return
			}
		}
	}
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tokenKind(tok) == tokenIdent && i+1 < len(tokens) && tokenIs(t.src, tokens[i+1], "(") {
			if function, ok := t.lookupFunction(tokenText(t.src, tok)); ok && t.functions[function].variadic && !t.functions[function].defined {
				close := matchingToken(t.src, tokens, i+1, "(", ")")
				if close > i+1 && t.emitVariadicCall(function, tokens[i+2:close]) {
					i = close
					continue
				}
			}
		}
		if tokenIs(t.src, tok, "(") {
			close := matchingToken(t.src, tokens, i, "(", ")")
			if close > i+1 && close+1 < len(tokens) && tokenIs(t.src, tokens[close+1], "{") {
				end := matchingToken(t.src, tokens, close+1, "{", "}")
				if typeID, ok := t.typeFromTokens(tokens[i+1 : close]); ok && end > close+1 {
					kind := t.typeInfo(typeID).kind
					if kind == cTypeStruct || kind == cTypeArray || kind == cTypeUnion {
						t.emitInitializerValue(typeID, tokens[close+1:end+1])
						i = end
						continue
					}
				}
			}
		}
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
		if tokenKind(tok) == tokenIdent {
			if access, ok := t.memberAccess(tokens, i); ok {
				t.out = append(t.out, access.expr...)
				i = access.end - 1
				continue
			}
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
			if object, ok := t.lookupObject(text); ok && object.goName != "" {
				text = []byte(object.goName)
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

func (t *translator) emitVariadicCall(function int, tokens []token) bool {
	fn := t.functions[function]
	args := splitTopLevel(t.src, tokens, ",")
	if len(tokens) == 0 {
		args = nil
	}
	if len(args) < fn.paramCount {
		t.fail(TranslateErrUnsupported)
		return true
	}
	start := len(t.functionParams)
	for i := 0; i < len(args); i++ {
		typeID := cTypeInt32ID
		if i < fn.paramCount {
			typeID = t.functionParams[fn.paramStart+i]
		} else {
			typeID = t.defaultArgumentType(t.expressionType(args[i]))
		}
		t.functionParams = append(t.functionParams, typeID)
	}
	callIndex := -1
	for i := 0; i < len(t.variadicCalls); i++ {
		call := t.variadicCalls[i]
		if call.function != function || call.paramCount != len(args) {
			continue
		}
		equal := true
		for j := 0; j < call.paramCount; j++ {
			if t.functionParams[call.paramStart+j] != t.functionParams[start+j] {
				equal = false
				break
			}
		}
		if equal {
			callIndex = i
			break
		}
	}
	if callIndex < 0 {
		t.typeSerial++
		callIndex = len(t.variadicCalls)
		t.variadicCalls = append(t.variadicCalls, cVariadicCall{
			function: function, paramStart: start, paramCount: len(args),
			goName: "__c_variadic_" + decimalString(t.typeSerial),
		})
	} else {
		t.functionParams = t.functionParams[:start]
	}
	call := t.variadicCalls[callIndex]
	t.out = append(t.out, call.goName...)
	t.out = append(t.out, '(')
	for i := 0; i < len(args); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		typeID := t.functionParams[call.paramStart+i]
		original := t.expressionType(args[i])
		if i >= fn.paramCount && typeID != original && t.typeInfo(typeID).kind != cTypePointer {
			t.emitType(typeID)
			t.out = append(t.out, '(')
			t.emitExpression(args[i])
			t.out = append(t.out, ')')
		} else {
			t.emitExpression(args[i])
		}
	}
	t.out = append(t.out, ')')
	return true
}

func (t *translator) defaultArgumentType(typeID int) int {
	info := t.typeInfo(typeID)
	if info.kind == cTypeBool || (info.kind == cTypeInt || info.kind == cTypeUint) && info.size < 4 {
		return cTypeInt32ID
	}
	if typeID == cTypeFloat32ID {
		return cTypeFloat64ID
	}
	if info.kind == cTypeArray {
		return t.pointerType(info.base)
	}
	if info.kind == cTypeFunction {
		return t.pointerType(typeID)
	}
	return typeID
}

func (t *translator) expressionType(tokens []token) int {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) == 0 {
		return cTypeInt32ID
	}
	if tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close < len(tokens)-1 {
			if typeID, ok := t.typeFromTokens(tokens[1:close]); ok {
				return typeID
			}
		}
	}
	if access, ok := t.memberAccess(tokens, 0); ok && access.end == len(tokens) {
		return access.typeID
	}
	if tokenKind(tokens[0]) == tokenIdent {
		if len(tokens) > 1 && tokenIs(t.src, tokens[1], "(") {
			if function, ok := t.lookupFunction(tokenText(t.src, tokens[0])); ok {
				return t.functions[function].resultType
			}
		}
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			return object.typeID
		}
		if _, ok := t.lookupValue(tokenText(t.src, tokens[0])); ok {
			return cTypeInt32ID
		}
	}
	if tokenKind(tokens[0]) == tokenString {
		return t.pointerType(cTypeInt8ID)
	}
	if tokenKind(tokens[0]) == tokenNumber {
		text := tokenText(t.src, tokens[0])
		floating := false
		float32 := false
		for i := 0; i < len(text); i++ {
			if text[i] == '.' || text[i] == 'e' || text[i] == 'E' || text[i] == 'p' || text[i] == 'P' {
				floating = true
			}
			if text[i] == 'f' || text[i] == 'F' {
				float32 = true
			}
		}
		if floating {
			if float32 {
				return cTypeFloat32ID
			}
			return cTypeFloat64ID
		}
	}
	return cTypeInt32ID
}

func (t *translator) emitAggregateInitializer(typeID int, tokens []token) {
	info := t.typeInfo(typeID)
	if len(tokens) < 2 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") ||
		(info.kind != cTypeStruct && info.kind != cTypeArray) {
		t.fail(TranslateErrUnsupported)
		return
	}
	items := splitTopLevel(t.src, tokens[1:len(tokens)-1], ",")
	fieldIndex := 0
	arrayIndex := 0
	t.out = append(t.out, '{')
	written := 0
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) == 0 {
			continue
		}
		valueType := cTypeVoidID
		value := item
		keyed := false
		if info.kind == cTypeStruct && len(item) >= 4 && tokenIs(t.src, item[0], ".") && tokenKind(item[1]) == tokenIdent && tokenIs(t.src, item[2], "=") {
			field, ok := t.lookupField(typeID, tokenText(t.src, item[1]))
			if !ok || !field.emit || field.bitWidth != 0 {
				t.fail(TranslateErrUnsupported)
				return
			}
			if written > 0 {
				t.out = append(t.out, ',')
			}
			t.out = append(t.out, field.name...)
			t.out = append(t.out, ':')
			valueType, value, keyed = field.typeID, item[3:], true
			for j := 0; j < info.fieldCount; j++ {
				if t.fields[info.fieldStart+j].name == field.name {
					fieldIndex = j + 1
					break
				}
			}
		} else if info.kind == cTypeArray && len(item) >= 4 && tokenIs(t.src, item[0], "[") {
			close := matchingToken(t.src, item, 0, "[", "]")
			if close <= 1 || close+1 >= len(item) || !tokenIs(t.src, item[close+1], "=") {
				t.fail(TranslateErrUnsupported)
				return
			}
			index, ok := t.constantExpression(item[1:close])
			if !ok || index < 0 || index >= info.count {
				t.fail(TranslateErrUnsupported)
				return
			}
			if written > 0 {
				t.out = append(t.out, ',')
			}
			t.appendDecimal(index)
			t.out = append(t.out, ':')
			valueType, value, keyed = info.base, item[close+2:], true
			arrayIndex = index + 1
		}
		if !keyed {
			if written > 0 {
				t.out = append(t.out, ',')
			}
			if info.kind == cTypeArray {
				if arrayIndex >= info.count {
					t.fail(TranslateErrUnsupported)
					return
				}
				valueType = info.base
				arrayIndex++
			} else {
				for fieldIndex < info.fieldCount && t.fields[info.fieldStart+fieldIndex].synthetic {
					fieldIndex++
				}
				if fieldIndex >= info.fieldCount {
					t.fail(TranslateErrUnsupported)
					return
				}
				field := t.fields[info.fieldStart+fieldIndex]
				valueType = field.typeID
				fieldIndex++
				t.out = append(t.out, field.name...)
				t.out = append(t.out, ':')
			}
		}
		t.emitInitializerValue(valueType, value)
		written++
	}
	t.out = append(t.out, '}')
}

func (t *translator) emitInitializerValue(typeID int, value []token) {
	info := t.typeInfo(typeID)
	if len(value) > 0 && tokenIs(t.src, value[0], "{") && (info.kind == cTypeStruct || info.kind == cTypeArray || info.kind == cTypeUnion) {
		if info.kind == cTypeUnion {
			t.emitUnionInitializer(typeID, value)
		} else {
			if t.emitMutationInitializer(typeID, value) {
				return
			}
			t.emitType(typeID)
			t.emitAggregateInitializer(typeID, value)
		}
		return
	}
	t.expression(value)
}

func (t *translator) emitMutationInitializer(typeID int, tokens []token) bool {
	info := t.typeInfo(typeID)
	items := splitTopLevel(t.src, tokens[1:len(tokens)-1], ",")
	paths := make([]initializerPath, 0, len(items))
	values := make([][]token, 0, len(items))
	steps := make([]initializerStep, 0, len(items))
	next := 0
	mutation := false
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) == 0 {
			continue
		}
		path := initializerPath{start: len(steps), typeID: typeID}
		at := 0
		if tokenIs(t.src, item[0], ".") || tokenIs(t.src, item[0], "[") {
			mutation = true
			first := true
			for at < len(item) && (tokenIs(t.src, item[at], ".") || tokenIs(t.src, item[at], "[")) {
				current := t.typeInfo(path.typeID)
				if tokenIs(t.src, item[at], ".") {
					if at+1 >= len(item) || tokenKind(item[at+1]) != tokenIdent || current.kind != cTypeStruct && current.kind != cTypeUnion {
						t.fail(TranslateErrUnsupported)
						return true
					}
					fieldIndex := -1
					for j := 0; j < current.fieldCount; j++ {
						field := t.fields[current.fieldStart+j]
						if !field.synthetic && textEquals(tokenText(t.src, item[at+1]), field.name) {
							fieldIndex = current.fieldStart + j
							break
						}
					}
					if fieldIndex < 0 {
						t.fail(TranslateErrUnsupported)
						return true
					}
					steps = append(steps, initializerStep{field: fieldIndex})
					path.typeID = t.fields[fieldIndex].typeID
					if first && info.kind == cTypeStruct {
						next = fieldIndex - info.fieldStart + 1
					}
					at += 2
				} else {
					close := matchingToken(t.src, item, at, "[", "]")
					if close <= at+1 || current.kind != cTypeArray {
						t.fail(TranslateErrUnsupported)
						return true
					}
					index, ok := t.constantExpression(item[at+1 : close])
					if !ok || index < 0 || index >= current.count {
						t.fail(TranslateErrUnsupported)
						return true
					}
					steps = append(steps, initializerStep{field: -1, index: index})
					path.typeID = current.base
					if first && info.kind == cTypeArray {
						next = index + 1
					}
					at = close + 1
				}
				first = false
			}
			if at >= len(item) || !tokenIs(t.src, item[at], "=") {
				t.fail(TranslateErrUnsupported)
				return true
			}
			at++
		} else if info.kind == cTypeArray {
			if next >= info.count {
				t.fail(TranslateErrUnsupported)
				return true
			}
			steps = append(steps, initializerStep{field: -1, index: next})
			path.typeID = info.base
			next++
		} else {
			for next < info.fieldCount && t.fields[info.fieldStart+next].synthetic {
				next++
			}
			if next >= info.fieldCount {
				t.fail(TranslateErrUnsupported)
				return true
			}
			fieldIndex := info.fieldStart + next
			steps = append(steps, initializerStep{field: fieldIndex})
			path.typeID = t.fields[fieldIndex].typeID
			if t.fields[fieldIndex].bitWidth > 0 {
				mutation = true
			}
			next++
		}
		path.count = len(steps) - path.start
		paths = append(paths, path)
		values = append(values, item[at:])
	}
	if !mutation {
		return false
	}
	t.typeSerial++
	name := "__c_aggregate_init_" + decimalString(t.typeSerial)
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	for i := 0; i < len(paths); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.emitInitializerValue(paths[i].typeID, values[i])
	}
	t.out = append(t.out, ')')
	outer := t.out
	t.out = t.staticOut
	t.out = append(t.out, "func "...)
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	for i := 0; i < len(paths); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, 'v')
		t.appendDecimal(i)
		t.out = append(t.out, ' ')
		last := steps[paths[i].start+paths[i].count-1]
		if last.field >= 0 && t.typeInfo(paths[i].typeID).kind == cTypeBool && t.fields[last.field].bitWidth > 0 {
			t.out = append(t.out, "uint8"...)
		} else {
			t.emitType(paths[i].typeID)
		}
	}
	t.out = append(t.out, ") "...)
	t.emitType(typeID)
	t.out = append(t.out, "{var p "...)
	t.emitType(typeID)
	t.out = append(t.out, ';')
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		last := steps[path.start+path.count-1]
		if last.field >= 0 && t.fields[last.field].bitWidth > 0 {
			t.emitInitializerPath(steps[path.start:path.start+path.count-1], typeID)
			t.out = append(t.out, ".__c_set_"...)
			t.out = append(t.out, t.fields[last.field].name...)
			t.out = append(t.out, "(v"...)
			t.appendDecimal(i)
			t.out = append(t.out, ");"...)
		} else {
			t.emitInitializerPath(steps[path.start:path.start+path.count], typeID)
			t.out = append(t.out, "=v"...)
			t.appendDecimal(i)
			t.out = append(t.out, ';')
		}
	}
	t.out = append(t.out, "return p}\n"...)
	t.staticOut = t.out
	t.out = outer
	return true
}

func (t *translator) emitInitializerPath(steps []initializerStep, typeID int) {
	outer := t.out
	t.out = nil
	t.out = append(t.out, 'p')
	for i := 0; i < len(steps); i++ {
		step := steps[i]
		if step.field < 0 {
			t.out = append(t.out, '[')
			t.appendDecimal(step.index)
			t.out = append(t.out, ']')
			typeID = t.typeInfo(typeID).base
			continue
		}
		field := t.fields[step.field]
		if t.typeInfo(typeID).kind == cTypeUnion {
			receiver := t.out
			t.out = append([]byte("(*"), receiver...)
			t.out = append(t.out, ".__c_ptr_"...)
			t.out = append(t.out, field.name...)
			t.out = append(t.out, "())"...)
		} else {
			t.out = append(t.out, '.')
			t.out = append(t.out, field.name...)
		}
		typeID = field.typeID
	}
	outer = append(outer, t.out...)
	t.out = outer
}

func (t *translator) emitUnionInitializer(typeID int, tokens []token) {
	info := t.typeInfo(typeID)
	if info.kind != cTypeUnion || len(tokens) < 2 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		t.fail(TranslateErrUnsupported)
		return
	}
	items := splitTopLevel(t.src, tokens[1:len(tokens)-1], ",")
	var item []token
	for i := 0; i < len(items); i++ {
		if len(items[i]) == 0 {
			continue
		}
		if item != nil {
			t.fail(TranslateErrUnsupported)
			return
		}
		item = items[i]
	}
	if item == nil {
		t.fail(TranslateErrUnsupported)
		return
	}
	field := cField{}
	value := item
	if len(item) >= 4 && tokenIs(t.src, item[0], ".") && tokenKind(item[1]) == tokenIdent && tokenIs(t.src, item[2], "=") {
		var ok bool
		field, ok = t.lookupField(typeID, tokenText(t.src, item[1]))
		if !ok || !field.emit {
			t.fail(TranslateErrUnsupported)
			return
		}
		value = item[3:]
	} else {
		for i := 0; i < info.fieldCount; i++ {
			candidate := t.fields[info.fieldStart+i]
			if candidate.emit && candidate.name != "" {
				field = candidate
				break
			}
		}
	}
	if field.name == "" {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.typeSerial++
	name := "__c_union_init_" + decimalString(t.typeSerial)
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	t.emitInitializerValue(field.typeID, value)
	t.out = append(t.out, ')')
	outer := t.out
	t.out = t.staticOut
	t.out = append(t.out, "func "...)
	t.out = append(t.out, name...)
	t.out = append(t.out, "(v "...)
	if t.typeInfo(field.typeID).kind == cTypeBool && field.bitWidth > 0 {
		t.out = append(t.out, "uint8"...)
	} else {
		t.emitType(field.typeID)
	}
	t.out = append(t.out, ") "...)
	t.emitType(typeID)
	t.out = append(t.out, "{var p "...)
	t.emitType(typeID)
	t.out = append(t.out, ';')
	if field.bitWidth > 0 {
		t.out = append(t.out, "p.__c_set_"...)
		t.out = append(t.out, field.name...)
		t.out = append(t.out, "(v);"...)
	} else {
		t.out = append(t.out, "(*p.__c_ptr_"...)
		t.out = append(t.out, field.name...)
		t.out = append(t.out, "())=v;"...)
	}
	t.out = append(t.out, "return p}\n"...)
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) emitBitfieldMutation(tokens []token) bool {
	depth := 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") || tokenIs(t.src, tokens[i], "{"):
			depth++
		case tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") || tokenIs(t.src, tokens[i], "}"):
			depth--
		}
		if depth != 0 || !isAssignmentToken(t.src, tokens[i]) {
			continue
		}
		access, ok := t.memberAccess(tokens[:i], 0)
		if !ok || !access.bitfield || access.end != i {
			return false
		}
		t.out = append(t.out, access.receiver...)
		t.out = append(t.out, ".__c_set_"...)
		t.out = append(t.out, access.field.name...)
		t.out = append(t.out, '(')
		if !tokenIs(t.src, tokens[i], "=") {
			t.out = append(t.out, access.expr...)
			op := tokenText(t.src, tokens[i])
			t.out = append(t.out, op[:len(op)-1]...)
		}
		t.emitExpression(tokens[i+1:])
		t.out = append(t.out, ')')
		return true
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[len(tokens)-1], "++") || tokenIs(t.src, tokens[len(tokens)-1], "--")) {
		access, ok := t.memberAccess(tokens[:len(tokens)-1], 0)
		if ok && access.bitfield && access.end == len(tokens)-1 {
			t.emitBitfieldStep(access, tokens[len(tokens)-1])
			return true
		}
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[0], "++") || tokenIs(t.src, tokens[0], "--")) {
		access, ok := t.memberAccess(tokens[1:], 0)
		if ok && access.bitfield && access.end == len(tokens)-1 {
			t.emitBitfieldStep(access, tokens[0])
			return true
		}
	}
	return false
}

func (t *translator) emitBitfieldStep(access cMemberAccess, op token) {
	t.out = append(t.out, access.receiver...)
	t.out = append(t.out, ".__c_set_"...)
	t.out = append(t.out, access.field.name...)
	t.out = append(t.out, '(')
	t.out = append(t.out, access.expr...)
	if tokenIs(t.src, op, "++") {
		t.out = append(t.out, "+1)"...)
	} else {
		t.out = append(t.out, "-1)"...)
	}
}

func isAssignmentToken(src []byte, tok token) bool {
	return tokenIs(src, tok, "=") || tokenIs(src, tok, "+=") || tokenIs(src, tok, "-=") ||
		tokenIs(src, tok, "*=") || tokenIs(src, tok, "/=") || tokenIs(src, tok, "%=") ||
		tokenIs(src, tok, "<<=") || tokenIs(src, tok, ">>=") || tokenIs(src, tok, "&=") ||
		tokenIs(src, tok, "^=") || tokenIs(src, tok, "|=")
}

func (t *translator) memberAccess(tokens []token, start int) (cMemberAccess, bool) {
	if start < 0 || start >= len(tokens) || tokenKind(tokens[start]) != tokenIdent {
		return cMemberAccess{}, false
	}
	object, ok := t.lookupObject(tokenText(t.src, tokens[start]))
	if !ok {
		return cMemberAccess{}, false
	}
	expr := append([]byte{}, object.goName...)
	result := cMemberAccess{expr: expr, typeID: object.typeID, end: start + 1}
	for result.end+1 < len(tokens) && (tokenIs(t.src, tokens[result.end], ".") || tokenIs(t.src, tokens[result.end], "->")) {
		arrow := tokenIs(t.src, tokens[result.end], "->")
		aggregateType := result.typeID
		if arrow {
			pointer := t.typeInfo(aggregateType)
			if pointer.kind != cTypePointer {
				break
			}
			aggregateType = pointer.base
		}
		aggregate := t.typeInfo(aggregateType)
		if (aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion) || tokenKind(tokens[result.end+1]) != tokenIdent {
			break
		}
		field, found := t.lookupField(aggregateType, tokenText(t.src, tokens[result.end+1]))
		if !found {
			break
		}
		fieldType := field.typeID
		if aggregate.qualifiers&cQualifierConst != 0 {
			fieldType = t.qualifiedType(fieldType, cQualifierConst)
		}
		receiver := append([]byte{}, result.expr...)
		if field.bitWidth > 0 {
			expr = append(expr, ".__c_get_"...)
			expr = append(expr, field.name...)
			expr = append(expr, '(', ')')
			result.expr = expr
			result.receiver = receiver
			result.field = field
			result.typeID = fieldType
			result.end += 2
			result.bitfield = true
			return result, true
		}
		if aggregate.kind == cTypeUnion {
			wrapped := make([]byte, 0, len(expr)+len(field.name)+15)
			wrapped = append(wrapped, '(', '*')
			wrapped = append(wrapped, expr...)
			wrapped = append(wrapped, ".__c_ptr_"...)
			wrapped = append(wrapped, field.name...)
			wrapped = append(wrapped, '(', ')', ')')
			expr = wrapped
		} else if field.emit {
			expr = append(expr, '.')
			expr = append(expr, field.name...)
		} else {
			break
		}
		result.expr = expr
		result.field = field
		result.typeID = fieldType
		result.end += 2
	}
	return result, result.end > start+1
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
		if field.bitWidth == 0 && textEquals(name, field.name) {
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

func (t *translator) findFieldSeparator() int {
	paren, bracket, brace := 0, 0, 0
	for i := t.pos; i < len(t.tokens); i++ {
		if paren == 0 && bracket == 0 && brace == 0 && (tokenIs(t.src, t.tokens[i], ",") || tokenIs(t.src, t.tokens[i], ";")) {
			return i
		}
		switch {
		case tokenIs(t.src, t.tokens[i], "("):
			paren++
		case tokenIs(t.src, t.tokens[i], ")"):
			paren--
		case tokenIs(t.src, t.tokens[i], "["):
			bracket++
		case tokenIs(t.src, t.tokens[i], "]"):
			bracket--
		case tokenIs(t.src, t.tokens[i], "{"):
			brace++
		case tokenIs(t.src, t.tokens[i], "}"):
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
		tokenIs(src, tok, "_Bool") || tokenIs(src, tok, "_Atomic") || tokenIs(src, tok, "_Thread_local") || tokenIs(src, tok, "struct") ||
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
