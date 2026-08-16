package c11

import "renvo.dev/internal/arena"

const (
	TranslateOK = iota
	TranslateErrScan
	TranslateErrDeclaration
	TranslateErrStatement
	TranslateErrUnsupported
	TranslateErrVLA
)

const (
	storageNone = iota
	storageOther
	storageStatic
	storageThread
	storageThreadStatic
	storageThreadExtern
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
	name            token
	typeID          int
	attributes      cAttributes
	params          []parameter
	function        bool
	variadic        bool
	initializer     []token
	functionType    int
	incompleteArray bool
	storage         int
}

// cAttributes contains only properties which affect C semantics at the shared
// frontend boundary. Diagnostic and optimizer hints are recognized by name but
// deliberately do not grow this record.
type cAttributes struct {
	align             int
	packed            bool
	transparentUnion  bool
	section           string
	alias             string
	visibility        string
	weak              bool
	callingConvention bool
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
	incomplete bool
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
	src              []byte
	tokens           []token
	pos              int
	packageName      string
	out              []byte
	object           bool
	checkOnly        bool
	speculativeType  bool
	ok               bool
	err              int
	errorAt          int
	types            []cTypeInfo
	fields           []cField
	opaqueTypes      []cTypeName
	names            []cNamespaceName
	objects          []cObjectName
	tentatives       []cObjectName
	definitions      []cObjectName
	threadNames      []cObjectName
	functions        []cFunctionName
	variadicCalls    []cVariadicCall
	functionParams   []int
	ordinaryNames    []int
	tagNames         []int
	pointerTypes     []int
	qualifiedTypes   []int
	pointerChanges   []cTypeCacheChange
	qualifiedChanges []cTypeCacheChange
	functionSections bool
	dataSections     bool
	baseAttributes   cAttributes
	typeSerial       int
	dataModel        int
	pointerSize      int
	packageHeader    int
	usesUnsafe       bool
	staticOut        []byte
	resultType       int
	boolHelper       bool
	tlsRuntime       bool
	localStart       int
	initializing     int
	nameStart        int
}

type checkScratchMark struct {
	mark                                   int
	out, staticOut                         []byte
	typesLen, typesCap                     int
	fieldsLen, fieldsCap                   int
	opaqueTypesLen, opaqueTypesCap         int
	namesLen, namesCap                     int
	objectsLen, objectsCap                 int
	tentativesLen, tentativesCap           int
	definitionsLen, definitionsCap         int
	threadNamesLen, threadNamesCap         int
	functionsLen, functionsCap             int
	variadicCallsLen, variadicCallsCap     int
	functionParamsLen, functionParamsCap   int
	pointerTypesCap, qualifiedTypesCap     int
	pointerChangesCap, qualifiedChangesCap int
}

// Translate lowers one preprocessed C translation unit into the shared Go
// frontend syntax. It deliberately performs no backend-specific lowering.
func Translate(packageName string, src []byte) Result {
	return translate(packageName, src, nil, false, DataModelLP64, false)
}

// TranslateObject lowers a hosted C translation unit. Prelude contains
// declarations recovered from the translation unit's included headers; it is
// kept separate so diagnostics in src retain their original byte offsets.
func TranslateObject(packageName string, src []byte, prelude []byte) Result {
	return TranslateObjectForDataModel(packageName, src, prelude, DataModelLP64)
}

func TranslateObjectForDataModel(packageName string, src []byte, prelude []byte, dataModel int) Result {
	return TranslateObjectWithConfig(packageName, src, prelude, ObjectConfig{DataModel: dataModel})
}

// ObjectConfig contains object-placement policy selected by the compiler
// driver. Declaration attributes remain properties of the declaration itself;
// these booleans only implement the corresponding GCC command-line switches.
type ObjectConfig struct {
	DataModel        int
	FunctionSections bool
	DataSections     bool
}

func TranslateObjectWithConfig(packageName string, src []byte, prelude []byte, config ObjectConfig) Result {
	return translateObjectConfig(packageName, src, prelude, true, config, false)
}

// CheckObjectForDataModel parses and type-checks a preprocessed C translation
// unit without requiring all target code-generation operations to exist yet.
// It is the M4 boundary used for constructs such as inline assembly whose
// lowering belongs to a later milestone.
func CheckObjectForDataModel(src []byte, dataModel int) Result {
	return translateObjectConfig("main", src, nil, true, ObjectConfig{DataModel: dataModel}, true)
}

func translate(packageName string, src []byte, prelude []byte, object bool, dataModel int, checkOnly bool) Result {
	return translateObjectConfig(packageName, src, prelude, object, ObjectConfig{DataModel: dataModel}, checkOnly)
}

func translateObjectConfig(packageName string, src []byte, prelude []byte, object bool, config ObjectConfig, checkOnly bool) Result {
	t := translator{
		packageName:      packageName,
		out:              make([]byte, 0, len(src)+len(src)/4+len(prelude)+64),
		object:           object,
		checkOnly:        checkOnly,
		functionSections: config.FunctionSections,
		dataSections:     config.DataSections,
		ok:               true,
		errorAt:          -1,
		localStart:       -1,
		initializing:     -1,
	}
	t.initTypes(config.DataModel)
	t.appendText("package ")
	t.out = append(t.out, packageName...)
	t.out = append(t.out, '\n')
	t.packageHeader = len(t.out)
	if len(prelude) > 0 && !t.translateSource(prelude) {
		return Result{Ok: false, Error: t.err, ErrorAt: -1}
	}
	if !t.translateSource(src) {
		return Result{Ok: false, Error: t.err, ErrorAt: t.errorAt}
	}
	if checkOnly {
		return Result{Ok: true, Error: TranslateOK, ErrorAt: -1}
	}
	t.emitPendingDeclarations()
	if !t.ok {
		return Result{Ok: false, Error: t.err, ErrorAt: t.errorAt}
	}
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
	t.ensureNameIndex(len(t.tokens))
	if t.checkOnly {
		t.reserveCheckTables(len(t.tokens))
	}
	for t.ok && t.kind() != tokenEOF {
		t.skipDirectives()
		if t.kind() == tokenEOF {
			break
		}
		if t.take(";") {
			continue
		}
		if t.checkOnly {
			mark := t.beginCheckScratch()
			t.externalDeclaration()
			t.endCheckScratch(mark)
		} else {
			t.externalDeclaration()
		}
	}
	return t.ok
}

func (t *translator) ensureNameIndex(tokens int) {
	want := 128
	for want < tokens/8+1 {
		want *= 2
	}
	if len(t.ordinaryNames) >= want {
		return
	}
	t.ordinaryNames = make([]int, want)
	t.tagNames = make([]int, want)
	for i := 0; i < len(t.names); i++ {
		buckets := t.ordinaryNames
		if t.names[i].kind == cNameTag {
			buckets = t.tagNames
		}
		bucket := cNameHashString(t.names[i].name) & (len(buckets) - 1)
		t.names[i].hashPrev = buckets[bucket]
		buckets[bucket] = i + 1
	}
}

func (t *translator) reserveCheckTables(tokens int) {
	// A self-hosted compiler cannot rely on garbage collection to recover old
	// slice backing arrays. Reserve compact semantic tables from input size so
	// ordinary growth does not pin a declaration's scratch arena. Correctness
	// does not depend on these estimates: endCheckScratch simply retains the
	// low arena when a pathological input exceeds one of them.
	types := make([]cTypeInfo, len(t.types), len(t.types)+tokens/32+64)
	copy(types, t.types)
	t.types = types
	t.pointerTypes = make([]int, cap(types))
	t.qualifiedTypes = make([]int, cap(types)*16)
	t.pointerChanges = make([]cTypeCacheChange, 0, cap(types))
	t.qualifiedChanges = make([]cTypeCacheChange, 0, cap(types))
	fields := make([]cField, len(t.fields), len(t.fields)+tokens/48+64)
	copy(fields, t.fields)
	t.fields = fields
	opaqueTypes := make([]cTypeName, len(t.opaqueTypes), len(t.opaqueTypes)+tokens/256+16)
	copy(opaqueTypes, t.opaqueTypes)
	t.opaqueTypes = opaqueTypes
	names := make([]cNamespaceName, len(t.names), len(t.names)+tokens/32+64)
	copy(names, t.names)
	t.names = names
	objects := make([]cObjectName, len(t.objects), len(t.objects)+tokens/128+32)
	copy(objects, t.objects)
	t.objects = objects
	tentatives := make([]cObjectName, len(t.tentatives), len(t.tentatives)+tokens/256+16)
	copy(tentatives, t.tentatives)
	t.tentatives = tentatives
	definitions := make([]cObjectName, len(t.definitions), len(t.definitions)+tokens/256+16)
	copy(definitions, t.definitions)
	t.definitions = definitions
	threadNames := make([]cObjectName, len(t.threadNames), len(t.threadNames)+tokens/256+16)
	copy(threadNames, t.threadNames)
	t.threadNames = threadNames
	functions := make([]cFunctionName, len(t.functions), len(t.functions)+tokens/48+32)
	copy(functions, t.functions)
	t.functions = functions
	variadicCalls := make([]cVariadicCall, len(t.variadicCalls), len(t.variadicCalls)+tokens/256+16)
	copy(variadicCalls, t.variadicCalls)
	t.variadicCalls = variadicCalls
	functionParams := make([]int, len(t.functionParams), len(t.functionParams)+tokens/16+64)
	copy(functionParams, t.functionParams)
	t.functionParams = functionParams
}

func (t *translator) beginCheckScratch() checkScratchMark {
	return checkScratchMark{
		mark: arena.Mark(), out: t.out, staticOut: t.staticOut,
		typesLen: len(t.types), typesCap: cap(t.types),
		fieldsLen: len(t.fields), fieldsCap: cap(t.fields),
		opaqueTypesLen: len(t.opaqueTypes), opaqueTypesCap: cap(t.opaqueTypes),
		namesLen: len(t.names), namesCap: cap(t.names),
		objectsLen: len(t.objects), objectsCap: cap(t.objects),
		tentativesLen: len(t.tentatives), tentativesCap: cap(t.tentatives),
		definitionsLen: len(t.definitions), definitionsCap: cap(t.definitions),
		threadNamesLen: len(t.threadNames), threadNamesCap: cap(t.threadNames),
		functionsLen: len(t.functions), functionsCap: cap(t.functions),
		variadicCallsLen: len(t.variadicCalls), variadicCallsCap: cap(t.variadicCalls),
		functionParamsLen: len(t.functionParams), functionParamsCap: cap(t.functionParams),
		pointerTypesCap: cap(t.pointerTypes), qualifiedTypesCap: cap(t.qualifiedTypes),
		pointerChangesCap: cap(t.pointerChanges), qualifiedChangesCap: cap(t.qualifiedChanges),
	}
}

func (t *translator) endCheckScratch(mark checkScratchMark) {
	// Strings retained by the semantic symbol/type tables must outlive the
	// low-arena declaration scratch that produced them.
	for i := mark.typesLen; i < len(t.types); i++ {
		t.types[i].goName = arena.PersistString(t.types[i].goName)
	}
	for i := mark.fieldsLen; i < len(t.fields); i++ {
		t.fields[i].name = arena.PersistString(t.fields[i].name)
		t.fields[i].carrier = arena.PersistString(t.fields[i].carrier)
	}
	for i := mark.opaqueTypesLen; i < len(t.opaqueTypes); i++ {
		t.opaqueTypes[i].name = arena.PersistString(t.opaqueTypes[i].name)
	}
	for i := mark.namesLen; i < len(t.names); i++ {
		t.names[i].name = arena.PersistString(t.names[i].name)
	}
	for i := mark.objectsLen; i < len(t.objects); i++ {
		t.objects[i].name = arena.PersistString(t.objects[i].name)
		t.objects[i].goName = arena.PersistString(t.objects[i].goName)
	}
	for i := mark.tentativesLen; i < len(t.tentatives); i++ {
		t.tentatives[i].name = arena.PersistString(t.tentatives[i].name)
		t.tentatives[i].goName = arena.PersistString(t.tentatives[i].goName)
	}
	for i := mark.definitionsLen; i < len(t.definitions); i++ {
		t.definitions[i].name = arena.PersistString(t.definitions[i].name)
		t.definitions[i].goName = arena.PersistString(t.definitions[i].goName)
	}
	for i := mark.threadNamesLen; i < len(t.threadNames); i++ {
		t.threadNames[i].name = arena.PersistString(t.threadNames[i].name)
		t.threadNames[i].goName = arena.PersistString(t.threadNames[i].goName)
	}
	for i := mark.functionsLen; i < len(t.functions); i++ {
		t.functions[i].name = arena.PersistString(t.functions[i].name)
	}
	for i := mark.variadicCallsLen; i < len(t.variadicCalls); i++ {
		t.variadicCalls[i].goName = arena.PersistString(t.variadicCalls[i].goName)
	}

	t.out = mark.out
	t.staticOut = mark.staticOut
	stable := mark.typesCap == cap(t.types) && mark.fieldsCap == cap(t.fields) &&
		mark.opaqueTypesCap == cap(t.opaqueTypes) && mark.namesCap == cap(t.names) &&
		mark.objectsCap == cap(t.objects) && mark.tentativesCap == cap(t.tentatives) &&
		mark.definitionsCap == cap(t.definitions) && mark.threadNamesCap == cap(t.threadNames) &&
		mark.functionsCap == cap(t.functions) && mark.variadicCallsCap == cap(t.variadicCalls) &&
		mark.functionParamsCap == cap(t.functionParams) && mark.pointerTypesCap == cap(t.pointerTypes) &&
		mark.qualifiedTypesCap == cap(t.qualifiedTypes) && mark.pointerChangesCap == cap(t.pointerChanges) &&
		mark.qualifiedChangesCap == cap(t.qualifiedChanges)
	if stable || !t.ok {
		arena.Rewind(mark.mark)
	}
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
	if t.currentIs("asm") || t.currentIs("__asm") || t.currentIs("__asm__") {
		if !t.checkOnly || !t.takeAsmStatement() {
			t.fail(TranslateErrUnsupported)
		}
		return
	}
	if t.currentIs("_Static_assert") || t.currentIs("static_assert") {
		if !t.takeStaticAssert() {
			t.fail(TranslateErrDeclaration)
		}
		return
	}
	base, storage, ok := t.parseType()
	if !ok {
		t.fail(TranslateErrDeclaration)
		return
	}
	if t.typeInfo(base).qualifiers&cQualifierRestrict != 0 && t.typeInfo(base).kind != cTypePointer {
		t.fail(TranslateErrDeclaration)
		return
	}
	typeAttributes := t.baseAttributes
	if t.take(";") {
		return
	}
	decl, ok := t.parseDeclarator(base, true)
	if !ok {
		t.fail(TranslateErrDeclaration)
		return
	}
	decl.attributes.merge(typeAttributes)
	if storage == storageTypedef && decl.function {
		t.rememberTypedef(string(tokenText(t.src, decl.name)), decl.functionType)
		for t.take(",") {
			next, valid := t.parseDeclarator(base, true)
			if !valid || !next.function {
				t.fail(TranslateErrDeclaration)
				return
			}
			t.rememberTypedef(string(tokenText(t.src, next.name)), next.functionType)
		}
		if !t.take(";") {
			t.fail(TranslateErrDeclaration)
		}
		return
	}
	if decl.function {
		if storage == storageThread || storage == storageThreadStatic || storage == storageThreadExtern {
			t.fail(TranslateErrDeclaration)
			return
		}
		if t.take(";") {
			if t.object && decl.attributes.alias != "" {
				t.emitObjectAlias(decl, storage, true)
			}
			if (t.checkOnly || t.object && storage != storageStatic) && !t.rememberFunction(&decl, false) {
				t.fail(TranslateErrDeclaration)
			}
			return // Otherwise a C or Go definition in the package satisfies it.
		}
		if t.take(",") {
			if (t.checkOnly || t.object && storage != storageStatic) && !t.rememberFunction(&decl, false) {
				t.fail(TranslateErrDeclaration)
				return
			}
			for {
				next, valid := t.parseDeclarator(base, true)
				if !valid || !next.function || (t.checkOnly || t.object && storage != storageStatic) && !t.rememberFunction(&next, false) {
					t.fail(TranslateErrDeclaration)
					return
				}
				if !t.take(",") {
					break
				}
			}
			if !t.take(";") {
				t.fail(TranslateErrDeclaration)
			}
			return
		}
		if !t.currentIs("{") {
			t.fail(TranslateErrDeclaration)
			return
		}
		if !t.rememberFunction(&decl, true) {
			t.fail(TranslateErrDeclaration)
			return
		}
		if t.checkOnly {
			t.checkFunction(decl, storage)
			return
		}
		t.emitFunction(decl, storage)
		return
	}
	decl.initializer = t.takeInitializer()
	autoDeclaration := t.typeInfo(decl.typeID).kind == cTypeAuto
	if !t.resolveAutoDeclarator(&decl) {
		t.fail(TranslateErrDeclaration)
		return
	}
	decls := []declarator{decl}
	for t.take(",") {
		if autoDeclaration {
			t.fail(TranslateErrDeclaration)
			return
		}
		next, valid := t.parseDeclarator(base, false)
		if !valid {
			t.fail(TranslateErrDeclaration)
			return
		}
		next.attributes.merge(typeAttributes)
		next.initializer = t.takeInitializer()
		decls = append(decls, next)
	}
	if !t.take(";") {
		t.fail(TranslateErrDeclaration)
		return
	}
	if storage == storageTypedef {
		for i := 0; i < len(decls); i++ {
			if decls[i].attributes.transparentUnion {
				if t.typeInfo(decls[i].typeID).kind != cTypeUnion {
					t.fail(TranslateErrDeclaration)
					return
				}
				t.types[decls[i].typeID].transparentUnion = true
				if !t.checkOnly {
					t.fail(TranslateErrUnsupported)
					return
				}
			}
			t.rememberTypedef(string(tokenText(t.src, decls[i].name)), decls[i].typeID)
		}
		return
	}
	if storage == storageThread || storage == storageThreadStatic {
		for i := 0; i < len(decls); i++ {
			t.emitThreadVariable(decls[i], true)
		}
		return
	}
	if storage == storageThreadExtern {
		for i := 0; i < len(decls); i++ {
			t.rememberThreadExtern(decls[i])
		}
		return
	}
	if storage == storageExtern {
		for i := 0; i < len(decls); i++ {
			if t.object && decls[i].attributes.alias != "" {
				t.emitObjectAlias(decls[i], storage, false)
			}
			t.rememberObject(tokenText(t.src, decls[i].name), decls[i].typeID)
		}
		return
	}
	if storage == storageOther {
		if t.checkOnly {
			for i := 0; i < len(decls); i++ {
				t.rememberObject(tokenText(t.src, decls[i].name), decls[i].typeID)
			}
			return
		}
		t.fail(TranslateErrDeclaration)
		return
	}
	for i := 0; i < len(decls); i++ {
		decls[i].storage = storage
		if !t.fileVariable(decls[i]) {
			t.fail(TranslateErrDeclaration)
			return
		}
	}
}

// checkFunction retains file-scope declarations while giving each function
// body a scratch lifetime. The native frontend has a garbage collector for
// these short-lived slices; a self-hosted compiler uses a bump arena instead,
// so retaining every expression temporary until the translation unit ends
// makes large preprocessed inputs needlessly scale with total body complexity.
func (t *translator) checkFunction(decl declarator, storage int) {
	mark := arena.Mark()
	persistMark := arena.PersistMark()
	out := t.out
	staticOut := t.staticOut
	types := t.types
	fields := t.fields
	opaqueTypes := t.opaqueTypes
	names := t.names
	objects := t.objects
	tentatives := t.tentatives
	definitions := t.definitions
	threadNames := t.threadNames
	functions := t.functions
	variadicCalls := t.variadicCalls
	functionParams := t.functionParams
	pointerTypes := t.pointerTypes
	qualifiedTypes := t.qualifiedTypes
	pointerChanges := t.pointerChanges
	qualifiedChanges := t.qualifiedChanges
	pointerChangeMark := len(t.pointerChanges)
	qualifiedChangeMark := len(t.qualifiedChanges)
	t.emitFunction(decl, storage)
	for i := len(t.pointerChanges) - 1; i >= pointerChangeMark; i-- {
		change := t.pointerChanges[i]
		if change.index >= 0 && change.index < len(pointerTypes) {
			pointerTypes[change.index] = change.previous
		}
	}
	for i := len(t.qualifiedChanges) - 1; i >= qualifiedChangeMark; i-- {
		change := t.qualifiedChanges[i]
		if change.index >= 0 && change.index < len(qualifiedTypes) {
			qualifiedTypes[change.index] = change.previous
		}
	}

	t.out = out
	t.staticOut = staticOut
	t.types = types
	t.fields = fields
	t.opaqueTypes = opaqueTypes
	t.names = names
	t.objects = objects
	t.tentatives = tentatives
	t.definitions = definitions
	t.threadNames = threadNames
	t.functions = functions
	t.variadicCalls = variadicCalls
	t.functionParams = functionParams
	t.pointerTypes = pointerTypes
	t.qualifiedTypes = qualifiedTypes
	t.pointerChanges = pointerChanges
	t.qualifiedChanges = qualifiedChanges
	arena.PersistReset(persistMark)
	arena.Rewind(mark)
}

func (t *translator) parseAggregateType(union bool) (int, bool) {
	kind := cTypeStruct
	prefix := "__c_struct_"
	if union {
		kind = cTypeUnion
		prefix = "__c_union_"
	}
	t.pos++
	attributes, ok := t.parseAttributes()
	if !ok {
		return cTypeVoidID, false
	}
	name := ""
	if t.kind() == tokenIdent {
		name = string(tokenText(t.src, t.tokens[t.pos]))
		t.pos++
	}
	afterName, ok := t.parseAttributes()
	if !ok {
		return cTypeVoidID, false
	}
	attributes.merge(afterName)
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
		t.appendName(name, typeID, cNameTag)
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
		prior, current := cTypeVoidID, false
		if entry := t.lookupNameEntryString(name, true); entry >= t.nameStart {
			prior, current = t.names[entry].index, true
		}
		if current && t.types[prior].size == 0 {
			t.types = t.types[:len(t.types)-1]
			typeID = prior
			t.types[typeID].goName = arena.PersistString(goName)
			t.types[typeID].kind = kind
		} else {
			t.appendName(name, typeID, cNameTag)
		}
	}
	t.pos++
	var aggregateFields []cField
	offset := 0
	maxAlign := 1
	maxSize := 0
	indirect := false
	bitActive := false
	bitSize := 0
	bitAlign := 0
	bitOffset := 0
	bitUsed := 0
	bitCarrier := ""
	for !t.currentIs("}") && t.kind() != tokenEOF {
		if t.take(";") {
			continue
		}
		if t.currentIs("_Static_assert") || t.currentIs("static_assert") {
			if !t.takeStaticAssert() {
				return cTypeVoidID, false
			}
			continue
		}
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
			if field.attributes.align > alignment {
				alignment = field.attributes.align
				indirect = true
			}
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
			aggregateFields = append(aggregateFields, cField{name: string(tokenText(t.src, field.name)), typeID: field.typeID, offset: fieldOffset, align: field.attributes.align, emit: true})
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
	afterBody, ok := t.parseAttributes()
	if !ok {
		return cTypeVoidID, false
	}
	attributes.merge(afterBody)
	if attributes.transparentUnion {
		if !union {
			return cTypeVoidID, false
		}
		t.types[typeID].transparentUnion = true
		if !t.checkOnly {
			t.fail(TranslateErrUnsupported)
			return cTypeVoidID, false
		}
	}
	naturalAlign := maxAlign
	if attributes.packed {
		offset, maxSize, maxAlign = t.packAggregateFields(aggregateFields, union)
		indirect = true
	}
	if attributes.align > maxAlign {
		maxAlign = attributes.align
	}
	if attributes.align > naturalAlign {
		indirect = true
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
	t.types[typeID].indirect = indirect
	t.types[typeID].fieldStart = fieldStart
	t.types[typeID].fieldCount = len(t.fields) - fieldStart
	t.emitAggregateDeclaration(typeID)
	t.emitAggregateAccessors(typeID)
	return typeID, true
}

func (attributes *cAttributes) merge(other cAttributes) {
	if other.align > attributes.align {
		attributes.align = other.align
	}
	attributes.packed = attributes.packed || other.packed
	attributes.transparentUnion = attributes.transparentUnion || other.transparentUnion
	if other.section != "" {
		attributes.section = other.section
	}
	if other.alias != "" {
		attributes.alias = other.alias
	}
	if other.visibility != "" {
		attributes.visibility = other.visibility
	}
	attributes.weak = attributes.weak || other.weak
	attributes.callingConvention = attributes.callingConvention || other.callingConvention
}

func (attributes cAttributes) requiresObjectMetadata() bool {
	return attributes.section != "" || attributes.alias != "" || attributes.visibility != "" ||
		attributes.weak || attributes.callingConvention
}

func (t *translator) packAggregateFields(fields []cField, union bool) (int, int, int) {
	offset := 0
	maxSize := 0
	maxAlign := 1
	containerOld := 0
	containerNew := 0
	carrier := ""
	carrierOffset := 0
	for i := 0; i < len(fields); i++ {
		field := &fields[i]
		if !field.emit {
			if field.carrier != "" && field.carrier == carrier {
				field.offset = carrierOffset
			} else {
				field.offset = containerNew + field.offset - containerOld
			}
			continue
		}
		alignment := 1
		if field.align > alignment {
			alignment = field.align
		}
		if alignment > maxAlign {
			maxAlign = alignment
		}
		containerOld = field.offset
		containerNew = 0
		if !union {
			offset = alignC(offset, alignment)
			containerNew = offset
		}
		field.offset = containerNew
		size := t.typeSize(field.typeID)
		if size > maxSize {
			maxSize = size
		}
		if !union {
			offset += size
		}
		if field.synthetic {
			carrier = field.name
			carrierOffset = field.offset
		} else {
			carrier = ""
		}
	}
	return offset, maxSize, maxAlign
}

func (t *translator) parseEnumType() (int, bool) {
	t.pos++
	attributes, ok := t.parseAttributes()
	if !ok {
		return cTypeVoidID, false
	}
	name := ""
	if t.kind() == tokenIdent {
		name = string(tokenText(t.src, t.tokens[t.pos]))
		t.pos++
	}
	afterName, ok := t.parseAttributes()
	if !ok {
		return cTypeVoidID, false
	}
	attributes.merge(afterName)
	if !t.take("{") {
		if name == "" {
			return cTypeVoidID, false
		}
		if typeID, ok := t.lookupTag([]byte(name)); ok {
			return typeID, true
		}
		t.appendName(name, cTypeInt32ID, cNameTag)
		return cTypeInt32ID, true
	}
	value := 0
	minimum, maximum := 0, 0
	haveValue := false
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
		if !haveValue || value < minimum {
			minimum = value
		}
		if !haveValue || value > maximum {
			maximum = value
		}
		haveValue = true
		value++
		if !t.take(",") {
			break
		}
	}
	if !t.take("}") {
		return cTypeVoidID, false
	}
	afterDefinition, ok := t.parseAttributes()
	if !ok {
		return cTypeVoidID, false
	}
	attributes.merge(afterDefinition)
	typeID := cTypeInt32ID
	if attributes.packed {
		switch {
		case minimum >= 0 && maximum <= 1<<8-1:
			typeID = cTypeUint8ID
		case minimum >= -1<<7 && maximum <= 1<<7-1:
			typeID = cTypeInt8ID
		case minimum >= 0 && maximum <= 1<<16-1:
			typeID = cTypeUint16ID
		case minimum >= -1<<15 && maximum <= 1<<15-1:
			typeID = cTypeInt16ID
		case minimum >= 0:
			typeID = cTypeUint32ID
		}
	}
	if name != "" {
		if entry := t.lookupNameEntryString(name, true); entry >= t.nameStart {
			t.names[entry].index = typeID
			return typeID, true
		}
		t.appendName(name, typeID, cNameTag)
	}
	return typeID, true
}

func (t *translator) rememberValue(name string, value int) {
	t.rememberName(name, value, cNameValue)
}

func (t *translator) lookupValue(name []byte) (int, bool) {
	return t.lookupName(name, cNameValue)
}

func (t *translator) rememberObject(name []byte, typeID int) {
	if len(name) == 0 {
		return
	}
	if t.localStart >= 0 {
		if entry := t.lookupNameEntry(name, false); entry >= t.nameStart {
			t.fail(TranslateErrDeclaration)
			return
		}
	}
	t.objects = append(t.objects, cObjectName{name: string(name), goName: string(name), typeID: typeID, auto: t.localStart >= 0})
	t.rememberName(string(name), len(t.objects)-1, cNameObject)
}

func (t *translator) rememberAliasedObject(name []byte, goName string, typeID int) {
	if len(name) == 0 {
		return
	}
	t.objects = append(t.objects, cObjectName{name: string(name), goName: goName, typeID: typeID})
	t.rememberName(string(name), len(t.objects)-1, cNameObject)
}

func (t *translator) rememberFunction(decl *declarator, definition bool) bool {
	name := string(tokenText(t.src, decl.name))
	function := -1
	if entry := t.lookupNameEntryString(name, false); entry >= 0 {
		if t.names[entry].kind == cNameFunction {
			function = t.names[entry].index
		} else {
			// A block-scope ordinary identifier can shadow a file-scope function.
			// Preserve the function declaration table independently of that
			// namespace lookup; this uncommon path does not affect normal hashing.
			for i := 0; i < len(t.functions); i++ {
				if t.functions[i].name == name {
					function = i
					break
				}
			}
		}
	}
	if function >= 0 {
		fn := &t.functions[function]
		if !t.compatibleParameterType(fn.resultType, decl.typeID) || fn.paramCount != len(decl.params) || fn.variadic != decl.variadic || definition && fn.defined {
			return false
		}
		for j := 0; j < fn.paramCount; j++ {
			if !t.compatibleParameterType(t.functionParams[fn.paramStart+j], decl.params[j].typeID) {
				return false
			}
		}
		if definition {
			decl.attributes.merge(fn.attributes)
			fn.defined = true
		}
		fn.attributes.merge(decl.attributes)
		return true
	}
	start := len(t.functionParams)
	for i := 0; i < len(decl.params); i++ {
		t.functionParams = append(t.functionParams, decl.params[i].typeID)
	}
	t.functions = append(t.functions, cFunctionName{
		name: name, typeID: decl.functionType, resultType: decl.typeID, paramStart: start,
		paramCount: len(decl.params), variadic: decl.variadic, defined: definition, attributes: decl.attributes,
	})
	return t.rememberName(name, len(t.functions)-1, cNameFunction)
}

func (t *translator) rememberName(name string, index int, kind int) bool {
	if kind != cNameTag {
		if entry := t.lookupNameEntryString(name, false); entry >= t.nameStart && t.names[entry].kind != kind {
			t.fail(TranslateErrDeclaration)
			return false
		}
	}
	t.appendName(name, index, kind)
	return true
}

func (t *translator) compatibleParameterType(left int, right int) bool {
	if left == right {
		return true
	}
	a, b := t.typeInfo(left), t.typeInfo(right)
	if a.kind != b.kind || a.count != b.count || a.fieldStart != b.fieldStart || a.fieldCount != b.fieldCount ||
		a.paramCount != b.paramCount || a.variadic != b.variadic || a.size != b.size {
		return false
	}
	if a.kind == cTypePointer || a.kind == cTypeArray {
		return t.compatibleParameterType(a.base, b.base)
	}
	if a.kind == cTypeFunction {
		if !t.compatibleParameterType(a.base, b.base) {
			return false
		}
		for i := 0; i < a.paramCount; i++ {
			if !t.compatibleParameterType(t.functionParams[a.paramStart+i], t.functionParams[b.paramStart+i]) {
				return false
			}
		}
	}
	return true
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
			t.tentatives = append(t.tentatives, cObjectName{name: name, typeID: decl.typeID, attributes: decl.attributes, storage: decl.storage})
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
	if info.kind != cTypeArray || !decl.incompleteArray {
		return true
	}
	if value, ok := t.cStringBytes(decl.initializer); ok && t.typeInfo(info.base).size == 1 {
		decl.typeID = t.arrayType(info.base, len(value))
		decl.incompleteArray = false
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
			rangeAt := topLevelToken(t.src, items[i][1:close], "...")
			indexTokens := items[i][1:close]
			if rangeAt >= 0 {
				indexTokens = items[i][1 : 1+rangeAt]
			}
			index, ok := t.constantExpression(indexTokens)
			if !ok || index < 0 {
				return false
			}
			if rangeAt >= 0 {
				last, valid := t.constantExpression(items[i][2+rangeAt : close])
				if !valid || last < index {
					return false
				}
				index = last
			}
			next = index
		}
		next++
		if next > count {
			count = next
		}
	}
	decl.typeID = t.arrayType(info.base, count)
	decl.incompleteArray = false
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
		t.emitObjectMetadata("variable", t.tentatives[i].name, t.tentatives[i].attributes, t.tentatives[i].storage, nil, typeID)
		t.appendText("var ")
		t.out = append(t.out, t.tentatives[i].name...)
		t.out = append(t.out, ' ')
		t.emitType(typeID)
		t.out = append(t.out, ';', '\n')
	}
	for i := 0; i < len(t.threadNames); i++ {
		if !t.threadNames[i].auto {
			t.fail(TranslateErrUnsupported)
			return
		}
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
	index, ok := t.lookupName(name, cNameObject)
	if ok {
		return t.objects[index], true
	}
	return cObjectName{}, false
}

func (t *translator) lookupFunction(name []byte) (int, bool) {
	return t.lookupName(name, cNameFunction)
}

func (t *translator) lookupName(name []byte, kind int) (int, bool) {
	entry := t.lookupNameEntry(name, kind == cNameTag)
	if entry >= 0 {
		return t.names[entry].index, t.names[entry].kind == kind
	}
	return 0, false
}

func (t *translator) lookupNameEntry(name []byte, tag bool) int {
	buckets := t.ordinaryNames
	if tag {
		buckets = t.tagNames
	}
	if len(buckets) == 0 {
		for i := len(t.names) - 1; i >= 0; i-- {
			if (t.names[i].kind == cNameTag) == tag && textEquals(name, t.names[i].name) {
				return i
			}
		}
		return -1
	}
	entry := buckets[cNameHash(name)&(len(buckets)-1)]
	for entry > 0 {
		i := entry - 1
		if textEquals(name, t.names[i].name) {
			return i
		}
		entry = t.names[i].hashPrev
	}
	return -1
}

func (t *translator) lookupNameEntryString(name string, tag bool) int {
	buckets := t.ordinaryNames
	if tag {
		buckets = t.tagNames
	}
	if len(buckets) == 0 {
		for i := len(t.names) - 1; i >= 0; i-- {
			if (t.names[i].kind == cNameTag) == tag && t.names[i].name == name {
				return i
			}
		}
		return -1
	}
	entry := buckets[cNameHashString(name)&(len(buckets)-1)]
	for entry > 0 {
		i := entry - 1
		if t.names[i].name == name {
			return i
		}
		entry = t.names[i].hashPrev
	}
	return -1
}

func (t *translator) appendName(name string, index int, kind int) {
	buckets := t.ordinaryNames
	if kind == cNameTag {
		buckets = t.tagNames
	}
	previous := 0
	if len(buckets) > 0 {
		bucket := cNameHashString(name) & (len(buckets) - 1)
		previous = buckets[bucket]
		buckets[bucket] = len(t.names) + 1
	}
	t.names = append(t.names, cNamespaceName{name: name, index: index, kind: kind, hashPrev: previous})
}

func (t *translator) truncateNames(length int) {
	for i := len(t.names) - 1; i >= length; i-- {
		buckets := t.ordinaryNames
		if t.names[i].kind == cNameTag {
			buckets = t.tagNames
		}
		if len(buckets) > 0 {
			bucket := cNameHashString(t.names[i].name) & (len(buckets) - 1)
			buckets[bucket] = t.names[i].hashPrev
		}
	}
	t.names = t.names[:length]
}

func cNameHash(name []byte) int {
	hash := 5381
	for i := 0; i < len(name); i++ {
		hash = ((hash << 5) + hash + int(name[i])) & 2147483647
	}
	return hash
}

func cNameHashString(name string) int {
	hash := 5381
	for i := 0; i < len(name); i++ {
		hash = ((hash << 5) + hash + int(name[i])) & 2147483647
	}
	return hash
}

func (t *translator) beginScope() cScopeMark {
	mark := cScopeMark{names: len(t.names), nameStart: t.nameStart}
	t.nameStart = mark.names
	return mark
}

func (t *translator) endScope(mark cScopeMark) {
	t.truncateNames(mark.names)
	t.nameStart = mark.nameStart
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
	t.appendText("type ")
	t.out = append(t.out, info.goName...)
	if info.kind == cTypeUnion || info.indirect {
		t.appendText(" struct{__c_align ")
		switch info.align {
		case 8:
			t.appendText("uint64")
		case 4:
			t.appendText("uint32")
		case 2:
			t.appendText("uint16")
		default:
			t.appendText("uint8")
		}
		if tail := info.size - info.align; tail > 0 {
			t.appendText(";__c_tail [")
			t.appendDecimal(tail)
			t.appendText("]byte")
		}
		t.appendText("};\n")
		return
	}
	t.appendText(" struct{")
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
		if field.name == "" || field.emit && info.kind != cTypeUnion && !info.indirect {
			continue
		}
		if field.bitWidth == 0 {
			if info.kind != cTypeUnion && !info.indirect {
				continue
			}
			t.usesUnsafe = true
			t.appendText("func(p *")
			t.out = append(t.out, info.goName...)
			t.appendText(")__c_ptr_")
			t.out = append(t.out, field.name...)
			t.appendText("()*")
			t.emitType(field.typeID)
			t.appendText("{return (*")
			t.emitType(field.typeID)
			t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(p))+uintptr(")
			t.appendDecimal(field.offset)
			t.appendText(")))}\n")
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
	t.appendText("func(p *")
	t.out = append(t.out, info.goName...)
	t.appendText(")__c_get_")
	t.out = append(t.out, field.name...)
	t.appendText("() ")
	if fieldInfo.kind == cTypeBool {
		t.appendText("uint8")
	} else {
		t.emitType(field.typeID)
	}
	t.appendText("{return ")
	if fieldInfo.kind == cTypeInt {
		t.emitType(field.typeID)
		t.out = append(t.out, '(')
		t.emitBitfieldCarrier(info, field, storageType)
		t.appendText("<<")
		t.appendDecimal(bits - field.bitOffset - field.bitWidth)
		t.appendText(")>>")
		t.appendDecimal(bits - field.bitWidth)
	} else {
		t.out = append(t.out, '(')
		t.emitBitfieldCarrier(info, field, storageType)
		if field.bitOffset > 0 {
			t.appendText(">>")
			t.appendDecimal(field.bitOffset)
		}
		t.appendText(")&")
		t.emitBitfieldMask(storageType, field.bitWidth, bits)
	}
	t.appendText("}\nfunc(p *")
	t.out = append(t.out, info.goName...)
	t.appendText(")__c_set_")
	t.out = append(t.out, field.name...)
	t.appendText("(v ")
	if fieldInfo.kind == cTypeBool {
		t.appendText("uint8")
	} else {
		t.emitType(field.typeID)
	}
	t.appendText("){")
	if info.kind == cTypeUnion || info.indirect {
		t.usesUnsafe = true
		t.appendText("q:=(*")
		t.emitType(storageType)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(p))+uintptr(")
		t.appendDecimal(field.offset)
		t.appendText(")));*q=")
		t.emitBitfieldUpdated("*q", storageType, field, bits)
	} else {
		t.appendText("p.")
		t.out = append(t.out, field.carrier...)
		t.out = append(t.out, '=')
		t.emitBitfieldUpdated("p."+field.carrier, storageType, field, bits)
	}
	t.appendText("}\n")
}

func (t *translator) emitBitfieldCarrier(info cTypeInfo, field cField, storageType int) {
	if info.kind == cTypeUnion || info.indirect {
		t.usesUnsafe = true
		t.appendText("*(*")
		t.emitType(storageType)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(p))+uintptr(")
		t.appendDecimal(field.offset)
		t.appendText(")))")
		return
	}
	t.appendText("p.")
	t.out = append(t.out, field.carrier...)
}

func (t *translator) emitBitfieldMask(storageType int, width int, bits int) {
	if width == bits {
		t.out = append(t.out, '^')
		t.emitType(storageType)
		t.appendText("(0)")
		return
	}
	t.appendText("((")
	t.emitType(storageType)
	t.appendText("(1)<<")
	t.appendDecimal(width)
	t.appendText(")-1)")
}

func (t *translator) emitBitfieldUpdated(carrier string, storageType int, field cField, bits int) {
	t.out = append(t.out, '(')
	t.out = append(t.out, carrier...)
	t.appendText("&^(")
	t.emitBitfieldMask(storageType, field.bitWidth, bits)
	if field.bitOffset > 0 {
		t.appendText("<<")
		t.appendDecimal(field.bitOffset)
	}
	t.appendText("))|((")
	t.emitType(storageType)
	t.appendText("(v)&")
	t.emitBitfieldMask(storageType, field.bitWidth, bits)
	t.out = append(t.out, ')')
	if field.bitOffset > 0 {
		t.appendText("<<")
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
	t.appendText("// renvo:linkstatic libc,")
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
		t.appendText("return nil")
	} else if resultType != cTypeVoidID {
		t.appendText("return 0")
	}
	t.out = append(t.out, '}', '\n')
}

func (t *translator) emitVariadicForeignFunction(call cVariadicCall) {
	fn := t.functions[call.function]
	if fn.defined {
		t.appendText("func ")
		t.out = append(t.out, call.goName...)
		t.out = append(t.out, '(')
		for i := 0; i < call.paramCount; i++ {
			if i > 0 {
				t.out = append(t.out, ',')
			}
			t.out = append(t.out, 'p')
			t.appendDecimal(i)
			t.out = append(t.out, ' ')
			t.emitType(t.functionParams[call.paramStart+i])
		}
		t.out = append(t.out, ')')
		if fn.resultType != cTypeVoidID {
			t.out = append(t.out, ' ')
			t.emitType(fn.resultType)
		}
		t.out = append(t.out, '{')
		if fn.resultType != cTypeVoidID {
			t.appendText("return ")
		}
		t.out = append(t.out, fn.name...)
		t.out = append(t.out, '(')
		for i := 0; i < fn.paramCount; i++ {
			if i > 0 {
				t.out = append(t.out, ',')
			}
			t.out = append(t.out, 'p')
			t.appendDecimal(i)
		}
		t.appendText(")}\n")
		return
	}
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

func (t *translator) appendText(value string) {
	t.out = append(t.out, value...)
}

func (t *translator) appendDecimalSigned(value int) {
	if value < 0 {
		t.out = append(t.out, '-')
		value = -value
	}
	t.appendDecimal(value)
}

func (t *translator) isGNUAttribute() bool {
	return t.currentIs("__attribute__") || t.currentIs("__attribute")
}

func (t *translator) parseAttributes() (cAttributes, bool) {
	var attributes cAttributes
	for t.isGNUAttribute() {
		t.pos++
		if !t.take("(") || !t.take("(") {
			return attributes, false
		}
		for {
			if t.take(")") {
				if !t.take(")") {
					return attributes, false
				}
				break
			}
			if t.kind() != tokenIdent {
				return attributes, false
			}
			name := string(tokenText(t.src, t.tokens[t.pos]))
			t.pos++
			var arguments []token
			if t.take("(") {
				end := t.findClosing(")")
				if end < t.pos {
					return attributes, false
				}
				arguments = t.tokens[t.pos:end]
				t.pos = end + 1
			}
			switch name {
			case "aligned", "__aligned__":
				alignment := 16
				if len(arguments) > 0 {
					var valid bool
					alignment, valid = t.constantExpression(arguments)
					if !valid {
						return attributes, false
					}
				}
				if alignment < 1 || alignment > 1<<30 || alignment&(alignment-1) != 0 {
					return attributes, false
				}
				if alignment > attributes.align {
					attributes.align = alignment
				}
			case "packed", "__packed__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.packed = true
			case "transparent_union", "__transparent_union__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.transparentUnion = true
			case "alias", "__alias__":
				if len(arguments) != 1 || tokenKind(arguments[0]) != tokenString {
					return attributes, false
				}
				value, valid := t.attributeString(arguments[0])
				if !valid || value == "" {
					return attributes, false
				}
				attributes.alias = value
			case "visibility", "__visibility__":
				if len(arguments) != 1 || tokenKind(arguments[0]) != tokenString {
					return attributes, false
				}
				value, valid := t.attributeString(arguments[0])
				if !valid || value != "default" && value != "internal" && value != "hidden" && value != "protected" {
					return attributes, false
				}
				attributes.visibility = value
			case "__section__", "section":
				if len(arguments) != 1 || tokenKind(arguments[0]) != tokenString {
					return attributes, false
				}
				value, valid := t.attributeString(arguments[0])
				if !valid || value == "" || value[0] != '.' {
					return attributes, false
				}
				attributes.section = value
			case "weak", "__weak__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.weak = true
			case "ms_abi", "__ms_abi__", "sysv_abi", "__sysv_abi__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.callingConvention = true
			case "cleanup", "__cleanup__":
				if len(arguments) != 1 || tokenKind(arguments[0]) != tokenIdent {
					return attributes, false
				}
				if !t.checkOnly {
					t.fail(TranslateErrUnsupported)
					return attributes, false
				}
			case "__gnu_inline__", "__unused__", "unused", "__no_instrument_function__", "no_instrument_function",
				"__always_inline__", "always_inline", "__warn_unused_result__", "warn_unused_result", "__pure__", "pure",
				"__noinline__", "noinline",
				"__const__", "const",
				"__format__", "format", "__noreturn__", "noreturn", "__malloc__", "malloc",
				"__error__", "error", "__warning__", "warning",
				"__designated_init__", "designated_init",
				"__externally_visible__", "externally_visible",
				"__alloc_size__", "alloc_size",
				"__assume_aligned__", "assume_aligned",
				"__nonnull__", "nonnull",
				"cold", "__cold__", "hot", "__hot__", "used", "__used__", "nocf_check", "__nocf_check__",
				"no_sanitize_address", "__no_sanitize_address__", "no_profile_instrument_function", "__no_profile_instrument_function__",
				"noclone", "__noclone__", "no_stack_protector", "__no_stack_protector__":
				// These attributes affect diagnostics, optimization, liveness, or
				// later object placement, but not C type layout or evaluation.
			default:
				t.fail(TranslateErrUnsupported)
				return attributes, false
			}
			if t.take(",") {
				continue
			}
			if !t.take(")") || !t.take(")") {
				return attributes, false
			}
			break
		}
	}
	return attributes, true
}

func (t *translator) attributeString(tok token) (string, bool) {
	value, ok := decodeCString(tokenText(t.src, tok))
	if !ok {
		return "", false
	}
	return string(value), true
}

func (t *translator) takeAsmClause() bool {
	if !t.currentIs("asm") && !t.currentIs("__asm") && !t.currentIs("__asm__") {
		return false
	}
	t.pos++
	for t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") ||
		t.currentIs("inline") || t.currentIs("__inline") || t.currentIs("__inline__") || t.currentIs("goto") {
		t.pos++
	}
	end := matchingToken(t.src, t.tokens, t.pos, "(", ")")
	if end < 0 {
		return false
	}
	t.pos = end + 1
	return true
}

func (t *translator) takeAsmStatement() bool {
	return t.takeAsmClause() && t.take(";")
}

func (t *translator) takeStaticAssert() bool {
	t.pos++
	if !t.take("(") {
		return false
	}
	end := t.findClosing(")")
	if end < t.pos {
		return false
	}
	items := splitTopLevel(t.src, t.tokens[t.pos:end], ",")
	if len(items) < 1 || len(items) > 2 || len(items[0]) == 0 {
		return false
	}
	value, ok := t.constantExpression(items[0])
	if !ok || value == 0 {
		return false
	}
	if len(items) == 2 && (len(items[1]) != 1 || tokenKind(items[1][0]) != tokenString) {
		return false
	}
	t.pos = end + 1
	return t.take(";")
}

func (t *translator) emitForeignType(typeID int) {
	// A Go string is used only as the shared frontend carrier for a C string
	// pointer. Object-mode call lowering emits its data word alone, so the ELF
	// boundary is the exact one-register System V `char *` ABI.
	info := t.typeInfo(typeID)
	base := t.typeInfo(info.base)
	if info.kind == cTypePointer && base.kind == cTypeInt && base.size == 1 {
		t.appendText("string")
		return
	}
	t.emitType(typeID)
}

func (t *translator) parseType() (int, int, bool) {
	t.baseAttributes = cAttributes{}
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
		case t.isGNUAttribute():
			if seen {
				goto done
			}
			attributes, ok := t.parseAttributes()
			if !ok {
				return cTypeVoidID, storage, false
			}
			t.baseAttributes.merge(attributes)
		case t.currentIs("static"):
			if storage == storageNone {
				storage = storageStatic
			} else if storage == storageThread {
				storage = storageThreadStatic
			} else {
				storage = storageInvalid
			}
			t.pos++
		case t.currentIs("_Thread_local"):
			if storage == storageNone {
				storage = storageThread
			} else if storage == storageStatic {
				storage = storageThreadStatic
			} else if storage == storageExtern {
				storage = storageThreadExtern
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
		case t.currentIs("inline") || t.currentIs("__inline") || t.currentIs("__inline__"):
			t.pos++
		case t.currentIs("extern"):
			if storage == storageNone {
				storage = storageExtern
			} else if storage == storageThread {
				storage = storageThreadExtern
			} else {
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
		case t.currentIs("const") || t.currentIs("__const") || t.currentIs("__const__"):
			qualifiers |= cQualifierConst
			t.pos++
		case t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__"):
			qualifiers |= cQualifierVolatile
			t.pos++
		case t.currentIs("restrict") || t.currentIs("__restrict") || t.currentIs("__restrict__"):
			qualifiers |= cQualifierRestrict
			t.pos++
		case t.currentIs("_Atomic"):
			t.pos++
			if t.take("(") {
				if seen {
					return cTypeVoidID, storage, false
				}
				end := t.findClosing(")")
				if end <= t.pos {
					return cTypeVoidID, storage, false
				}
				var valid bool
				typeID, valid = t.typeFromTokens(t.tokens[t.pos:end])
				if !valid || t.typeInfo(typeID).kind == cTypeArray || t.typeInfo(typeID).kind == cTypeFunction || t.typeInfo(typeID).qualifiers&cQualifierAtomic != 0 {
					return cTypeVoidID, storage, false
				}
				t.pos = end + 1
				qualifiers |= cQualifierAtomic
				hasTypeID, seen = true, true
			} else {
				qualifiers |= cQualifierAtomic
			}
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
		case t.currentIs("__int128"):
			base, seen = "int128", true
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
		case t.currentIs("__auto_type"):
			if seen {
				return cTypeVoidID, storage, false
			}
			typeID, hasTypeID, seen = t.autoType(), true, true
			t.pos++
		case t.currentIs("typeof") || t.currentIs("__typeof") || t.currentIs("__typeof__"):
			if seen {
				goto done
			}
			t.pos++
			if !t.take("(") {
				return cTypeVoidID, storage, false
			}
			end := t.findClosing(")")
			if end < t.pos {
				return cTypeVoidID, storage, false
			}
			operand := t.tokens[t.pos:end]
			if len(operand) == 0 {
				return cTypeVoidID, storage, false
			}
			outerTokens, outerPos := t.tokens, t.pos
			t.tokens, t.pos = operand, 0
			typeOperand := t.isTypeStart()
			t.tokens, t.pos = outerTokens, outerPos
			var valid bool
			if typeOperand {
				typeID, valid = t.typeFromTokens(operand)
			} else {
				typeID = t.lvalueType(operand)
				if typeID == cTypeVoidID {
					typeID = t.expressionType(operand)
				}
				valid = true
			}
			if !valid {
				return cTypeVoidID, storage, false
			}
			t.pos = end + 1
			hasTypeID, seen = true, true
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
				if !known && textEquals(typeName, "__builtin_va_list") {
					// x86_64 SysV defines va_list as an array containing one
					// 24-byte, 8-byte-aligned record. Three uintptr slots preserve
					// its size, alignment, and array-parameter decay at the ABI
					// boundary without teaching later Go phases a C-only type.
					typeID, known = t.arrayType(cTypeUintptrID, 3), true
				}
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
					if t.speculativeType {
						return cTypeVoidID, storage, false
					}
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
		if qualifiers&cQualifierAtomic != 0 && (t.typeInfo(typeID).kind == cTypeArray || t.typeInfo(typeID).kind == cTypeFunction || t.typeInfo(typeID).qualifiers&cQualifierAtomic != 0) {
			return cTypeVoidID, storage, false
		}
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
	case "int128":
		typeID = t.integer128Type(unsigned)
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

func (t *translator) integer128Type(unsigned bool) int {
	kind := cTypeInt
	name := "[2]int64"
	if unsigned {
		kind = cTypeUint
		name = "[2]uint64"
	}
	for i := cTypeUintptrID + 1; i < len(t.types); i++ {
		if t.types[i].kind == kind && t.types[i].size == 16 && t.types[i].align == 16 {
			return i
		}
	}
	t.types = append(t.types, cTypeInfo{kind: kind, size: 16, align: 16, goName: name})
	return len(t.types) - 1
}

func (t *translator) autoType() int {
	for i := cTypeUintptrID + 1; i < len(t.types); i++ {
		if t.types[i].kind == cTypeAuto && t.types[i].qualifiers == 0 {
			return i
		}
	}
	t.types = append(t.types, cTypeInfo{kind: cTypeAuto, align: 1})
	return len(t.types) - 1
}

func (t *translator) resolveAutoDeclarator(decl *declarator) bool {
	info := t.typeInfo(decl.typeID)
	if info.kind != cTypeAuto {
		return true
	}
	if len(decl.initializer) == 0 {
		return false
	}
	inferred := t.expressionType(decl.initializer)
	if inferred == cTypeVoidID || t.typeInfo(inferred).kind == cTypeFunction {
		return false
	}
	if inferredInfo := t.typeInfo(inferred); inferredInfo.kind == cTypeArray {
		inferred = t.pointerType(inferredInfo.base)
	}
	decl.typeID = t.qualifiedType(inferred, info.qualifiers)
	return true
}

func (t *translator) parseDeclarator(base int, allowFunction bool) (declarator, bool) {
	name, ops, attributes, ok := t.parseDeclaratorOps(false)
	result := declarator{name: name, typeID: base, attributes: attributes}
	if !ok {
		return result, false
	}
	result.typeID, ok = t.applyDeclaratorOps(base, ops)
	if !ok {
		return result, false
	}
	if len(ops) > 0 && ops[0].kind == declaratorArray && ops[0].incomplete {
		result.incompleteArray = true
	}
	if result.attributes.callingConvention && !t.checkOnly ||
		result.attributes.requiresObjectMetadata() && !t.checkOnly && !t.object {
		t.fail(TranslateErrUnsupported)
		return result, false
	}
	info := t.typeInfo(result.typeID)
	if info.kind == cTypeFunction {
		if !allowFunction {
			return result, false
		}
		result.function = true
		result.functionType = result.typeID
		if len(ops) > 0 && ops[0].kind == declaratorFunction {
			result.params = ops[0].params
			result.variadic = ops[0].variadic
		} else {
			for i := 0; i < info.paramCount; i++ {
				result.params = append(result.params, parameter{typeID: t.functionParams[info.paramStart+i]})
			}
			result.variadic = info.variadic
		}
		result.typeID = info.base
	} else if info.kind == cTypeOpaque {
		// The ABI size of an unknown scalar typedef cannot be inferred safely
		// from an unpreprocessed header. Opaque typedefs are exact when used
		// behind a pointer, which is the normal system-library handle shape.
		return result, false
	}
	return result, true
}

func (t *translator) applyDeclaratorOps(typeID int, ops []declaratorOp) (int, bool) {
	for i := len(ops) - 1; i >= 0; i-- {
		switch ops[i].kind {
		case declaratorPointer:
			typeID = t.qualifiedType(t.pointerType(typeID), ops[i].qualifiers)
		case declaratorArray:
			typeID = t.qualifiedType(t.arrayType(typeID, ops[i].count), ops[i].qualifiers)
		case declaratorFunction:
			kind := t.typeInfo(typeID).kind
			if kind == cTypeArray || kind == cTypeFunction {
				return typeID, false
			}
			typeID = t.functionType(typeID, ops[i].params, ops[i].variadic)
		}
	}
	return typeID, true
}

func (t *translator) parseDeclaratorOps(abstract bool) (token, []declaratorOp, cAttributes, bool) {
	attributes, ok := t.parseAttributes()
	if !ok {
		return token{}, nil, attributes, false
	}
	var pointerQualifiers []int
	for t.take("*") {
		qualifiers := 0
		for t.currentIs("const") || t.currentIs("__const") || t.currentIs("__const__") ||
			t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") ||
			t.currentIs("restrict") || t.currentIs("__restrict") || t.currentIs("__restrict__") ||
			t.currentIs("_Atomic") || t.isGNUAttribute() {
			if t.isGNUAttribute() {
				parsed, valid := t.parseAttributes()
				if !valid {
					return token{}, nil, attributes, false
				}
				attributes.merge(parsed)
				continue
			}
			if t.currentIs("const") || t.currentIs("__const") || t.currentIs("__const__") {
				qualifiers |= cQualifierConst
			} else if t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") {
				qualifiers |= cQualifierVolatile
			} else if t.currentIs("restrict") || t.currentIs("__restrict") || t.currentIs("__restrict__") {
				qualifiers |= cQualifierRestrict
			} else {
				qualifiers |= cQualifierAtomic
			}
			t.pos++
		}
		pointerQualifiers = append(pointerQualifiers, qualifiers)
	}
	parsed, valid := t.parseAttributes()
	if !valid {
		return token{}, nil, attributes, false
	}
	attributes.merge(parsed)
	var name token
	var ops []declaratorOp
	if t.kind() == tokenIdent {
		name = t.tokens[t.pos]
		t.pos++
	} else if t.take("(") {
		var nested cAttributes
		name, ops, nested, ok = t.parseDeclaratorOps(abstract)
		attributes.merge(nested)
		if !ok || !t.take(")") {
			return name, ops, attributes, false
		}
	} else if !abstract {
		return name, ops, attributes, false
	}
	for {
		parsed, valid = t.parseAttributes()
		if !valid {
			return name, ops, attributes, false
		}
		attributes.merge(parsed)
		if t.currentIs("asm") || t.currentIs("__asm") || t.currentIs("__asm__") {
			if !t.checkOnly || !t.takeAsmClause() {
				t.fail(TranslateErrUnsupported)
				return name, ops, attributes, false
			}
			continue
		}
		if t.take("[") {
			qualifiers := 0
			for t.currentIs("static") || t.currentIs("const") || t.currentIs("__const") || t.currentIs("__const__") ||
				t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") ||
				t.currentIs("restrict") || t.currentIs("__restrict") || t.currentIs("__restrict__") {
				if t.currentIs("const") || t.currentIs("__const") || t.currentIs("__const__") {
					qualifiers |= cQualifierConst
				} else if t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") {
					qualifiers |= cQualifierVolatile
				} else if t.currentIs("restrict") || t.currentIs("__restrict") || t.currentIs("__restrict__") {
					qualifiers |= cQualifierRestrict
				}
				t.pos++
			}
			end := t.findClosing("]")
			if end < t.pos {
				return name, ops, attributes, false
			}
			count := 0
			if end > t.pos {
				var valid bool
				count, valid = t.constantExpression(t.tokens[t.pos:end])
				if !valid {
					t.fail(TranslateErrVLA)
					return name, ops, attributes, false
				}
				if count < 0 {
					return name, ops, attributes, false
				}
			}
			ops = append(ops, declaratorOp{kind: declaratorArray, count: count, qualifiers: qualifiers, incomplete: end == t.pos})
			t.pos = end + 1
			continue
		}
		if t.take("(") {
			params, variadic, valid := t.parseParameterList()
			if !valid {
				return name, ops, attributes, false
			}
			ops = append(ops, declaratorOp{kind: declaratorFunction, params: params, variadic: variadic})
			continue
		}
		break
	}
	for i := len(pointerQualifiers) - 1; i >= 0; i-- {
		ops = append(ops, declaratorOp{kind: declaratorPointer, qualifiers: pointerQualifiers[i]})
	}
	return name, ops, attributes, true
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
		name, ops, _, ok := t.parseDeclaratorOps(true)
		if !ok {
			return params, false, false
		}
		paramType, ok := t.applyDeclaratorOps(paramBase, ops)
		if !ok {
			return params, false, false
		}
		paramInfo := t.typeInfo(paramType)
		if paramInfo.kind == cTypeArray {
			paramType = t.qualifiedType(t.pointerType(paramInfo.base), paramInfo.qualifiers)
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
	if !t.checkOnly && tokenIs(t.src, decl.name, "main") && t.packageName == "main" &&
		(len(decl.params) != 0 || decl.typeID != cTypeInt32ID) {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.object {
		t.emitObjectMetadata("function", string(tokenText(t.src, decl.name)), decl.attributes, storage, nil, decl.functionType)
	}
	if t.object && storage != storageStatic {
		t.appendText("//export ")
		t.out = append(t.out, tokenText(t.src, decl.name)...)
		t.out = append(t.out, '\n')
	}
	t.appendText("func ")
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
	functionScope := t.beginScope()
	oldLocalStart := t.localStart
	t.localStart = objectMark
	oldResult := t.resultType
	t.resultType = decl.typeID
	for i := 0; i < len(decl.params); i++ {
		t.rememberObject(tokenText(t.src, decl.params[i].name), decl.params[i].typeID)
	}
	t.blockInCurrentScope()
	t.resultType = oldResult
	t.localStart = oldLocalStart
	t.objects = t.objects[:objectMark]
	t.endScope(functionScope)
	t.out = append(t.out, '\n')
}

func (t *translator) emitVariable(decl declarator) {
	t.rememberObject(tokenText(t.src, decl.name), decl.typeID)
	name := string(tokenText(t.src, decl.name))
	oldInitializing := t.initializing
	t.initializing = len(t.objects) - 1
	t.emitVariableName(name, decl)
	t.initializing = oldInitializing
	if t.localStart >= 0 {
		t.appendText("_=&")
		t.out = append(t.out, name...)
		t.appendText(";")
	}
}

func (t *translator) emitVariableName(name string, decl declarator) {
	if t.localStart < 0 {
		t.emitObjectMetadata("variable", name, decl.attributes, decl.storage, decl.initializer, decl.typeID)
	}
	t.appendText("var ")
	t.out = append(t.out, name...)
	t.out = append(t.out, ' ')
	t.emitType(decl.typeID)
	if len(decl.initializer) > 0 {
		t.out = append(t.out, '=')
		if relocationKind, _ := t.objectInitializerRelocation(decl.typeID, decl.initializer); t.object && relocationKind != 0 {
			if t.typeInfo(decl.typeID).kind == cTypePointer {
				t.appendText("nil")
			} else {
				t.appendText("0")
			}
			t.out = append(t.out, ';', '\n')
			return
		}
		kind := t.typeInfo(decl.typeID).kind
		if t.checkOnly {
			t.checkInitializerValue(decl.typeID, decl.initializer)
		} else if kind == cTypeArray && t.emitStringArrayInitializer(decl.typeID, decl.initializer) {
		} else if (kind == cTypeArray || kind == cTypeStruct) && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitInitializerValue(decl.typeID, decl.initializer)
		} else if kind == cTypeUnion && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitUnionInitializer(decl.typeID, decl.initializer)
		} else {
			t.convertedExpression(decl.typeID, decl.initializer)
		}
	}
	t.out = append(t.out, ';', '\n')
}

func (t *translator) emitObjectAlias(decl declarator, storage int, function bool) {
	if !t.object || decl.attributes.alias == "" {
		return
	}
	t.typeSerial++
	kind := "variable-alias"
	if function {
		kind = "function-alias"
	}
	t.emitObjectMetadata(kind, string(tokenText(t.src, decl.name)), decl.attributes, storage, nil, decl.typeID)
	t.appendText("var __c_object_alias_")
	t.appendDecimal(t.typeSerial)
	t.appendText(" uint8\n")
}

func (t *translator) emitObjectMetadata(kind string, name string, attributes cAttributes, storage int, initializer []token, typeID int) {
	if !t.object {
		return
	}
	section := attributes.section
	if section == "" {
		if kind == "function" && t.functionSections {
			section = ".text." + name
		} else if kind == "variable" && t.dataSections {
			prefix := ".bss."
			if len(initializer) != 0 {
				prefix = ".data."
				if t.typeInfo(typeID).qualifiers&cQualifierConst != 0 {
					prefix = ".rodata."
				}
			}
			section = prefix + name
		}
	}
	if section == "" {
		section = "-"
	}
	binding := 1
	if storage == storageStatic {
		binding = 0
	}
	if attributes.weak {
		binding = 2
	}
	visibility := 0
	switch attributes.visibility {
	case "internal":
		visibility = 1
	case "hidden":
		visibility = 2
	case "protected":
		visibility = 3
	}
	alias := attributes.alias
	if alias == "" {
		alias = "-"
	}
	t.appendText("// renvo:object ")
	t.appendText(kind)
	t.out = append(t.out, ' ')
	t.appendText(name)
	t.out = append(t.out, ' ')
	t.appendText(section)
	t.out = append(t.out, ' ')
	alignment := attributes.align
	if kind != "function" && kind != "function-alias" && t.typeInfo(typeID).align > alignment {
		alignment = t.typeInfo(typeID).align
	}
	t.appendDecimal(alignment)
	t.out = append(t.out, ' ')
	t.appendDecimal(binding)
	t.out = append(t.out, ' ')
	t.appendDecimal(visibility)
	t.out = append(t.out, ' ')
	t.appendText(alias)
	t.out = append(t.out, ' ')
	t.appendDecimal(t.typeSize(typeID))
	relocationKind, relocationTarget := t.objectInitializerRelocation(typeID, initializer)
	t.out = append(t.out, ' ')
	t.appendDecimal(relocationKind)
	t.out = append(t.out, ' ')
	if relocationTarget == "" {
		relocationTarget = "-"
	}
	t.appendText(relocationTarget)
	t.appendText(" 0")
	t.out = append(t.out, '\n')
}

func (t *translator) objectInitializerRelocation(typeID int, initializer []token) (int, string) {
	if len(initializer) < 2 {
		return 0, ""
	}
	ampersand := -1
	for i := len(initializer) - 2; i >= 0; i-- {
		if tokenIs(t.src, initializer[i], "&") && tokenKind(initializer[i+1]) == tokenIdent && i+2 == len(initializer) {
			ampersand = i
			break
		}
	}
	if ampersand < 0 {
		return 0, ""
	}
	size := t.typeSize(typeID)
	if size == 8 {
		return 1, string(tokenText(t.src, initializer[ampersand+1]))
	}
	if size == 4 {
		if t.typeInfo(typeID).kind == cTypeUint {
			return 10, string(tokenText(t.src, initializer[ampersand+1]))
		}
		return 11, string(tokenText(t.src, initializer[ampersand+1]))
	}
	return 0, ""
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

func (t *translator) rememberThreadExtern(decl declarator) {
	name := string(tokenText(t.src, decl.name))
	index := t.threadName(name)
	if index >= 0 {
		if !t.compatibleParameterType(t.threadNames[index].typeID, decl.typeID) {
			t.fail(TranslateErrDeclaration)
			return
		}
		t.rememberThreadObject(name, t.threadNames[index].goName, decl.typeID)
		return
	}
	t.typeSerial++
	getter := "__c_tls_get_" + decimalString(t.typeSerial)
	t.threadNames = append(t.threadNames, cObjectName{name: name, goName: getter, typeID: decl.typeID})
	t.rememberThreadObject(name, getter, decl.typeID)
}

func (t *translator) threadName(name string) int {
	for i := 0; i < len(t.threadNames); i++ {
		if t.threadNames[i].name == name {
			return i
		}
	}
	return -1
}

func (t *translator) rememberThreadObject(name string, getter string, typeID int) {
	t.rememberAliasedObject([]byte(name), "(*"+getter+"())", typeID)
}

func (t *translator) emitThreadVariable(decl declarator, linked bool) {
	t.ensureThreadRuntime()
	name := string(tokenText(t.src, decl.name))
	getter, number := "", ""
	if linked {
		index := t.threadName(name)
		if index >= 0 {
			if t.threadNames[index].auto || !t.compatibleParameterType(t.threadNames[index].typeID, decl.typeID) {
				t.fail(TranslateErrDeclaration)
				return
			}
			t.threadNames[index].auto = true
			getter = t.threadNames[index].goName
			number = getter[len("__c_tls_get_"):]
		}
	}
	if getter == "" {
		t.typeSerial++
		number = decimalString(t.typeSerial)
		getter = "__c_tls_get_" + number
		if linked {
			t.threadNames = append(t.threadNames, cObjectName{name: name, goName: getter, typeID: decl.typeID, auto: true})
		}
	}
	t.rememberThreadObject(name, getter, decl.typeID)
	var initializer []byte
	if len(decl.initializer) != 0 {
		outer := t.out
		t.out = nil
		info := t.typeInfo(decl.typeID)
		if (info.kind == cTypeArray || info.kind == cTypeStruct || info.kind == cTypeUnion) && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitInitializerValue(decl.typeID, decl.initializer)
		} else {
			t.convertedExpression(decl.typeID, decl.initializer)
		}
		initializer = t.out
		t.out = outer
	}
	outer := t.out
	t.out = t.staticOut
	t.appendText("var __c_tls_key_")
	t.out = append(t.out, number...)
	t.appendText(" uint32\nvar __c_tls_ready_")
	t.out = append(t.out, number...)
	t.appendText(" uint8\nfunc ")
	t.out = append(t.out, getter...)
	t.appendText("() *")
	t.emitType(decl.typeID)
	t.appendText("{__c_pthread_mutex_lock((*byte)(__c_unsafe.Pointer(&__c_tls_mutex)));if __c_tls_ready_")
	t.out = append(t.out, number...)
	t.appendText("==0{__c_pthread_key_create(&__c_tls_key_")
	t.out = append(t.out, number...)
	t.appendText(",nil);__c_tls_ready_")
	t.out = append(t.out, number...)
	t.appendText("=1};__c_pthread_mutex_unlock((*byte)(__c_unsafe.Pointer(&__c_tls_mutex)));raw:=__c_pthread_getspecific(__c_tls_key_")
	t.out = append(t.out, number...)
	t.appendText(");if raw==nil{raw=__c_calloc(1,")
	t.appendDecimal(t.typeSize(decl.typeID))
	t.appendText(");__c_pthread_setspecific(__c_tls_key_")
	t.out = append(t.out, number...)
	t.appendText(",raw)")
	if len(initializer) != 0 {
		t.appendText(";*(*")
		t.emitType(decl.typeID)
		t.appendText(")(__c_unsafe.Pointer(raw))=")
		t.out = append(t.out, initializer...)
	}
	t.appendText("};return (*")
	t.emitType(decl.typeID)
	t.appendText(")(__c_unsafe.Pointer(raw))}\n")
	t.staticOut = t.out
	t.out = outer
	t.usesUnsafe = true
}

func (t *translator) ensureThreadRuntime() {
	if t.tlsRuntime {
		return
	}
	t.tlsRuntime = true
	t.staticOut = append(t.staticOut, "var __c_tls_mutex [40]byte\n// renvo:linkstatic libc,pthread_key_create\nfunc __c_pthread_key_create(p0 *uint32,p1 *byte) int32{return 0}\n// renvo:linkstatic libc,pthread_mutex_lock\nfunc __c_pthread_mutex_lock(p0 *byte) int32{return 0}\n// renvo:linkstatic libc,pthread_mutex_unlock\nfunc __c_pthread_mutex_unlock(p0 *byte) int32{return 0}\n// renvo:linkstatic libc,pthread_getspecific\nfunc __c_pthread_getspecific(p0 uint32) *byte{return nil}\n// renvo:linkstatic libc,pthread_setspecific\nfunc __c_pthread_setspecific(p0 uint32,p1 *byte) int32{return 0}\n// renvo:linkstatic libc,calloc\nfunc __c_calloc(p0 uintptr,p1 uintptr) *byte{return nil}\n"...)
}

func (t *translator) block() {
	t.blockWithScope(true)
}

func (t *translator) blockInCurrentScope() {
	t.blockWithScope(false)
}

func (t *translator) blockWithScope(scoped bool) {
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	objectMark := len(t.objects)
	var scope cScopeMark
	if scoped {
		scope = t.beginScope()
	}
	t.out = append(t.out, '{')
	for t.ok && t.kind() != tokenEOF && !t.currentIs("}") {
		t.skipDirectives()
		if t.currentIs("}") {
			break
		}
		if t.checkOnly {
			mark := t.beginCheckScratch()
			t.statement()
			t.endCheckScratch(mark)
		} else {
			t.statement()
		}
		if t.ok && !t.checkOnly {
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
	if scoped {
		t.endScope(scope)
	}
}

func (t *translator) statement() {
	if t.currentIs("_Static_assert") || t.currentIs("static_assert") {
		if !t.takeStaticAssert() {
			t.fail(TranslateErrDeclaration)
		}
		return
	}
	if t.currentIs("{") {
		t.block()
		return
	}
	if t.isTypeStart() {
		t.localDeclaration()
		return
	}
	if t.take("if") {
		t.appendText("if ")
		t.parenthesizedCondition()
		t.out = append(t.out, ' ')
		t.controlledStatement()
		if t.take("else") {
			t.appendText(" else ")
			t.controlledStatement()
		}
		return
	}
	if t.take("while") {
		t.appendText("for ")
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
		t.appendText("goto ")
		t.out = append(t.out, tokenText(t.src, t.tokens[t.pos])...)
		t.pos++
		if !t.take(";") {
			t.fail(TranslateErrStatement)
			return
		}
		t.out = append(t.out, ';')
		return
	}
	if t.currentIs("asm") || t.currentIs("__asm") || t.currentIs("__asm__") {
		if !t.checkOnly || !t.takeAsmStatement() {
			t.fail(TranslateErrUnsupported)
			return
		}
		t.out = append(t.out, ';')
		return
	}
	if t.currentIs("case") || t.currentIs("default") {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.take("return") {
		t.appendText("return")
		start := t.pos
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		if end > start {
			t.out = append(t.out, ' ')
			t.convertedExpression(t.resultType, t.tokens[start:end])
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
	typeAttributes := t.baseAttributes
	if t.typeInfo(base).qualifiers&cQualifierRestrict != 0 && t.typeInfo(base).kind != cTypePointer {
		t.fail(TranslateErrDeclaration)
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
		decl.attributes.merge(typeAttributes)
		decl.initializer = t.takeInitializer()
		autoDeclaration := t.typeInfo(decl.typeID).kind == cTypeAuto
		if !t.resolveAutoDeclarator(&decl) {
			t.fail(TranslateErrDeclaration)
			return
		}
		if storage != storageExtern && (!t.completeInitializerArray(&decl) || t.typeInfo(decl.typeID).kind == cTypeArray && decl.incompleteArray) {
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
			t.objects[len(t.objects)-1].auto = false
		} else if storage == storageThread || storage == storageThreadStatic {
			t.emitThreadVariable(decl, false)
		} else if storage == storageThreadExtern {
			t.rememberThreadExtern(decl)
		} else if storage == storageStatic {
			t.emitStaticLocal(decl)
		} else {
			t.emitVariable(decl)
		}
		if !t.take(",") {
			break
		}
		if autoDeclaration {
			t.fail(TranslateErrDeclaration)
			return
		}
	}
	if !t.take(";") {
		t.fail(TranslateErrDeclaration)
	}
}

func (t *translator) forStatement() {
	objectMark := len(t.objects)
	scope := t.beginScope()
	t.forStatementBody()
	t.objects = t.objects[:objectMark]
	t.endScope(scope)
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
		t.appendText("for ;")
	} else {
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		t.appendText("for ")
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
	scope := t.beginScope()
	t.switchStatementBody()
	t.objects = t.objects[:objectMark]
	t.endScope(scope)
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
	switchType := t.integerPromotion(t.expressionType(t.tokens[t.pos:close]))
	t.appendText("switch ")
	t.convertedExpression(switchType, t.tokens[t.pos:close])
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
				t.appendText("fallthrough;")
			}
			terminal = false
			haveCase = true
			if t.take("case") {
				end := t.findAtDepth(":")
				if end < 0 || end == t.pos {
					t.fail(TranslateErrStatement)
					return
				}
				caseTokens := t.tokens[t.pos:end]
				if rangeAt := topLevelToken(t.src, caseTokens, "..."); rangeAt >= 0 {
					if !t.checkOnly {
						t.fail(TranslateErrUnsupported)
						return
					}
					first, firstOK := t.constantExpression(caseTokens[:rangeAt])
					last, lastOK := t.constantExpression(caseTokens[rangeAt+1:])
					if !firstOK || !lastOK || first > last {
						t.fail(TranslateErrStatement)
						return
					}
					t.pos = end + 1
					continue
				}
				t.appendText("case ")
				t.convertedExpression(switchType, caseTokens)
				t.out = append(t.out, ':')
				t.pos = end + 1
			} else {
				t.pos++
				if !t.take(":") {
					t.fail(TranslateErrStatement)
					return
				}
				t.appendText("default:")
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
		if t.checkOnly {
			mark := t.beginCheckScratch()
			t.statement()
			t.endCheckScratch(mark)
		} else {
			t.statement()
		}
		if t.ok && !t.checkOnly {
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
	t.appendText("{var ")
	t.out = append(t.out, name...)
	t.appendText(" bool=true;for ")
	t.out = append(t.out, name...)
	t.appendText("||")
	t.condition(t.tokens[t.pos:close])
	t.appendText(" {")
	t.out = append(t.out, name...)
	t.appendText("=false;")
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
	if t.checkOnly {
		t.checkExpression(tokens)
		return
	}
	if len(tokens) == 0 {
		t.appendText("true")
		return
	}
	t.emitConditionExpression(tokens)
}

func (t *translator) expression(tokens []token) {
	if t.checkOnly {
		t.checkExpression(tokens)
		return
	}
	if t.rejectConstMutation(tokens) {
		return
	}
	if t.emitBitfieldMutation(tokens) {
		return
	}
	if t.emitConvertedAssignment(tokens) {
		return
	}
	t.emitExpression(tokens)
}

func (t *translator) checkExpression(tokens []token) {
	if len(tokens) == 0 || t.rejectConstMutation(tokens) {
		return
	}
	for i := 0; i+2 < len(tokens); i++ {
		if !tokenIs(t.src, tokens[i], "(") || !tokenIs(t.src, tokens[i+1], "{") {
			continue
		}
		bodyEnd := matchingToken(t.src, tokens, i+1, "{", "}")
		if bodyEnd < 0 || bodyEnd+1 >= len(tokens) || !tokenIs(t.src, tokens[bodyEnd+1], ")") {
			t.fail(TranslateErrStatement)
			return
		}
		outerTokens, outerPos := t.tokens, t.pos
		t.tokens, t.pos = tokens[i+1:bodyEnd+1], 0
		t.block()
		t.tokens, t.pos = outerTokens, outerPos
		if !t.ok {
			return
		}
		i = bodyEnd + 1
	}
	_ = t.expressionType(tokens)
}

func (t *translator) emitConvertedAssignment(tokens []token) bool {
	depth := 0
	for i := 0; i < len(tokens); i++ {
		if tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") {
			depth++
		} else if tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") {
			depth--
		} else if depth == 0 && tokenIs(t.src, tokens[i], "=") {
			t.emitExpression(tokens[:i])
			t.out = append(t.out, '=')
			t.convertedExpression(t.lvalueType(tokens[:i]), tokens[i+1:])
			return true
		}
	}
	return false
}

func (t *translator) convertedExpression(typeID int, tokens []token) {
	if t.checkOnly {
		t.checkExpression(tokens)
		return
	}
	if t.typeInfo(typeID).kind == cTypePointer {
		if value, ok := t.constantExpression(tokens); ok && value == 0 {
			t.appendText("nil")
			return
		}
	}
	if len(tokens) > 3 && tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close+1 < len(tokens) && tokenIs(t.src, tokens[close+1], "{") {
			end := matchingToken(t.src, tokens, close+1, "{", "}")
			if end == len(tokens)-1 {
				t.emitInitializerValue(typeID, tokens[close+1:])
				return
			}
		}
	}
	if t.typeInfo(typeID).kind == cTypeBool {
		t.appendText("uint8(")
		if t.booleanExpression(tokens) {
			t.emitBoolInteger(tokens)
		} else {
			t.emitBoolIntegerComparison(tokens)
		}
		t.out = append(t.out, ')')
		return
	}
	info := t.typeInfo(typeID)
	if info.kind == cTypeInt || info.kind == cTypeUint {
		if value, ok := t.constantExpression(tokens); ok {
			if value, ok = t.convertIntegerConstant(typeID, value); ok {
				if info.kind == cTypeUint && info.size == 8 && value < 0 {
					t.appendText("^uint64(")
					t.appendDecimal(-value - 1)
					t.out = append(t.out, ')')
				} else {
					t.appendDecimalSigned(value)
				}
				return
			}
		}
	}
	sourceType := t.expressionType(tokens)
	sourceInfo := t.typeInfo(sourceType)
	if t.typeInfo(typeID).kind == cTypePointer && sourceInfo.kind == cTypeArray &&
		t.compatibleParameterType(t.typeInfo(typeID).base, sourceInfo.base) {
		t.appendText("&(")
		t.expression(tokens)
		t.appendText(")[0]")
		return
	}
	if typeID == cTypeVoidID || sourceType == typeID && !t.booleanExpression(tokens) || t.pointerConversionCompatible(typeID, sourceType) {
		t.expression(tokens)
		return
	}
	if t.booleanExpression(tokens) {
		t.emitBoolInteger(tokens)
		return
	}
	if info.kind == cTypeStruct || info.kind == cTypeUnion || info.kind == cTypeArray || info.kind == cTypeFunction {
		t.expression(tokens)
		return
	}
	if info.kind == cTypePointer {
		t.out = append(t.out, '(')
		t.emitType(typeID)
		t.out = append(t.out, ')')
	} else {
		t.emitType(typeID)
	}
	t.out = append(t.out, '(')
	t.expression(tokens)
	t.out = append(t.out, ')')
}

func (t *translator) pointerConversionCompatible(left int, right int) bool {
	a, b := t.typeInfo(left), t.typeInfo(right)
	if a.kind != cTypePointer || b.kind != cTypePointer {
		return false
	}
	return t.compatibleParameterType(a.base, b.base)
}

func (t *translator) emitBoolIntegerComparison(tokens []token) {
	t.ensureBoolHelper()
	t.appendText("__c_bool_int((")
	t.expression(tokens)
	if t.typeInfo(t.expressionType(tokens)).kind == cTypePointer {
		t.appendText(")!=nil)")
	} else {
		t.appendText(")!=0)")
	}
}

func (t *translator) emitBoolInteger(tokens []token) {
	t.ensureBoolHelper()
	t.appendText("__c_bool_int(")
	t.emitExpression(tokens)
	t.out = append(t.out, ')')
}

func (t *translator) ensureBoolHelper() {
	if !t.boolHelper {
		t.staticOut = append(t.staticOut, "func __c_bool_int(v bool) int32{if v{return 1};return 0}\n"...)
		t.boolHelper = true
	}
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
		if t.typeInfo(t.postfixMutationType(tokens[:len(tokens)-1])).qualifiers&cQualifierConst != 0 {
			t.fail(TranslateErrUnsupported)
			return true
		}
	}
	return false
}

func (t *translator) postfixMutationType(tokens []token) int {
	// Postfix operators bind before an unparenthesized unary prefix: *p++ is
	// *(p++), while (*p)++ still targets the pointee.
	for len(tokens) > 1 && !tokenIs(t.src, tokens[0], "(") && t.unaryOperator(tokens[0]) {
		tokens = tokens[1:]
	}
	return t.lvalueType(tokens)
}

func (t *translator) lvalueType(tokens []token) int {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) == 1 && tokenKind(tokens[0]) == tokenIdent {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			return object.typeID
		}
		if function, ok := t.lookupFunction(tokenText(t.src, tokens[0])); ok {
			return t.functions[function].typeID
		}
	}
	if access, ok := t.memberAccess(tokens, 0); ok && access.end == len(tokens) {
		return access.typeID
	}
	if memberType, ok := t.memberExpressionType(tokens); ok {
		return memberType
	}
	if subscriptType, ok := t.subscriptExpressionType(tokens); ok {
		return subscriptType
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
	if consumed := t.nullPointerPrefix(tokens); consumed == len(tokens) {
		t.appendText("nil")
		return
	}
	if len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		t.out = append(t.out, '(')
		t.emitExpression(tokens[1 : len(tokens)-1])
		t.out = append(t.out, ')')
		return
	}
	if at := t.commaOperator(tokens); at >= 0 {
		t.emitCommaExpression(tokens[:at], tokens[at+1:])
		return
	}
	if question, colon := t.conditionalOperator(tokens); question >= 0 {
		t.emitConditionalExpression(tokens, question, colon)
		return
	}
	if at := t.binaryOperator(tokens); at >= 0 {
		t.emitBinaryExpression(tokens, at)
		return
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "!") {
		t.out = append(t.out, '!')
		t.emitConditionExpression(tokens[1:])
		return
	}
	if len(tokens) > 1 && t.unaryArithmetic(tokens[0]) {
		typeID := t.integerPromotion(t.expressionType(tokens[1:]))
		t.emitType(typeID)
		t.out = append(t.out, '(')
		if tokenIs(t.src, tokens[0], "~") {
			t.out = append(t.out, '-')
			t.convertedExpression(typeID, tokens[1:])
			t.appendText("-1)")
			return
		} else {
			t.out = append(t.out, tokenText(t.src, tokens[0])...)
		}
		t.convertedExpression(typeID, tokens[1:])
		t.out = append(t.out, ')')
		return
	}
	if len(tokens) > 1 && t.sizeOperator(tokens[0]) {
		t.emitSizeof(tokens)
		return
	}
	if len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close < len(tokens)-1 && !tokenIs(t.src, tokens[close+1], "{") {
			if typeID, ok := t.typeFromTokens(tokens[1:close]); ok {
				pointer := t.typeInfo(typeID).kind == cTypePointer
				if pointer {
					t.out = append(t.out, '(')
				}
				t.emitType(typeID)
				if pointer {
					t.out = append(t.out, ')')
				}
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
			if function, ok := t.lookupFunction(tokenText(t.src, tok)); ok {
				close := matchingToken(t.src, tokens, i+1, "(", ")")
				if close > i+1 && t.functions[function].variadic && t.emitVariadicCall(function, tokens[i+2:close]) {
					i = close
					continue
				}
				if close > i+1 && !t.functions[function].variadic && t.emitFixedCall(function, tokens[i+2:close]) {
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
		if tokenIs(t.src, tok, "sizeof") || t.isAlignof(tok) {
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
			t.appendText("nil")
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
			end := i + 1
			for end < len(tokens) && tokenKind(tokens[end]) == tokenString {
				end++
			}
			t.emitCStringValues(tokens[i:end])
			i = end - 1
			continue
		} else if tokenKind(tok) == tokenNumber {
			text = trimIntegerSuffix(text)
			if len(text) > 0 && (text[len(text)-1] == 'f' || text[len(text)-1] == 'F') {
				text = text[:len(text)-1]
			}
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

func (t *translator) emitSizeof(tokens []token) {
	operand := tokens[1:]
	if t.isAlignof(tokens[0]) && (len(operand) < 3 || !tokenIs(t.src, operand[0], "(")) {
		t.fail(TranslateErrUnsupported)
		return
	}
	if len(operand) > 2 && tokenIs(t.src, operand[0], "(") && matchingToken(t.src, operand, 0, "(", ")") == len(operand)-1 {
		inside := operand[1 : len(operand)-1]
		if typeID, ok := t.typeFromTokens(inside); ok {
			if tokenIs(t.src, tokens[0], "sizeof") {
				t.appendDecimal(t.typeSize(typeID))
			} else {
				t.appendDecimal(t.typeAlign(typeID))
			}
			return
		}
		operand = inside
	}
	if t.isAlignof(tokens[0]) {
		t.fail(TranslateErrUnsupported)
		return
	}
	if value, ok := t.cStringBytes(operand); ok {
		t.appendDecimal(len(value))
		return
	}
	t.appendDecimal(t.typeSize(t.expressionType(operand)))
}

func (t *translator) emitFixedCall(function int, tokens []token) bool {
	fn := t.functions[function]
	args := splitTopLevel(t.src, tokens, ",")
	if len(tokens) == 0 {
		args = nil
	}
	if len(args) != fn.paramCount {
		t.fail(TranslateErrUnsupported)
		return true
	}
	t.out = append(t.out, fn.name...)
	t.out = append(t.out, '(')
	for i := 0; i < len(args); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.convertedExpression(t.functionParams[fn.paramStart+i], args[i])
	}
	t.out = append(t.out, ')')
	return true
}

func (t *translator) conditionalOperator(tokens []token) (int, int) {
	paren, bracket, question := 0, 0, -1
	nested := 0
	for i := 0; i < len(tokens); i++ {
		switch string(tokenText(t.src, tokens[i])) {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			bracket++
		case "]":
			bracket--
		case "?":
			if paren == 0 && bracket == 0 {
				if question < 0 {
					question = i
				} else {
					nested++
				}
			}
		case ":":
			if paren == 0 && bracket == 0 && question >= 0 {
				if nested == 0 {
					return question, i
				}
				nested--
			}
		}
	}
	return -1, -1
}

func (t *translator) emitConditionalExpression(tokens []token, question int, colon int) {
	if colon == question+1 {
		t.fail(TranslateErrUnsupported)
		return
	}
	left, right := t.expressionType(tokens[question+1:colon]), t.expressionType(tokens[colon+1:])
	typeID := t.usualArithmeticType(left, right)
	if t.typeInfo(left).kind == cTypePointer {
		typeID = left
	} else if t.typeInfo(right).kind == cTypePointer {
		typeID = right
	}
	t.typeSerial++
	name := "__c_conditional_" + decimalString(t.typeSerial)
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	t.emitConditionExpression(tokens[:question])
	t.emitCaptureArguments(true)
	t.out = append(t.out, ')')
	outer, names := t.beginHelper()
	t.appendText("func ")
	t.out = append(t.out, name...)
	t.appendText("(c bool")
	t.emitCaptureParameters(true)
	t.appendText(") ")
	t.emitType(typeID)
	t.appendText("{if c{return ")
	t.convertedExpression(typeID, tokens[question+1:colon])
	t.appendText("};return ")
	t.convertedExpression(typeID, tokens[colon+1:])
	t.appendText("}\n")
	t.endHelper(outer, names)
}

func (t *translator) commaOperator(tokens []token) int {
	paren, bracket, brace := 0, 0, 0
	for i := len(tokens) - 1; i >= 0; i-- {
		switch {
		case tokenIs(t.src, tokens[i], ")"):
			paren++
		case tokenIs(t.src, tokens[i], "("):
			paren--
		case tokenIs(t.src, tokens[i], "]"):
			bracket++
		case tokenIs(t.src, tokens[i], "["):
			bracket--
		case tokenIs(t.src, tokens[i], "}"):
			brace++
		case tokenIs(t.src, tokens[i], "{"):
			brace--
		case paren == 0 && bracket == 0 && brace == 0 && tokenIs(t.src, tokens[i], ","):
			return i
		}
	}
	return -1
}

func (t *translator) emitCommaExpression(left []token, right []token) {
	typeID := t.expressionType(right)
	t.typeSerial++
	name := "__c_comma_" + decimalString(t.typeSerial)
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	t.emitCaptureArguments(false)
	t.out = append(t.out, ')')
	outer, names := t.beginHelper()
	t.appendText("func ")
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	t.emitCaptureParameters(false)
	t.appendText(") ")
	t.emitType(typeID)
	t.out = append(t.out, '{')
	t.expression(left)
	t.appendText(";return ")
	t.convertedExpression(typeID, right)
	t.appendText("}\n")
	t.endHelper(outer, names)
}

func (t *translator) emitCaptureArguments(comma bool) {
	if t.localStart < 0 {
		return
	}
	for i := t.localStart; i < len(t.objects); i++ {
		if t.captureObject(i) {
			if comma {
				t.out = append(t.out, ',')
			}
			t.out = append(t.out, '&')
			t.out = append(t.out, t.objects[i].goName...)
			comma = true
		}
	}
}

func (t *translator) emitCaptureParameters(comma bool) {
	parameter := 0
	for i := t.localStart; i >= 0 && i < len(t.objects); i++ {
		if !t.captureObject(i) {
			continue
		}
		if comma || parameter > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, 'p')
		t.appendDecimal(parameter)
		t.appendText(" *")
		t.emitType(t.objects[i].typeID)
		parameter++
	}
}

func (t *translator) beginHelper() ([]byte, []string) {
	outer := t.out
	t.out = nil
	var names []string
	parameter := 0
	for i := t.localStart; i >= 0 && i < len(t.objects); i++ {
		if !t.captureObject(i) {
			continue
		}
		names = append(names, t.objects[i].goName)
		t.objects[i].goName = "(*p" + decimalString(parameter) + ")"
		parameter++
	}
	return outer, names
}

func (t *translator) endHelper(outer []byte, names []string) {
	body := t.out
	name := 0
	for i := t.localStart; i >= 0 && i < len(t.objects); i++ {
		if name < len(names) && t.objects[i].auto && t.objects[i].goName == "(*p"+decimalString(name)+")" {
			t.objects[i].goName = names[name]
			name++
		}
	}
	t.staticOut = append(t.staticOut, body...)
	t.out = outer
}

func (t *translator) captureObject(index int) bool {
	return index >= 0 && index < len(t.objects) && index != t.initializing && t.objects[index].auto
}

func (t *translator) binaryOperator(tokens []token) int {
	best, precedence := -1, 100
	paren, bracket := 0, 0
	operand := true
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
			operand = true
			continue
		case tokenIs(t.src, tokens[i], ")"):
			paren--
			operand = false
			continue
		case tokenIs(t.src, tokens[i], "["):
			bracket++
			continue
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
			continue
		}
		if paren != 0 || bracket != 0 {
			continue
		}
		if operand && t.unaryOperator(tokens[i]) {
			continue
		}
		p := t.binaryPrecedence(tokens[i])
		if p != 0 {
			if p <= precedence {
				best, precedence = i, p
			}
			operand = true
		} else {
			operand = false
		}
	}
	return best
}

func (t *translator) binaryPrecedence(tok token) int {
	return cBinaryPrecedence(tokenText(t.src, tok))
}

func (t *translator) emitBinaryExpression(tokens []token, at int) {
	op := tokens[at]
	left, right := tokens[:at], tokens[at+1:]
	if tokenIs(t.src, op, "&&") || tokenIs(t.src, op, "||") {
		t.emitConditionExpression(left)
		t.out = append(t.out, tokenText(t.src, op)...)
		t.emitConditionExpression(right)
		return
	}
	leftType, rightType := t.expressionType(left), t.expressionType(right)
	shift := tokenIs(t.src, op, "<<") || tokenIs(t.src, op, ">>")
	typeID := t.usualArithmeticType(leftType, rightType)
	if shift {
		typeID = t.integerPromotion(leftType)
	}
	t.convertedExpression(typeID, left)
	t.out = append(t.out, tokenText(t.src, op)...)
	if shift {
		t.convertedExpression(t.integerPromotion(rightType), right)
	} else {
		t.convertedExpression(typeID, right)
	}
}

func (t *translator) emitConditionExpression(tokens []token) {
	if t.booleanExpression(tokens) {
		t.emitExpression(tokens)
	} else {
		t.out = append(t.out, '(')
		t.emitExpression(tokens)
		t.appendText(")!=0")
	}
}

func (t *translator) booleanExpression(tokens []token) bool {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if question, _ := t.conditionalOperator(tokens); question >= 0 {
		return false
	}
	at := t.binaryOperator(tokens)
	if at < 0 {
		return len(tokens) > 0 && tokenIs(t.src, tokens[0], "!")
	}
	return t.binaryPrecedence(tokens[at]) <= 2 || t.binaryPrecedence(tokens[at]) == 6 || t.binaryPrecedence(tokens[at]) == 7
}

func (t *translator) integerPromotion(typeID int) int {
	info := t.typeInfo(typeID)
	if info.kind == cTypeBool || (info.kind == cTypeInt || info.kind == cTypeUint) && info.size < 4 {
		return cTypeInt32ID
	}
	return typeID
}

func (t *translator) usualArithmeticType(left int, right int) int {
	a, b := t.integerPromotion(left), t.integerPromotion(right)
	ai, bi := t.typeInfo(a), t.typeInfo(b)
	if ai.kind == cTypeFloat || bi.kind == cTypeFloat {
		if ai.size > bi.size {
			return a
		}
		return b
	}
	if ai.kind != cTypeInt && ai.kind != cTypeUint || bi.kind != cTypeInt && bi.kind != cTypeUint {
		return a
	}
	if ai.kind == bi.kind {
		if ai.size >= bi.size {
			return a
		}
		return b
	}
	unsigned, signed := a, b
	ui, si := ai, bi
	if ai.kind == cTypeInt {
		unsigned, signed, ui, si = b, a, bi, ai
	}
	if ui.size >= si.size {
		return unsigned
	}
	return signed
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
	if typeID, ok := t.statementExpressionType(tokens); ok {
		return typeID
	}
	if typeID, ok := t.genericExpressionType(tokens); ok {
		return typeID
	}
	if at := t.commaOperator(tokens); at >= 0 {
		return t.expressionType(tokens[at+1:])
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "!") {
		return cTypeInt32ID
	}
	if len(tokens) > 1 && t.unaryArithmetic(tokens[0]) {
		return t.integerPromotion(t.expressionType(tokens[1:]))
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "*") {
		info := t.typeInfo(t.expressionType(tokens[1:]))
		if info.kind == cTypePointer {
			return info.base
		}
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "&") {
		return t.pointerType(t.lvalueType(tokens[1:]))
	}
	if tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close < len(tokens)-1 {
			if typeID, ok := t.typeFromTokens(tokens[1:close]); ok {
				return typeID
			}
		}
	}
	if question, colon := t.conditionalOperator(tokens); question >= 0 {
		leftTokens := tokens[question+1 : colon]
		if len(leftTokens) == 0 {
			leftTokens = tokens[:question]
		}
		left, right := t.expressionType(leftTokens), t.expressionType(tokens[colon+1:])
		if t.typeInfo(left).kind == cTypePointer {
			return left
		}
		if t.typeInfo(right).kind == cTypePointer {
			return right
		}
		return t.usualArithmeticType(left, right)
	}
	if at := t.binaryOperator(tokens); at >= 0 {
		if t.booleanExpression(tokens) {
			return cTypeInt32ID
		}
		if tokenIs(t.src, tokens[at], "<<") || tokenIs(t.src, tokens[at], ">>") {
			return t.integerPromotion(t.expressionType(tokens[:at]))
		}
		return t.usualArithmeticType(t.expressionType(tokens[:at]), t.expressionType(tokens[at+1:]))
	}
	if subscriptType, ok := t.subscriptExpressionType(tokens); ok {
		return subscriptType
	}
	if memberType, ok := t.memberExpressionType(tokens); ok {
		return memberType
	}
	if access, ok := t.memberAccess(tokens, 0); ok && access.end == len(tokens) {
		return access.typeID
	}
	if len(tokens) > 3 && tokenKind(tokens[0]) == tokenIdent && tokenIs(t.src, tokens[1], "[") && tokenIs(t.src, tokens[len(tokens)-1], "]") {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			info := t.typeInfo(object.typeID)
			if info.kind == cTypeArray || info.kind == cTypePointer {
				return info.base
			}
		}
	}
	if tokenKind(tokens[0]) == tokenIdent {
		if len(tokens) > 1 && tokenIs(t.src, tokens[1], "(") {
			if function, ok := t.lookupFunction(tokenText(t.src, tokens[0])); ok {
				return t.functions[function].resultType
			}
			if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
				info := t.typeInfo(object.typeID)
				if info.kind == cTypePointer {
					info = t.typeInfo(info.base)
				}
				if info.kind == cTypeFunction {
					return info.base
				}
			}
		}
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			return object.typeID
		}
		if _, ok := t.lookupValue(tokenText(t.src, tokens[0])); ok {
			return cTypeInt32ID
		}
		if function, ok := t.lookupFunction(tokenText(t.src, tokens[0])); ok {
			return t.pointerType(t.functions[function].typeID)
		}
	}
	if tokenKind(tokens[0]) == tokenChar {
		return cTypeInt32ID
	}
	if tokenKind(tokens[0]) == tokenString {
		return t.pointerType(cTypeInt8ID)
	}
	if tokenKind(tokens[0]) == tokenNumber {
		text := tokenText(t.src, tokens[0])
		floating := false
		float32 := false
		hexadecimal := len(text) > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')
		for i := 0; i < len(text); i++ {
			if text[i] == '.' || hexadecimal && (text[i] == 'p' || text[i] == 'P') || !hexadecimal && (text[i] == 'e' || text[i] == 'E') {
				floating = true
			}
		}
		if floating && len(text) > 0 && (text[len(text)-1] == 'f' || text[len(text)-1] == 'F') {
			float32 = true
		}
		if floating {
			if float32 {
				return cTypeFloat32ID
			}
			return cTypeFloat64ID
		}
		unsigned, long := false, false
		for i := len(text) - 1; i >= 0; i-- {
			if text[i] == 'u' || text[i] == 'U' {
				unsigned = true
			} else if text[i] == 'l' || text[i] == 'L' {
				long = true
			} else {
				break
			}
		}
		if long {
			if unsigned {
				return cTypeUint64ID
			}
			return cTypeInt64ID
		}
		if unsigned {
			return cTypeUint32ID
		}
		if value, ok := integerTokenValue(t.src, tokens[:1]); ok && value > 0x7fffffff {
			if len(text) > 1 && text[0] == '0' && value <= 0xffffffff {
				return cTypeUint32ID
			}
			return cTypeInt64ID
		}
	}
	return cTypeInt32ID
}

func (t *translator) statementExpressionType(tokens []token) (int, bool) {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		return cTypeVoidID, false
	}
	start, lastStart, lastEnd := 1, -1, -1
	paren, bracket, brace := 0, 0, 0
	for i := 1; i < len(tokens)-1; i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		case tokenIs(t.src, tokens[i], "{"):
			brace++
		case tokenIs(t.src, tokens[i], "}"):
			brace--
		case paren == 0 && bracket == 0 && brace == 0 && tokenIs(t.src, tokens[i], ";"):
			if i > start {
				lastStart, lastEnd = start, i
			}
			start = i + 1
		}
	}
	if lastStart < 0 {
		return cTypeVoidID, false
	}
	if !t.checkOnly {
		return t.expressionType(tokens[lastStart:lastEnd]), true
	}
	outerTokens, outerPos := t.tokens, t.pos
	outerOut, outerStaticOut := t.out, t.staticOut
	objectMark := len(t.objects)
	scope := t.beginScope()
	t.tokens, t.pos, t.out = tokens, 1, nil
	for t.ok && t.pos < lastStart {
		t.statement()
	}
	typeID := cTypeVoidID
	if t.ok && t.pos == lastStart {
		typeID = t.expressionType(tokens[lastStart:lastEnd])
	}
	t.tokens, t.pos, t.out, t.staticOut = outerTokens, outerPos, outerOut, outerStaticOut
	t.objects = t.objects[:objectMark]
	t.endScope(scope)
	return typeID, t.ok && typeID != cTypeVoidID
}

func (t *translator) genericExpressionType(tokens []token) (int, bool) {
	if len(tokens) < 4 || !tokenIs(t.src, tokens[0], "_Generic") || !tokenIs(t.src, tokens[1], "(") {
		return cTypeVoidID, false
	}
	end := matchingToken(t.src, tokens, 1, "(", ")")
	if end != len(tokens)-1 {
		return cTypeVoidID, false
	}
	items := splitTopLevel(t.src, tokens[2:end], ",")
	if len(items) < 2 {
		return cTypeVoidID, false
	}
	controlType := t.expressionType(items[0])
	defaultType, haveDefault := cTypeVoidID, false
	for i := 1; i < len(items); i++ {
		colon := topLevelToken(t.src, items[i], ":")
		if colon <= 0 || colon+1 >= len(items[i]) {
			return cTypeVoidID, false
		}
		resultType := t.expressionType(items[i][colon+1:])
		if colon == 1 && tokenIs(t.src, items[i][0], "default") {
			defaultType, haveDefault = resultType, true
			continue
		}
		associationType, ok := t.typeFromTokens(items[i][:colon])
		if !ok {
			return cTypeVoidID, false
		}
		if t.compatibleParameterType(controlType, associationType) {
			return resultType, true
		}
	}
	return defaultType, haveDefault
}

func topLevelToken(src []byte, tokens []token, text string) int {
	paren, bracket, brace := 0, 0, 0
	for i := 0; i < len(tokens); i++ {
		if paren == 0 && bracket == 0 && brace == 0 && tokenIs(src, tokens[i], text) {
			return i
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
	return -1
}

func (t *translator) subscriptExpressionType(tokens []token) (int, bool) {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[len(tokens)-1], "]") {
		return cTypeVoidID, false
	}
	paren, bracket := 0, 0
	open := -1
	for i := 0; i < len(tokens)-1; i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			if paren == 0 && bracket == 0 {
				open = i
			}
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		}
	}
	if open <= 0 || matchingToken(t.src, tokens, open, "[", "]") != len(tokens)-1 {
		return cTypeVoidID, false
	}
	base := t.typeInfo(t.expressionType(tokens[:open]))
	if base.kind != cTypeArray && base.kind != cTypePointer {
		return cTypeVoidID, false
	}
	return base.base, true
}

func (t *translator) memberExpressionType(tokens []token) (int, bool) {
	paren, bracket := 0, 0
	member := -1
	for i := 0; i+1 < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		case paren == 0 && bracket == 0 && (tokenIs(t.src, tokens[i], ".") || tokenIs(t.src, tokens[i], "->")):
			member = i
		}
	}
	if member <= 0 || member+1 >= len(tokens) || tokenKind(tokens[member+1]) != tokenIdent || member+2 != len(tokens) {
		return cTypeVoidID, false
	}
	base := t.expressionType(tokens[:member])
	if tokenIs(t.src, tokens[member], "->") {
		pointer := t.typeInfo(base)
		if pointer.kind != cTypePointer {
			return cTypeVoidID, false
		}
		base = pointer.base
	}
	field, ok := t.lookupField(base, tokenText(t.src, tokens[member+1]))
	if !ok {
		return cTypeVoidID, false
	}
	return field.typeID, true
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
	t.out = append(t.out, '{')
	written := 0
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) == 0 {
			continue
		}
		valueType := cTypeVoidID
		if written > 0 {
			t.out = append(t.out, ',')
		}
		if info.kind == cTypeArray {
			if written >= info.count {
				t.fail(TranslateErrUnsupported)
				return
			}
			valueType = info.base
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
		t.emitInitializerValue(valueType, item)
		written++
	}
	t.out = append(t.out, '}')
}

func (t *translator) emitInitializerValue(typeID int, value []token) {
	if t.checkOnly {
		t.checkInitializerValue(typeID, value)
		return
	}
	info := t.typeInfo(typeID)
	if info.kind == cTypeArray && t.emitStringArrayInitializer(typeID, value) {
		return
	}
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
	t.convertedExpression(typeID, value)
}

func (t *translator) checkInitializerValue(typeID int, value []token) {
	if len(value) == 0 {
		t.fail(TranslateErrDeclaration)
		return
	}
	if !tokenIs(t.src, value[0], "{") {
		t.checkExpression(value)
		return
	}
	if len(value) < 2 || !tokenIs(t.src, value[len(value)-1], "}") {
		t.fail(TranslateErrDeclaration)
		return
	}
	info := t.typeInfo(typeID)
	items := value[1 : len(value)-1]
	if info.kind != cTypeArray && info.kind != cTypeStruct && info.kind != cTypeUnion {
		seen := false
		for start := 0; ; {
			item, nextStart, done := nextTopLevelItem(t.src, items, start)
			if len(item) != 0 && (seen || tokenIs(t.src, item[0], ".") || tokenIs(t.src, item[0], "[")) {
				t.fail(TranslateErrDeclaration)
				return
			}
			if len(item) != 0 {
				seen = true
				t.checkInitializerValue(typeID, item)
			}
			if done {
				break
			}
			start = nextStart
		}
		return
	}
	next := 0
	for start := 0; t.ok; {
		item, nextStart, done := nextTopLevelItem(t.src, items, start)
		if len(item) == 0 {
			if done {
				break
			}
			start = nextStart
			continue
		}
		mark := t.beginCheckScratch()
		next = t.checkAggregateInitializerItem(typeID, info, item, next)
		t.endCheckScratch(mark)
		if done {
			break
		}
		start = nextStart
	}
}

func (t *translator) checkAggregateInitializerItem(typeID int, info cTypeInfo, item []token, next int) int {
	targetType, at := typeID, 0
	designated := tokenIs(t.src, item[0], ".") || tokenIs(t.src, item[0], "[")
	if designated {
		for at < len(item) && (tokenIs(t.src, item[at], ".") || tokenIs(t.src, item[at], "[")) {
			current := t.typeInfo(targetType)
			if tokenIs(t.src, item[at], ".") {
				if at+1 >= len(item) || tokenKind(item[at+1]) != tokenIdent || current.kind != cTypeStruct && current.kind != cTypeUnion {
					t.fail(TranslateErrDeclaration)
					return next
				}
				field, ok := t.lookupField(targetType, tokenText(t.src, item[at+1]))
				if !ok {
					t.fail(TranslateErrDeclaration)
					return next
				}
				targetType = field.typeID
				at += 2
				continue
			}
			close := matchingToken(t.src, item, at, "[", "]")
			if close <= at+1 || current.kind != cTypeArray {
				t.fail(TranslateErrDeclaration)
				return next
			}
			designator := item[at+1 : close]
			rangeAt := topLevelToken(t.src, designator, "...")
			firstTokens := designator
			if rangeAt >= 0 {
				firstTokens = designator[:rangeAt]
			}
			first, ok := t.constantExpression(firstTokens)
			last := first
			if rangeAt >= 0 {
				last, ok = t.constantExpression(designator[rangeAt+1:])
			}
			if !ok || first < 0 || last < first || current.count > 0 && last >= current.count {
				t.fail(TranslateErrDeclaration)
				return next
			}
			targetType = current.base
			if at == 0 {
				next = last + 1
			}
			at = close + 1
		}
		if at >= len(item) || !tokenIs(t.src, item[at], "=") {
			t.fail(TranslateErrDeclaration)
			return next
		}
		at++
	} else if info.kind == cTypeArray {
		if info.count > 0 && next >= info.count {
			t.fail(TranslateErrDeclaration)
			return next
		}
		targetType = info.base
		next++
	} else {
		for next < info.fieldCount && t.fields[info.fieldStart+next].synthetic {
			next++
		}
		if next >= info.fieldCount || info.kind == cTypeUnion && next > 0 {
			t.fail(TranslateErrDeclaration)
			return next
		}
		targetType = t.fields[info.fieldStart+next].typeID
		next++
	}
	t.checkInitializerValue(targetType, item[at:])
	return next
}

func nextTopLevelItem(src []byte, tokens []token, start int) ([]token, int, bool) {
	paren, bracket, brace := 0, 0, 0
	for i := start; i < len(tokens); i++ {
		if paren == 0 && bracket == 0 && brace == 0 && tokenIs(src, tokens[i], ",") {
			return tokens[start:i], i + 1, false
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
	return tokens[start:], len(tokens), true
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
	t.appendText("func ")
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
			t.appendText("uint8")
		} else {
			t.emitType(paths[i].typeID)
		}
	}
	t.appendText(") ")
	t.emitType(typeID)
	t.appendText("{var p ")
	t.emitType(typeID)
	t.out = append(t.out, ';')
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		last := steps[path.start+path.count-1]
		if last.field >= 0 && t.fields[last.field].bitWidth > 0 {
			t.emitInitializerPath(steps[path.start:path.start+path.count-1], typeID)
			t.appendText(".__c_set_")
			t.out = append(t.out, t.fields[last.field].name...)
			t.appendText("(v")
			t.appendDecimal(i)
			t.appendText(");")
		} else {
			t.emitInitializerPath(steps[path.start:path.start+path.count], typeID)
			t.appendText("=v")
			t.appendDecimal(i)
			t.out = append(t.out, ';')
		}
	}
	t.appendText("return p}\n")
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
		if info := t.typeInfo(typeID); info.kind == cTypeUnion || info.indirect {
			receiver := t.out
			t.out = append([]byte("(*"), receiver...)
			t.appendText(".__c_ptr_")
			t.out = append(t.out, field.name...)
			t.appendText("())")
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
	t.appendText("func ")
	t.out = append(t.out, name...)
	t.appendText("(v ")
	if t.typeInfo(field.typeID).kind == cTypeBool && field.bitWidth > 0 {
		t.appendText("uint8")
	} else {
		t.emitType(field.typeID)
	}
	t.appendText(") ")
	t.emitType(typeID)
	t.appendText("{var p ")
	t.emitType(typeID)
	t.out = append(t.out, ';')
	if field.bitWidth > 0 {
		t.appendText("p.__c_set_")
		t.out = append(t.out, field.name...)
		t.appendText("(v);")
	} else {
		t.appendText("(*p.__c_ptr_")
		t.out = append(t.out, field.name...)
		t.appendText("())=v;")
	}
	t.appendText("return p}\n")
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
		t.appendText(".__c_set_")
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
	t.appendText(".__c_set_")
	t.out = append(t.out, access.field.name...)
	t.out = append(t.out, '(')
	t.out = append(t.out, access.expr...)
	if tokenIs(t.src, op, "++") {
		t.appendText("+1)")
	} else {
		t.appendText("-1)")
	}
}

func isAssignmentToken(src []byte, tok token) bool {
	return cTokenSetContains("=,+=,-=,*=,/=,%=,<<=,>>=,&=,^=,|=", tokenText(src, tok))
}

func (t *translator) unaryArithmetic(tok token) bool {
	text := tokenText(t.src, tok)
	return len(text) == 1 && (text[0] == '+' || text[0] == '-' || text[0] == '~')
}

func (t *translator) sizeOperator(tok token) bool {
	return tokenIs(t.src, tok, "sizeof") || t.isAlignof(tok)
}

func (t *translator) isAlignof(tok token) bool {
	return cTokenSetContains("_Alignof,__alignof,__alignof__", tokenText(t.src, tok))
}

func (t *translator) unaryOperator(tok token) bool {
	text := tokenText(t.src, tok)
	if len(text) == 1 {
		return text[0] == '+' || text[0] == '-' || text[0] == '*' || text[0] == '&' || text[0] == '!' || text[0] == '~'
	}
	return t.sizeOperator(tok)
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
		if aggregate.kind == cTypeUnion || aggregate.indirect {
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
	if len(tokens) == 0 {
		return cTypeVoidID, false
	}
	outerTokens, outerPos := t.tokens, t.pos
	outerOK, outerErr, outerErrorAt := t.ok, t.err, t.errorAt
	outerSpeculative := t.speculativeType
	t.tokens, t.pos, t.speculativeType = tokens, 0, true
	typeID, storage, ok := t.parseType()
	if ok && storage == storageNone && t.pos < len(t.tokens) {
		name, ops, _, valid := t.parseDeclaratorOps(true)
		if valid && len(tokenText(t.src, name)) == 0 {
			typeID, valid = t.applyDeclaratorOps(typeID, ops)
		}
		ok = valid
	}
	ok = ok && storage == storageNone && t.pos == len(t.tokens)
	t.tokens, t.pos, t.speculativeType = outerTokens, outerPos, outerSpeculative
	if !ok {
		t.ok, t.err, t.errorAt = outerOK, outerErr, outerErrorAt
	}
	return typeID, ok
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
	offset := 0
	for pos := 0; pos < len(tokens); {
		info := t.typeInfo(typeID)
		if (info.kind != cTypeStruct && info.kind != cTypeUnion) || tokenKind(tokens[pos]) != tokenIdent {
			return 0, false
		}
		field, found := t.lookupField(typeID, tokenText(t.src, tokens[pos]))
		if !found || field.bitWidth != 0 {
			return 0, false
		}
		offset += field.offset
		typeID = field.typeID
		pos++
		for pos < len(tokens) && tokenIs(t.src, tokens[pos], "[") {
			end := matchingToken(t.src, tokens, pos, "[", "]")
			info = t.typeInfo(typeID)
			if end < 0 || info.kind != cTypeArray {
				return 0, false
			}
			index, ok := t.constantExpression(tokens[pos+1 : end])
			if !ok || index < 0 {
				return 0, false
			}
			offset += index * t.typeSize(info.base)
			typeID = info.base
			pos = end + 1
		}
		if pos == len(tokens) {
			return offset, true
		}
		if !tokenIs(t.src, tokens[pos], ".") {
			return 0, false
		}
		pos++
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

func (t *translator) emitCStringValues(tokens []token) {
	value, ok := t.cStringBytes(tokens)
	if !ok {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.out = append(t.out, '"')
	hex := "0123456789abcdef"
	for i := 0; i < len(value); i++ {
		t.out = append(t.out, '\\', 'x', hex[value[i]>>4], hex[value[i]&15])
	}
	t.out = append(t.out, '"')
}

func (t *translator) cStringBytes(tokens []token) ([]byte, bool) {
	var out []byte
	if len(tokens) == 0 {
		return nil, false
	}
	for i := 0; i < len(tokens); i++ {
		if tokenKind(tokens[i]) != tokenString {
			return nil, false
		}
		text := tokenText(t.src, tokens[i])
		quote := 0
		for quote < len(text) && text[quote] != '"' {
			quote++
		}
		if quote != 0 && !(quote == 2 && text[0] == 'u' && text[1] == '8') {
			return nil, false
		}
		value, ok := decodeCString(text)
		if !ok {
			return nil, false
		}
		out = append(out, value...)
	}
	return append(out, 0), true
}

func (t *translator) emitStringArrayInitializer(typeID int, tokens []token) bool {
	info := t.typeInfo(typeID)
	base := t.typeInfo(info.base)
	value, ok := t.cStringBytes(tokens)
	if !ok || (base.kind != cTypeInt && base.kind != cTypeUint) || base.size != 1 {
		return false
	}
	if len(value)-1 > info.count {
		t.fail(TranslateErrDeclaration)
		return true
	}
	if len(value) > info.count {
		value = value[:info.count]
	}
	t.emitType(typeID)
	t.out = append(t.out, '{')
	for i := 0; i < len(value); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		if base.kind == cTypeInt && value[i] >= 128 {
			t.appendDecimalSigned(int(value[i]) - 256)
		} else {
			t.appendDecimal(int(value[i]))
		}
	}
	t.out = append(t.out, '}')
	return true
}

func decodeCString(text []byte) ([]byte, bool) {
	return decodeCLiteral(text, '"')
}

func decodeCCharacter(text []byte) (int, bool) {
	value, ok := decodeCLiteral(text, '\'')
	if !ok || len(value) != 1 {
		return 0, false
	}
	return int(value[0]), true
}

func decodeCLiteral(text []byte, quote byte) ([]byte, bool) {
	start := 0
	for start < len(text) && text[start] != quote {
		start++
	}
	if start >= len(text) || len(text) < start+2 || text[len(text)-1] != quote {
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
		case 'e':
			out = append(out, 27)
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
			if ch == 'x' {
				value := 0
				digits := 0
				for i+1 < len(text)-1 {
					digit := hexDigit(text[i+1])
					if digit < 0 {
						break
					}
					i++
					value = value*16 + digit
					digits++
				}
				if digits == 0 {
					return nil, false
				}
				out = append(out, byte(value))
				continue
			}
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

func hexDigit(ch byte) int {
	if ch >= '0' && ch <= '9' {
		return int(ch - '0')
	}
	if ch >= 'a' && ch <= 'f' {
		return int(ch-'a') + 10
	}
	if ch >= 'A' && ch <= 'F' {
		return int(ch-'A') + 10
	}
	return -1
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
	// Ordinary identifiers in the innermost scope shadow typedef names (and
	// provisional opaque header types) when deciding whether a statement starts
	// with a declaration.
	if _, known := t.lookupObject(name); known {
		return false
	}
	if _, known := t.lookupValue(name); known {
		return false
	}
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
	return cTokenSetContains("auto,register,static,extern,typedef,const,__const,__const__,volatile,__volatile,__volatile__,restrict,__restrict,__restrict__,inline,__inline,__inline__,void,char,short,int,long,float,double,signed,unsigned,_Bool,_Atomic,_Thread_local,struct,union,enum,__extension__,__signed__,__int128,__auto_type,typeof,__typeof,__typeof__", tokenText(src, tok))
}

func cTokenSetContains(set string, text []byte) bool {
	for start := 0; start < len(set); {
		end := start
		for end < len(set) && set[end] != ',' {
			end++
		}
		if end-start == len(text) {
			equal := true
			for i := 0; i < len(text); i++ {
				if set[start+i] != text[i] {
					equal = false
					break
				}
			}
			if equal {
				return true
			}
		}
		start = end + 1
	}
	return false
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
