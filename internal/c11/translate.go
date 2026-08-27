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
	objectTypeID    int
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
	inline            bool
	gnuInline         bool
	diagnosticError   bool
	section           string
	alias             string
	visibility        string
	weak              bool
	callingConvention int
	asmRegister       string
	cleanup           string
	used              bool
}

type cDeferredFunction struct {
	name    string
	source  []byte
	emitted bool
}

type cAsmOperand struct {
	name       string
	constraint string
	expression []token
}

type cAsm struct {
	template []byte
	outputs  []cAsmOperand
	inputs   []cAsmOperand
	clobbers []string
	labels   []string
	volatile bool
	gotoAsm  bool
}

type parameter struct {
	name              token
	typeID            int
	transparentTypeID int
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

type generatedIdentifierReference struct {
	name  string
	next  int
	found bool
}

type generatedIdentifierReferences struct {
	buckets []int
	names   []generatedIdentifierReference
}

type translator struct {
	src                     []byte
	tokens                  []token
	pos                     int
	packageName             string
	isolateGoBuiltins       bool
	goExports               []GoExport
	out                     []byte
	object                  bool
	checkOnly               bool
	assemblyOutput          bool
	assemblyOut             []byte
	speculativeType         bool
	ok                      bool
	err                     int
	errorAt                 int
	types                   []cTypeInfo
	fields                  []cField
	opaqueTypes             []cTypeName
	names                   []cNamespaceName
	objects                 []cObjectName
	externals               []cObjectName
	tentatives              []cObjectName
	definitions             []cObjectName
	threadNames             []cObjectName
	functions               []cFunctionName
	deferredFunctions       []cDeferredFunction
	deferredVariables       []cDeferredFunction
	deferredStaticOut       []byte
	capturingDeferred       bool
	diagnosticErrorNames    []string
	translationUnitDefs     []string
	variadicCalls           []cVariadicCall
	functionParams          []int
	ordinaryNames           []int
	tagNames                []int
	pointerTypes            []int
	qualifiedTypes          []int
	pointerChanges          []cTypeCacheChange
	qualifiedChanges        []cTypeCacheChange
	functionSections        bool
	dataSections            bool
	unsignedChar            bool
	kernelCodeModel         bool
	pruneUnusedStatics      bool
	baseAttributes          cAttributes
	typeSerial              int
	dataModel               int
	pointerSize             int
	longSize                int
	wcharSize               int
	packageHeader           int
	usesUnsafe              bool
	c11DirectiveEmitted     bool
	staticOut               []byte
	resultType              int
	helperSection           string
	boolHelper              bool
	kernelLinkAddressHelper bool
	asmMemoryHelper         bool
	asmCCHelper             bool
	asmChecksumHelper       bool
	asmChecksumAddHelper    bool
	asmChecksumAdd3Helper   bool
	asmChecksumAdd64        int
	asmIPChecksumHelper     bool
	asmStackHelper          bool
	asmInstructionHelper    bool
	asmControlHelper        int
	asmXControl             int
	asmDebugHelper          int
	asmFPUHelper            bool
	asmFPUStateHelpers      []string
	asmFPUPrepare           bool
	asmFPUWait              bool
	asmFlagsHelper          int
	asmCPUIDHelper          bool
	asmMSRHelper            bool
	asmCounterOps           int
	asmPauseHelper          bool
	asmDelayLoop            bool
	asmMonitorWait          int
	asmCPUNODEHelper        bool
	asmFarCallHelper        bool
	asmFarJumpHelper        bool
	asmInterruptFlags       int
	asmDescriptorOps        int
	asmIOHelpers            int
	asmStringIOHelpers      int
	asmSyscallHelper        bool
	asmInterruptStackHelper bool
	asmBitSetHelper         bool
	asmByteOps              int
	asmBitOps64             int
	asmBitScan              int
	asmPopulationCount      int
	asmByteSwap             int
	asmMultiplyDivide       bool
	asmMultiply32           bool
	asmMultiplyShift        bool
	asmCompareExchange128   bool
	asmAtomicArithmetic     int
	asmAtomicExchange       int
	asmCompareExchange      int
	asmNonTemporalStore     int
	asmVerifySegment        bool
	asmInvalidateHelper     bool
	asmInvalidatePage       bool
	asmInt3Selftest         bool
	asmBreakpointHelper     bool
	asmRandomOps            int
	asmFDIVBug              bool
	asmHaltOps              int
	asmCacheFlush           bool
	asmPrefetch             bool
	asmFenceOps             int
	asmDirectOps            int
	asmLoadGS               bool
	uint128MultiplyShift    bool
	asmDivideHelper         bool
	asmBoundsMaskHelper     bool
	asmCopyHelper           bool
	asmUserCopyHelper       bool
	asmUserClearHelper      bool
	asmUserMemoryHelpers    []string
	asmExceptionLoad        bool
	asmGetUserOps           int
	asmPutUserOps           int
	asmSegmentHelpers       int
	asmSafeSegmentHelpers   int
	asmAccessRightsHelper   bool
	asmStartupSegments      bool
	asmBaseRegisterOps      int
	asmSegmentMemory        int
	asmSegmentCompare       int
	vaListHelper            bool
	vaCopyHelper            bool
	stringPointerHelper     bool
	wideStringHelpers       int
	stringLengthHelper      bool
	variadicFunction        bool
	overflowHelpers         int
	builtinBitHelpers       int
	builtinAddressHelpers   int
	builtinIsDigitHelper    bool
	postfixDerefTypes       []int
	postfixCastedDerefKeys  []string
	postfixScalarTypes      []int
	prefixScalarTypes       []int
	postfixAssignTypes      []int
	prefixDerefAssignTypes  []int
	assignmentValueTypes    []int
	pointerAssignValueKey   []int
	compoundAssignKeys      []int
	pointerStepTypes        []int
	pointerMutationTypes    []int
	pointerDiffTypes        []int
	pointerIndexTypes       []int
	arrayIndexTypes         []int
	discardTypes            []int
	asmClampTypes           []int
	asmRuntimePointers      []string
	asmIRQStackHelpers      []string
	msABICallTypes          []int
	nestedAccessorKeys      []string
	localLabels             []cLocalLabel
	hoistedGotos            []string
	tlsRuntime              bool
	localStart              int
	initializing            int
	captureExclusions       []int
	nameStart               int
}

type checkScratchMark struct {
	mark                                   int
	out, staticOut                         []byte
	typesLen, typesCap                     int
	fieldsLen, fieldsCap                   int
	opaqueTypesLen, opaqueTypesCap         int
	namesLen, namesCap                     int
	objectsLen, objectsCap                 int
	externalsLen, externalsCap             int
	tentativesLen, tentativesCap           int
	definitionsLen, definitionsCap         int
	threadNamesLen, threadNamesCap         int
	functionsLen, functionsCap             int
	variadicCallsLen, variadicCallsCap     int
	functionParamsLen, functionParamsCap   int
	pointerTypesCap, qualifiedTypesCap     int
	pointerChangesCap, qualifiedChangesCap int
}

type cLocalLabel struct {
	name   string
	goName string
}

// Translate lowers one preprocessed C translation unit into the shared Go
// frontend syntax. It deliberately performs no backend-specific lowering.
func Translate(packageName string, src []byte) Result {
	return translate(packageName, src, nil, false, DataModelLP64, false)
}

// TranslateForDataModel lowers a freestanding C translation unit using the
// scalar widths of the selected executable target.
func TranslateForDataModel(packageName string, src []byte, dataModel int) Result {
	return translate(packageName, src, nil, false, dataModel, false)
}

// TranslateWithConfig lowers an executable C translation unit with the
// target's command-line scalar and optimization policy.
func TranslateWithConfig(packageName string, src []byte, config ObjectConfig) Result {
	return translateObjectConfig(packageName, src, nil, false, config, false)
}

// TranslateWithPreludeConfig lowers an executable C translation unit after
// parsing declarations supplied by a synthetic header. Prelude diagnostics do
// not use source offsets from src.
func TranslateWithPreludeConfig(packageName string, src []byte, prelude []byte, config ObjectConfig) Result {
	return translateObjectConfig(packageName, src, prelude, false, config, false)
}

// InspectDeclarationsWithConfig parses a cgo preamble. Source contains one
// declared function name per line so the loader can retain the explicit set.
func InspectDeclarationsWithConfig(src []byte, config ObjectConfig) Result {
	return translateObjectConfig("", src, nil, false, config, true)
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
	DataModel          int
	FunctionSections   bool
	DataSections       bool
	ShortWChar         bool
	UnsignedChar       bool
	KernelCodeModel    bool
	PruneUnusedStatics bool
	IsolateGoBuiltins  bool
	GoExports          []GoExport
}

// GoExport maps an external C identifier to the Go function named by a
// //export directive in an explicit mixed-language package.
type GoExport struct {
	CName  string
	GoName string
}

func TranslateObjectWithConfig(packageName string, src []byte, prelude []byte, config ObjectConfig) Result {
	return translateObjectConfig(packageName, src, prelude, true, config, false)
}

// TranslateAssemblyMetadata evaluates constant inline-assembly records used
// by build-time layout generators. It deliberately accepts only translation
// units which emit at least one .ascii record beginning with "->"; ordinary
// compiler assembly output remains unsupported rather than being approximated.
func TranslateAssemblyMetadata(src []byte, prelude []byte, config ObjectConfig) Result {
	return translateObjectConfigMode("main", src, prelude, true, config, false, true)
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
	return translateObjectConfigMode(packageName, src, prelude, object, config, checkOnly, false)
}

func translateObjectConfigMode(packageName string, src []byte, prelude []byte, object bool, config ObjectConfig, checkOnly bool, assemblyOutput bool) Result {
	t := translator{
		packageName:        packageName,
		isolateGoBuiltins:  config.IsolateGoBuiltins,
		goExports:          config.GoExports,
		out:                make([]byte, 0, len(src)+len(src)/4+len(prelude)+64),
		object:             object,
		checkOnly:          checkOnly,
		assemblyOutput:     assemblyOutput,
		functionSections:   config.FunctionSections,
		dataSections:       config.DataSections,
		unsignedChar:       config.UnsignedChar,
		kernelCodeModel:    config.KernelCodeModel,
		pruneUnusedStatics: config.PruneUnusedStatics,
		ok:                 true,
		errorAt:            -1,
		localStart:         -1,
		initializing:       -1,
	}
	t.initTypes(config.DataModel)
	t.wcharSize = 4
	if config.ShortWChar {
		t.wcharSize = 2
	}
	t.appendText("package ")
	t.out = append(t.out, packageName...)
	t.out = append(t.out, '\n')
	if object && !checkOnly {
		// C permits the backend to omit checks whose only purpose is to define
		// behavior for operations that C itself leaves undefined. Keep this an
		// explicit unit-level contract rather than inferring C from helper names.
		t.appendText("// renvo:c11\n")
	}
	t.packageHeader = len(t.out)
	var preludeScan scanResult
	if len(prelude) > 0 {
		preludeScan = scan(prelude)
	}
	sourceScan := scan(src)
	if object {
		t.rememberTranslationUnitFunctionDefinitions(prelude, preludeScan.tokens)
		t.rememberTranslationUnitFunctionDefinitions(src, sourceScan.tokens)
	}
	if len(prelude) > 0 && !t.translateScannedSource(prelude, preludeScan) {
		return Result{Ok: false, Error: t.err, ErrorAt: -1}
	}
	if !t.translateScannedSource(src, sourceScan) {
		return Result{Ok: false, Error: t.err, ErrorAt: t.errorAt}
	}
	if checkOnly {
		if packageName == "" {
			t.out = t.out[:0]
			for i := 0; i < len(t.functions); i++ {
				t.out = append(t.out, t.functions[i].name...)
				t.out = append(t.out, '\n')
			}
		}
		return Result{Source: t.out, Ok: true, Error: TranslateOK, ErrorAt: -1}
	}
	if assemblyOutput {
		if len(t.assemblyOut) == 0 {
			return Result{Ok: false, Error: TranslateErrUnsupported, ErrorAt: -1}
		}
		return Result{Source: t.assemblyOut, Ok: true, Error: TranslateOK, ErrorAt: -1}
	}
	t.emitReachableDeferredFunctions()
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
	t.out = compactGeneratedCLineBreaks(t.out)
	return Result{Source: t.out, Ok: true, Error: TranslateOK, ErrorAt: -1}
}

func compactGeneratedCLineBreaks(source []byte) []byte {
	lines := 1
	for i := 0; i < len(source); i++ {
		if source[i] == '\n' {
			lines++
		}
	}
	if lines <= generatedCLineSafeLimit {
		return source
	}
	for i := 1; i < len(source); i++ {
		if source[i] == '\n' && source[i-1] == ';' {
			source[i] = ' '
		}
	}
	return source
}

const generatedCLineSafeLimit = 60000

// rememberTranslationUnitFunctionDefinitions records ordinary function
// definitions before lowering any body. C permits a call through a prototype
// to precede the definition, while object-mode foreign char pointers use a
// deliberately different shared-Go carrier. Knowing which declarations become
// local definitions keeps that ABI choice independent of source order.
func (t *translator) rememberTranslationUnitFunctionDefinitions(src []byte, tokens []token) {
	if len(src) == 0 {
		return
	}
	segmentStart := 0
	braceDepth := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tokenIs(src, tok, "{") {
			if braceDepth == 0 {
				name := -1
				parenDepth := 0
				for j := segmentStart; j+1 < i; j++ {
					switch {
					case tokenIs(src, tokens[j], "("):
						parenDepth++
					case tokenIs(src, tokens[j], ")"):
						parenDepth--
					case parenDepth == 0 && tokenKind(tokens[j]) == tokenIdent &&
						tokenIs(src, tokens[j+1], "(") &&
						!cTokenSetContains("__attribute__,__attribute,__declspec,typeof,__typeof,__typeof__", tokenText(src, tokens[j])):
						name = j
					}
				}
				if name >= 0 {
					t.rememberTranslationUnitFunctionDefinition(string(tokenText(src, tokens[name])))
				}
			}
			braceDepth++
		} else if tokenIs(src, tok, "}") {
			if braceDepth > 0 {
				braceDepth--
			}
			if braceDepth == 0 {
				segmentStart = i + 1
			}
		} else if braceDepth == 0 && tokenIs(src, tok, ";") {
			segmentStart = i + 1
		}
	}
}

func (t *translator) rememberTranslationUnitFunctionDefinition(name string) {
	for i := 0; i < len(t.translationUnitDefs); i++ {
		if t.translationUnitDefs[i] == name {
			return
		}
	}
	t.translationUnitDefs = append(t.translationUnitDefs, arena.PersistString(name))
}

func (t *translator) translationUnitFunctionDefined(name string) bool {
	for i := 0; i < len(t.translationUnitDefs); i++ {
		if t.translationUnitDefs[i] == name {
			return true
		}
	}
	return false
}

func (t *translator) translateSource(src []byte) bool {
	return t.translateScannedSource(src, scan(src))
}

func (t *translator) translateScannedSource(src []byte, scanned scanResult) bool {
	if !scanned.ok {
		t.ok = false
		t.err = TranslateErrScan
		t.errorAt = scanned.errorAt
		return false
	}
	t.src = src
	t.tokens = scanned.tokens
	if cap(t.diagnosticErrorNames)-len(t.diagnosticErrorNames) < len(t.tokens)/256+16 {
		names := make([]string, len(t.diagnosticErrorNames), len(t.diagnosticErrorNames)+len(t.tokens)/256+16)
		copy(names, t.diagnosticErrorNames)
		t.diagnosticErrorNames = names
	}
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
	externals := make([]cObjectName, len(t.externals), len(t.externals)+tokens/256+16)
	copy(externals, t.externals)
	t.externals = externals
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
		externalsLen: len(t.externals), externalsCap: cap(t.externals),
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
		t.fields[i].goName = arena.PersistString(t.fields[i].goName)
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
	for i := mark.externalsLen; i < len(t.externals); i++ {
		t.externals[i].name = arena.PersistString(t.externals[i].name)
		t.externals[i].goName = arena.PersistString(t.externals[i].goName)
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
		mark.objectsCap == cap(t.objects) && mark.externalsCap == cap(t.externals) && mark.tentativesCap == cap(t.tentatives) &&
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
		operation, ok := t.parseAsmClause()
		if !ok || !t.take(";") || !t.checkOnly &&
			!t.emitFileScopeWeakAlias(operation) && !t.emitFileScopeStaticCallTrampKey(operation) && !t.emitFileScopeRelativeData(operation) && !t.emitFileScopeStaticCall(operation) &&
			!t.emitFileScopeInt3Magic(operation) && !t.emitFileScopeNopFunction(operation) {
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
			if !t.rememberFunction(&decl, false) {
				t.fail(TranslateErrDeclaration)
			}
			return // Otherwise a C or Go definition in the package satisfies it.
		}
		if t.take(",") {
			if !t.rememberFunction(&decl, false) {
				t.fail(TranslateErrDeclaration)
				return
			}
			for {
				next, valid := t.parseDeclarator(base, true)
				if !valid || !next.function || !t.rememberFunction(&next, false) {
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
		t.rememberInlineConstantReturn(decl)
		if t.checkOnly {
			t.checkFunction(decl, storage)
			return
		}
		if t.object && t.pruneUnusedStatics && decl.attributes.inline && !decl.attributes.used &&
			(storage == storageStatic || storage == storageExtern && decl.attributes.gnuInline) {
			start := len(t.out)
			oldDeferredStatic, oldCapturing := t.deferredStaticOut, t.capturingDeferred
			t.deferredStaticOut = nil
			t.capturingDeferred = true
			t.emitFunction(decl, storage)
			deferred := append([]byte{}, t.deferredStaticOut...)
			deferred = append(deferred, t.out[start:]...)
			t.out = t.out[:start]
			t.deferredStaticOut = oldDeferredStatic
			t.capturingDeferred = oldCapturing
			t.deferredFunctions = append(t.deferredFunctions, cDeferredFunction{
				name: string(tokenText(t.src, decl.name)), source: arena.PersistBytes(deferred),
			})
		} else {
			t.emitFunction(decl, storage)
		}
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
			if t.object && len(decls[i].initializer) == 0 {
				t.rememberExternalObject(decls[i])
			}
		}
		return
	}
	if storage == storageOther {
		for i := 0; i < len(decls); i++ {
			if decls[i].attributes.asmRegister == "rsp" || decls[i].attributes.asmRegister == "esp" {
				if len(decls[i].initializer) != 0 {
					t.fail(TranslateErrDeclaration)
					return
				}
				if t.checkOnly {
					t.rememberObject(tokenText(t.src, decls[i].name), decls[i].typeID)
				} else {
					t.ensureAsmStackHelper(decls[i].typeID)
					t.rememberAliasedObject(tokenText(t.src, decls[i].name), "renvo_runtime_CReadStackPointer()", decls[i].typeID)
				}
				continue
			}
			if t.checkOnly {
				t.rememberObject(tokenText(t.src, decls[i].name), decls[i].typeID)
				continue
			}
			t.fail(TranslateErrDeclaration)
			return
		}
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

func (t *translator) emitFileScopeWeakAlias(operation cAsm) bool {
	if !t.object || operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 0 {
		return false
	}
	statements, ok := splitAsmStatements(operation.template)
	if !ok {
		return false
	}
	alias, target := "", ""
	for _, statement := range statements {
		line := trimAsmLine(statement)
		if len(line) == 0 {
			continue
		}
		if asmLinePrefix(line, ".weak") {
			candidate := trimAsmLine(line[len(".weak"):])
			if alias != "" || !validAsmMetadataWord(candidate) {
				return false
			}
			alias = string(candidate)
			continue
		}
		if asmLinePrefix(line, ".set") {
			value := trimAsmLine(line[len(".set"):])
			comma := -1
			for i := 0; i < len(value); i++ {
				if value[i] == ',' {
					comma = i
					break
				}
			}
			if comma <= 0 || comma+1 >= len(value) {
				return false
			}
			left := trimAsmLine(value[:comma])
			right := trimAsmLine(value[comma+1:])
			if target != "" || !validAsmMetadataWord(left) || !validAsmMetadataWord(right) || string(left) != alias {
				return false
			}
			target = string(right)
			continue
		}
		return false
	}
	if alias == "" || target == "" {
		return false
	}
	functionType := cTypeVoidID
	for i := 0; i < len(t.functions); i++ {
		if t.functions[i].name == target {
			functionType = t.functions[i].typeID
			break
		}
	}
	if functionType == cTypeVoidID {
		return false
	}
	t.emitObjectMetadata("function-alias", alias, cAttributes{weak: true, alias: target}, storageExtern, nil, functionType, false)
	t.typeSerial++
	t.appendText("var __c_object_alias_")
	t.appendDecimal(t.typeSerial)
	t.appendText(" uint8\n")
	return true
}

func (t *translator) emitFileScopeStaticCallTrampKey(operation cAsm) bool {
	if !t.object || operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 0 {
		return false
	}
	statements, ok := splitAsmStatements(operation.template)
	if !ok {
		return false
	}
	section, closed := "", false
	var targets []string
	for _, statement := range statements {
		line := trimAsmLine(statement)
		if len(line) == 0 {
			continue
		}
		if asmLinePrefix(line, ".pushsection") {
			if section != "" {
				return false
			}
			value := trimAsmLine(line[len(".pushsection"):])
			end := 0
			for end < len(value) && value[end] != ',' && value[end] != ' ' && value[end] != '\t' {
				end++
			}
			if end == 0 {
				return false
			}
			section = string(value[:end])
		} else if asmLinePrefix(line, ".long") {
			expression := trimAsmLine(line[len(".long"):])
			minus := -1
			for i := 0; i < len(expression); i++ {
				if expression[i] == '-' {
					minus = i
					break
				}
			}
			if minus < 0 || string(trimAsmLine(expression[minus+1:])) != "." {
				return false
			}
			target := trimAsmLine(expression[:minus])
			if !validAsmMetadataWord(target) {
				return false
			}
			targets = append(targets, string(target))
		} else if asmLinePrefix(line, ".popsection") {
			closed = true
		} else {
			return false
		}
	}
	if section != ".static_call_tramp_key" || !closed || len(targets) != 2 {
		return false
	}
	for _, target := range targets {
		t.typeSerial++
		t.emitAsmDataSegment("__c_asm_reloc_"+decimalString(t.typeSerial), section, []byte{0, 0, 0, 0}, 4, 2, target)
	}
	return true
}

func (t *translator) emitFileScopeRelativeData(operation cAsm) bool {
	if !t.object || operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 0 {
		return false
	}
	section, symbol, target := "", "", ""
	relocationOffset, relocationSize, relocationKind := -1, 0, 0
	var data []byte
	statements, ok := splitAsmStatements(operation.template)
	if !ok {
		return false
	}
	for _, statement := range statements {
		line := trimAsmLine(statement)
		if len(line) == 0 {
			continue
		}
		if asmLinePrefix(line, ".section") {
			value := trimAsmLine(line[len(".section"):])
			if len(value) == 0 {
				return false
			}
			if value[0] == '"' {
				close := 1
				for close < len(value) && value[close] != '"' {
					close++
				}
				if close >= len(value) {
					return false
				}
				section = string(value[1:close])
			} else {
				end := 0
				for end < len(value) && value[end] != ',' && value[end] != ' ' && value[end] != '\t' {
					end++
				}
				if end == 0 {
					return false
				}
				section = string(value[:end])
			}
		} else if len(line) > 1 && line[len(line)-1] == ':' {
			candidate := trimAsmLine(line[:len(line)-1])
			if validAsmMetadataWord(candidate) && (candidate[0] < '0' || candidate[0] > '9') {
				symbol = string(candidate)
			} else {
				return false
			}
		} else if asmLinePrefix(line, ".short") {
			values, valid := asmIntegerValues(trimAsmLine(line[len(".short"):]), 2)
			if !valid {
				return false
			}
			for _, value := range values {
				data = append(data, byte(value), byte(value>>8))
			}
		} else if asmLinePrefix(line, ".long") {
			if relocationOffset >= 0 {
				return false
			}
			expression := trimAsmLine(line[len(".long"):])
			minus := -1
			for i := 0; i < len(expression); i++ {
				if expression[i] == '-' {
					minus = i
					break
				}
			}
			if minus >= 0 && string(trimAsmLine(expression[minus+1:])) == "." {
				candidate := trimAsmLine(expression[:minus])
				if !validAsmMetadataWord(candidate) {
					return false
				}
				target = string(candidate)
				relocationOffset, relocationSize, relocationKind = len(data), 4, 2
				data = append(data, 0, 0, 0, 0)
			} else {
				values, valid := asmIntegerValues(expression, 4)
				if !valid {
					return false
				}
				for _, value := range values {
					data = append(data, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
				}
			}
		} else if asmLinePrefix(line, ".quad") {
			if relocationOffset >= 0 {
				return false
			}
			candidate := trimAsmLine(line[len(".quad"):])
			if !validAsmMetadataWord(candidate) {
				return false
			}
			target = string(candidate)
			relocationOffset, relocationSize, relocationKind = len(data), 8, 1
			data = append(data, 0, 0, 0, 0, 0, 0, 0, 0)
		} else if asmLinePrefix(line, ".ascii") || asmLinePrefix(line, ".asciz") {
			zeroTerminated := asmLinePrefix(line, ".asciz")
			prefix := ".ascii"
			if zeroTerminated {
				prefix = ".asciz"
			}
			values, count, valid := decodeAsmStrings(trimAsmLine(line[len(prefix):]))
			if !valid || count == 0 {
				return false
			}
			data = append(data, values...)
			if zeroTerminated {
				for i := 0; i < count; i++ {
					data = append(data, 0)
				}
			}
		} else if asmLinePrefix(line, ".balign") {
			alignment, valid := asmDecimal(trimAsmLine(line[len(".balign"):]))
			if !valid || alignment <= 0 {
				return false
			}
			for len(data)%alignment != 0 {
				data = append(data, 0)
			}
		} else if !asmLinePrefix(line, ".previous") {
			return false
		}
	}
	if symbol == "" {
		t.typeSerial++
		symbol = "__c_asm_data_" + decimalString(t.typeSerial)
	}
	if section == "" || target == "" ||
		!validAsmMetadataWord([]byte(section)) || !validAsmMetadataWord([]byte(symbol)) || !validAsmMetadataWord([]byte(target)) {
		return false
	}
	if relocationOffset < 0 || relocationOffset+relocationSize > len(data) {
		return false
	}
	if relocationOffset > 0 {
		t.emitAsmDataSegment(symbol, section, data[:relocationOffset], 1, 0, "-")
	}
	relocationName := symbol
	relocationAlignment := relocationSize
	if relocationOffset > 0 {
		t.typeSerial++
		relocationName = "__c_asm_reloc_" + decimalString(t.typeSerial)
		relocationAlignment = 1
	}
	t.emitAsmDataSegment(relocationName, section, data[relocationOffset:relocationOffset+relocationSize], relocationAlignment, relocationKind, target)
	if relocationOffset+relocationSize < len(data) {
		t.typeSerial++
		t.emitAsmDataSegment("__c_asm_data_"+decimalString(t.typeSerial), section, data[relocationOffset+relocationSize:], 1, 0, "-")
	}
	return true
}

func asmIntegerValues(value []byte, width int) ([]uint64, bool) {
	var result []uint64
	start, depth := 0, 0
	for i := 0; i <= len(value); i++ {
		if i < len(value) {
			if value[i] == '(' {
				depth++
			} else if value[i] == ')' {
				depth--
				if depth < 0 {
					return nil, false
				}
			}
		}
		if i < len(value) && (value[i] != ',' || depth != 0) {
			continue
		}
		item := trimAsmLine(value[start:i])
		parsed, ok := asmInteger(item, width)
		if !ok {
			return nil, false
		}
		result = append(result, parsed)
		start = i + 1
	}
	return result, depth == 0 && len(result) > 0
}

func asmInteger(value []byte, width int) (uint64, bool) {
	for len(value) > 1 && value[0] == '(' && value[len(value)-1] == ')' {
		value = trimAsmLine(value[1 : len(value)-1])
	}
	if len(value) > 1 && value[0] == '~' {
		inner, ok := asmInteger(trimAsmLine(value[1:]), width)
		if !ok {
			return 0, false
		}
		mask := uint64(^uint64(0))
		if width < 8 {
			mask = (uint64(1) << (width * 8)) - 1
		}
		return (^inner) & mask, true
	}
	base, start := uint64(10), 0
	if len(value) > 2 && value[0] == '0' && (value[1] == 'x' || value[1] == 'X') {
		base, start = 16, 2
	}
	if start >= len(value) {
		return 0, false
	}
	result := uint64(0)
	limit := uint64(^uint64(0))
	if width < 8 {
		limit = (uint64(1) << (width * 8)) - 1
	}
	for i := start; i < len(value); i++ {
		digit := uint64(16)
		if value[i] >= '0' && value[i] <= '9' {
			digit = uint64(value[i] - '0')
		} else if value[i] >= 'a' && value[i] <= 'f' {
			digit = uint64(value[i]-'a') + 10
		} else if value[i] >= 'A' && value[i] <= 'F' {
			digit = uint64(value[i]-'A') + 10
		}
		if digit >= base || result > (limit-digit)/base {
			return 0, false
		}
		result = result*base + digit
	}
	return result, true
}

func (t *translator) emitAsmDataSegment(name string, section string, data []byte, alignment int, relocationKind int, target string) {
	t.appendText("// renvo:object variable ")
	t.appendText(name)
	t.appendText(" ")
	t.appendText(section)
	t.out = append(t.out, ' ')
	t.appendDecimal(alignment)
	t.appendText(" 0 0 - ")
	t.appendDecimal(len(data))
	t.out = append(t.out, ' ')
	t.appendDecimal(relocationKind)
	t.out = append(t.out, ' ')
	t.appendText(target)
	t.appendText(" 0\nvar ")
	t.appendText(cGoIdentifier(name))
	t.appendText(" [")
	t.appendDecimal(len(data))
	t.appendText("]uint8=[")
	t.appendDecimal(len(data))
	t.appendText("]uint8{")
	for i := 0; i < len(data); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.appendDecimal(int(data[i]))
	}
	t.appendText("}\n")
}

func (t *translator) emitFileScopeStaticCall(operation cAsm) bool {
	if len(operation.outputs) != 0 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 ||
		len(operation.labels) != 0 || operation.gotoAsm {
		return false
	}
	plainReturn := t.asmTemplateCompactContains(operation.template, "ret;int3;nop;nop;nop")
	directJump := t.asmTemplateCompactContains(operation.template, ".byte0xe9;.long")
	if (!plainReturn && !directJump) || !t.asmTemplateCompactContains(operation.template, ".pushsection.static_call.text") ||
		!t.asmTemplateCompactContains(operation.template, ".globl") ||
		!t.asmTemplateCompactContains(operation.template, ".byte0x0f,0xb9,0xcc") ||
		!t.asmTemplateCompactContains(operation.template, ".popsection") {
		return false
	}
	statements, ok := splitAsmStatements(operation.template)
	if !ok {
		return false
	}
	name, target := "", ""
	for i := 0; i < len(statements); i++ {
		line := trimAsmLine(statements[i])
		if asmLinePrefix(line, ".globl") {
			candidate := trimAsmLine(line[len(".globl"):])
			if !validAsmMetadataWord(candidate) || name != "" {
				return false
			}
			name = string(candidate)
			continue
		}
		if directJump && asmLinePrefix(line, ".long") {
			value := trimAsmLine(line[len(".long"):])
			minus := -1
			for j := 0; j < len(value); j++ {
				if value[j] == '-' {
					minus = j
					break
				}
			}
			if minus > 0 {
				candidate := trimAsmLine(value[:minus])
				if validAsmMetadataWord(candidate) {
					target = string(candidate)
				}
			}
		}
	}
	if name == "" || directJump && target == "" {
		return false
	}
	if _, exists := t.lookupFunction([]byte(name)); !exists {
		return false
	}
	t.typeSerial++
	t.appendText("// renvo:object static-call ")
	t.appendText(name)
	t.appendText(" .static_call.text 4 1 0 - 8 ")
	if directJump {
		t.appendText("2 ")
		t.appendText(target)
	} else {
		t.appendText("0 -")
	}
	t.appendText(" 0\nvar __c_static_call_")
	t.appendDecimal(t.typeSerial)
	t.appendText(" [8]uint8=[8]uint8{")
	if directJump {
		t.appendText("233,0,0,0,0,15,185,204")
	} else {
		t.appendText("195,204,144,144,144,15,185,204")
	}
	t.appendText("}\n")
	return true
}

func (t *translator) emitFileScopeNopFunction(operation cAsm) bool {
	if !t.object || operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 0 ||
		!t.asmTemplateCompactContains(operation.template, "ret") ||
		!t.asmTemplateCompactContains(operation.template, ".balign16,0x90") {
		return false
	}
	entrySection := t.asmTemplateCompactContains(operation.template, ".pushsection.entry.text,\"ax\"")
	plainReturn := t.asmTemplateCompactContains(operation.template, "ret;int3")
	if !entrySection && !plainReturn || entrySection && !t.asmTemplateCompactContains(operation.template, ".popsection") {
		return false
	}
	statements, ok := splitAsmStatements(operation.template)
	if !ok {
		return false
	}
	name := ""
	for i := 0; i < len(statements); i++ {
		line := trimAsmLine(statements[i])
		if asmLinePrefix(line, ".global") {
			candidate := trimAsmLine(line[len(".global"):])
			if !validAsmMetadataWord(candidate) || name != "" {
				return false
			}
			name = string(candidate)
		}
	}
	compactName := cGoIdentifier(name)
	if name == "" || !t.asmTemplateCompactContains(operation.template, name+":") ||
		!t.asmTemplateCompactContains(operation.template, ".type"+name+",@function") ||
		!t.asmTemplateCompactContains(operation.template, ".size"+name+",.-"+name) {
		return false
	}
	functionType := cTypeVoidID
	function := -1
	for i := 0; i < len(t.functions); i++ {
		if t.functions[i].name == name {
			function = i
			break
		}
	}
	if function >= 0 {
		fn := &t.functions[function]
		if fn.resultType != cTypeVoidID || fn.paramCount != 0 || fn.variadic {
			return false
		}
		fn.defined = true
		functionType = fn.typeID
	} else {
		functionType = t.functionType(cTypeVoidID, nil, false)
		t.functions = append(t.functions, cFunctionName{name: name, typeID: functionType, resultType: cTypeVoidID, defined: true})
		if !t.rememberName(name, len(t.functions)-1, cNameFunction) {
			return false
		}
	}
	section := ""
	if entrySection {
		section = ".entry.text"
	}
	t.emitObjectMetadata("function", name, cAttributes{section: section, align: 16}, storageExtern, nil, functionType, false)
	t.appendText("//export ")
	t.appendText(name)
	t.appendText("\nfunc ")
	t.appendText(compactName)
	t.appendText("(){}\n")
	return true
}

func (t *translator) emitFileScopeInt3Magic(operation cAsm) bool {
	const name = "int3_magic"
	if !t.object || operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 0 ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection.init.text,\"ax\",@progbits") ||
		!t.asmTemplateCompactContains(operation.template, ".type"+name+",@function") ||
		!t.asmTemplateCompactContains(operation.template, name+":") ||
		!t.asmTemplateCompactContains(operation.template, "movl$1,(%rdi)") ||
		!t.asmTemplateCompactContains(operation.template, "ret") ||
		!t.asmTemplateCompactContains(operation.template, ".size"+name+",.-"+name) ||
		!t.asmTemplateCompactContains(operation.template, ".popsection") {
		return false
	}
	function := -1
	for i := 0; i < len(t.functions); i++ {
		if t.functions[i].name == name {
			function = i
			break
		}
	}
	if function < 0 {
		return false
	}
	fn := &t.functions[function]
	if fn.defined || fn.resultType != cTypeVoidID || fn.paramCount != 1 {
		return false
	}
	parameterType := t.typeInfo(t.functionParams[fn.paramStart])
	if parameterType.kind != cTypePointer || t.typeSize(parameterType.base) != 4 {
		return false
	}
	fn.defined = true
	t.emitObjectMetadata("function", name, cAttributes{section: ".init.text"}, storageStatic, nil, fn.typeID, false)
	t.appendText("func int3_magic(ptr *uint32){*ptr=1}\n")
	return true
}

func splitAsmStatements(value []byte) ([][]byte, bool) {
	var result [][]byte
	start := 0
	quoted, escaped := false, false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if quoted {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				quoted = false
			}
			continue
		}
		if ch == '"' {
			quoted = true
		} else if ch == ';' || ch == '\n' {
			result = append(result, value[start:i])
			start = i + 1
		}
	}
	if quoted || escaped {
		return nil, false
	}
	result = append(result, value[start:])
	return result, true
}

func decodeAsmStrings(value []byte) ([]byte, int, bool) {
	var result []byte
	count := 0
	for pos := 0; pos < len(value); {
		for pos < len(value) && (value[pos] == ' ' || value[pos] == '\t' || value[pos] == ',') {
			pos++
		}
		if pos == len(value) {
			break
		}
		if value[pos] != '"' {
			return nil, 0, false
		}
		pos++
		for pos < len(value) && value[pos] != '"' {
			ch := value[pos]
			pos++
			if ch == '\\' {
				if pos >= len(value) {
					return nil, 0, false
				}
				ch = value[pos]
				pos++
				if ch >= '0' && ch <= '7' {
					n := int(ch - '0')
					for digits := 1; digits < 3 && pos < len(value) && value[pos] >= '0' && value[pos] <= '7'; digits++ {
						n = n*8 + int(value[pos]-'0')
						pos++
					}
					ch = byte(n)
				} else if ch == 'n' {
					ch = '\n'
				} else if ch == 't' {
					ch = '\t'
				}
			}
			result = append(result, ch)
		}
		if pos >= len(value) || value[pos] != '"' {
			return nil, 0, false
		}
		pos++
		count++
	}
	return result, count, count > 0
}

func asmDecimal(value []byte) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		result = result*10 + int(value[i]-'0')
		if result > 1<<20 {
			return 0, false
		}
	}
	return result, true
}

func trimAsmLine(value []byte) []byte {
	start, end := 0, len(value)
	for start < end && (value[start] == ' ' || value[start] == '\t' || value[start] == '\r') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}

func asmLinePrefix(value []byte, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return len(value) == len(prefix) || value[len(prefix)] == ' ' || value[len(prefix)] == '\t'
}

func validAsmMetadataWord(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch != '_' && ch != '.' && ch != '$' && ch != '-' &&
			(ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
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
	externals := t.externals
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
	t.externals = externals
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
		// Attributes following a referenced aggregate tag belong to the
		// surrounding declaration. In particular, Linux commonly writes a
		// function section attribute between `struct tag` and the return
		// pointer star.
		t.baseAttributes.merge(attributes)
		if typeID, ok := t.lookupTag([]byte(name)); ok {
			return typeID, true
		}
		t.typeSerial++
		goName := prefix + name
		for i := 0; i < len(t.types); i++ {
			if t.types[i].goName == goName {
				goName += "_" + decimalString(t.typeSerial)
				break
			}
		}
		t.types = append(t.types, cTypeInfo{kind: kind, size: 0, align: 1, goName: goName})
		typeID := len(t.types) - 1
		t.appendName(name, typeID, cNameTag)
		return typeID, true
	}
	t.typeSerial++
	goName := prefix + name
	if name == "" {
		goName = prefix + "anon_" + decimalString(t.typeSerial)
	} else {
		currentIncomplete := -1
		if entry := t.lookupNameEntryString(name, true); entry >= t.nameStart {
			if typeID := t.names[entry].index; !t.types[typeID].complete {
				currentIncomplete = typeID
			}
		}
		if currentIncomplete >= 0 {
			goName = t.types[currentIncomplete].goName
		} else {
			for i := 0; i < len(t.types); i++ {
				if t.types[i].goName == goName {
					goName += "_" + decimalString(t.typeSerial)
					break
				}
			}
		}
	}
	t.types = append(t.types, cTypeInfo{kind: kind, align: 1, goName: goName})
	typeID := len(t.types) - 1
	if name != "" {
		prior, current := cTypeVoidID, false
		if entry := t.lookupNameEntryString(name, true); entry >= t.nameStart {
			prior, current = t.names[entry].index, true
		}
		if current && !t.types[prior].complete {
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
			if fieldInfo.indirect {
				indirect = true
			}
			if fieldInfo.kind != cTypeStruct && fieldInfo.kind != cTypeUnion {
				return cTypeVoidID, false
			}
			if fieldInfo.size == 0 {
				// Go gives an empty aggregate field addressable storage. C's GNU
				// empty aggregate extension does not, so every enclosing layout
				// must use explicit C offsets instead of native Go field offsets.
				indirect = true
			}
			alignment := t.typeAlign(fieldBase)
			if alignment > cAggregateCarrierSize(alignment) {
				indirect = true
			}
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
				promoted.promoted = true
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
			if fieldInfo.indirect {
				indirect = true
			}
			if fieldInfo.size == 0 {
				indirect = true
			}
			alignment := t.typeAlign(field.typeID)
			if alignment > cAggregateCarrierSize(alignment) {
				indirect = true
			}
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
	if !union {
		// Renvo's backend aggregate ABI reserves at least one machine word for
		// each ordinary Go struct field. Use native fields only when that layout
		// is also the C layout; otherwise the existing byte carrier and accessors
		// preserve the C offsets exactly. This matters for adjacent narrow fields
		// such as `uint32_t size, count` on a 64-bit target.
		nativeOffset := 0
		for i := 0; i < len(aggregateFields); i++ {
			field := aggregateFields[i]
			if !field.emit {
				continue
			}
			nativeOffset = alignC(nativeOffset, t.pointerSize)
			if field.offset != nativeOffset {
				indirect = true
			}
			fieldSize := t.typeSize(field.typeID)
			if fieldSize < t.pointerSize {
				fieldSize = t.pointerSize
			}
			nativeOffset += alignC(fieldSize, t.pointerSize)
		}
		if size != alignC(nativeOffset, t.pointerSize) {
			indirect = true
		}
	}
	fieldStart := len(t.fields)
	for i := 0; i < len(aggregateFields); i++ {
		if aggregateFields[i].name != "" {
			aggregateFields[i].goName = cGoIdentifier(aggregateFields[i].name)
		}
	}
	t.fields = append(t.fields, aggregateFields...)
	t.types[typeID].size = size
	t.types[typeID].align = maxAlign
	t.types[typeID].complete = true
	t.types[typeID].indirect = indirect
	t.types[typeID].fieldStart = fieldStart
	t.types[typeID].fieldCount = len(t.fields) - fieldStart
	outer := t.out
	t.out = t.staticOut
	t.emitAggregateDeclaration(typeID)
	t.emitAggregateAccessors(typeID)
	t.staticOut = t.out
	t.out = outer
	return typeID, true
}

func (attributes *cAttributes) merge(other cAttributes) {
	if other.align > attributes.align {
		attributes.align = other.align
	}
	attributes.packed = attributes.packed || other.packed
	attributes.transparentUnion = attributes.transparentUnion || other.transparentUnion
	attributes.inline = attributes.inline || other.inline
	attributes.gnuInline = attributes.gnuInline || other.gnuInline
	attributes.diagnosticError = attributes.diagnosticError || other.diagnosticError
	attributes.used = attributes.used || other.used
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
	if other.callingConvention != 0 {
		attributes.callingConvention = other.callingConvention
	}
	if other.asmRegister != "" {
		attributes.asmRegister = other.asmRegister
	}
	if other.cleanup != "" {
		attributes.cleanup = other.cleanup
	}
}

func (attributes cAttributes) requiresObjectMetadata() bool {
	return attributes.section != "" || attributes.alias != "" || attributes.visibility != "" ||
		attributes.weak || attributes.callingConvention != 0
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
	t.objects = append(t.objects, cObjectName{name: string(name), goName: t.cSourceGoIdentifier(string(name)), typeID: typeID, auto: t.localStart >= 0})
	t.rememberName(string(name), len(t.objects)-1, cNameObject)
}

func (t *translator) rememberAliasedObject(name []byte, goName string, typeID int) {
	if len(name) == 0 {
		return
	}
	t.objects = append(t.objects, cObjectName{name: string(name), goName: goName, typeID: typeID})
	t.rememberName(string(name), len(t.objects)-1, cNameObject)
}

func (t *translator) rememberExternalObject(decl declarator) {
	name := string(tokenText(t.src, decl.name))
	for i := 0; i < len(t.externals); i++ {
		if t.externals[i].name != name {
			continue
		}
		if !t.compatibleType(t.externals[i].typeID, decl.typeID) {
			t.fail(TranslateErrDeclaration)
		}
		return
	}
	t.externals = append(t.externals, cObjectName{name: name, goName: t.cSourceGoIdentifier(name), typeID: decl.typeID,
		attributes: decl.attributes, storage: storageExtern})
}

func (t *translator) rememberFunction(decl *declarator, definition bool) bool {
	name := string(tokenText(t.src, decl.name))
	if decl.attributes.diagnosticError && !t.checkOnly {
		t.rememberDiagnosticErrorName(name)
	}
	function := -1
	if entry := t.lookupNameEntryString(name, false); entry >= 0 {
		if t.names[entry].kind == cNameFunction {
			function = t.names[entry].index
		}
	}
	if function < 0 {
		// Block-scope function declarations have external linkage even after
		// their ordinary-identifier scope ends. Preserve that declaration table
		// independently of the current namespace lookup.
		for i := 0; i < len(t.functions); i++ {
			if t.functions[i].name == name {
				function = i
				break
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
			fn.consumesVariadic = t.functionConsumesVariadicArguments()
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
		paramCount: len(decl.params), variadic: decl.variadic, defined: definition,
		consumesVariadic: definition && t.functionConsumesVariadicArguments(), attributes: decl.attributes,
	})
	return t.rememberName(name, len(t.functions)-1, cNameFunction)
}

func (t *translator) functionConsumesVariadicArguments() bool {
	if !t.currentIs("{") {
		return false
	}
	close := matchingToken(t.src, t.tokens, t.pos, "{", "}")
	for i := t.pos + 1; i >= 0 && i < close; i++ {
		if tokenIs(t.src, t.tokens[i], "__builtin_va_start") || tokenIs(t.src, t.tokens[i], "va_start") ||
			tokenIs(t.src, t.tokens[i], "__va_start") {
			return true
		}
	}
	return false
}

func (t *translator) rememberDiagnosticErrorName(name string) {
	for i := 0; i < len(t.diagnosticErrorNames); i++ {
		if t.diagnosticErrorNames[i] == name {
			return
		}
	}
	t.diagnosticErrorNames = append(t.diagnosticErrorNames, arena.PersistString(name))
}

func (t *translator) rememberInlineConstantReturn(decl declarator) {
	if !decl.attributes.inline || !t.currentIs("{") || decl.typeID == cTypeVoidID {
		return
	}
	close := matchingToken(t.src, t.tokens, t.pos, "{", "}")
	if close <= t.pos+3 || !tokenIs(t.src, t.tokens[t.pos+1], "return") ||
		!tokenIs(t.src, t.tokens[close-1], ";") {
		return
	}
	for i := t.pos + 2; i < close-1; i++ {
		if tokenIs(t.src, t.tokens[i], ";") || tokenIs(t.src, t.tokens[i], "{") || tokenIs(t.src, t.tokens[i], "}") {
			return
		}
	}
	info := t.typeInfo(decl.typeID)
	if info.kind != cTypeInt && info.kind != cTypeUint && info.kind != cTypeBool {
		return
	}
	value, ok := t.constantExpression(t.tokens[t.pos+2 : close-1])
	if !ok {
		value, ok = t.inlineRuntimeAnnihilatorCondition(t.tokens[t.pos+2:close-1], true)
	}
	if !ok {
		return
	}
	name := string(tokenText(t.src, decl.name))
	for i := 0; i < len(t.functions); i++ {
		if t.functions[i].name == name {
			t.functions[i].constantReturn = value
			t.functions[i].constantReturnOK = true
			return
		}
	}
}

func (t *translator) inlinePureConstantCondition(tokens []token) (int, bool) {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if value, ok := t.inlineRuntimeAnnihilatorCondition(tokens, true); ok {
		return value, true
	}
	if t.conditionContainsRuntimeObject(tokens) {
		return 0, false
	}
	return t.constantCondition(tokens)
}

// inlineRuntimeAnnihilatorCondition recognizes constant-return inline calls
// and value-independent masks without asking the general C constant evaluator
// to assign a value to a runtime object. Logical folding is limited to cases
// whose short-circuit behavior makes the runtime operand unreachable.
func (t *translator) inlineRuntimeAnnihilatorCondition(tokens []token, logical bool) (int, bool) {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	// The constant parser rejects ordinary object reads, but it must still see
	// the complete expression: __builtin_constant_p and a selected constant
	// conditional arm can make an expression constant even when an unselected
	// fallback names a runtime object.
	if value, ok := t.constantExpression(tokens); ok {
		return value, true
	}
	if value, ok := t.inlineConstantCallValue(tokens); ok {
		return value, true
	}
	if question, colon := t.conditionalOperator(tokens); question >= 0 {
		condition, known := t.inlineRuntimeAnnihilatorCondition(tokens[:question], logical)
		if known {
			selected := tokens[colon+1:]
			if condition != 0 {
				selected = tokens[question+1 : colon]
				if len(selected) == 0 {
					return condition, true
				}
			}
			return t.inlineRuntimeAnnihilatorCondition(selected, logical)
		}
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "!") && t.binaryOperator(tokens) < 0 {
		if value, ok := t.inlineRuntimeAnnihilatorCondition(tokens[1:], logical); ok {
			if value == 0 {
				return 1, true
			}
			return 0, true
		}
	}
	at := t.binaryOperator(tokens)
	if at <= 0 || at+1 >= len(tokens) {
		return 0, false
	}
	left, right := tokens[:at], tokens[at+1:]
	leftValue, leftOK := t.inlineRuntimeAnnihilatorCondition(left, logical)
	rightValue, rightOK := t.inlineRuntimeAnnihilatorCondition(right, logical)
	if logical && tokenIs(t.src, tokens[at], "&&") && leftOK {
		if leftValue == 0 {
			return 0, true
		}
		if rightOK {
			if rightValue != 0 {
				return 1, true
			}
			return 0, true
		}
	}
	if logical && tokenIs(t.src, tokens[at], "||") && leftOK {
		if leftValue != 0 {
			return 1, true
		}
		if rightOK {
			if rightValue != 0 {
				return 1, true
			}
			return 0, true
		}
	}
	if logical && tokenIs(t.src, tokens[at], "&&") && rightOK && rightValue == 0 &&
		t.constantCallArgumentsSideEffectFree([][]token{left}) {
		return 0, true
	}
	if logical && tokenIs(t.src, tokens[at], "||") && rightOK && rightValue != 0 &&
		t.constantCallArgumentsSideEffectFree([][]token{left}) {
		return 1, true
	}
	if tokenIs(t.src, tokens[at], "&") {
		if (rightOK && rightValue == 0 || valueIsConstantZero(t, right)) &&
			t.constantCallArgumentsSideEffectFree([][]token{left}) {
			return 0, true
		}
		if (leftOK && leftValue == 0 || valueIsConstantZero(t, left)) &&
			t.constantCallArgumentsSideEffectFree([][]token{right}) {
			return 0, true
		}
	}
	if leftOK && rightOK {
		value := 0
		switch {
		case tokenIs(t.src, tokens[at], "=="):
			if leftValue == rightValue {
				value = 1
			}
		case tokenIs(t.src, tokens[at], "!="):
			if leftValue != rightValue {
				value = 1
			}
		case tokenIs(t.src, tokens[at], "<"):
			if leftValue < rightValue {
				value = 1
			}
		case tokenIs(t.src, tokens[at], "<="):
			if leftValue <= rightValue {
				value = 1
			}
		case tokenIs(t.src, tokens[at], ">"):
			if leftValue > rightValue {
				value = 1
			}
		case tokenIs(t.src, tokens[at], ">="):
			if leftValue >= rightValue {
				value = 1
			}
		default:
			return 0, false
		}
		return value, true
	}
	return 0, false
}

func valueIsConstantZero(t *translator, tokens []token) bool {
	value, ok := t.constantExpression(tokens)
	return ok && value == 0
}

func (t *translator) diagnosticErrorName(name []byte) bool {
	for i := 0; i < len(t.diagnosticErrorNames); i++ {
		if textEquals(name, t.diagnosticErrorNames[i]) {
			return true
		}
	}
	return false
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
	if a.kind == cTypeFunction && a.msABI != b.msABI {
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
		} else {
			t.tentatives[tentative].attributes.merge(decl.attributes)
		}
		return true
	}
	if tentative >= 0 {
		decl.attributes.merge(t.tentatives[tentative].attributes)
		copy(t.tentatives[tentative:], t.tentatives[tentative+1:])
		t.tentatives = t.tentatives[:len(t.tentatives)-1]
	}
	if t.object && !t.checkOnly {
		decl.objectTypeID = t.objectFlexibleInitializerType(decl.typeID, decl.initializer)
	}
	t.definitions = append(t.definitions, cObjectName{name: name, typeID: decl.typeID})
	if t.object && t.pruneUnusedStatics && decl.storage == storageStatic && !decl.attributes.used {
		start := len(t.out)
		t.emitVariable(decl)
		deferred := append([]byte{}, t.out[start:]...)
		t.out = t.out[:start]
		t.deferredVariables = append(t.deferredVariables, cDeferredFunction{
			name: name, source: arena.PersistBytes(deferred),
		})
	} else {
		t.emitVariable(decl)
	}
	return true
}

func (t *translator) objectFlexibleInitializerType(typeID int, initializer []token) int {
	info := t.typeInfo(typeID)
	if info.kind != cTypeStruct || len(initializer) < 2 || !tokenIs(t.src, initializer[0], "{") ||
		!tokenIs(t.src, initializer[len(initializer)-1], "}") {
		return 0
	}
	flexibleIndex := -1
	for i := info.fieldCount - 1; i >= 0; i-- {
		field := t.fields[info.fieldStart+i]
		if field.synthetic || field.promoted {
			continue
		}
		fieldInfo := t.typeInfo(field.typeID)
		if !field.emit || fieldInfo.kind != cTypeArray || fieldInfo.count != 0 {
			return 0
		}
		flexibleIndex = info.fieldStart + i
		break
	}
	if flexibleIndex < 0 {
		return 0
	}
	flexible := t.fields[flexibleIndex]
	count := 0
	items := splitTopLevel(t.src, initializer[1:len(initializer)-1], ",")
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) < 4 || !tokenIs(t.src, item[0], ".") || tokenKind(item[1]) != tokenIdent ||
			!textEquals(tokenText(t.src, item[1]), flexible.name) {
			continue
		}
		at := 2
		if !tokenIs(t.src, item[at], "=") {
			return 0
		}
		var ok bool
		count, ok = t.initializerArrayCount(item[at+1:])
		if !ok || count == 0 {
			return 0
		}
		break
	}
	if count == 0 {
		return 0
	}
	flexibleInfo := t.typeInfo(flexible.typeID)
	expandedArray := t.arrayType(flexibleInfo.base, count)
	fieldStart := len(t.fields)
	for i := 0; i < info.fieldCount; i++ {
		field := t.fields[info.fieldStart+i]
		if info.fieldStart+i == flexibleIndex {
			field.typeID = expandedArray
		}
		t.fields = append(t.fields, field)
	}
	t.typeSerial++
	expanded := info
	expanded.goName = "__c_struct_flexible_" + decimalString(t.typeSerial)
	expanded.fieldStart = fieldStart
	expanded.indirect = true
	expanded.size = alignC(flexible.offset+t.typeSize(expandedArray), info.align)
	if expanded.size < info.size {
		expanded.size = info.size
	}
	t.types = append(t.types, expanded)
	expandedType := len(t.types) - 1
	outer := t.out
	t.out = t.staticOut
	t.emitAggregateDeclaration(expandedType)
	t.emitAggregateAccessors(expandedType)
	t.staticOut = t.out
	t.out = outer
	return expandedType
}

func (t *translator) initializerArrayCount(initializer []token) (int, bool) {
	if len(initializer) < 2 || !tokenIs(t.src, initializer[0], "{") || !tokenIs(t.src, initializer[len(initializer)-1], "}") {
		return 0, false
	}
	items := splitTopLevel(t.src, initializer[1:len(initializer)-1], ",")
	next, count := 0, 0
	for i := 0; i < len(items); i++ {
		if len(items[i]) == 0 {
			continue
		}
		if tokenIs(t.src, items[i][0], "[") {
			close := matchingToken(t.src, items[i], 0, "[", "]")
			if close <= 1 {
				return 0, false
			}
			rangeAt := topLevelToken(t.src, items[i][1:close], "...")
			indexTokens := items[i][1:close]
			if rangeAt >= 0 {
				indexTokens = items[i][1 : 1+rangeAt]
			}
			index, ok := t.constantExpression(indexTokens)
			if !ok || index < 0 {
				return 0, false
			}
			if rangeAt >= 0 {
				last, valid := t.constantExpression(items[i][2+rangeAt : close])
				if !valid || last < index {
					return 0, false
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
	return count, true
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
	count, ok := t.initializerArrayCount(decl.initializer)
	if !ok {
		return false
	}
	decl.typeID = t.arrayType(info.base, count)
	decl.incompleteArray = false
	return true
}

func (t *translator) emitReachableDeferredFunctions() {
	references := newGeneratedIdentifierReferences(len(t.deferredFunctions) + len(t.deferredVariables))
	for i := 0; i < len(t.deferredFunctions); i++ {
		references.add(t.deferredFunctions[i].name)
	}
	for i := 0; i < len(t.deferredVariables); i++ {
		references.add(t.deferredVariables[i].name)
	}
	references.scan(t.out)
	references.scan(t.staticOut)
	for i := 0; i < len(t.variadicCalls); i++ {
		function := t.variadicCalls[i].function
		if function >= 0 && function < len(t.functions) {
			references.mark(t.functions[function].name)
		}
	}
	for {
		progress := false
		for i := 0; i < len(t.deferredFunctions); i++ {
			deferred := &t.deferredFunctions[i]
			if deferred.emitted || !references.contains(deferred.name) {
				continue
			}
			deferred.emitted = true
			t.out = append(t.out, deferred.source...)
			references.scan(deferred.source)
			progress = true
		}
		for i := 0; i < len(t.deferredVariables); i++ {
			deferred := &t.deferredVariables[i]
			if deferred.emitted || !references.contains(deferred.name) {
				continue
			}
			deferred.emitted = true
			t.out = append(t.out, deferred.source...)
			references.scan(deferred.source)
			progress = true
		}
		if !progress {
			return
		}
	}
}

func (t *translator) emitPendingDeclarations() {
	t.out = append(t.out, t.staticOut...)
	references := newGeneratedIdentifierReferences(len(t.externals) + len(t.functions) + len(t.tentatives))
	for i := 0; i < len(t.externals); i++ {
		references.add(t.externals[i].goName)
	}
	for i := 0; i < len(t.functions); i++ {
		if !t.functions[i].variadic && (!t.functions[i].defined || t.object && t.functions[i].attributes.weak) {
			references.add(t.functions[i].name)
		}
	}
	for i := 0; i < len(t.tentatives); i++ {
		references.add(t.tentatives[i].name)
	}
	references.scan(t.out)
	for i := 0; i < len(t.types); i++ {
		info := t.typeInfo(i)
		if info.complete || info.goName == "" || info.kind != cTypeStruct && info.kind != cTypeUnion {
			continue
		}
		duplicate := false
		for j := 0; j < len(t.types); j++ {
			if j == i {
				continue
			}
			prior := t.typeInfo(j)
			if prior.goName == info.goName && (prior.kind == cTypeStruct || prior.kind == cTypeUnion) &&
				(prior.complete || j < i) {
				duplicate = true
			}
		}
		if !duplicate {
			t.appendText("type ")
			t.appendText(info.goName)
			t.appendText(" struct{};\n")
		}
	}
	for i := 0; i < len(t.externals); i++ {
		external := t.externals[i]
		defined := false
		for j := 0; j < len(t.definitions); j++ {
			defined = defined || t.definitions[j].name == external.name
		}
		for j := 0; j < len(t.tentatives); j++ {
			defined = defined || t.tentatives[j].name == external.name
		}
		if defined || !references.contains(external.goName) {
			continue
		}
		t.emitObjectMetadata("variable-extern", external.name, external.attributes, storageExtern, nil, external.typeID, false)
		t.appendText("var ")
		t.appendText(external.goName)
		t.out = append(t.out, ' ')
		t.emitType(external.typeID)
		t.out = append(t.out, ';', '\n')
	}
	for i := 0; i < len(t.functions); i++ {
		if t.object && !t.functions[i].variadic && references.contains(t.functions[i].name) &&
			(!t.functions[i].defined || t.object && t.functions[i].attributes.weak) {
			t.emitForeignFunctionName(t.functions[i])
		}
	}
	for i := 0; i < len(t.variadicCalls); i++ {
		t.emitVariadicForeignFunction(t.variadicCalls[i])
	}
	for i := 0; i < len(t.tentatives); i++ {
		if t.pruneUnusedStatics && t.tentatives[i].storage == storageStatic && !t.tentatives[i].attributes.used &&
			!references.contains(t.tentatives[i].name) {
			continue
		}
		typeID := t.tentatives[i].typeID
		info := t.typeInfo(typeID)
		if info.kind == cTypeArray && info.count == 0 {
			typeID = t.arrayType(info.base, 1)
		}
		t.emitObjectMetadata("variable", t.tentatives[i].name, t.tentatives[i].attributes, t.tentatives[i].storage, nil, typeID, false)
		t.appendText("var ")
		t.out = append(t.out, t.cSourceGoIdentifier(t.tentatives[i].name)...)
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

func newGeneratedIdentifierReferences(capacity int) generatedIdentifierReferences {
	size := 16
	for size < capacity*2 {
		size *= 2
	}
	return generatedIdentifierReferences{
		buckets: make([]int, size),
		names:   make([]generatedIdentifierReference, 0, capacity),
	}
}

func (r *generatedIdentifierReferences) add(name string) {
	if name == "" {
		return
	}
	bucket := cNameHashString(name) & (len(r.buckets) - 1)
	for entry := r.buckets[bucket]; entry != 0; entry = r.names[entry-1].next {
		if r.names[entry-1].name == name {
			return
		}
	}
	r.names = append(r.names, generatedIdentifierReference{name: name, next: r.buckets[bucket]})
	r.buckets[bucket] = len(r.names)
}

func (r *generatedIdentifierReferences) scan(source []byte) {
	for start := 0; start < len(source); {
		if !generatedIdentifierByte(source[start]) {
			start++
			continue
		}
		end := start + 1
		for end < len(source) && generatedIdentifierByte(source[end]) {
			end++
		}
		bucket := cNameHash(source[start:end]) & (len(r.buckets) - 1)
		for entry := r.buckets[bucket]; entry != 0; entry = r.names[entry-1].next {
			candidate := &r.names[entry-1]
			if len(candidate.name) == end-start && ppBytesStringEqual(source[start:end], candidate.name) {
				candidate.found = true
			}
		}
		start = end
	}
}

func (r *generatedIdentifierReferences) mark(name string) {
	bucket := cNameHashString(name) & (len(r.buckets) - 1)
	for entry := r.buckets[bucket]; entry != 0; entry = r.names[entry-1].next {
		candidate := &r.names[entry-1]
		if candidate.name == name {
			candidate.found = true
			return
		}
	}
}

func (r *generatedIdentifierReferences) contains(name string) bool {
	bucket := cNameHashString(name) & (len(r.buckets) - 1)
	for entry := r.buckets[bucket]; entry != 0; entry = r.names[entry-1].next {
		candidate := r.names[entry-1]
		if candidate.name == name {
			return candidate.found
		}
	}
	return false
}

func generatedIdentifierByte(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

func (t *translator) compatibleType(left int, right int) bool {
	if left == right {
		return true
	}
	a, b := t.typeInfo(left), t.typeInfo(right)
	if a.kind == cTypeArray && b.kind == cTypeArray && (a.count == 0 || b.count == 0) {
		return t.compatibleParameterType(a.base, b.base)
	}
	return t.compatibleParameterType(left, right)
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

// Go's scalar types provide at most eight bytes of alignment. Indirect C
// aggregates use one scalar as an alignment carrier followed by a byte tail;
// the carrier's width, rather than the requested C alignment, determines how
// many bytes it occupies in the translated representation.
func cAggregateCarrierSize(alignment int) int {
	if alignment >= 8 {
		return 8
	}
	if alignment >= 4 {
		return 4
	}
	if alignment >= 2 {
		return 2
	}
	return 1
}

func (t *translator) emitAggregateDeclaration(typeID int) {
	info := t.typeInfo(typeID)
	t.appendText("type ")
	t.out = append(t.out, info.goName...)
	if info.kind == cTypeUnion || info.indirect {
		t.appendText(" struct{__c_align ")
		carrierSize := cAggregateCarrierSize(info.align)
		switch carrierSize {
		case 8:
			t.appendText("uint64")
		case 4:
			t.appendText("uint32")
		case 2:
			t.appendText("uint16")
		default:
			t.appendText("uint8")
		}
		if tail := info.size - carrierSize; tail > 0 {
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
		t.out = append(t.out, field.goName...)
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
			if info.kind != cTypeUnion && !info.indirect && field.emit {
				continue
			}
			t.usesUnsafe = true
			t.appendText("func(p *")
			t.out = append(t.out, info.goName...)
			t.appendText(")")
			t.appendText(cPointerAccessorName(field))
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
	if info.kind == cTypeUnion || info.indirect || !field.emit {
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
	if info.kind == cTypeUnion || info.indirect || !field.emit {
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

func cGoIdentifier(name string) string {
	if cTokenSetContains("break,default,func,interface,select,case,defer,go,map,struct,chan,else,goto,package,switch,const,fallthrough,if,range,type,continue,for,import,return,var", []byte(name)) {
		return "__c_name_" + name
	}
	return name
}

func (t *translator) cSourceGoIdentifier(name string) string {
	if t.isolateGoBuiltins && !t.object && t.cIdentifierIsGoPredeclared(name) {
		return "__c_name_" + name
	}
	return cGoIdentifier(name)
}

func (t *translator) cIdentifierIsGoPredeclared(name string) bool {
	return cTokenSetContains("append,cap,close,complex,complex64,complex128,copy,delete,error,false,float32,float64,imag,int,int8,int16,int32,int64,len,make,new,nil,panic,print,println,real,recover,rune,string,true,uint,uint8,uint16,uint32,uint64,uintptr", []byte(name))
}

func (t *translator) cFunctionGoName(name string) string {
	for i := 0; i < len(t.goExports); i++ {
		if t.goExports[i].CName == name {
			return t.goExports[i].GoName
		}
	}
	if t.object {
		return name
	}
	return t.cSourceGoIdentifier(name)
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
	resultKind := t.typeInfo(resultType).kind
	if resultKind == cTypePointer {
		t.appendText("return nil")
	} else if resultKind == cTypeStruct || resultKind == cTypeUnion || resultKind == cTypeArray {
		t.appendText("return ")
		t.emitType(resultType)
		t.appendText("{}")
	} else if resultType != cTypeVoidID {
		t.appendText("return 0")
	}
	t.out = append(t.out, '}', '\n')
}

func (t *translator) emitVariadicForeignFunction(call cVariadicCall) {
	fn := t.functions[call.function]
	if fn.defined || !t.object {
		extraCount := call.paramCount - fn.paramCount
		if extraCount < 0 || extraCount > 64 {
			t.fail(TranslateErrUnsupported)
			return
		}
		for i := fn.paramCount; (fn.consumesVariadic || !t.object) && i < call.paramCount; i++ {
			info := t.typeInfo(t.functionParams[call.paramStart+i])
			if info.kind != cTypePointer && info.kind != cTypeInt && info.kind != cTypeUint && info.kind != cTypeBool && info.kind != cTypeFloat ||
				info.kind != cTypePointer && (info.size < 1 || info.size > 8) {
				t.fail(TranslateErrUnsupported)
				return
			}
		}
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
		if !fn.consumesVariadic && t.object {
			t.appendText("{")
			if fn.resultType != cTypeVoidID {
				t.appendText("return ")
			}
			t.out = append(t.out, t.cFunctionGoName(fn.name)...)
			t.out = append(t.out, '(')
			for i := 0; i < fn.paramCount; i++ {
				if i > 0 {
					t.out = append(t.out, ',')
				}
				t.out = append(t.out, 'p')
				t.appendDecimal(i)
			}
			if fn.paramCount != 0 {
				t.out = append(t.out, ',')
			}
			t.appendText("nil)}\n")
			return
		}
		wordCount := 0
		for i := 0; i < extraCount; i++ {
			size := t.typeInfo(t.functionParams[call.paramStart+fn.paramCount+i]).size
			if size < t.pointerSize {
				size = t.pointerSize
			}
			wordCount += (size + t.pointerSize - 1) / t.pointerSize
		}
		if wordCount < 6 {
			wordCount = 6
		}
		t.appendText("{var __c_va_words [")
		t.appendDecimal(wordCount)
		// C automatic objects are intentionally not zeroed by the object backend.
		// The synthetic va_list is compiler state rather than a source automatic:
		// initialize all three words so unwritten register/overflow cursors cannot
		// inherit stale stack contents.
		t.appendText("]uintptr;var __c_va [3]uintptr=[3]uintptr{};")
		wordOffset := 0
		for i := 0; i < extraCount; i++ {
			typeID := t.functionParams[call.paramStart+fn.paramCount+i]
			info := t.typeInfo(typeID)
			words := (info.size + t.pointerSize - 1) / t.pointerSize
			if words < 1 {
				words = 1
			}
			if info.kind == cTypeFloat {
				t.usesUnsafe = true
				t.appendText("__c_va_bits_")
				t.appendDecimal(i)
				t.appendText(":=uint64(0);*(*float64)(__c_unsafe.Pointer(&__c_va_bits_")
				t.appendDecimal(i)
				t.appendText("))=")
				t.out = append(t.out, 'p')
				t.appendDecimal(fn.paramCount + i)
				t.appendText(";")
				for word := 0; word < words; word++ {
					t.appendText("__c_va_words[")
					t.appendDecimal(wordOffset + word)
					t.appendText("]=uintptr(__c_va_bits_")
					t.appendDecimal(i)
					if word != 0 {
						t.appendText(">>32")
					}
					t.appendText(");")
				}
			} else {
				t.appendText("__c_va_words[")
				t.appendDecimal(wordOffset)
				t.appendText("]=uintptr(")
				if info.kind == cTypePointer {
					t.usesUnsafe = true
					t.appendText("__c_unsafe.Pointer(")
				}
				t.out = append(t.out, 'p')
				t.appendDecimal(fn.paramCount + i)
				if info.kind == cTypePointer {
					t.appendText(")")
				}
				t.appendText(");")
				if words == 2 {
					t.appendText("__c_va_words[")
					t.appendDecimal(wordOffset + 1)
					t.appendText("]=uintptr(uint64(p")
					t.appendDecimal(fn.paramCount + i)
					t.appendText(")>>32);")
				}
			}
			wordOffset += words
		}
		t.appendText("__c_va[0]=uintptr(__c_unsafe.Pointer(&__c_va_words[0]));")
		if extraCount > 6 {
			t.appendText("__c_va[2]=uintptr(__c_unsafe.Pointer(&__c_va_words[6]));")
		}
		t.usesUnsafe = true
		if fn.resultType != cTypeVoidID {
			t.appendText("return ")
		}
		t.out = append(t.out, t.cFunctionGoName(fn.name)...)
		t.out = append(t.out, '(')
		for i := 0; i < fn.paramCount; i++ {
			if i > 0 {
				t.out = append(t.out, ',')
			}
			t.out = append(t.out, 'p')
			t.appendDecimal(i)
		}
		if fn.paramCount != 0 {
			t.out = append(t.out, ',')
		}
		t.appendText("&__c_va")
		t.appendText(")}\n")
		return
	}
	t.emitForeignFunction(fn.name, call.goName, fn.resultType, call.paramStart, call.paramCount)
}

func (t *translator) appendDecimal(value int) {
	t.appendDecimalUnsigned(uint64(value))
}

func (t *translator) appendDecimalUnsigned(value uint64) {
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
		// Negating the minimum signed integer overflows. Form its magnitude
		// without ever representing that positive value as an int.
		t.appendDecimalUnsigned(uint64(-(value + 1)) + 1)
		return
	}
	t.appendDecimal(value)
}

func (t *translator) isGNUAttribute() bool {
	return t.currentIs("__attribute__") || t.currentIs("__attribute")
}

func (t *translator) isGNUAttributeStatement() bool {
	at := t.pos
	for at < len(t.tokens) && (tokenIs(t.src, t.tokens[at], "__attribute__") || tokenIs(t.src, t.tokens[at], "__attribute")) {
		if at+1 >= len(t.tokens) || !tokenIs(t.src, t.tokens[at+1], "(") {
			return false
		}
		close := matchingToken(t.src, t.tokens, at+1, "(", ")")
		if close < 0 {
			return false
		}
		at = close + 1
	}
	return at < len(t.tokens) && tokenIs(t.src, t.tokens[at], ";")
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
				value, valid := t.attributeStrings(arguments)
				if !valid || value == "" {
					return attributes, false
				}
				attributes.section = value
			case "weak", "__weak__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.weak = true
			case "ms_abi", "__ms_abi__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.callingConvention = 1
			case "sysv_abi", "__sysv_abi__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.callingConvention = 2
			case "cleanup", "__cleanup__":
				if len(arguments) != 1 || tokenKind(arguments[0]) != tokenIdent {
					return attributes, false
				}
				attributes.cleanup = string(tokenText(t.src, arguments[0]))
			case "__gnu_inline__", "gnu_inline":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.gnuInline = true
			case "__error__", "error":
				if message, valid := t.attributeStrings(arguments); !valid || message == "" {
					return attributes, false
				}
				attributes.diagnosticError = true
			case "used", "__used__":
				if len(arguments) != 0 {
					return attributes, false
				}
				attributes.used = true
			case "__unused__", "unused", "__no_instrument_function__", "no_instrument_function",
				"__always_inline__", "always_inline", "__warn_unused_result__", "warn_unused_result", "__pure__", "pure",
				"__leaf__", "leaf", "__nothrow__", "nothrow",
				"__noinline__", "noinline",
				"__const__", "const",
				"__format__", "format", "__noreturn__", "noreturn", "__malloc__", "malloc",
				"__warning__", "warning",
				"__designated_init__", "designated_init",
				"__externally_visible__", "externally_visible",
				"__alloc_size__", "alloc_size",
				"__assume_aligned__", "assume_aligned",
				"__nonnull__", "nonnull",
				"cold", "__cold__", "hot", "__hot__", "nocf_check", "__nocf_check__",
				"no_sanitize_address", "__no_sanitize_address__", "no_profile_instrument_function", "__no_profile_instrument_function__",
				"noclone", "__noclone__", "no_stack_protector", "__no_stack_protector__",
				"fallthrough", "__fallthrough__":
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

func (t *translator) attributeStrings(tokens []token) (string, bool) {
	if len(tokens) == 0 {
		return "", false
	}
	value := ""
	for i := 0; i < len(tokens); i++ {
		if tokenKind(tokens[i]) != tokenString {
			return "", false
		}
		part, ok := t.attributeString(tokens[i])
		if !ok {
			return "", false
		}
		value += part
	}
	return value, true
}

func (t *translator) attributeString(tok token) (string, bool) {
	value, ok := decodeCString(tokenText(t.src, tok))
	if !ok {
		return "", false
	}
	return string(value), true
}

func (t *translator) takeAsmClause() bool {
	_, ok := t.parseAsmClause()
	return ok
}

func (t *translator) parseAsmClause() (cAsm, bool) {
	var result cAsm
	if !t.currentIs("asm") && !t.currentIs("__asm") && !t.currentIs("__asm__") {
		return result, false
	}
	t.pos++
	for t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") ||
		t.currentIs("inline") || t.currentIs("__inline") || t.currentIs("__inline__") || t.currentIs("goto") {
		if t.currentIs("volatile") || t.currentIs("__volatile") || t.currentIs("__volatile__") {
			result.volatile = true
		} else if t.currentIs("goto") {
			result.gotoAsm = true
		}
		t.pos++
	}
	end := matchingToken(t.src, t.tokens, t.pos, "(", ")")
	if end < 0 {
		return result, false
	}
	groups := splitTopLevel(t.src, t.tokens[t.pos+1:end], ":")
	if len(groups) == 0 || len(groups) > 5 {
		return result, false
	}
	value, ok := t.asmString(groups[0])
	if !ok {
		return result, false
	}
	result.template = value
	if len(groups) > 1 {
		result.outputs, ok = t.parseAsmOperands(groups[1])
		if !ok {
			return result, false
		}
	}
	if len(groups) > 2 {
		result.inputs, ok = t.parseAsmOperands(groups[2])
		if !ok {
			return result, false
		}
	}
	if len(groups) > 3 {
		for _, item := range splitTopLevel(t.src, groups[3], ",") {
			if len(item) == 0 {
				continue
			}
			clobber, valid := t.asmString(item)
			if !valid || len(clobber) == 0 {
				return result, false
			}
			result.clobbers = append(result.clobbers, string(clobber))
		}
	}
	if len(groups) > 4 {
		for _, item := range splitTopLevel(t.src, groups[4], ",") {
			if len(item) != 1 || tokenKind(item[0]) != tokenIdent {
				return result, false
			}
			result.labels = append(result.labels, string(tokenText(t.src, item[0])))
		}
	}
	if result.gotoAsm != (len(groups) == 5) {
		return result, false
	}
	t.pos = end + 1
	return result, true
}

func (t *translator) asmString(tokens []token) ([]byte, bool) {
	value, ok := t.cStringBytes(tokens)
	if !ok || len(value) == 0 {
		return nil, false
	}
	return value[:len(value)-1], true
}

func (t *translator) parseAsmOperands(tokens []token) ([]cAsmOperand, bool) {
	var result []cAsmOperand
	if len(tokens) == 0 {
		return result, true
	}
	for _, item := range splitTopLevel(t.src, tokens, ",") {
		if len(item) == 0 {
			return nil, false
		}
		operand := cAsmOperand{}
		at := 0
		if tokenIs(t.src, item[at], "[") {
			if len(item) < 3 || tokenKind(item[1]) != tokenIdent || !tokenIs(t.src, item[2], "]") {
				return nil, false
			}
			operand.name = string(tokenText(t.src, item[1]))
			at = 3
		}
		constraintStart := at
		for at < len(item) && tokenKind(item[at]) == tokenString {
			at++
		}
		constraint, ok := t.asmString(item[constraintStart:at])
		if !ok || at >= len(item) || !tokenIs(t.src, item[at], "(") {
			return nil, false
		}
		close := matchingToken(t.src, item, at, "(", ")")
		if close != len(item)-1 || close == at+1 {
			return nil, false
		}
		operand.constraint = string(constraint)
		operand.expression = item[at+1 : close]
		result = append(result, operand)
	}
	return result, true
}

func (t *translator) takeAsmStatement() bool {
	return t.takeAsmClause() && t.take(";")
}

func (t *translator) emitAsmStatement() bool {
	operation, ok := t.parseAsmClause()
	if !ok || !t.take(";") {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		t.checkExpression(operation.outputs[i].expression)
	}
	for i := 0; i < len(operation.inputs); i++ {
		t.checkExpression(operation.inputs[i].expression)
	}
	if !t.ok {
		return false
	}
	if t.checkOnly {
		return true
	}
	if t.assemblyOutput {
		// Header-only inline functions contain many target instructions which do
		// not contribute to a layout generator's assembly. Preserve only the
		// constant metadata records that the -S contract explicitly supports.
		t.emitAsmConstantMetadata(operation)
		return t.ok
	}
	if operation.gotoAsm || len(operation.labels) != 0 {
		return t.emitAsmGotoCPUFeature(operation) || t.emitAsmGotoUserLoad(operation) || t.emitAsmGotoUserStore(operation)
	}
	identity := string(operation.template) == "leaq %c1(%%rip), %0" &&
		len(operation.outputs) == 1 && operation.outputs[0].constraint == "=r" &&
		len(operation.inputs) == 1 && operation.inputs[0].constraint == "i" &&
		t.asmExpressionsEqual(operation.outputs[0].expression, operation.inputs[0].expression)
	handled := identity
	if !handled {
		handled = t.emitAsmInstructionPointer(operation)
	}
	if !handled {
		handled = t.emitAsmTiedIdentity(operation)
	}
	if !handled {
		handled = t.emitAsmChecksumFold(operation)
	}
	if !handled {
		handled = t.emitAsmChecksumAdd(operation)
	}
	if !handled {
		handled = t.emitAsmChecksumAdd64(operation)
	}
	if !handled {
		handled = t.emitAsmIPChecksum(operation)
	}
	if !handled {
		handled = t.emitAsmClampAddress(operation)
	}
	if !handled {
		handled = t.emitAsmRuntimePointer(operation)
	}
	if !handled {
		handled = t.emitAsmRuntimeShift(operation)
	}
	if !handled {
		handled = t.emitAsmBitTest(operation)
	}
	if !handled {
		handled = t.emitAsmCompareBytes(operation)
	}
	if !handled {
		handled = t.emitAsmDivide32(operation)
	}
	if !handled {
		handled = t.emitAsmBoundsMask(operation)
	}
	if !handled {
		handled = t.emitAsmEndBranchValue(operation)
	}
	if !handled {
		handled = t.emitAsmCopyBytes(operation)
	}
	if !handled {
		handled = t.emitAsmUserCopy(operation)
	}
	if !handled {
		handled = t.emitAsmInlineGetUser(operation)
	}
	if !handled {
		handled = t.emitAsmExceptionLoad(operation)
	}
	if !handled {
		handled = t.emitAsmRepeatedCopy(operation)
	}
	if !handled {
		handled = t.emitAsmByteOperation(operation)
	}
	if !handled {
		handled = t.emitAsmByteTest(operation)
	}
	if !handled {
		handled = t.emitAsmBitOperation64(operation)
	}
	if !handled {
		handled = t.emitAsmBitScan(operation)
	}
	if !handled {
		handled = t.emitAsmPopulationCount(operation)
	}
	if !handled {
		handled = t.emitAsmByteSwap(operation)
	}
	if !handled {
		handled = t.emitAsmMultiplyDivide64(operation)
	}
	if !handled {
		handled = t.emitAsmMultiply32(operation)
	}
	if !handled {
		handled = t.emitAsmMultiplyShift32(operation)
	}
	if !handled {
		handled = t.emitAsmAlternativeCall(operation)
	}
	if !handled {
		handled = t.emitAsmAlternativeRelocSelftest(operation)
	}
	if !handled {
		handled = t.emitAsmCompareExchange128(operation)
	}
	if !handled {
		handled = t.emitAsmAtomicArithmetic(operation)
	}
	if !handled {
		handled = t.emitAsmAtomicExchange(operation)
	}
	if !handled {
		handled = t.emitAsmCompareExchange(operation)
	}
	if !handled {
		handled = t.emitAsmRepeatedStore(operation)
	}
	if !handled {
		handled = t.emitAsmUserClear(operation)
	}
	if !handled {
		handled = t.emitAsmGetUser(operation)
	}
	if !handled {
		handled = t.emitAsmPutUser(operation)
	}
	if !handled {
		handled = t.emitAsmNonTemporalStore(operation)
	}
	if !handled {
		handled = t.emitAsmEmptyOperands(operation)
	}
	if !handled {
		handled = t.emitAsmEmptyReadWrite(operation)
	}
	if !handled {
		handled = t.emitAsmScalarLoad(operation)
	}
	if !handled {
		handled = t.emitAsmScalarStore(operation)
	}
	if !handled {
		handled = t.emitAsmVerifySegment(operation)
	}
	if !handled {
		handled = t.emitAsmInvalidateProcessContext(operation)
	}
	if !handled {
		handled = t.emitAsmUserAccess(operation)
	}
	if !handled {
		handled = t.emitAsmReadFlags(operation)
	}
	if !handled {
		handled = t.emitAsmWriteFlags(operation)
	}
	if !handled {
		handled = t.emitAsmHalt(operation)
	}
	if !handled {
		handled = t.emitAsmWriteBackInvalidate(operation)
	}
	if !handled {
		handled = t.emitAsmSwapGS(operation)
	}
	if !handled {
		handled = t.emitAsmCacheFlush(operation)
	}
	if !handled {
		handled = t.emitAsmPrefetch(operation)
	}
	if !handled {
		handled = t.emitAsmAlternativeDivider(operation)
	}
	if !handled {
		handled = t.emitAsmAlternativeStore(operation)
	}
	if !handled {
		handled = t.emitAsmFence(operation)
	}
	if !handled {
		handled = t.emitAsmSerialize(operation)
	}
	if !handled {
		handled = t.emitAsmHypercall(operation)
	}
	if !handled {
		handled = t.emitAsmFixedRegisterMove(operation)
	}
	if !handled {
		handled = t.emitAsmLoadGS(operation)
	}
	if !handled {
		handled = t.emitAsmDirectMove(operation)
	}
	if !handled {
		handled = t.emitAsmRandom(operation)
	}
	if !handled {
		handled = t.emitAsmBreakpoint(operation)
	}
	if !handled {
		handled = t.emitAsmTileRelease(operation)
	}
	if !handled {
		handled = t.emitAsmUndefinedInstruction(operation)
	}
	if !handled {
		handled = t.emitAsmInt3Selftest(operation)
	}
	if !handled {
		handled = t.emitAsmIRETToSelf(operation)
	}
	if !handled {
		handled = t.emitAsmBitSet(operation)
	}
	if !handled {
		handled = t.emitAsmControlRegister(operation)
	}
	if !handled {
		handled = t.emitAsmXControl(operation)
	}
	if !handled {
		handled = t.emitAsmFPUState(operation)
	}
	if !handled {
		handled = t.emitAsmExtendedState(operation)
	}
	if !handled {
		handled = t.emitAsmFPUInit(operation)
	}
	if !handled {
		handled = t.emitAsmFDIVBug(operation)
	}
	if !handled {
		handled = t.emitAsmEFlagsToggle(operation)
	}
	if !handled {
		handled = t.emitAsmCPUID(operation)
	}
	if !handled {
		handled = t.emitAsmCounter(operation)
	}
	if !handled {
		handled = t.emitAsmMSR(operation)
	}
	if !handled {
		handled = t.emitAsmPause(operation)
	}
	if !handled {
		handled = t.emitAsmDelayLoop(operation)
	}
	if !handled {
		handled = t.emitAsmMonitorWait(operation)
	}
	if !handled {
		handled = t.emitAsmCPUNODE(operation)
	}
	if !handled {
		handled = t.emitAsmFarCall16(operation)
	}
	if !handled {
		handled = t.emitAsmInterruptFlag(operation)
	}
	if !handled {
		handled = t.emitAsmDescriptorTable(operation)
	}
	if !handled {
		handled = t.emitAsmPortIO(operation)
	}
	if !handled {
		handled = t.emitAsmStringPortIO(operation)
	}
	if !handled {
		handled = t.emitAsmSyscall(operation)
	}
	if !handled {
		handled = t.emitAsmInterruptWithStack(operation)
	}
	if !handled {
		handled = t.emitAsmSegmentRegister(operation)
	}
	if !handled {
		handled = t.emitAsmAccessRights(operation)
	}
	if !handled {
		handled = t.emitAsmBaseRegister(operation)
	}
	if !handled {
		handled = t.emitAsmVideoTextFill(operation)
	}
	if !handled {
		handled = t.emitAsmSegmentMemory(operation)
	}
	if !handled {
		handled = t.emitAsmSegmentCompare(operation)
	}
	if !handled {
		handled = t.emitAsmIRQStackCall(operation)
	}
	if len(operation.template) != 0 && !handled {
		return false
	}
	if len(operation.template) == 0 && !handled && (len(operation.outputs) != 0 || len(operation.inputs) != 0) {
		return false
	}
	for i := 0; i < len(operation.clobbers); i++ {
		switch operation.clobbers[i] {
		case "memory":
			t.ensureAsmMemoryHelper()
			t.appendText("renvo_runtime_CMemoryBarrier();")
		case "cc":
			t.ensureAsmCCHelper()
			t.appendText("renvo_runtime_CConditionClobber();")
		case "eax", "ebx", "ecx", "edx", "esi", "edi", "rax", "rbx", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "r11":
			if !handled {
				return false
			}
		default:
			return false
		}
	}
	if len(operation.clobbers) == 0 {
		t.out = append(t.out, ';')
	}
	return true
}

func (t *translator) emitAsmConstantMetadata(operation cAsm) bool {
	line := trimAsmLine(operation.template)
	for len(line) > 0 && line[0] == '\n' {
		line = trimAsmLine(line[1:])
	}
	if !asmLinePrefix(line, ".ascii") || !textContains(line, "\"->") ||
		len(operation.outputs) != 0 || len(operation.clobbers) != 0 || len(operation.labels) != 0 || operation.gotoAsm {
		return false
	}
	start := len(t.assemblyOut)
	for i := 0; i < len(line); i++ {
		if line[i] != '%' {
			t.assemblyOut = append(t.assemblyOut, line[i])
			continue
		}
		modifier := byte(0)
		at := i + 1
		if at < len(line) && line[at] == 'c' {
			modifier = 'c'
			at++
		}
		if at >= len(line) || line[at] < '0' || line[at] > '9' {
			t.assemblyOut = t.assemblyOut[:start]
			return false
		}
		index := int(line[at] - '0')
		if index >= len(operation.inputs) {
			t.assemblyOut = t.assemblyOut[:start]
			return false
		}
		value, ok := t.constantExpression(operation.inputs[index].expression)
		if !ok || operation.inputs[index].constraint != "i" {
			t.assemblyOut = t.assemblyOut[:start]
			return false
		}
		if modifier == 0 {
			t.assemblyOut = append(t.assemblyOut, '$')
		}
		if value < 0 {
			t.assemblyOut = append(t.assemblyOut, '-')
			value = -value
		}
		t.assemblyOut = append(t.assemblyOut, decimalString(value)...)
		i = at
	}
	t.assemblyOut = append(t.assemblyOut, '\n')
	return true
}

func (t *translator) emitAsmBreakpoint(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "int3") && !t.asmTemplateCompactPrefix(operation.template, "int$3") ||
		len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	if !t.asmBreakpointHelper {
		t.asmBreakpointHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CBreakpoint(){}\n"...)
	}
	t.appendText("renvo_runtime_CBreakpoint();")
	return true
}

func (t *translator) emitAsmRandom(operation cAsm) bool {
	kind, name := 0, ""
	if t.asmTemplateCompactPrefix(operation.template, "rdrand%[out]") {
		kind, name = 1, "renvo_runtime_CRDRAND64"
	} else if t.asmTemplateCompactPrefix(operation.template, "rdseed%[out]") {
		kind, name = 2, "renvo_runtime_CRDSEED64"
	} else {
		return false
	}
	if len(operation.outputs) != 2 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 {
		return false
	}
	condition, output := -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].constraint == "=@ccc" {
			condition = i
		} else if operation.outputs[i].name == "out" && operation.outputs[i].constraint == "=r" {
			output = i
		}
	}
	if condition < 0 || output < 0 || t.typeInfo(t.expressionType(operation.outputs[condition].expression)).kind != cTypeBool ||
		t.typeSize(t.expressionType(operation.outputs[output].expression)) != 8 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.outputs[output].expression)
	if !ok {
		return false
	}
	if t.asmRandomOps&kind == 0 {
		t.asmRandomOps |= kind
		t.staticOut = append(t.staticOut, ("func " + name + "(value *uint64) uint8{return 0}\n")...)
	}
	t.emitExpression(operation.outputs[condition].expression)
	t.appendText("=")
	t.appendText(name)
	t.appendText("(")
	t.convertedExpression(t.pointerType(cTypeUint64ID), pointer)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmFDIVBug(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "fninitfldl%1fdivl%2fmull%2fldl%1fsubp%%st,%%st(1)fistpl%0fwaitfninit") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=m" ||
		len(operation.inputs) != 2 || operation.inputs[0].constraint != "m" || operation.inputs[1].constraint != "m" ||
		len(operation.clobbers) != 0 {
		return false
	}
	output, outputOK := t.asmDereferencePointer(operation.outputs[0].expression)
	x, xOK := t.asmDereferencePointer(operation.inputs[0].expression)
	y, yOK := t.asmDereferencePointer(operation.inputs[1].expression)
	if !outputOK || !xOK || !yOK || t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 ||
		t.expressionType(operation.inputs[0].expression) != cTypeFloat64ID ||
		t.expressionType(operation.inputs[1].expression) != cTypeFloat64ID {
		return false
	}
	if !t.asmFDIVBug {
		t.asmFDIVBug = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CFDIVBug(output *int32,x *float64,y *float64){}\n"...)
	}
	t.appendText("renvo_runtime_CFDIVBug(")
	t.convertedExpression(t.pointerType(cTypeInt32ID), output)
	t.appendText(",")
	t.convertedExpression(t.pointerType(cTypeFloat64ID), x)
	t.appendText(",")
	t.convertedExpression(t.pointerType(cTypeFloat64ID), y)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmInstructionPointer(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "lea0(%%rip),%0") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" ||
		len(operation.inputs) != 0 || len(operation.clobbers) != 0 {
		return false
	}
	typeID := t.expressionType(operation.outputs[0].expression)
	if t.typeSize(typeID) != t.pointerSize {
		return false
	}
	if !t.asmInstructionHelper {
		t.asmInstructionHelper = true
		outer := t.out
		t.out = t.staticOut
		t.appendText("func renvo_runtime_CReadInstructionPointer() ")
		t.emitType(typeID)
		t.appendText("{return 0}\n")
		t.staticOut = t.out
		t.out = outer
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CReadInstructionPointer();")
	return t.ok
}

func (t *translator) emitAsmIRQStackCall(operation cAsm) bool {
	if !t.object || len(operation.outputs) != 2 || len(operation.inputs) < 2 || len(operation.inputs) > 4 ||
		operation.outputs[0].constraint != "+r" || operation.outputs[1].constraint != "+r" ||
		!t.asmTemplateCompactContains(operation.template, "movq%%rsp,(%[tos])movq%[tos],%%rsp") ||
		!t.asmTemplateCompactContains(operation.template, "call%c[__func]") ||
		!t.asmTemplateCompactContains(operation.template, "popq%%rsp") {
		return false
	}
	hasEnter := t.asmTemplateCompactContains(operation.template, "callirq_enter_rcu")
	hasExit := t.asmTemplateCompactContains(operation.template, "callirq_exit_rcu")
	if hasEnter != hasExit {
		return false
	}
	functionOperand, tosOperand, arg1Operand, arg2Operand := -1, -1, -1, -1
	for i := 0; i < len(operation.inputs); i++ {
		switch operation.inputs[i].name {
		case "__func":
			functionOperand = i
		case "tos":
			tosOperand = i
		case "arg1":
			arg1Operand = i
		case "arg2":
			arg2Operand = i
		}
	}
	argCount := len(operation.inputs) - 2
	if functionOperand < 0 || tosOperand < 0 || argCount >= 1 && arg1Operand < 0 ||
		argCount == 2 && arg2Operand < 0 || argCount == 0 && (arg1Operand >= 0 || arg2Operand >= 0) ||
		argCount == 1 && arg2Operand >= 0 ||
		operation.inputs[functionOperand].constraint != "i" || operation.inputs[tosOperand].constraint != "r" ||
		arg1Operand >= 0 && operation.inputs[arg1Operand].constraint != "r" ||
		arg2Operand >= 0 && operation.inputs[arg2Operand].constraint != "r" ||
		!t.asmExpressionsEqual(operation.outputs[0].expression, operation.inputs[tosOperand].expression) {
		return false
	}
	targetExpression := operation.inputs[functionOperand].expression
	for len(targetExpression) > 2 && tokenIs(t.src, targetExpression[0], "(") &&
		matchingToken(t.src, targetExpression, 0, "(", ")") == len(targetExpression)-1 {
		targetExpression = targetExpression[1 : len(targetExpression)-1]
	}
	if len(targetExpression) != 1 || tokenKind(targetExpression[0]) != tokenIdent {
		return false
	}
	targetIndex, ok := t.lookupFunction(tokenText(t.src, targetExpression[0]))
	if !ok {
		return false
	}
	target := &t.functions[targetIndex]
	if target.resultType != cTypeVoidID || target.paramCount != argCount || cGoIdentifier(target.name) != target.name ||
		t.typeSize(t.expressionType(operation.inputs[tosOperand].expression)) != t.pointerSize {
		return false
	}
	argOperands := []int{arg1Operand, arg2Operand}
	for i := 0; i < argCount; i++ {
		if argOperands[i] < 0 || t.typeSize(t.expressionType(operation.inputs[argOperands[i]].expression)) > t.pointerSize {
			return false
		}
	}
	name := "renvo_runtime_CIRQStackCall" + decimalString(argCount) + "_" + target.name
	found := false
	for i := 0; i < len(t.asmIRQStackHelpers); i++ {
		found = found || t.asmIRQStackHelpers[i] == name
	}
	if !found {
		t.asmIRQStackHelpers = append(t.asmIRQStackHelpers, name)
		outer := t.out
		t.out = t.staticOut
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(tos ")
		t.emitType(t.expressionType(operation.inputs[tosOperand].expression))
		for i := 0; i < argCount; i++ {
			t.appendText(",arg")
			t.appendDecimal(i + 1)
			t.appendText(" ")
			t.emitType(t.expressionType(operation.inputs[argOperands[i]].expression))
		}
		t.appendText("){}\n")
		t.staticOut = t.out
		t.out = outer
	}
	t.appendText(name)
	t.appendText("(")
	t.emitExpression(operation.inputs[tosOperand].expression)
	for i := 0; i < argCount; i++ {
		t.appendText(",")
		t.convertedExpression(t.functionParams[target.paramStart+i], operation.inputs[argOperands[i]].expression)
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmChecksumFold(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "addl%1,%0adcl$0xffff,%0") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "r" || operation.inputs[1].constraint != "0" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.inputs[1].expression)) != 4 {
		return false
	}
	if !t.asmChecksumHelper {
		t.asmChecksumHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CChecksumFold(initial uint32,addend uint32) uint32{wide:=uint64(initial)+uint64(addend);return uint32(wide+65535+(wide>>32))}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CChecksumFold(uint32(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("),uint32(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmChecksumAdd(operation cAsm) bool {
	three := t.asmTemplateCompactContains(operation.template, "addl%1,%0adcl%2,%0adcl%3,%0adcl$0,%0")
	one := t.asmTemplateCompactContains(operation.template, "addl%2,%0adcl$0,%0")
	if !three && !one || len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 {
		return false
	}
	if three {
		if len(operation.inputs) != 4 || operation.inputs[0].constraint != "g" || operation.inputs[1].constraint != "g" ||
			operation.inputs[2].constraint != "g" || operation.inputs[3].constraint != "0" {
			return false
		}
		for i := 0; i < len(operation.inputs); i++ {
			if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 4 {
				return false
			}
		}
		if !t.asmChecksumAdd3Helper {
			t.asmChecksumAdd3Helper = true
			t.staticOut = append(t.staticOut, "func renvo_runtime_CChecksumAdd3(initial uint32,a uint32,b uint32,c uint32) uint32{wide:=uint64(initial)+uint64(a);sum:=uint32(wide);carry:=wide>>32;wide=uint64(sum)+uint64(b)+carry;sum=uint32(wide);carry=wide>>32;wide=uint64(sum)+uint64(c)+carry;return uint32(wide)+uint32(wide>>32)}\n"...)
		}
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CChecksumAdd3(uint32(")
		t.emitExpression(operation.inputs[3].expression)
		for i := 0; i < 3; i++ {
			t.appendText("),uint32(")
			t.emitExpression(operation.inputs[i].expression)
		}
		t.appendText("));")
		return t.ok
	}
	if len(operation.inputs) != 2 || operation.inputs[0].constraint != "0" || operation.inputs[1].constraint != "rm" ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.inputs[1].expression)) != 4 {
		return false
	}
	if !t.asmChecksumAddHelper {
		t.asmChecksumAddHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CChecksumAdd(initial uint32,addend uint32) uint32{wide:=uint64(initial)+uint64(addend);return uint32(wide)+uint32(wide>>32)}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CChecksumAdd(uint32(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),uint32(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) ensureAsmChecksumAdd64Helper(kind int) {
	if t.asmChecksumAdd64&kind != 0 {
		return
	}
	t.asmChecksumAdd64 |= kind
	if kind == 1 {
		t.staticOut = append(t.staticOut, "func renvo_runtime_CChecksumAdd64(initial uint64,addend uint64) uint64{sum:=initial+addend;carry:=uint64(0);if sum<initial{carry=1};return sum+carry}\n"...)
		return
	}
	if kind == 2 {
		t.staticOut = append(t.staticOut, "func renvo_runtime_CChecksumAdd64Five(initial uint64,a uint64,b uint64,c uint64,d uint64,e uint64) uint64{sum:=initial;carry:=uint64(0);values:=[5]uint64{a,b,c,d,e};for i:=0;i<5;i++{next:=sum+values[i];nextCarry:=uint64(0);if next<sum{nextCarry=1};withCarry:=next+carry;if withCarry<next{nextCarry=1};sum=withCarry;carry=nextCarry};return sum+carry}\n"...)
		return
	}
	t.staticOut = append(t.staticOut, "func renvo_runtime_CChecksumAdd64Memory(initial uint64,address *uint64,count uintptr) uint64{sum:=initial;carry:=uint64(0);for i:=uintptr(0);i<count;i++{value:=*(*uint64)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(address))+i*8));next:=sum+value;nextCarry:=uint64(0);if next<sum{nextCarry=1};withCarry:=next+carry;if withCarry<next{nextCarry=1};sum=withCarry;carry=nextCarry};return sum+carry}\n"...)
	t.usesUnsafe = true
}

func (t *translator) emitAsmChecksumAdd64(operation cAsm) bool {
	if len(operation.outputs) == 1 && operation.outputs[0].name == "sum" && operation.outputs[0].constraint == "=r" &&
		len(operation.inputs) == 3 && operation.inputs[0].constraint == "[sum]" &&
		operation.inputs[1].name == "saddr" && operation.inputs[1].constraint == "r" &&
		operation.inputs[2].name == "daddr" && operation.inputs[2].constraint == "r" && len(operation.clobbers) == 0 &&
		t.typeSize(t.expressionType(operation.outputs[0].expression)) == 8 &&
		t.typeSize(t.expressionType(operation.inputs[0].expression)) == 8 &&
		t.typeInfo(t.expressionType(operation.inputs[1].expression)).kind == cTypePointer &&
		t.typeInfo(t.expressionType(operation.inputs[2].expression)).kind == cTypePointer &&
		t.asmTemplateCompactPrefix(operation.template, "addq(%[saddr]),%[sum]adcq8(%[saddr]),%[sum]adcq(%[daddr]),%[sum]adcq8(%[daddr]),%[sum]adcq$0,%[sum]") {
		t.ensureAsmChecksumAdd64Helper(2)
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CChecksumAdd64Five(uint64(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(")")
		for i := 1; i < 3; i++ {
			t.appendText(",uint64(*(*uint64)(__c_unsafe.Pointer(")
			t.emitExpression(operation.inputs[i].expression)
			t.appendText("))),uint64(*(*uint64)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
			t.emitExpression(operation.inputs[i].expression)
			t.appendText("))+8)))")
		}
		t.appendText(",0);")
		t.usesUnsafe = true
		return t.ok
	}
	if len(operation.outputs) != 1 || operation.outputs[0].constraint != "+r" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 8 || len(operation.clobbers) != 0 {
		return false
	}
	five := t.asmTemplateCompactPrefix(operation.template, "addq%1,%0adcq%2,%0adcq%3,%0adcq%4,%0adcq%5,%0adcq$0,%0")
	one := t.asmTemplateCompactContains(operation.template, "adcq$0,%0") &&
		t.asmTemplateCompactPrefix(operation.template, "addq%1,%0") ||
		t.asmTemplateCompactContains(operation.template, "adcq$0,%[res]") &&
			t.asmTemplateCompactPrefix(operation.template, "addq%[trail],%[res]")
	memoryCount := 0
	if t.asmTemplateCompactPrefix(operation.template, "addq0*8(%[src]),%[res]") &&
		t.asmTemplateCompactContains(operation.template, "adcq$0,%[res]") {
		memoryCount = 1
		if t.asmTemplateCompactContains(operation.template, "adcq1*8(%[src]),%[res]") {
			memoryCount = 2
		}
		if t.asmTemplateCompactContains(operation.template, "adcq3*8(%[src]),%[res]") {
			memoryCount = 4
		}
	}
	if five {
		if len(operation.inputs) != 5 {
			return false
		}
		for i := 0; i < 5; i++ {
			if operation.inputs[i].constraint != "m" || t.typeSize(t.expressionType(operation.inputs[i].expression)) != 8 {
				return false
			}
		}
		t.ensureAsmChecksumAdd64Helper(2)
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CChecksumAdd64Five(uint64(")
		t.emitExpression(operation.outputs[0].expression)
		for i := 0; i < 5; i++ {
			t.appendText("),uint64(")
			t.emitExpression(operation.inputs[i].expression)
		}
		t.appendText("));")
		return t.ok
	}
	if one {
		if len(operation.inputs) != 1 || operation.inputs[0].constraint != "r" ||
			t.typeSize(t.expressionType(operation.inputs[0].expression)) != 8 {
			return false
		}
		t.ensureAsmChecksumAdd64Helper(1)
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CChecksumAdd64(uint64(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("),uint64(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
		return t.ok
	}
	if memoryCount == 0 || len(operation.inputs) != 2 || operation.inputs[0].name != "src" ||
		operation.inputs[0].constraint != "r" || operation.inputs[1].constraint != "m" ||
		t.typeInfo(t.expressionType(operation.inputs[0].expression)).kind != cTypePointer {
		return false
	}
	t.ensureAsmChecksumAdd64Helper(4)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CChecksumAdd64Memory(uint64(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("),(*uint64)(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")),")
	t.appendDecimal(memoryCount)
	t.appendText(");")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmIPChecksum(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "movl(%1),%0") ||
		!t.asmTemplateCompactContains(operation.template, "subl$4,%2") ||
		!t.asmTemplateCompactContains(operation.template, "adcl16(%1),%0") ||
		!t.asmTemplateCompactContains(operation.template, "notl%0") ||
		len(operation.outputs) != 3 || len(operation.inputs) != 2 ||
		operation.outputs[0].constraint != "=r" || operation.outputs[1].constraint != "=r" || operation.outputs[2].constraint != "=r" ||
		operation.inputs[0].constraint != "1" || operation.inputs[1].constraint != "2" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 ||
		t.typeInfo(t.expressionType(operation.outputs[1].expression)).kind != cTypePointer ||
		t.typeSize(t.expressionType(operation.outputs[2].expression)) != 4 {
		return false
	}
	if !t.asmIPChecksumHelper {
		t.asmIPChecksumHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CIPChecksum(pointer **byte,length *uint32) uint32{p:=*pointer;words:=*length;sum:=*(*uint32)(__c_unsafe.Pointer(p));if words<=4{*length=words-4;return sum};carry:=uint64(0);for i:=uint32(1);i<words;i++{value:=uint64(*(*uint32)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(p))+uintptr(i)*4)));wide:=uint64(sum)+value+carry;sum=uint32(wide);carry=wide>>32};sum+=uint32(carry);*pointer=(*byte)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(p))+uintptr(words-4)*4));*length=0;folded:=uint32(sum>>16)+uint32(sum&65535);return ^uint32((folded&65535)+(folded>>16))}\n"...)
		t.usesUnsafe = true
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CIPChecksum(&(")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("),&(")
	t.emitExpression(operation.outputs[2].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmClampAddress(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "cmp%1,%0cmova%1,%0") || len(operation.outputs) != 1 ||
		operation.outputs[0].constraint != "=r" || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "r" || operation.inputs[1].constraint != "0" {
		return false
	}
	typeID := t.expressionType(operation.outputs[0].expression)
	if t.typeInfo(typeID).kind != cTypePointer {
		return false
	}
	name := "__c_clamp_address_" + decimalString(typeID)
	found := false
	for i := 0; i < len(t.asmClampTypes); i++ {
		found = found || t.asmClampTypes[i] == typeID
	}
	if !found {
		t.asmClampTypes = append(t.asmClampTypes, typeID)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(value ")
		t.emitType(typeID)
		t.appendText(",limit uintptr) ")
		t.emitType(typeID)
		t.appendText("{if uintptr(__c_unsafe.Pointer(value))>limit{return (")
		t.emitType(typeID)
		t.appendText(")(__c_unsafe.Pointer(limit))};return value}\n")
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.appendText(name)
	t.appendText("(")
	t.convertedExpression(typeID, operation.inputs[1].expression)
	t.appendText(",uintptr(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmRuntimePointer(operation cAsm) bool {
	section, ok := asmRuntimePointerSection(operation.template)
	if !ok || !t.asmTemplateCompactPrefix(operation.template, "mov%1,%0") || len(operation.outputs) != 1 ||
		operation.outputs[0].constraint != "=r" || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "i" || operation.inputs[1].constraint != "i" {
		return false
	}
	typeID := t.expressionType(operation.outputs[0].expression)
	if t.typeSize(typeID) != t.pointerSize {
		return false
	}
	wordSize, constant := t.constantExpression(operation.inputs[1].expression)
	if !constant || wordSize != t.pointerSize {
		return false
	}
	name := "renvo_runtime_CRuntimePointer_" + cGoIdentifier(section)
	found := false
	for i := 0; i < len(t.asmRuntimePointers); i++ {
		found = found || t.asmRuntimePointers[i] == name
	}
	if !found {
		t.asmRuntimePointers = append(t.asmRuntimePointers, name)
		t.staticOut = append(t.staticOut, ("func " + name + "(value uintptr) uintptr{return value}\n")...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.out = append(t.out, '=')
	pointer := t.typeInfo(typeID).kind == cTypePointer
	if pointer {
		t.out = append(t.out, '(')
		t.emitType(typeID)
		t.appendText(")(__c_unsafe.Pointer(")
	} else {
		t.emitType(typeID)
		t.out = append(t.out, '(')
	}
	t.appendText(name)
	t.appendText("(uintptr(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")))")
	if pointer {
		t.out = append(t.out, ')')
		t.usesUnsafe = true
	}
	return t.ok
}

func asmRuntimePointerSection(template []byte) (string, bool) {
	prefix := []byte(".pushsection runtime_ptr_")
	for start := 0; start+len(prefix) <= len(template); start++ {
		match := true
		for i := 0; i < len(prefix); i++ {
			match = match && template[start+i] == prefix[i]
		}
		if !match {
			continue
		}
		end := start + len(prefix)
		for end < len(template) && template[end] != ',' && template[end] != '\n' && template[end] != ' ' && template[end] != '\t' {
			end++
		}
		section := template[start+len(".pushsection ") : end]
		if validAsmMetadataWord(section) {
			return string(section), true
		}
		return "", false
	}
	return "", false
}

func (t *translator) emitAsmRuntimeShift(operation cAsm) bool {
	section, ok := asmRuntimeMetadataSection(operation.template, ".pushsection runtime_shift_")
	if !ok || !t.asmTemplateCompactPrefix(operation.template, "shrl$12,%k0") || len(operation.outputs) != 1 ||
		operation.outputs[0].constraint != "+r" || len(operation.inputs) != 0 {
		return false
	}
	typeID := t.expressionType(operation.outputs[0].expression)
	info := t.typeInfo(typeID)
	if info.kind != cTypeInt && info.kind != cTypeUint || info.size != 4 && info.size != 8 {
		return false
	}
	name := "renvo_runtime_CRuntimeShift_" + cGoIdentifier(section)
	found := false
	for i := 0; i < len(t.asmRuntimePointers); i++ {
		found = found || t.asmRuntimePointers[i] == name
	}
	if !found {
		t.asmRuntimePointers = append(t.asmRuntimePointers, name)
		outer := t.out
		t.out = t.staticOut
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(value ")
		t.emitType(typeID)
		t.appendText(") ")
		t.emitType(typeID)
		t.appendText("{return value}\n")
		t.staticOut = t.out
		t.out = outer
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.appendText(name)
	t.appendText("(")
	t.convertedExpression(typeID, operation.outputs[0].expression)
	t.appendText(")")
	return t.ok
}

func asmRuntimeMetadataSection(template []byte, marker string) (string, bool) {
	prefix := []byte(marker)
	for start := 0; start+len(prefix) <= len(template); start++ {
		match := true
		for i := 0; i < len(prefix); i++ {
			match = match && template[start+i] == prefix[i]
		}
		if !match {
			continue
		}
		end := start + len(prefix)
		for end < len(template) && template[end] != ',' && template[end] != '\n' && template[end] != ' ' && template[end] != '\t' {
			end++
		}
		section := template[start+len(".pushsection ") : end]
		if validAsmMetadataWord(section) {
			return string(section), true
		}
		return "", false
	}
	return "", false
}

func (t *translator) emitAsmGotoCPUFeature(operation cAsm) bool {
	if !operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 3 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 2 ||
		!t.asmTemplateCompactContains(operation.template, "testb%[bitnum],%a[cap_byte]") ||
		!t.asmTemplateCompactContains(operation.template, "jnz%l[t_yes]") ||
		!t.asmTemplateCompactContains(operation.template, "jmp%l[t_no]") {
		return false
	}
	feature, bitnum, capability := -1, -1, -1
	for i := 0; i < len(operation.inputs); i++ {
		switch operation.inputs[i].name {
		case "feature":
			if operation.inputs[i].constraint == "i" {
				feature = i
			}
		case "bitnum":
			if operation.inputs[i].constraint == "i" {
				bitnum = i
			}
		case "cap_byte":
			if operation.inputs[i].constraint == "i" {
				capability = i
			}
		}
	}
	if feature < 0 || bitnum < 0 || capability < 0 ||
		t.typeInfo(t.expressionType(operation.inputs[capability].expression)).kind != cTypePointer {
		return false
	}
	// GCC evaluates every input before entering the asm. The feature operand is
	// consumed only by the alternative metadata, so retain its evaluation even
	// though the portable baseline is the capability-byte test below.
	t.appendText("_=")
	t.emitExpression(operation.inputs[feature].expression)
	t.appendText(";if (uint8(*(")
	t.emitExpression(operation.inputs[capability].expression)
	t.appendText("))&uint8(")
	t.emitExpression(operation.inputs[bitnum].expression)
	t.appendText("))!=0{goto ")
	t.appendText(operation.labels[0])
	t.appendText("};goto ")
	t.appendText(operation.labels[1])
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmGotoUserLoad(operation cAsm) bool {
	if !t.object || !operation.gotoAsm || len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 1 ||
		operation.outputs[0].name != "output" || operation.inputs[0].name != "umem" ||
		operation.inputs[0].constraint != "m" ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\",\"a\"") ||
		!t.asmTemplateCompactContains(operation.template, ".long(%l2)-.") ||
		!t.asmTemplateCompactContains(operation.template, ".long3") {
		return false
	}
	width := 0
	if operation.outputs[0].constraint == "=q" && t.asmTemplateCompactPrefix(operation.template, "1:movb%[umem],%[output]") {
		width = 1
	} else if operation.outputs[0].constraint == "=r" && t.asmTemplateCompactPrefix(operation.template, "1:movw%[umem],%[output]") {
		width = 2
	} else if operation.outputs[0].constraint == "=r" && t.asmTemplateCompactPrefix(operation.template, "1:movl%[umem],%[output]") {
		width = 4
	} else if operation.outputs[0].constraint == "=r" && t.asmTemplateCompactPrefix(operation.template, "1:movq%[umem],%[output]") {
		width = 8
	}
	outputInfo := t.typeInfo(t.expressionType(operation.outputs[0].expression))
	if width == 0 || outputInfo.kind != cTypeInt && outputInfo.kind != cTypeUint || outputInfo.size > 8 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.inputs[0].expression)
	if !ok || t.typeInfo(t.expressionType(pointer)).kind != cTypePointer {
		return false
	}
	label := t.localLabelGoName(operation.labels[0])
	name := "renvo_runtime_CUserLoad" + decimalString(width*8) + "_" + label
	found := false
	for i := 0; i < len(t.asmUserMemoryHelpers); i++ {
		found = found || t.asmUserMemoryHelpers[i] == name
	}
	if !found {
		t.asmUserMemoryHelpers = append(t.asmUserMemoryHelpers, name)
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr) uint64{return 0}\n")...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(")
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText("))));if false{goto ")
	t.appendText(label)
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmGotoUserStore(operation cAsm) bool {
	if !t.object || !operation.gotoAsm || len(operation.outputs) != 0 || len(operation.inputs) != 2 ||
		len(operation.clobbers) != 0 || len(operation.labels) != 1 ||
		operation.inputs[1].constraint != "m" ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\"") ||
		!t.asmTemplateCompactContains(operation.template, ".long(%l2)-.") ||
		!t.asmTemplateCompactContains(operation.template, ".long3") {
		return false
	}
	width := 0
	if t.asmTemplateCompactPrefix(operation.template, "1:movb%0,%1") {
		width = 1
	} else if t.asmTemplateCompactPrefix(operation.template, "1:movw%0,%1") {
		width = 2
	} else if t.asmTemplateCompactPrefix(operation.template, "1:movl%0,%1") {
		width = 4
	} else if t.asmTemplateCompactPrefix(operation.template, "1:movq%0,%1") {
		width = 8
	}
	if width == 0 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.inputs[1].expression)
	if !ok || t.typeInfo(t.expressionType(pointer)).kind != cTypePointer {
		return false
	}
	label := t.localLabelGoName(operation.labels[0])
	name := "renvo_runtime_CUserStore" + decimalString(width*8) + "_" + label
	found := false
	for i := 0; i < len(t.asmUserMemoryHelpers); i++ {
		found = found || t.asmUserMemoryHelpers[i] == name
	}
	if !found {
		t.asmUserMemoryHelpers = append(t.asmUserMemoryHelpers, name)
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,value uint64){}\n")...)
	}
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText(")),uint64(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("));if false{goto ")
	t.appendText(label)
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmInlineGetUser(operation cAsm) bool {
	width := 0
	for index := 0; index < 4; index++ {
		suffix := "b"
		candidateWidth := 1
		if index == 1 {
			suffix = "w"
			candidateWidth = 2
		} else if index == 2 {
			suffix = "l"
			candidateWidth = 4
		} else if index == 3 {
			suffix = "q"
			candidateWidth = 8
		}
		if t.asmTemplateCompactPrefix(operation.template, "1:mov"+suffix+"%[umem],%[output]2:") {
			width = candidateWidth
			break
		}
	}
	if width == 0 || len(operation.outputs) != 2 || len(operation.inputs) != 2 || len(operation.clobbers) != 0 ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\",\"a\"") ||
		!t.asmTemplateCompactContains(operation.template, ".long(1b)-.") ||
		!t.asmTemplateCompactContains(operation.template, ".long(2b)-.") ||
		!t.asmTemplateCompactContains(operation.template, "extable_type_regreg=%[errout],type=(17|((-14)<<16))") {
		return false
	}
	errout, output, memory, tied := -1, -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].name == "errout" && operation.outputs[i].constraint == "=r" {
			errout = i
		} else if operation.outputs[i].name == "output" && operation.outputs[i].constraint == "=a" {
			output = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].name == "umem" && operation.inputs[i].constraint == "m" {
			memory = i
		} else if operation.inputs[i].constraint == "0" {
			tied = i
		}
	}
	if errout < 0 || output < 0 || memory < 0 || tied < 0 ||
		!t.asmExpressionsEqual(operation.outputs[errout].expression, operation.inputs[tied].expression) ||
		t.typeSize(t.expressionType(operation.outputs[errout].expression)) != 4 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.inputs[memory].expression)
	if !ok || t.typeInfo(t.expressionType(pointer)).kind != cTypePointer {
		return false
	}
	bit := width
	name := "renvo_runtime_CGetUser" + decimalString(width)
	if t.asmGetUserOps&bit == 0 {
		t.asmGetUserOps |= bit
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,output uintptr) int32{return 0}\n")...)
	}
	t.emitExpression(operation.outputs[errout].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[errout].expression))
	t.appendText("(")
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText(")),uintptr(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.outputs[output].expression)
	t.appendText(")))));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmExceptionLoad(operation cAsm) bool {
	if !t.object || len(operation.outputs) != 1 || len(operation.inputs) != 1 || len(operation.clobbers) != 0 ||
		operation.outputs[0].constraint != "=r" || operation.inputs[0].constraint != "m" ||
		!t.asmTemplateCompactPrefix(operation.template, "1:mov%[mem],%[ret]2:") ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\",\"a\"") ||
		!t.asmTemplateCompactContains(operation.template, ".long(1b)-.") ||
		!t.asmTemplateCompactContains(operation.template, ".long(2b)-.") ||
		!t.asmTemplateCompactContains(operation.template, ".long20") ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 8 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.inputs[0].expression)
	if !ok || t.typeInfo(t.expressionType(pointer)).kind != cTypePointer {
		return false
	}
	if !t.asmExceptionLoad {
		t.asmExceptionLoad = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CExceptionLoad64(address uintptr) uint64{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(renvo_runtime_CExceptionLoad64(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText("))));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmAlternativeDivider(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "#ALT:oldinstr771:772:") ||
		!t.asmTemplateCompactContains(operation.template, "div%2") || len(operation.outputs) != 0 || len(operation.inputs) != 3 ||
		operation.inputs[0].constraint != "a" || operation.inputs[1].constraint != "d" || operation.inputs[2].constraint != "r" {
		return false
	}
	for i := 0; i < len(operation.inputs); i++ {
		t.appendText("_=")
		t.emitExpression(operation.inputs[i].expression)
		t.appendText(";")
	}
	return t.ok
}

func (t *translator) emitAsmAlternativeStore(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "#ALT:oldinstr") ||
		!t.asmTemplateCompactContains(operation.template, "movl%0,%1") ||
		!t.asmTemplateCompactContains(operation.template, "xchgl%0,%1") ||
		len(operation.outputs) != 2 || operation.outputs[0].constraint != "=r" ||
		operation.outputs[1].constraint != "=m" || len(operation.inputs) != 3 ||
		operation.inputs[0].constraint != "i" || operation.inputs[1].constraint != "0" ||
		operation.inputs[2].constraint != "m" ||
		!t.asmExpressionsEqual(operation.outputs[0].expression, operation.inputs[1].expression) ||
		!t.asmExpressionsEqual(operation.outputs[1].expression, operation.inputs[2].expression) {
		return false
	}
	// Preserve the old-instruction path. The replacement xchg has the same
	// externally visible store; its register result is tied to an otherwise
	// dead input in this Linux idiom. M8 can add the alternatives metadata.
	t.appendText("_=")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(";")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=")
	t.convertedExpression(t.expressionType(operation.outputs[1].expression), operation.inputs[1].expression)
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmBoundsMask(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "cmp%1,%2;sbb%0,%0") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" ||
		len(operation.inputs) != 2 || operation.inputs[0].constraint != "g" ||
		operation.inputs[1].constraint != "r" {
		return false
	}
	if !t.asmBoundsMaskHelper {
		t.asmBoundsMaskHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CBoundsMask(index uint64,size uint64) uint64{if index<size{return ^uint64(0)};return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(renvo_runtime_CBoundsMask(uint64(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("),uint64(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")));")
	return t.ok
}

func (t *translator) emitAsmEndBranchValue(operation cAsm) bool {
	if len(operation.outputs) != 1 || operation.outputs[0].name != "endbr" ||
		operation.outputs[0].constraint != "=&r" || len(operation.inputs) != 0 || len(operation.clobbers) != 0 ||
		!t.asmTemplateCompactPrefix(operation.template, "mov$~0xfa1e0ff3,%[endbr]not%[endbr]") {
		return false
	}
	// Linux constructs the ENDBR64 instruction word this way so the compiler
	// cannot fold it into an instruction stream that objtool may mistake for an
	// actual landing pad. Preserve the resulting scalar value in generated code.
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(0xfa1e0ff3);")
	return t.ok
}

func (t *translator) emitAsmFence(operation cAsm) bool {
	if len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	kind, name := 0, ""
	if t.asmTemplateCompactContains(operation.template, "mfence;lfence") {
		kind, name = 1, "renvo_runtime_CMemoryReadFence"
	} else if t.asmTemplateCompactContains(operation.template, "mfence") {
		kind, name = 2, "renvo_runtime_CMemoryFence"
	} else if t.asmTemplateCompactContains(operation.template, "lock;addl$0,-4(%%rsp)") {
		// Linux uses a locked no-op on the current stack as an x86 full
		// barrier. Keep the semantic fence without retaining the dummy access.
		kind, name = 2, "renvo_runtime_CMemoryFence"
	} else if t.asmTemplateCompactContains(operation.template, "lfence") {
		kind, name = 4, "renvo_runtime_CReadFence"
	} else if t.asmTemplateCompactContains(operation.template, "sfence") {
		kind, name = 8, "renvo_runtime_CWriteFence"
	}
	if kind == 0 {
		return false
	}
	if t.asmFenceOps&kind == 0 {
		t.asmFenceOps |= kind
		t.staticOut = append(t.staticOut, ("func " + name + "(){}\n")...)
	}
	t.appendText(name)
	t.appendText("();")
	return true
}

func (t *translator) emitAsmPrefetch(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "prefetcht0%1") || len(operation.outputs) != 0 ||
		len(operation.inputs) != 2 || operation.inputs[0].constraint != "i" || operation.inputs[1].constraint != "m" {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.inputs[1].expression)
	if !ok {
		return false
	}
	if !t.asmPrefetch {
		t.asmPrefetch = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CPrefetch(address uintptr){}\n"...)
	}
	t.appendText("renvo_runtime_CPrefetch(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText(")));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmSwapGS(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "swapgs") || len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	if t.asmHaltOps&8 == 0 {
		t.asmHaltOps |= 8
		t.staticOut = append(t.staticOut, "func renvo_runtime_CSwapGS(){}\n"...)
	}
	t.appendText("renvo_runtime_CSwapGS();")
	return true
}

func (t *translator) emitAsmScalarStore(operation cAsm) bool {
	width, positional, fixedAccumulator := 0, false, false
	zeroWidth := 0
	for index, suffix := range []string{"b", "w", "l", "q"} {
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"$0,%0") {
			zeroWidth = []int{1, 2, 4, 8}[index]
			break
		}
	}
	if zeroWidth != 0 {
		if len(operation.outputs) != 1 || operation.outputs[0].constraint != "+m" || len(operation.inputs) != 0 ||
			t.typeSize(t.expressionType(operation.outputs[0].expression)) != zeroWidth {
			return false
		}
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=0;")
		return t.ok
	}
	for index, suffix := range []string{"b", "w", "l", "q"} {
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"%[val],%[var]") {
			width = []int{1, 2, 4, 8}[index]
			break
		}
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"%0,%1") {
			width, positional = []int{1, 2, 4, 8}[index], true
			break
		}
		accumulator := []string{"%%al", "%%ax", "%%eax", "%%rax"}[index]
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+accumulator+",(%1)") {
			width, fixedAccumulator = []int{1, 2, 4, 8}[index], true
			break
		}
	}
	if fixedAccumulator {
		if len(operation.outputs) != 0 || len(operation.inputs) != 2 || operation.inputs[0].constraint != "a" ||
			operation.inputs[1].constraint != "r" || t.typeSize(t.expressionType(operation.inputs[0].expression)) != width ||
			t.typeSize(t.expressionType(operation.inputs[1].expression)) != t.pointerSize {
			return false
		}
		t.appendText("*(*")
		t.emitType(t.expressionType(operation.inputs[0].expression))
		t.appendText(")(__c_unsafe.Pointer(")
		t.emitExpression(operation.inputs[1].expression)
		t.appendText("))=")
		t.convertedExpression(t.expressionType(operation.inputs[0].expression), operation.inputs[0].expression)
		t.appendText(";")
		t.usesUnsafe = true
		return t.ok
	}
	if positional {
		if len(operation.outputs) != 0 || len(operation.inputs) != 2 ||
			(operation.inputs[0].constraint != "q" && operation.inputs[0].constraint != "r") ||
			operation.inputs[1].constraint != "m" ||
			t.typeSize(t.expressionType(operation.inputs[0].expression)) != width ||
			t.typeSize(t.expressionType(operation.inputs[1].expression)) != width {
			return false
		}
		t.emitExpression(operation.inputs[1].expression)
		t.appendText("=")
		t.convertedExpression(t.expressionType(operation.inputs[1].expression), operation.inputs[0].expression)
		t.appendText(";")
		return t.ok
	}
	if width == 0 || len(operation.outputs) != 1 || operation.outputs[0].name != "var" ||
		operation.outputs[0].constraint != "=m" || len(operation.inputs) != 1 || operation.inputs[0].name != "val" ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != width {
		return false
	}
	constraint := operation.inputs[0].constraint
	if constraint != "qi" && constraint != "ri" && constraint != "re" {
		return false
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.convertedExpression(t.expressionType(operation.outputs[0].expression), operation.inputs[0].expression)
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmEmptyReadWrite(operation cAsm) bool {
	if len(operation.template) != 0 || len(operation.outputs) == 0 || len(operation.inputs) != 0 {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		constraint := operation.outputs[i].constraint
		if len(constraint) < 2 || constraint[0] != '+' {
			return false
		}
		t.appendText("_=")
		t.emitExpression(operation.outputs[i].expression)
		t.appendText(";")
	}
	return t.ok
}

func (t *translator) emitAsmTileRelease(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, ".byte0xc4,0xe2,0x78,0x49,0xc0") ||
		len(operation.outputs) != 0 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 {
		return false
	}
	if t.asmDirectOps&8 == 0 {
		t.asmDirectOps |= 8
		t.staticOut = append(t.staticOut, "func renvo_runtime_CTileRelease(){}\n"...)
	}
	t.appendText("renvo_runtime_CTileRelease();")
	return true
}

func (t *translator) emitAsmUndefinedInstruction(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, ".byte0x0f,0x0b") ||
		len(operation.outputs) != 0 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 {
		return false
	}
	t.ensureUndefinedInstructionHelper()
	t.appendText("renvo_runtime_CUndefinedInstruction();")
	return true
}

func (t *translator) emitAsmInt3Selftest(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "int3_selftest_ip:") ||
		!t.asmTemplateCompactContains(operation.template, "int3;nop;nop;nop;nop") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "+r" ||
		len(operation.inputs) != 1 || operation.inputs[0].constraint != "D" {
		return false
	}
	pointerType := t.expressionType(operation.inputs[0].expression)
	pointer := t.typeInfo(pointerType)
	if pointer.kind != cTypePointer || t.typeSize(pointer.base) != 4 {
		return false
	}
	if !t.asmInt3Selftest {
		t.asmInt3Selftest = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CInt3Selftest(value *uint32){}\n"...)
	}
	t.appendText("renvo_runtime_CInt3Selftest(")
	t.convertedExpression(t.pointerType(cTypeUint32ID), operation.inputs[0].expression)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmIRETToSelf(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "mov%%ss,%0") ||
		!t.asmTemplateCompactContains(operation.template, "pushq%q0") ||
		!t.asmTemplateCompactContains(operation.template, "pushfq") ||
		!t.asmTemplateCompactContains(operation.template, "mov%%cs,%0") ||
		!t.asmTemplateCompactContains(operation.template, "iretq") ||
		len(operation.outputs) != 2 || operation.outputs[0].constraint != "=&r" ||
		operation.outputs[1].constraint != "+r" || len(operation.inputs) != 0 {
		return false
	}
	if t.asmDirectOps&32 == 0 {
		t.asmDirectOps |= 32
		t.staticOut = append(t.staticOut, "func renvo_runtime_CIRETToSelf(){}\n"...)
	}
	// The tied stack-register operand is unchanged by a successful same-ring
	// IRET, but GCC still evaluates its input side before entering the asm.
	t.appendText("_=")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(";renvo_runtime_CIRETToSelf();")
	return t.ok
}

func (t *translator) emitAsmSerialize(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, ".byte0xf,0x1,0xe8") || len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	if t.asmDirectOps&1 == 0 {
		t.asmDirectOps |= 1
		t.staticOut = append(t.staticOut, "func renvo_runtime_CSerialize(){}\n"...)
	}
	t.appendText("renvo_runtime_CSerialize();")
	return true
}

func (t *translator) emitAsmHypercall(operation cAsm) bool {
	vmcall := t.asmTemplateCompactContains(operation.template, "vmcall")
	vmmcall := t.asmTemplateCompactContains(operation.template, "vmmcall")
	if !vmcall && !vmmcall || len(operation.outputs) != 1 || operation.outputs[0].constraint != "=a" ||
		len(operation.inputs) < 1 || len(operation.inputs) > 5 {
		return false
	}
	constraints := "abcdS"
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint != constraints[i:i+1] || t.typeSize(t.expressionType(operation.inputs[i].expression)) > 8 {
			return false
		}
	}
	// The kernel alternative starts with VMCALL and patches VMMCALL on AMD.
	// Retain that baseline choice when both encodings occur in the template;
	// an explicit VMMCALL-only SEV hypercall remains VMMCALL.
	name := "renvo_runtime_CVMCall"
	keyShift := 8 + len(operation.inputs)
	if vmmcall && !vmcall {
		name = "renvo_runtime_CVMMCall"
		keyShift = 16 + len(operation.inputs)
	}
	name += decimalString(len(operation.inputs))
	key := 1 << keyShift
	if t.asmDirectOps&key == 0 {
		t.asmDirectOps |= key
		declaration := "func " + name + "("
		for i := 0; i < len(operation.inputs); i++ {
			if i > 0 {
				declaration += ","
			}
			declaration += "p" + decimalString(i) + " uint64"
		}
		declaration += ") int64{return 0}\n"
		t.staticOut = append(t.staticOut, declaration...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.appendText(name)
	t.appendText("(")
	for i := 0; i < len(operation.inputs); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.appendText("uint64(")
		t.emitExpression(operation.inputs[i].expression)
		t.out = append(t.out, ')')
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmFixedRegisterMove(operation cAsm) bool {
	if len(operation.outputs) != 0 || len(operation.inputs) != 1 || operation.inputs[0].name == "" ||
		operation.inputs[0].constraint != "g" || t.typeSize(t.expressionType(operation.inputs[0].expression)) != 8 {
		return false
	}
	template := "mov%[" + operation.inputs[0].name + "],%%r9"
	if !t.asmTemplateCompactPrefix(operation.template, template) {
		return false
	}
	if t.asmDirectOps&64 == 0 {
		t.asmDirectOps |= 64
		t.staticOut = append(t.staticOut, "func renvo_runtime_CLoadR9(value uint64){}\n"...)
	}
	t.appendText("renvo_runtime_CLoadR9(uint64(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmLoadGS(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, ".byte0xf2,0x0f,0x00,0xf7") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "+D" || len(operation.inputs) != 0 {
		return false
	}
	if !t.asmLoadGS {
		t.asmLoadGS = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CLoadGS(selector uint32){}\n"...)
	}
	t.appendText("renvo_runtime_CLoadGS(uint32(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmDirectMove(operation cAsm) bool {
	enqueue := t.asmTemplateCompactPrefix(operation.template, ".byte0xf3,0x0f,0x38,0xf8,0x02")
	move := t.asmTemplateCompactPrefix(operation.template, ".byte0x66,0x0f,0x38,0xf8,0x02")
	if !enqueue && !move {
		return false
	}
	destination, source, condition := -1, -1, -1
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint == "a" {
			destination = i
		} else if operation.inputs[i].constraint == "d" {
			source = i
		}
	}
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].constraint == "=@ccz" {
			condition = i
		}
	}
	if destination < 0 || source < 0 || enqueue && condition < 0 {
		return false
	}
	name, bit := "renvo_runtime_CDirectMove64", 2
	if enqueue {
		name, bit = "renvo_runtime_CEnqueueCommand", 4
	}
	if t.asmDirectOps&bit == 0 {
		t.asmDirectOps |= bit
		declaration := "func " + name + "(destination uintptr,source uintptr){}\n"
		if enqueue {
			declaration = "func " + name + "(destination uintptr,source uintptr) uint8{return 0}\n"
		}
		t.staticOut = append(t.staticOut, declaration...)
	}
	if enqueue {
		t.emitExpression(operation.outputs[condition].expression)
		t.appendText("=")
	}
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[destination].expression)
	t.appendText(")),uintptr(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[source].expression)
	t.appendText(")));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmCacheFlush(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "clflush") {
		return false
	}
	var pointer []token
	if len(operation.outputs) == 1 && operation.outputs[0].constraint == "+m" {
		var ok bool
		pointer, ok = t.asmDereferencePointer(operation.outputs[0].expression)
		if !ok {
			return false
		}
		for i := 0; i < len(operation.inputs); i++ {
			if _, constant := t.constantExpression(operation.inputs[i].expression); !constant &&
				!t.asmExpressionsEqual(operation.inputs[i].expression, pointer) {
				return false
			}
		}
	} else if len(operation.outputs) == 0 {
		for i := 0; i < len(operation.inputs); i++ {
			input := operation.inputs[i]
			if (input.name == "addr" || input.constraint != "i") && t.typeInfo(t.expressionType(input.expression)).kind == cTypePointer {
				if pointer != nil {
					return false
				}
				pointer = input.expression
				continue
			}
			if _, ok := t.constantExpression(input.expression); !ok {
				return false
			}
		}
		if pointer == nil {
			return false
		}
	} else {
		return false
	}
	if !t.asmCacheFlush {
		t.asmCacheFlush = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCacheLineFlush(address uintptr){}\n"...)
	}
	t.appendText("renvo_runtime_CCacheLineFlush(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText(")));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmWriteBackInvalidate(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "wbinvd") || len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	if t.asmHaltOps&4 == 0 {
		t.asmHaltOps |= 4
		t.staticOut = append(t.staticOut, "func renvo_runtime_CWriteBackInvalidateCache(){}\n"...)
	}
	t.appendText("renvo_runtime_CWriteBackInvalidateCache();")
	return true
}

func (t *translator) emitAsmReadFlags(operation cAsm) bool {
	legacy := t.asmTemplateCompactContains(operation.template, "pushfl;popl%0")
	if (!t.asmTemplateCompactContains(operation.template, "pushf;pop%0") && !legacy) || len(operation.outputs) != 1 ||
		(operation.outputs[0].constraint != "=rm" && (!legacy || operation.outputs[0].constraint != "=r")) || len(operation.inputs) != 0 {
		return false
	}
	typeID := t.expressionType(operation.outputs[0].expression)
	if t.typeSize(typeID) != t.pointerSize {
		return false
	}
	t.ensureAsmFlagsHelper(false, typeID)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CReadFlags();")
	if t.asmTemplateCompactContains(operation.template, ".byte0x0f,0x01,0xca") {
		t.ensureAsmUserAccessHelper(false)
		t.appendText("renvo_runtime_CDisableUserAccess();")
	}
	return t.ok
}

func (t *translator) emitAsmWriteFlags(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "push%0;popf") || len(operation.outputs) != 0 ||
		len(operation.inputs) != 1 || operation.inputs[0].constraint != "g" {
		return false
	}
	typeID := t.expressionType(operation.inputs[0].expression)
	if t.typeSize(typeID) != t.pointerSize {
		return false
	}
	t.ensureAsmFlagsHelper(true, typeID)
	t.appendText("renvo_runtime_CWriteFlags(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmUserAccess(operation cAsm) bool {
	enable := t.asmTemplateCompactContains(operation.template, ".byte0x0f,0x01,0xcb")
	disable := t.asmTemplateCompactContains(operation.template, ".byte0x0f,0x01,0xca")
	if enable == disable || len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	t.ensureAsmUserAccessHelper(enable)
	if enable {
		t.appendText("renvo_runtime_CEnableUserAccess();")
	} else {
		t.appendText("renvo_runtime_CDisableUserAccess();")
	}
	return true
}

func (t *translator) ensureAsmUserAccessHelper(enable bool) {
	bit, name := 64, "renvo_runtime_CDisableUserAccess"
	if enable {
		bit, name = 128, "renvo_runtime_CEnableUserAccess"
	}
	if t.asmDirectOps&bit != 0 {
		return
	}
	t.asmDirectOps |= bit
	t.staticOut = append(t.staticOut, ("func " + name + "(){}\n")...)
}

func (t *translator) emitAsmHalt(operation cAsm) bool {
	kind, name := 0, ""
	if t.asmTemplateCompactPrefix(operation.template, "sti;hlt") {
		kind, name = 2, "renvo_runtime_CEnableInterruptsAndHalt"
	} else if t.asmTemplateCompactPrefix(operation.template, "hlt") {
		kind, name = 1, "renvo_runtime_CHalt"
	}
	if kind == 0 || len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	if t.asmHaltOps&kind == 0 {
		t.asmHaltOps |= kind
		t.staticOut = append(t.staticOut, ("func " + name + "(){}\n")...)
	}
	t.appendText(name)
	t.appendText("();")
	return true
}

func (t *translator) emitAsmVerifySegment(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "verw%[ds]") || len(operation.outputs) != 0 ||
		len(operation.inputs) != 1 || operation.inputs[0].name != "ds" || operation.inputs[0].constraint != "m" ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 2 {
		return false
	}
	if !t.asmVerifySegment {
		t.asmVerifySegment = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CVerifySegment(address uintptr){}\n"...)
	}
	t.appendText("renvo_runtime_CVerifySegment(uintptr(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("))));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmInvalidateProcessContext(operation cAsm) bool {
	if t.asmTemplateCompactPrefix(operation.template, "invlpg(%0)") && len(operation.outputs) == 0 &&
		len(operation.inputs) == 1 && operation.inputs[0].constraint == "r" &&
		t.typeSize(t.expressionType(operation.inputs[0].expression)) == t.pointerSize {
		if !t.asmInvalidatePage {
			t.asmInvalidatePage = true
			t.staticOut = append(t.staticOut, "func renvo_runtime_CInvalidatePage(address uintptr){}\n"...)
		}
		t.appendText("renvo_runtime_CInvalidatePage(uintptr(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
		return t.ok
	}
	if !t.asmTemplateCompactPrefix(operation.template, "invpcid%[desc],%[type]") || len(operation.outputs) != 0 ||
		len(operation.inputs) != 2 || operation.inputs[0].name != "desc" || operation.inputs[0].constraint != "m" ||
		operation.inputs[1].name != "type" || operation.inputs[1].constraint != "r" {
		return false
	}
	if !t.asmInvalidateHelper {
		t.asmInvalidateHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CInvalidateProcessContext(address uintptr,kind uintptr){}\n"...)
	}
	t.appendText("renvo_runtime_CInvalidateProcessContext(uintptr(&(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")),uintptr(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmScalarLoad(operation cAsm) bool {
	width, positional, fixedAccumulator := 0, false, false
	for index, suffix := range []string{"b", "w", "l", "q"} {
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"%[var],%[val]") ||
			t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"%a[var],%[val]") {
			width = []int{1, 2, 4, 8}[index]
			break
		}
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"%1,%0") {
			width, positional = []int{1, 2, 4, 8}[index], true
			break
		}
		accumulator := []string{"%%al", "%%ax", "%%eax", "%%rax"}[index]
		if t.asmTemplateCompactPrefix(operation.template, "mov"+suffix+"(%1),"+accumulator) {
			width, fixedAccumulator = []int{1, 2, 4, 8}[index], true
			break
		}
	}
	if fixedAccumulator {
		if len(operation.outputs) != 1 || operation.outputs[0].constraint != "=a" || len(operation.inputs) != 1 ||
			operation.inputs[0].constraint != "r" || t.typeSize(t.expressionType(operation.outputs[0].expression)) != width ||
			t.typeSize(t.expressionType(operation.inputs[0].expression)) != t.pointerSize {
			return false
		}
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=*(*")
		t.emitType(t.expressionType(operation.outputs[0].expression))
		t.appendText(")(__c_unsafe.Pointer(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
		t.usesUnsafe = true
		return t.ok
	}
	if width == 0 || len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
		(!positional && operation.outputs[0].name != "val") || (operation.outputs[0].constraint != "=q" && operation.outputs[0].constraint != "=r") ||
		(!positional && operation.inputs[0].name != "var") || (operation.inputs[0].constraint != "m" && operation.inputs[0].constraint != "i") ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != width {
		return false
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	if operation.inputs[0].constraint == "i" {
		t.appendText("*(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(")")
	} else {
		t.convertedExpression(t.expressionType(operation.outputs[0].expression), operation.inputs[0].expression)
	}
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmEmptyOperands(operation cAsm) bool {
	if len(operation.template) != 0 || len(operation.outputs) != 0 || len(operation.inputs) == 0 {
		return false
	}
	for i := 0; i < len(operation.inputs); i++ {
		t.appendText("_=")
		t.emitExpression(operation.inputs[i].expression)
		t.appendText(";")
	}
	return t.ok
}

func (t *translator) emitAsmNonTemporalStore(operation cAsm) bool {
	if t.emitAsmNonTemporalCopy(operation) {
		return true
	}
	width := 0
	if t.asmTemplateCompactPrefix(operation.template, "movntil%1,%0") {
		width = 4
	} else if t.asmTemplateCompactPrefix(operation.template, "movntiq%1,%0") {
		width = 8
	}
	if width == 0 || len(operation.outputs) != 1 || operation.outputs[0].constraint != "=m" ||
		len(operation.inputs) != 1 || operation.inputs[0].constraint != "r" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != width ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != width {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.outputs[0].expression)
	if !ok {
		return false
	}
	name := "renvo_runtime_CNonTemporalStore" + decimalString(width*8)
	if t.asmNonTemporalStore&(1<<width) == 0 {
		t.asmNonTemporalStore |= 1 << width
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,value uint64){}\n")...)
	}
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText(")),uint64(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmNonTemporalCopy(operation cAsm) bool {
	storeWidth, count := 0, 0
	if t.asmTemplateCompactPrefix(operation.template, "movq(%0),%%r8") &&
		t.asmTemplateCompactContains(operation.template, "movnti%%r11,24(%1)") {
		storeWidth, count = 8, 4
	} else if t.asmTemplateCompactPrefix(operation.template, "movq(%0),%%r8") &&
		t.asmTemplateCompactContains(operation.template, "movnti%%r8,(%1)") {
		storeWidth, count = 8, 1
	} else if t.asmTemplateCompactPrefix(operation.template, "movl(%0),%%r8d") &&
		t.asmTemplateCompactContains(operation.template, "movnti%%r8d,(%1)") {
		storeWidth, count = 4, 1
	}
	if storeWidth == 0 || len(operation.outputs) != 0 || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "r" || operation.inputs[1].constraint != "r" ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 8 ||
		t.typeSize(t.expressionType(operation.inputs[1].expression)) != 8 {
		return false
	}
	name := "renvo_runtime_CNonTemporalStore" + decimalString(storeWidth*8)
	if t.asmNonTemporalStore&(1<<storeWidth) == 0 {
		t.asmNonTemporalStore |= 1 << storeWidth
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,value uint64){}\n")...)
	}
	for i := 0; i < count; i++ {
		offset := i * storeWidth
		t.appendText(name)
		t.appendText("(uintptr(")
		t.emitExpression(operation.inputs[1].expression)
		if offset != 0 {
			t.appendText(")+")
			t.appendDecimal(offset)
		} else {
			t.appendText(")")
		}
		t.appendText(",uint64(*(*uint")
		t.appendDecimal(storeWidth * 8)
		t.appendText(")(__c_unsafe.Pointer(uintptr(")
		t.emitExpression(operation.inputs[0].expression)
		if offset != 0 {
			t.appendText(")+")
			t.appendDecimal(offset)
		} else {
			t.appendText(")")
		}
		t.appendText("))));")
	}
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmRepeatedStore(operation cAsm) bool {
	width := 0
	if t.asmTemplateCompactPrefix(operation.template, "rep;stosb") || t.asmTemplateCompactPrefix(operation.template, "repstosb") ||
		t.asmTemplateCompactPrefix(operation.template, "cld;repstosb") {
		width = 1
	} else if t.asmTemplateCompactPrefix(operation.template, "rep;stosw") || t.asmTemplateCompactPrefix(operation.template, "repstosw") ||
		t.asmTemplateCompactPrefix(operation.template, "cld;repstosw") {
		width = 2
	} else if t.asmTemplateCompactPrefix(operation.template, "rep;stosl") || t.asmTemplateCompactPrefix(operation.template, "repstosl") ||
		t.asmTemplateCompactPrefix(operation.template, "cld;repstosl") {
		width = 4
	} else if t.asmTemplateCompactPrefix(operation.template, "rep;stosq") || t.asmTemplateCompactPrefix(operation.template, "repstosq") ||
		t.asmTemplateCompactPrefix(operation.template, "cld;repstosq") {
		width = 8
	}
	modern := len(operation.outputs) == 2 && operation.outputs[0].constraint == "+D" &&
		operation.outputs[1].constraint == "+c" && len(operation.inputs) == 1 && operation.inputs[0].constraint == "a"
	legacy := len(operation.outputs) == 2 && operation.outputs[0].constraint == "=D" &&
		operation.outputs[1].constraint == "=c" && len(operation.inputs) == 3 &&
		operation.inputs[0].constraint == "0" && operation.inputs[1].constraint == "1" &&
		operation.inputs[2].constraint == "a"
	if width == 0 || !modern && !legacy {
		return false
	}
	pointerType := t.expressionType(operation.outputs[0].expression)
	valueInput := 0
	if legacy {
		valueInput = 2
	}
	if t.typeInfo(pointerType).kind != cTypePointer ||
		modern && t.typeSize(t.typeInfo(pointerType).base) != width ||
		t.typeSize(t.expressionType(operation.inputs[valueInput].expression)) < width {
		return false
	}
	if legacy {
		t.appendText("for ")
		t.emitExpression(operation.outputs[1].expression)
		t.appendText(">0{*(*uint")
		t.appendDecimal(width * 8)
		t.appendText(")(__c_unsafe.Pointer(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("))=uint")
		t.appendDecimal(width * 8)
		t.appendText("(")
		t.emitExpression(operation.inputs[valueInput].expression)
		t.appendText(");")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=(")
		t.emitType(pointerType)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("))+")
		t.appendDecimal(width)
		t.appendText("));")
		t.emitExpression(operation.outputs[1].expression)
		t.appendText("--};")
		t.usesUnsafe = true
		return t.ok
	}
	helper := t.ensurePointerStepHelper(pointerType, false)
	t.appendText("for ")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(">0{*(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(")=")
	t.convertedExpression(t.typeInfo(pointerType).base, operation.inputs[valueInput].expression)
	t.appendText(";")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.appendText(helper)
	t.appendText("(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(",1);")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("--};")
	return t.ok
}

func (t *translator) emitAsmUserClear(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "repstosb") ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\"") ||
		len(operation.outputs) != 3 || len(operation.inputs) != 1 ||
		operation.outputs[0].constraint != "+c" || operation.outputs[1].constraint != "+D" ||
		operation.outputs[2].constraint != "+r" || operation.inputs[0].constraint != "a" {
		return false
	}
	zero, ok := t.constantExpression(operation.inputs[0].expression)
	pointerType := t.expressionType(operation.outputs[1].expression)
	if !ok || zero != 0 || t.typeInfo(pointerType).kind != cTypePointer {
		return false
	}
	if !t.asmUserClearHelper {
		t.asmUserClearHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CClearUserBytes___ex_table(dest *byte,count uintptr) uintptr{return count}\n"...)
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_clear_count_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(");__c_clear_dest_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(";__c_clear_remaining_")
	t.appendText(suffix)
	t.appendText(":=renvo_runtime_CClearUserBytes___ex_table((*byte)(__c_unsafe.Pointer(__c_clear_dest_")
	t.appendText(suffix)
	t.appendText(")),__c_clear_count_")
	t.appendText(suffix)
	t.appendText(");__c_clear_done_")
	t.appendText(suffix)
	t.appendText(":=__c_clear_count_")
	t.appendText(suffix)
	t.appendText("-__c_clear_remaining_")
	t.appendText(suffix)
	t.appendText(";")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(__c_clear_remaining_")
	t.appendText(suffix)
	t.appendText(");")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=(")
	t.emitType(pointerType)
	t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(__c_clear_dest_")
	t.appendText(suffix)
	t.appendText("))+__c_clear_done_")
	t.appendText(suffix)
	t.appendText("));_=")
	t.emitExpression(operation.outputs[2].expression)
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmGetUser(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "call__get_user_%c[size]") &&
		!t.asmTemplateCompactPrefix(operation.template, "call__get_user_nocheck_%c[size]") ||
		len(operation.outputs) != 3 || len(operation.inputs) != 2 || len(operation.clobbers) != 0 {
		return false
	}
	result, value, stack, address, sizeInput := -1, -1, -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		switch operation.outputs[i].constraint {
		case "=a":
			result = i
		case "=r":
			value = i
		case "+r":
			stack = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint == "0" {
			address = i
		} else if operation.inputs[i].name == "size" && operation.inputs[i].constraint == "i" {
			sizeInput = i
		}
	}
	if result < 0 || value < 0 || stack < 0 || address < 0 || sizeInput < 0 {
		return false
	}
	width, constant := t.constantExpression(operation.inputs[sizeInput].expression)
	bit := 0
	switch width {
	case 1:
		bit = 1
	case 2:
		bit = 2
	case 4:
		bit = 4
	case 8:
		bit = 8
	default:
		return false
	}
	if !constant {
		return false
	}
	name := "renvo_runtime_CGetUser" + decimalString(width)
	if t.asmGetUserOps&bit == 0 {
		t.asmGetUserOps |= bit
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,output uintptr) int32{return 0}\n")...)
	}
	t.emitExpression(operation.outputs[result].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[result].expression))
	t.appendText("(")
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[address].expression)
	t.appendText(")),uintptr(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.outputs[value].expression)
	t.appendText(")))));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmPutUser(operation cAsm) bool {
	checked := t.asmTemplateCompactPrefix(operation.template, "call__put_user_%c[size]")
	nocheck := t.asmTemplateCompactPrefix(operation.template, "call__put_user_nocheck_%c[size]")
	if !checked && !nocheck || len(operation.outputs) != 2 || len(operation.inputs) != 3 {
		return false
	}
	result, stack, address, valueInput, sizeInput := -1, -1, -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].constraint == "=c" {
			result = i
		} else if operation.outputs[i].constraint == "+r" {
			stack = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint == "0" {
			address = i
		} else if operation.inputs[i].name == "size" && operation.inputs[i].constraint == "i" {
			sizeInput = i
		} else if operation.inputs[i].constraint == "r" {
			valueInput = i
		}
	}
	if result < 0 || stack < 0 || address < 0 || valueInput < 0 || sizeInput < 0 ||
		t.typeSize(t.expressionType(operation.inputs[address].expression)) != t.pointerSize {
		return false
	}
	width, constant := t.constantExpression(operation.inputs[sizeInput].expression)
	if !constant || width != 1 && width != 2 && width != 4 && width != 8 ||
		t.typeSize(t.expressionType(operation.inputs[valueInput].expression)) != width {
		return false
	}
	bit := width
	name := "renvo_runtime_CPutUser" + decimalString(width)
	if nocheck {
		bit <<= 4
		name = "renvo_runtime_CPutUserNoCheck" + decimalString(width)
	}
	if t.asmPutUserOps&bit == 0 {
		t.asmPutUserOps |= bit
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,value uint64) int32{return 0}\n")...)
	}
	t.emitExpression(operation.outputs[result].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[result].expression))
	t.appendText("(")
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[address].expression)
	t.appendText(")),uint64(")
	t.emitExpression(operation.inputs[valueInput].expression)
	t.appendText(")));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmAtomicArithmetic(operation cAsm) bool {
	op, name, width := 0, "", 0
	for candidate, operationName := range []string{"", "Add", "Sub", "And", "Or", "Xor", "Inc", "Dec"} {
		if candidate == 0 {
			continue
		}
		mnemonic := []string{"", "add", "sub", "and", "or", "xor", "inc", "dec"}[candidate]
		for suffixIndex, suffix := range []string{"b", "w", "l", "q"} {
			if t.asmTemplateCompactPrefix(operation.template, mnemonic+suffix) {
				op, name, width = candidate, operationName, []int{1, 2, 4, 8}[suffixIndex]
				break
			}
		}
		if op != 0 {
			break
		}
	}
	if op == 0 {
		return false
	}
	memory, condition, value := -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].constraint == "+m" || operation.outputs[i].constraint == "=m" {
			memory = i
		} else if operation.outputs[i].constraint == "=@cce" || operation.outputs[i].constraint == "=@ccz" ||
			operation.outputs[i].constraint == "=@ccs" {
			condition = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		constraint := operation.inputs[i].constraint
		if constraint == "ir" || constraint == "ri" || constraint == "er" || constraint == "re" || constraint == "qi" {
			value = i
		}
	}
	if memory < 0 || op <= 5 && value < 0 || op >= 6 && value >= 0 {
		return false
	}
	helper := "renvo_runtime_CAtomic" + name + decimalString(width*8)
	widthKey := []int{0, 0, 1, 0, 2, 0, 0, 0, 3}[width]
	key := op*4 + widthKey
	if t.asmAtomicArithmetic&(1<<key) == 0 {
		t.asmAtomicArithmetic |= 1 << key
		valueType := "uint64"
		if t.pointerSize == 4 && width <= 4 {
			valueType = "uintptr"
		}
		t.staticOut = append(t.staticOut, ("func " + helper + "(address uintptr,value " + valueType + ") uint8{return 0}\n")...)
	}
	if condition >= 0 {
		t.emitExpression(operation.outputs[condition].expression)
		t.appendText("=uint8((")
	}
	t.appendText(helper)
	t.appendText("(uintptr(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.outputs[memory].expression)
	t.appendText("))),")
	if t.pointerSize == 4 && width <= 4 {
		t.appendText("uintptr(")
	} else {
		t.appendText("uint64(")
	}
	if value >= 0 {
		t.emitExpression(operation.inputs[value].expression)
	} else {
		t.appendText("1")
	}
	t.appendText("))")
	if condition >= 0 {
		bit := 0
		if operation.outputs[condition].constraint == "=@ccs" {
			bit = 1
		}
		t.appendText(">>")
		t.appendDecimal(bit)
		t.appendText(")&1)")
	}
	t.appendText(";")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmAtomicExchange(operation cAsm) bool {
	legacyXchg := t.asmTemplateCompactContains(operation.template, "xchgl%0,%1") &&
		len(operation.outputs) == 2 && len(operation.inputs) == 1 &&
		operation.outputs[0].constraint == "+m" && operation.outputs[1].constraint == "=a" &&
		operation.inputs[0].constraint == "1"
	if legacyXchg {
		const key = 6
		if t.asmAtomicExchange&(1<<key) == 0 {
			t.asmAtomicExchange |= 1 << key
			if t.pointerSize == 4 {
				t.staticOut = append(t.staticOut, "func renvo_runtime_CAtomicExchange32(address uintptr,value uintptr) uintptr{return 0}\n"...)
			} else {
				t.staticOut = append(t.staticOut, "func renvo_runtime_CAtomicExchange32(address uintptr,value uint64) uint64{return 0}\n"...)
			}
		}
		outputType := t.expressionType(operation.outputs[1].expression)
		t.emitExpression(operation.outputs[1].expression)
		t.appendText("=")
		t.emitType(outputType)
		t.appendText("(renvo_runtime_CAtomicExchange32(uintptr(__c_unsafe.Pointer(&(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("))),")
		if t.pointerSize == 4 {
			t.appendText("uintptr(")
		} else {
			t.appendText("uint64(")
		}
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(")));")
		t.usesUnsafe = true
		return t.ok
	}
	kind, width := 0, 0
	for index, suffix := range []string{"b", "w", "l", "q"} {
		if t.asmTemplateCompactPrefix(operation.template, "xadd"+suffix) {
			kind, width = 1, []int{1, 2, 4, 8}[index]
			break
		}
		if t.asmTemplateCompactPrefix(operation.template, "xchg"+suffix) {
			kind, width = 2, []int{1, 2, 4, 8}[index]
			break
		}
	}
	if kind == 0 || len(operation.outputs) != 2 || len(operation.inputs) != 0 ||
		(operation.outputs[0].constraint != "+q" && operation.outputs[0].constraint != "+r") ||
		operation.outputs[1].constraint != "+m" {
		return false
	}
	// Linux deliberately spells all four widths in a sizeof switch. GCC still
	// accepts the non-selected alternatives even though their C operand type has
	// a different width; the instruction suffix, not that type, controls the asm.
	if outputSize, memorySize := t.typeSize(t.expressionType(operation.outputs[0].expression)), t.typeSize(t.expressionType(operation.outputs[1].expression)); outputSize == 0 || outputSize > 8 || memorySize == 0 || memorySize > 8 {
		return false
	}
	name := "renvo_runtime_CAtomicFetchAdd"
	if kind == 2 {
		name = "renvo_runtime_CAtomicExchange"
	}
	name += decimalString(width * 8)
	widthKey := []int{0, 0, 1, 0, 2, 0, 0, 0, 3}[width]
	key := (kind-1)*4 + widthKey
	if t.asmAtomicExchange&(1<<key) == 0 {
		t.asmAtomicExchange |= 1 << key
		valueType := "uint64"
		if t.pointerSize == 4 && width <= 4 {
			valueType = "uintptr"
		}
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,value " + valueType + ") " + valueType + "{return 0}\n")...)
	}
	outputType := t.expressionType(operation.outputs[0].expression)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	if t.typeInfo(outputType).kind == cTypePointer {
		t.out = append(t.out, '(')
		t.emitType(outputType)
		t.appendText(")(__c_unsafe.Pointer(uintptr(")
	} else {
		t.emitType(outputType)
		t.out = append(t.out, '(')
	}
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("))),")
	if t.pointerSize == 4 && width <= 4 {
		t.appendText("uintptr(")
	} else {
		t.appendText("uint64(")
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(")))")
	if t.typeInfo(outputType).kind == cTypePointer {
		t.appendText("))")
	}
	t.appendText(";")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmCompareExchange(operation cAsm) bool {
	width := 0
	for index, suffix := range []string{"b", "w", "l", "q"} {
		if t.asmTemplateCompactPrefix(operation.template, "cmpxchg"+suffix) {
			width = []int{1, 2, 4, 8}[index]
			break
		}
	}
	if width == 0 {
		return false
	}
	condition, memory, observed, replacement, expected := -1, -1, -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		switch operation.outputs[i].constraint {
		case "=@ccz", "=@cce":
			condition = i
		case "+m":
			memory = i
		case "+a", "=a":
			observed = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint == "0" {
			expected = i
		} else if operation.inputs[i].constraint == "q" || operation.inputs[i].constraint == "r" {
			replacement = i
		}
	}
	if memory < 0 || observed < 0 || replacement < 0 || condition < 0 && expected < 0 {
		return false
	}
	if memorySize := t.typeSize(t.expressionType(operation.outputs[memory].expression)); memorySize == 0 || memorySize > 8 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.outputs[memory].expression)
	if !ok {
		return false
	}
	helper := "renvo_runtime_CCompareExchange" + decimalString(width*8)
	if t.asmCompareExchange&(1<<width) == 0 {
		t.asmCompareExchange |= 1 << width
		t.staticOut = append(t.staticOut, ("func " + helper + "(address uintptr,observed uintptr,replacement uint64) uint8{return 0}\n")...)
	}
	if condition < 0 {
		t.emitExpression(operation.outputs[observed].expression)
		t.appendText("=")
		t.convertedExpression(t.expressionType(operation.outputs[observed].expression), operation.inputs[expected].expression)
		t.appendText(";")
	} else {
		t.emitExpression(operation.outputs[condition].expression)
		t.appendText("=")
	}
	t.appendText(helper)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(pointer)
	t.appendText(")),uintptr(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.outputs[observed].expression)
	t.appendText("))),uint64(")
	t.emitExpression(operation.inputs[replacement].expression)
	t.appendText("));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmCompareExchange128(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "cmpxchg16b%[ptr]") {
		return false
	}
	condition, memory, oldLow, oldHigh, newLow, newHigh := -1, -1, -1, -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		switch operation.outputs[i].constraint {
		case "=@cce":
			condition = i
		case "+m":
			if operation.outputs[i].name == "ptr" {
				memory = i
			}
		case "+a":
			oldLow = i
		case "+d":
			oldHigh = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint == "b" {
			newLow = i
		} else if operation.inputs[i].constraint == "c" {
			newHigh = i
		}
	}
	if memory < 0 || oldLow < 0 || oldHigh < 0 || newLow < 0 || newHigh < 0 ||
		len(operation.outputs) != 3 && len(operation.outputs) != 4 || len(operation.inputs) != 2 {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.outputs[memory].expression)
	if !ok || t.typeSize(t.expressionType(operation.outputs[memory].expression)) != 16 ||
		t.typeSize(t.expressionType(operation.outputs[oldLow].expression)) != 8 ||
		t.typeSize(t.expressionType(operation.outputs[oldHigh].expression)) != 8 {
		return false
	}
	if !t.asmCompareExchange128 {
		t.asmCompareExchange128 = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCompareExchange128(address *[2]uint64,oldLow *uint64,oldHigh *uint64,newLow uint64,newHigh uint64) uint8{return 0}\n"...)
	}
	if condition >= 0 {
		t.emitExpression(operation.outputs[condition].expression)
		t.appendText("=")
	}
	t.appendText("renvo_runtime_CCompareExchange128(")
	t.emitExpression(pointer)
	t.appendText(",&(")
	t.emitExpression(operation.outputs[oldLow].expression)
	t.appendText("),&(")
	t.emitExpression(operation.outputs[oldHigh].expression)
	t.appendText("),uint64(")
	t.emitExpression(operation.inputs[newLow].expression)
	t.appendText("),uint64(")
	t.emitExpression(operation.inputs[newHigh].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmAlternativeCall(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "call%c[old]") {
		return false
	}
	old, argument := -1, -1
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].name == "old" && operation.inputs[i].constraint == "i" {
			old = i
		} else if operation.inputs[i].constraint == "D" {
			argument = i
		}
	}
	if old < 0 || argument < 0 {
		return false
	}
	t.emitExpression(operation.inputs[old].expression)
	t.appendText("(")
	t.emitExpression(operation.inputs[argument].expression)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmAlternativeRelocSelftest(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, ".pushsection.altinstructions,\"a\"") ||
		!t.asmTemplateCompactContains(operation.template, "lea%[mem],%%rdi;call__alt_reloc_selftest;") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "+r" ||
		len(operation.inputs) != 1 || operation.inputs[0].name != "mem" || operation.inputs[0].constraint != "m" ||
		len(operation.clobbers) != 1 || operation.clobbers[0] != "rdi" {
		return false
	}
	t.appendText("__alt_reloc_selftest((*byte)(__c_unsafe.Pointer(&(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("))));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmMultiplyDivide64(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "mulq%2;divq%3") || len(operation.outputs) != 1 ||
		operation.outputs[0].constraint != "=a" || len(operation.inputs) != 3 ||
		operation.inputs[0].constraint != "a" || operation.inputs[1].constraint != "rm" || operation.inputs[2].constraint != "rm" {
		return false
	}
	for i := 0; i < len(operation.inputs); i++ {
		if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 8 {
			return false
		}
	}
	if !t.asmMultiplyDivide {
		t.asmMultiplyDivide = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CMultiplyDivide64(value uint64,multiplier uint64,divisor uint64) uint64{return value*multiplier/divisor}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(renvo_runtime_CMultiplyDivide64(uint64(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),uint64(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("),uint64(")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText(")))")
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmMultiply32(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "mull%%edx") || len(operation.outputs) != 2 || len(operation.inputs) != 2 ||
		operation.outputs[0].constraint != "=d" || operation.outputs[1].constraint != "=&a" ||
		operation.inputs[0].constraint != "1" || operation.inputs[1].constraint != "0" {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		outputSize := t.typeSize(t.expressionType(operation.outputs[i].expression))
		inputSize := t.typeSize(t.expressionType(operation.inputs[i].expression))
		if outputSize < 4 || outputSize > 8 || inputSize < 4 || inputSize > 8 {
			return false
		}
	}
	if !t.asmMultiply32 {
		t.asmMultiply32 = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CMultiply32(left uint32,right uint32) uint64{return uint64(left)*uint64(right)}\n"...)
	}
	t.typeSerial++
	value := "__c_multiply32_" + decimalString(t.typeSerial)
	t.appendText("var ")
	t.appendText(value)
	t.appendText(" uint64=renvo_runtime_CMultiply32(uint32(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),uint32(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("));")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(")
	t.appendText(value)
	t.appendText(">>32);")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[1].expression))
	t.appendText("(")
	t.appendText(value)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmMultiplyShift32(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "mulq%[mul_frac];shrd$32,%[hi],%[lo]") ||
		len(operation.outputs) != 2 || operation.outputs[0].name != "lo" || operation.outputs[0].constraint != "=a" ||
		operation.outputs[1].name != "hi" || operation.outputs[1].constraint != "=d" || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "0" || operation.inputs[1].name != "mul_frac" || operation.inputs[1].constraint != "rm" {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		if t.typeSize(t.expressionType(operation.outputs[i].expression)) != 8 {
			return false
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 8 {
			return false
		}
	}
	if !t.asmMultiplyShift {
		t.asmMultiplyShift = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CMultiplyShift32(value uint64,multiplier uint64,high *uint64) uint64{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CMultiplyShift32(uint64(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),uint64(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("),&(")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmByteSwap(operation cAsm) bool {
	width := 0
	if t.asmTemplateCompactPrefix(operation.template, "bswapl%0") {
		width = 4
	} else if t.asmTemplateCompactPrefix(operation.template, "bswapq%0") {
		width = 8
	}
	if width == 0 || len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" ||
		len(operation.inputs) != 1 || operation.inputs[0].constraint != "0" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != width ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != width {
		return false
	}
	name := "__c_byte_swap_" + decimalString(width*8)
	t.ensureByteSwapHelper(width)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(")
	t.appendText(name)
	t.appendText("(")
	if width == 8 {
		t.appendText("uint64(")
	} else {
		t.appendText("uint32(")
	}
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")))")
	t.appendText(";")
	return t.ok
}

func (t *translator) ensureByteSwapHelper(width int) string {
	name := "__c_byte_swap_" + decimalString(width*8)
	if t.asmByteSwap&(1<<width) != 0 {
		return name
	}
	t.asmByteSwap |= 1 << width
	if width == 2 {
		t.staticOut = append(t.staticOut, "func __c_byte_swap_16(value uint16) uint16{return (value>>8)|(value<<8)}\n"...)
	} else if width == 4 {
		t.staticOut = append(t.staticOut, "func __c_byte_swap_32(value uint32) uint32{return (value>>24)|((value>>8)&0xff00)|((value<<8)&0xff0000)|(value<<24)}\n"...)
	} else {
		t.staticOut = append(t.staticOut, "func __c_byte_swap_64(value uint64) uint64{return (value>>56)|((value>>40)&0xff00)|((value>>24)&0xff0000)|((value>>8)&0xff000000)|((value<<8)&0xff00000000)|((value<<24)&0xff0000000000)|((value<<40)&0xff000000000000)|(value<<56)}\n"...)
	}
	return name
}

func (t *translator) emitAsmPopulationCount(operation cAsm) bool {
	width := 0
	if t.asmTemplateCompactContains(operation.template, "popcntl%1,%0") {
		width = 4
	} else if t.asmTemplateCompactContains(operation.template, "popcntq%1,%0") {
		width = 8
	}
	if width == 0 || len(operation.outputs) != 1 || operation.outputs[0].constraint != "=a" ||
		len(operation.inputs) != 1 || operation.inputs[0].constraint != "D" ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != width {
		return false
	}
	name := "__c_population_count_" + decimalString(width*8)
	if t.asmPopulationCount&(1<<width) == 0 {
		t.asmPopulationCount |= 1 << width
		typeName := "uint32"
		if width == 8 {
			typeName = "uint64"
		}
		t.staticOut = append(t.staticOut, ("func " + name + "(value " + typeName + ") " + typeName + "{var count " + typeName + ";for value!=0{count+=value&1;value>>=1};return count}\n")...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(")
	t.appendText(name)
	t.appendText("(")
	if width == 8 {
		t.appendText("uint64(")
	} else {
		t.appendText("uint32(")
	}
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")))")
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmBitScan(operation cAsm) bool {
	forward := t.asmTemplateCompactPrefix(operation.template, "rep;bsf%1,%0") ||
		t.asmTemplateCompactPrefix(operation.template, "bsfl%1,%0")
	reverse := t.asmTemplateCompactPrefix(operation.template, "bsr%1,%0") ||
		t.asmTemplateCompactPrefix(operation.template, "bsrl%1,%0") ||
		t.asmTemplateCompactPrefix(operation.template, "bsrq%1,%q0")
	if !forward && !reverse || len(operation.outputs) != 1 || len(operation.inputs) < 1 || len(operation.inputs) > 2 ||
		(operation.outputs[0].constraint != "=r" && operation.outputs[0].constraint != "+r") ||
		(operation.inputs[0].constraint != "r" && operation.inputs[0].constraint != "rm") ||
		len(operation.inputs) == 2 && operation.inputs[1].constraint != "0" {
		return false
	}
	width := t.typeSize(t.expressionType(operation.inputs[0].expression))
	if width != 4 && width != 8 {
		return false
	}
	kind := 1
	name := "renvo_runtime_CBitScanForward"
	if reverse {
		kind, name = 2, "renvo_runtime_CBitScanReverse"
	}
	if width == 8 {
		kind += 2
	}
	name += decimalString(width * 8)
	if t.asmBitScan&(1<<kind) == 0 {
		t.asmBitScan |= 1 << kind
		typeName := "int32"
		valueType := "uint32"
		if width == 8 {
			typeName, valueType = "int64", "uint64"
		}
		t.staticOut = append(t.staticOut, ("func " + name + "(value " + valueType + ",fallback " + typeName + ") " + typeName + "{return fallback}\n")...)
	}
	outputType := t.expressionType(operation.outputs[0].expression)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(outputType)
	t.appendText("(")
	t.appendText(name)
	t.appendText("(")
	if width == 8 {
		t.appendText("uint64(")
	} else {
		t.appendText("uint32(")
	}
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),")
	if width == 8 {
		t.appendText("int64(")
	} else {
		t.appendText("int32(")
	}
	if len(operation.inputs) == 2 {
		t.emitExpression(operation.inputs[1].expression)
	} else {
		t.emitExpression(operation.outputs[0].expression)
	}
	t.appendText(")))")
	t.appendText(";")
	return t.ok
}

func (t *translator) emitAsmByteTest(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "testb%2,%1") || len(operation.outputs) != 1 ||
		operation.outputs[0].constraint != "=@ccnz" || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "m" || operation.inputs[1].constraint != "i" ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 1 {
		return false
	}
	if t.asmByteOps&16 == 0 {
		t.asmByteOps |= 16
		t.staticOut = append(t.staticOut, "func renvo_runtime_CTestByte(address *uint8,value uint8) uint8{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CTestByte(&(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),uint8(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmByteOperation(operation cAsm) bool {
	kind, name := 0, ""
	switch {
	case t.asmTemplateCompactPrefix(operation.template, "orb%b1,%0"):
		kind, name = 1, "renvo_runtime_COrByte"
	case t.asmTemplateCompactPrefix(operation.template, "andb%b1,%0"):
		kind, name = 2, "renvo_runtime_CAndByte"
	case t.asmTemplateCompactPrefix(operation.template, "xorb%b1,%0"):
		kind, name = 4, "renvo_runtime_CXorByte"
	case t.asmTemplateCompactPrefix(operation.template, "xorb%2,%1"):
		kind, name = 4, "renvo_runtime_CXorByte"
	}
	if kind == 0 || len(operation.inputs) != 1 || operation.inputs[0].constraint != "iq" ||
		len(operation.outputs) < 1 || len(operation.outputs) > 2 {
		return false
	}
	memory, condition := -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].constraint == "+m" {
			memory = i
		} else if operation.outputs[i].constraint == "=@ccs" {
			condition = i
		} else {
			return false
		}
	}
	if memory < 0 || len(operation.outputs) == 2 && (condition < 0 || kind != 4) {
		return false
	}
	if condition >= 0 {
		kind, name = 8, "renvo_runtime_CXorByteNegative"
	}
	pointer, ok := t.asmDereferencePointer(operation.outputs[memory].expression)
	if !ok || t.typeSize(t.expressionType(operation.outputs[memory].expression)) != 1 {
		return false
	}
	if t.asmByteOps&kind == 0 {
		t.asmByteOps |= kind
		declaration := "func " + name + "(address *int8,value int8){}\n"
		if condition >= 0 {
			declaration = "func " + name + "(address *int8,value int8) uint8{return 0}\n"
		}
		t.staticOut = append(t.staticOut, declaration...)
	}
	if condition >= 0 {
		t.emitExpression(operation.outputs[condition].expression)
		t.appendText("=")
	}
	t.appendText(name)
	t.appendText("(")
	t.emitExpression(pointer)
	t.appendText(",")
	t.convertedExpression(cTypeInt8ID, operation.inputs[0].expression)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmBitOperation64(operation cAsm) bool {
	kind, name := 0, ""
	switch {
	case t.asmTemplateCompactContains(operation.template, "btsq"):
		kind, name = 1, "renvo_runtime_CBitSet64"
	case t.asmTemplateCompactContains(operation.template, "btrq"):
		kind, name = 2, "renvo_runtime_CBitReset64"
	case t.asmTemplateCompactContains(operation.template, "btcq"):
		kind, name = 3, "renvo_runtime_CBitComplement64"
	case t.asmTemplateCompactContains(operation.template, "btq"):
		kind, name = 4, "renvo_runtime_CBitTest64"
	default:
		return false
	}
	condition, memory, bit := -1, -1, -1
	for i := 0; i < len(operation.outputs); i++ {
		if operation.outputs[i].constraint == "=@ccc" {
			condition = i
		} else if operation.outputs[i].constraint == "+m" {
			memory = i
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint == "m" {
			if memory >= 0 {
				return false
			}
			memory = len(operation.outputs) + i
		} else if operation.inputs[i].constraint == "Ir" {
			bit = i
		}
	}
	if memory < 0 || bit < 0 || len(operation.outputs) > 2 || len(operation.inputs) > 2 {
		return false
	}
	var memoryExpression []token
	if memory < len(operation.outputs) {
		memoryExpression = operation.outputs[memory].expression
	} else {
		memoryExpression = operation.inputs[memory-len(operation.outputs)].expression
	}
	pointer, ok := t.asmDereferencePointer(memoryExpression)
	if !ok || t.typeSize(t.expressionType(memoryExpression)) != 8 {
		return false
	}
	if t.asmBitOps64&(1<<kind) == 0 {
		t.asmBitOps64 |= 1 << kind
		t.staticOut = append(t.staticOut, ("func " + name + "(address *uint64,bit int64) uint8{return 0}\n")...)
	}
	if condition >= 0 {
		t.emitExpression(operation.outputs[condition].expression)
		t.appendText("=")
	}
	t.appendText(name)
	t.appendText("(")
	t.emitExpression(pointer)
	t.appendText(",int64(")
	t.emitExpression(operation.inputs[bit].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmCopyBytes(operation cAsm) bool {
	if t.asmTemplateCompactPrefix(operation.template, "movsb") || t.asmTemplateCompactPrefix(operation.template, "movsw") {
		if t.emitAsmMoveString(operation) {
			return true
		}
	}
	if t.asmTemplateCompactPrefix(operation.template, "rep;movsl") &&
		t.asmTemplateCompactContains(operation.template, "testb$2,%b4") &&
		t.asmTemplateCompactContains(operation.template, "movsw") &&
		t.asmTemplateCompactContains(operation.template, "testb$1,%b4") &&
		t.asmTemplateCompactContains(operation.template, "movsb") {
		return t.emitAsmCopyBytesWithTail(operation)
	}
	if !t.asmTemplateCompactPrefix(operation.template, "rep;movsqmovq%4,%%rcxrep;movsb") ||
		len(operation.outputs) != 3 || operation.outputs[0].constraint != "=&c" ||
		operation.outputs[1].constraint != "=&D" || operation.outputs[2].constraint != "=&S" ||
		len(operation.inputs) != 4 || operation.inputs[0].constraint != "0" ||
		operation.inputs[1].constraint != "g" || operation.inputs[2].constraint != "1" ||
		operation.inputs[3].constraint != "2" {
		return false
	}
	if !t.asmCopyHelper {
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCopyBytes(dest *byte,src *byte,count uintptr){}\n"...)
		t.asmCopyHelper = true
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_asm_count_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("*8+")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText(");__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText(";__c_asm_src_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.inputs[3].expression)
	t.appendText(";renvo_runtime_CCopyBytes((*byte)(__c_asm_dest_")
	t.appendText(suffix)
	t.appendText("),(*byte)(__c_asm_src_")
	t.appendText(suffix)
	t.appendText("),__c_asm_count_")
	t.appendText(suffix)
	t.appendText(");")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=0;")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=int64(uintptr(__c_unsafe.Pointer(__c_asm_dest_")
	t.appendText(suffix)
	t.appendText("))+__c_asm_count_")
	t.appendText(suffix)
	t.appendText(");")
	t.emitExpression(operation.outputs[2].expression)
	t.appendText("=int64(uintptr(__c_unsafe.Pointer(__c_asm_src_")
	t.appendText(suffix)
	t.appendText("))+__c_asm_count_")
	t.appendText(suffix)
	t.appendText(")};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmMoveString(operation cAsm) bool {
	width := 0
	if t.asmTemplateCompactPrefix(operation.template, "movsb") {
		width = 1
	} else if t.asmTemplateCompactPrefix(operation.template, "movsw") {
		width = 2
	}
	if width == 0 || len(operation.outputs) != 2 || operation.outputs[0].constraint != "=&D" ||
		operation.outputs[1].constraint != "=&S" || len(operation.inputs) != 2 ||
		operation.inputs[0].constraint != "0" || operation.inputs[1].constraint != "1" ||
		len(operation.clobbers) != 1 || operation.clobbers[0] != "memory" {
		return false
	}
	destType, sourceType := t.expressionType(operation.outputs[0].expression), t.expressionType(operation.outputs[1].expression)
	if t.typeInfo(destType).kind != cTypePointer || t.typeInfo(sourceType).kind != cTypePointer ||
		t.typeInfo(t.expressionType(operation.inputs[0].expression)).kind != cTypePointer ||
		t.typeInfo(t.expressionType(operation.inputs[1].expression)).kind != cTypePointer {
		return false
	}
	if !t.asmCopyHelper {
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCopyBytes(dest *byte,src *byte,count uintptr){}\n"...)
		t.asmCopyHelper = true
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(";__c_asm_src_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText(";renvo_runtime_CCopyBytes((*byte)(__c_unsafe.Pointer(__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(")),(*byte)(__c_unsafe.Pointer(__c_asm_src_")
	t.appendText(suffix)
	t.appendText(")),")
	t.appendDecimal(width)
	t.appendText(");")
	for i, typeID := range []int{destType, sourceType} {
		t.emitExpression(operation.outputs[i].expression)
		t.appendText("=(")
		t.emitType(typeID)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
		if i == 0 {
			t.appendText("__c_asm_dest_")
		} else {
			t.appendText("__c_asm_src_")
		}
		t.appendText(suffix)
		t.appendText("))+")
		t.appendDecimal(width)
		t.appendText("));")
	}
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmCopyBytesWithTail(operation cAsm) bool {
	if len(operation.outputs) != 3 || operation.outputs[0].constraint != "=&c" ||
		operation.outputs[1].constraint != "=&D" || operation.outputs[2].constraint != "=&S" ||
		len(operation.inputs) != 4 || operation.inputs[0].constraint != "0" ||
		operation.inputs[1].constraint != "q" || operation.inputs[2].constraint != "1" ||
		operation.inputs[3].constraint != "2" || len(operation.clobbers) != 1 || operation.clobbers[0] != "memory" {
		return false
	}
	if !t.asmCopyHelper {
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCopyBytes(dest *byte,src *byte,count uintptr){}\n"...)
		t.asmCopyHelper = true
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_asm_count_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText(");__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText(");__c_asm_src_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.inputs[3].expression)
	t.appendText(");renvo_runtime_CCopyBytes((*byte)(__c_unsafe.Pointer(__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(")),(*byte)(__c_unsafe.Pointer(__c_asm_src_")
	t.appendText(suffix)
	t.appendText(")),__c_asm_count_")
	t.appendText(suffix)
	t.appendText(");")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=0;")
	for i := 1; i < 3; i++ {
		t.emitExpression(operation.outputs[i].expression)
		t.appendText("=")
		t.emitType(t.expressionType(operation.outputs[i].expression))
		t.appendText("(")
		if i == 1 {
			t.appendText("__c_asm_dest_")
		} else {
			t.appendText("__c_asm_src_")
		}
		t.appendText(suffix)
		t.appendText("+__c_asm_count_")
		t.appendText(suffix)
		t.appendText(");")
	}
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmUserCopy(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "repmovsb") ||
		!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\"") ||
		len(operation.outputs) != 4 || len(operation.inputs) != 0 ||
		operation.outputs[0].constraint != "+c" || operation.outputs[1].constraint != "+D" ||
		operation.outputs[2].constraint != "+S" || operation.outputs[3].constraint != "+r" {
		return false
	}
	destType, sourceType := t.expressionType(operation.outputs[1].expression), t.expressionType(operation.outputs[2].expression)
	if t.typeInfo(destType).kind != cTypePointer || t.typeInfo(sourceType).kind != cTypePointer {
		return false
	}
	if !t.asmUserCopyHelper {
		t.asmUserCopyHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCopyUserBytes___ex_table(dest *byte,src *byte,count uintptr) uintptr{return count}\n"...)
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_user_count_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(");__c_user_dest_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(";__c_user_src_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.outputs[2].expression)
	t.appendText(";__c_user_remaining_")
	t.appendText(suffix)
	t.appendText(":=renvo_runtime_CCopyUserBytes___ex_table((*byte)(__c_unsafe.Pointer(__c_user_dest_")
	t.appendText(suffix)
	t.appendText(")),(*byte)(__c_unsafe.Pointer(__c_user_src_")
	t.appendText(suffix)
	t.appendText(")),__c_user_count_")
	t.appendText(suffix)
	t.appendText(");__c_user_done_")
	t.appendText(suffix)
	t.appendText(":=__c_user_count_")
	t.appendText(suffix)
	t.appendText("-__c_user_remaining_")
	t.appendText(suffix)
	t.appendText(";")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitType(t.expressionType(operation.outputs[0].expression))
	t.appendText("(__c_user_remaining_")
	t.appendText(suffix)
	t.appendText(");")
	for i, typeID := range []int{destType, sourceType} {
		t.emitExpression(operation.outputs[i+1].expression)
		t.appendText("=(")
		t.emitType(typeID)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
		if i == 0 {
			t.appendText("__c_user_dest_")
		} else {
			t.appendText("__c_user_src_")
		}
		t.appendText(suffix)
		t.appendText("))+__c_user_done_")
		t.appendText(suffix)
		t.appendText("));")
	}
	t.appendText("_=")
	t.emitExpression(operation.outputs[3].expression)
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmRepeatedCopy(operation cAsm) bool {
	width := 0
	if t.asmTemplateCompactPrefix(operation.template, "rep;movsl") {
		width = 4
	} else if t.asmTemplateCompactPrefix(operation.template, "rep;movsq") {
		width = 8
	}
	if width == 0 || len(operation.outputs) != 3 || len(operation.inputs) != 3 ||
		operation.outputs[0].constraint != "=&c" || operation.outputs[1].constraint != "=&D" ||
		operation.outputs[2].constraint != "=&S" || operation.inputs[0].constraint != "0" ||
		operation.inputs[1].constraint != "1" || operation.inputs[2].constraint != "2" {
		return false
	}
	destType, sourceType := t.expressionType(operation.outputs[1].expression), t.expressionType(operation.outputs[2].expression)
	if t.typeInfo(destType).kind != cTypePointer || t.typeInfo(sourceType).kind != cTypePointer {
		return false
	}
	if !t.asmCopyHelper {
		t.staticOut = append(t.staticOut, "func renvo_runtime_CCopyBytes(dest *byte,src *byte,count uintptr){}\n"...)
		t.asmCopyHelper = true
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_asm_count_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")*")
	t.appendDecimal(width)
	t.appendText(";__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText(";__c_asm_src_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText(";renvo_runtime_CCopyBytes((*byte)(__c_unsafe.Pointer(__c_asm_dest_")
	t.appendText(suffix)
	t.appendText(")),(*byte)(__c_unsafe.Pointer(__c_asm_src_")
	t.appendText(suffix)
	t.appendText(")),__c_asm_count_")
	t.appendText(suffix)
	t.appendText(");")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=0;")
	for i, typeID := range []int{destType, sourceType} {
		t.emitExpression(operation.outputs[i+1].expression)
		t.appendText("=(")
		t.emitType(typeID)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
		if i == 0 {
			t.appendText("__c_asm_dest_")
		} else {
			t.appendText("__c_asm_src_")
		}
		t.appendText(suffix)
		t.appendText("))+__c_asm_count_")
		t.appendText(suffix)
		t.appendText("));")
	}
	t.appendText("};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmVideoTextFill(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "pushw%%es;movw%2,%%es;shrw%%cx;jnc1f;stosw1:rep;stosl;popw%%es") ||
		len(operation.outputs) != 2 || operation.outputs[0].constraint != "+D" || operation.outputs[1].constraint != "+c" ||
		len(operation.inputs) != 2 || operation.inputs[0].constraint != "bdS" || operation.inputs[1].constraint != "a" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.outputs[1].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 2 ||
		t.typeSize(t.expressionType(operation.inputs[1].expression)) != 4 {
		return false
	}
	t.ensureAsmSegmentMemoryHelper(0, 2, true)
	t.appendText("_=")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(";for ")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(">0{renvo_runtime_CWriteSegmentfs16((*uint16)(__c_unsafe.Pointer(uintptr(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("))),uint16(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("));")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("+2;")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("--};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmTiedIdentity(operation cAsm) bool {
	if len(operation.template) != 0 || len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
		operation.outputs[0].constraint != "=r" || operation.inputs[0].constraint != "0" {
		return false
	}
	outputType := t.lvalueType(operation.outputs[0].expression)
	if outputType == cTypeVoidID {
		return false
	}
	t.emitExpression(operation.outputs[0].expression)
	t.out = append(t.out, '=')
	t.convertedExpression(outputType, operation.inputs[0].expression)
	t.out = append(t.out, ';')
	return t.ok
}

func (t *translator) emitAsmSegmentCompare(operation cAsm) bool {
	segment := ""
	if t.asmTemplatePrefix(operation.template, "fs; repe; cmpsb") {
		segment = "fs"
	} else if t.asmTemplatePrefix(operation.template, "gs; repe; cmpsb") {
		segment = "gs"
	}
	if segment == "" || len(operation.outputs) != 4 || len(operation.inputs) != 0 ||
		operation.outputs[0].constraint != "=@ccnz" || operation.outputs[1].constraint != "+D" ||
		operation.outputs[2].constraint != "+S" || operation.outputs[3].constraint != "+c" ||
		t.typeInfo(t.expressionType(operation.outputs[0].expression)).kind != cTypeBool {
		return false
	}
	segmentIndex := 0
	if segment == "gs" {
		segmentIndex = 1
	}
	t.ensureAsmSegmentCompareHelper(segmentIndex, operation)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CSegmentCompare")
	t.appendText(segment)
	for i := 1; i < len(operation.outputs); i++ {
		if i == 1 {
			t.appendText("(&")
		} else {
			t.appendText(",&")
		}
		t.emitExpression(operation.outputs[i].expression)
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmSegmentMemory(operation cAsm) bool {
	for segmentIndex, segment := range []string{"fs", "gs"} {
		for _, size := range []int{1, 2, 4} {
			letter := "b"
			if size == 2 {
				letter = "w"
			} else if size == 4 {
				letter = "l"
			}
			if t.asmTemplateCompactPrefix(operation.template, "mov"+letter+"%%"+segment+":%1,%0") {
				outputConstraint := "=r"
				if size == 1 {
					outputConstraint = "=q"
				}
				if len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
					operation.outputs[0].constraint != outputConstraint || operation.inputs[0].constraint != "m" ||
					t.typeSize(t.expressionType(operation.outputs[0].expression)) != size {
					return false
				}
				pointer, ok := t.asmDereferencePointer(operation.inputs[0].expression)
				if !ok {
					return false
				}
				t.ensureAsmSegmentMemoryHelper(segmentIndex, size, false)
				t.emitExpression(operation.outputs[0].expression)
				t.appendText("=renvo_runtime_CReadSegment")
				t.appendText(segment)
				t.appendDecimal(size * 8)
				t.appendText("(")
				t.emitExpression(pointer)
				t.appendText(");")
				return t.ok
			}
			if t.asmTemplateCompactPrefix(operation.template, "mov"+letter+"%1,%%"+segment+":%0") {
				inputConstraint := "ri"
				if size == 1 {
					inputConstraint = "qi"
				}
				if len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
					operation.outputs[0].constraint != "+m" || operation.inputs[0].constraint != inputConstraint ||
					t.typeSize(t.expressionType(operation.inputs[0].expression)) != size {
					return false
				}
				pointer, ok := t.asmDereferencePointer(operation.outputs[0].expression)
				if !ok {
					return false
				}
				t.ensureAsmSegmentMemoryHelper(segmentIndex, size, true)
				t.appendText("renvo_runtime_CWriteSegment")
				t.appendText(segment)
				t.appendDecimal(size * 8)
				t.appendText("(")
				t.emitExpression(pointer)
				t.appendText(",")
				t.emitExpression(operation.inputs[0].expression)
				t.appendText(");")
				return t.ok
			}
		}
	}
	return false
}

func (t *translator) emitAsmSegmentRegister(operation cAsm) bool {
	if t.asmTemplateCompactPrefix(operation.template, "movl%%eax,%%dsmovl%%eax,%%ssmovl%%eax,%%es") {
		if len(operation.outputs) != 0 || len(operation.inputs) != 1 || operation.inputs[0].constraint != "a" ||
			t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 {
			return false
		}
		t.ensureAsmStartupSegmentsHelper()
		t.appendText("renvo_runtime_CWriteStartupSegments(uint16(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
		return t.ok
	}
	for index, segment := range []string{"es", "ds", "ss"} {
		if !t.asmTemplateCompactPrefix(operation.template, "1:movl%k0,%%"+segment) ||
			!t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\",\"a\"") ||
			!t.asmTemplateCompactContains(operation.template, "extable_type_regreg=%k0,type=(17|((0)<<16))") {
			continue
		}
		if len(operation.outputs) != 1 || operation.outputs[0].constraint != "+r" ||
			len(operation.inputs) != 0 || t.typeSize(t.expressionType(operation.outputs[0].expression)) != 2 {
			return false
		}
		t.ensureAsmSafeSegmentHelper(index)
		t.appendText("renvo_runtime_CWriteSegmentSafe")
		t.appendText(segment)
		t.appendText("(uint16(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("));")
		return t.ok
	}
	segments := []string{"es", "ds", "ss", "fs", "gs"}
	for index, segment := range segments {
		readSize := 0
		if t.asmTemplateCompactPrefix(operation.template, "movw%%"+segment+",%0") {
			readSize = 2
		} else if t.asmTemplateCompactPrefix(operation.template, "movl%%"+segment+",%0") {
			readSize = 4
		} else if t.asmTemplateCompactPrefix(operation.template, "mov%%"+segment+",%0") && len(operation.outputs) == 1 {
			readSize = t.typeSize(t.expressionType(operation.outputs[0].expression))
			if readSize != 2 && readSize != 4 {
				return false
			}
		}
		if readSize != 0 {
			if len(operation.outputs) != 1 || (operation.outputs[0].constraint != "=r" && operation.outputs[0].constraint != "=rm") ||
				len(operation.inputs) != 0 || t.typeSize(t.expressionType(operation.outputs[0].expression)) != readSize {
				return false
			}
			t.ensureAsmSegmentHelper(index, false)
			t.emitExpression(operation.outputs[0].expression)
			t.appendText("=")
			if readSize != 2 {
				t.emitType(t.expressionType(operation.outputs[0].expression))
				t.appendText("(")
			}
			t.appendText("renvo_runtime_CReadSegment")
			t.appendText(segment)
			t.appendText("()")
			if readSize != 2 {
				t.appendText(")")
			}
			t.appendText(";")
			return t.ok
		}
		wrappedWrite := t.asmTemplateCompactContains(operation.template, "movw%0,%%"+segment) &&
			t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\"")
		inferredWrite := t.asmTemplateCompactPrefix(operation.template, "mov%0,%%"+segment)
		if (segment == "fs" || segment == "gs") && (t.asmTemplateCompactPrefix(operation.template, "movw%0,%%"+segment) || wrappedWrite || inferredWrite) {
			if len(operation.outputs) != 0 || len(operation.inputs) != 1 ||
				(operation.inputs[0].constraint != "rm" && operation.inputs[0].constraint != "r") ||
				t.typeSize(t.expressionType(operation.inputs[0].expression)) != 2 &&
					(!inferredWrite || t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4) {
				return false
			}
			t.ensureAsmSegmentHelper(index, true)
			t.appendText("renvo_runtime_CWriteSegment")
			t.appendText(segment)
			t.appendText("(uint16(")
			t.emitExpression(operation.inputs[0].expression)
			t.appendText("));")
			return t.ok
		}
	}
	return false
}

func (t *translator) emitAsmBaseRegister(operation cAsm) bool {
	kind, name, write := 0, "", false
	switch {
	case t.asmTemplateCompactPrefix(operation.template, "rdfsbase%0"):
		kind, name = 1, "renvo_runtime_CReadFSBase"
	case t.asmTemplateCompactPrefix(operation.template, "rdgsbase%0"):
		kind, name = 2, "renvo_runtime_CReadGSBase"
	case t.asmTemplateCompactPrefix(operation.template, "wrfsbase%0"):
		kind, name, write = 3, "renvo_runtime_CWriteFSBase", true
	case t.asmTemplateCompactPrefix(operation.template, "wrgsbase%0"):
		kind, name, write = 4, "renvo_runtime_CWriteGSBase", true
	default:
		return false
	}
	if write {
		if len(operation.outputs) != 0 || len(operation.inputs) != 1 || operation.inputs[0].constraint != "r" ||
			t.typeSize(t.expressionType(operation.inputs[0].expression)) != t.pointerSize {
			return false
		}
	} else if len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" || len(operation.inputs) != 0 ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != t.pointerSize {
		return false
	}
	if t.asmBaseRegisterOps&(1<<kind) == 0 {
		t.asmBaseRegisterOps |= 1 << kind
		if write {
			t.staticOut = append(t.staticOut, ("func " + name + "(value uint64){}\n")...)
		} else {
			t.staticOut = append(t.staticOut, ("func " + name + "() uint64{return 0}\n")...)
		}
	}
	if write {
		t.appendText(name)
		t.appendText("(uint64(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
	} else {
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=")
		t.emitType(t.expressionType(operation.outputs[0].expression))
		t.appendText("(")
		t.appendText(name)
		t.appendText("());")
	}
	return t.ok
}

func (t *translator) emitAsmPortIO(operation cAsm) bool {
	if port, ok := t.asmDirectOutputPort(operation); ok {
		t.ensureAsmIOHelper(1, true)
		t.appendText("renvo_runtime_COut8(0,")
		t.appendDecimal(port)
		t.appendText(");")
		return true
	}
	for _, size := range []int{1, 2, 4} {
		letter, modifier := "b", "b"
		if size == 2 {
			letter, modifier = "w", "w"
		} else if size == 4 {
			letter, modifier = "l", ""
		}
		outTemplate := "out" + letter + "%" + modifier + "0,%w1"
		genericOutput := t.asmTemplateCompactPrefix(operation.template, "out%0,%1") &&
			len(operation.inputs) == 2 && t.typeSize(t.expressionType(operation.inputs[0].expression)) == size
		legacyOutput := t.asmTemplateCompactPrefix(operation.template, "out"+letter+"%0,%1") &&
			len(operation.inputs) == 2 && t.typeSize(t.expressionType(operation.inputs[0].expression)) == size
		if t.asmTemplateCompactPrefix(operation.template, outTemplate) || genericOutput || legacyOutput {
			if len(operation.outputs) != 0 || len(operation.inputs) != 2 ||
				operation.inputs[0].constraint != "a" ||
				(operation.inputs[1].constraint != "Nd" && (!genericOutput && !legacyOutput || operation.inputs[1].constraint != "d")) ||
				t.typeSize(t.expressionType(operation.inputs[0].expression)) != size ||
				t.typeSize(t.expressionType(operation.inputs[1].expression)) != 2 {
				return false
			}
			t.ensureAsmIOHelper(size, true)
			t.appendText("renvo_runtime_COut")
			t.appendDecimal(size * 8)
			t.appendText("(")
			t.emitExpression(operation.inputs[0].expression)
			t.appendText(",")
			t.emitExpression(operation.inputs[1].expression)
			t.appendText(");")
			return t.ok
		}
		inTemplate := "in" + letter + "%w1,%" + modifier + "0"
		genericInput := t.asmTemplateCompactPrefix(operation.template, "in%1,%0") &&
			len(operation.outputs) == 1 && t.typeSize(t.expressionType(operation.outputs[0].expression)) == size
		legacyInput := t.asmTemplateCompactPrefix(operation.template, "in"+letter+"%1,%0") &&
			len(operation.outputs) == 1 && t.typeSize(t.expressionType(operation.outputs[0].expression)) == size
		if t.asmTemplateCompactPrefix(operation.template, inTemplate) || genericInput || legacyInput {
			if len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
				operation.outputs[0].constraint != "=a" ||
				(operation.inputs[0].constraint != "Nd" && (!genericInput && !legacyInput || operation.inputs[0].constraint != "d")) ||
				t.typeSize(t.expressionType(operation.outputs[0].expression)) != size ||
				t.typeSize(t.expressionType(operation.inputs[0].expression)) != 2 {
				return false
			}
			t.ensureAsmIOHelper(size, false)
			t.emitExpression(operation.outputs[0].expression)
			t.appendText("=renvo_runtime_CIn")
			t.appendDecimal(size * 8)
			t.appendText("(")
			t.emitExpression(operation.inputs[0].expression)
			t.appendText(");")
			return t.ok
		}
	}
	return false
}

func (t *translator) asmDirectOutputPort(operation cAsm) (int, bool) {
	if len(operation.outputs) != 0 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 ||
		len(operation.labels) != 0 || operation.gotoAsm {
		return 0, false
	}
	compact := make([]byte, 0, len(operation.template))
	for i := 0; i < len(operation.template); i++ {
		ch := operation.template[i]
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			compact = append(compact, ch)
		}
	}
	const prefix = "outb%al,$"
	if len(compact) <= len(prefix) || !textEquals(compact[:len(prefix)], prefix) {
		return 0, false
	}
	digits, base := compact[len(prefix):], 10
	if len(digits) > 2 && digits[0] == '0' && (digits[1] == 'x' || digits[1] == 'X') {
		digits, base = digits[2:], 16
	}
	if len(digits) == 0 {
		return 0, false
	}
	port := 0
	for i := 0; i < len(digits); i++ {
		digit := -1
		if digits[i] >= '0' && digits[i] <= '9' {
			digit = int(digits[i] - '0')
		} else if base == 16 && digits[i] >= 'a' && digits[i] <= 'f' {
			digit = int(digits[i]-'a') + 10
		} else if base == 16 && digits[i] >= 'A' && digits[i] <= 'F' {
			digit = int(digits[i]-'A') + 10
		}
		if digit < 0 || digit >= base {
			return 0, false
		}
		port = port*base + digit
		if port > 0xffff {
			return 0, false
		}
	}
	return port, true
}

func (t *translator) emitAsmStringPortIO(operation cAsm) bool {
	width, output := 0, false
	for index, suffix := range []string{"b", "w", "l"} {
		if t.asmTemplateCompactPrefix(operation.template, "rep;outs"+suffix) ||
			t.asmTemplateCompactPrefix(operation.template, "cld;repouts"+suffix) {
			width, output = []int{1, 2, 4}[index], true
			break
		}
		if t.asmTemplateCompactPrefix(operation.template, "rep;ins"+suffix) ||
			t.asmTemplateCompactPrefix(operation.template, "cld;repins"+suffix) {
			width = []int{1, 2, 4}[index]
			break
		}
	}
	pointerConstraint := "+D"
	if output {
		pointerConstraint = "+S"
	}
	modern := len(operation.outputs) == 2 && len(operation.inputs) == 1 &&
		operation.outputs[0].constraint == pointerConstraint && operation.outputs[1].constraint == "+c" &&
		operation.inputs[0].constraint == "d"
	legacyPointer := "=D"
	if output {
		legacyPointer = "=S"
	}
	legacy := len(operation.outputs) == 2 && len(operation.inputs) == 3 &&
		operation.outputs[0].constraint == legacyPointer && operation.outputs[1].constraint == "=c" &&
		operation.inputs[0].constraint == "d" && operation.inputs[1].constraint == "0" &&
		operation.inputs[2].constraint == "1"
	portSize := 0
	if modern || legacy {
		portSize = t.typeSize(t.expressionType(operation.inputs[0].expression))
	}
	if width == 0 || (!modern && !legacy) || (portSize != 2 && portSize != 4) {
		return false
	}
	pointerType := t.expressionType(operation.outputs[0].expression)
	if t.typeInfo(pointerType).kind != cTypePointer {
		return false
	}
	name := "renvo_runtime_CPortInString"
	key := width
	if output {
		name = "renvo_runtime_CPortOutString"
		key += 4
	}
	name += decimalString(width * 8)
	if t.asmStringIOHelpers&(1<<key) == 0 {
		t.asmStringIOHelpers |= 1 << key
		t.staticOut = append(t.staticOut, ("func " + name + "(port uint16,address uintptr,count uintptr){}\n")...)
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	t.appendText("{__c_io_address_")
	t.appendText(suffix)
	t.appendText(":=")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(";__c_io_count_")
	t.appendText(suffix)
	t.appendText(":=uintptr(")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(");__c_io_port_")
	t.appendText(suffix)
	t.appendText(":=uint16(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(");")
	t.appendText(name)
	t.appendText("(__c_io_port_")
	t.appendText(suffix)
	t.appendText(",uintptr(__c_unsafe.Pointer(__c_io_address_")
	t.appendText(suffix)
	t.appendText(")),__c_io_count_")
	t.appendText(suffix)
	t.appendText(");")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=(")
	t.emitType(pointerType)
	t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(__c_io_address_")
	t.appendText(suffix)
	t.appendText("))+__c_io_count_")
	t.appendText(suffix)
	t.appendText("*")
	t.appendDecimal(width)
	t.appendText("));")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=0;};")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmSyscall(operation cAsm) bool {
	if string(operation.template) != "syscall" || len(operation.outputs) < 1 || len(operation.outputs) > 2 ||
		operation.outputs[0].constraint != "=a" || len(operation.inputs) < 3 || len(operation.inputs) > 4 ||
		operation.inputs[0].constraint != "0" || operation.inputs[1].constraint != "D" ||
		operation.inputs[2].constraint != "S" || len(operation.inputs) == 4 && operation.inputs[3].constraint != "d" {
		return false
	}
	if len(operation.outputs) == 2 && operation.outputs[1].constraint != "=m" {
		return false
	}
	if t.typeSize(t.expressionType(operation.outputs[0].expression)) != 8 {
		return false
	}
	if !t.asmSyscallHelper {
		t.asmSyscallHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CVDSOSyscall(number uint64,arg0 uint64,arg1 uint64,arg2 uint64) int64{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=int64(renvo_runtime_CVDSOSyscall(")
	for i := 0; i < 4; i++ {
		if i > 0 {
			t.appendText(",")
		}
		if i >= len(operation.inputs) {
			t.appendText("0")
			continue
		}
		expression := operation.inputs[i].expression
		if t.typeInfo(t.expressionType(expression)).kind == cTypePointer {
			t.appendText("uint64(uintptr(__c_unsafe.Pointer(")
			t.emitExpression(expression)
			t.appendText(")))")
			t.usesUnsafe = true
		} else {
			t.appendText("uint64(")
			t.emitExpression(expression)
			t.appendText(")")
		}
	}
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmInterruptWithStack(operation cAsm) bool {
	if !t.asmTemplateCompactContains(operation.template, "mov%%esp,%%ebx") ||
		!t.asmTemplateCompactContains(operation.template, "mov%3,%%esp") ||
		!t.asmTemplateCompactContains(operation.template, "int%2") ||
		!t.asmTemplateCompactContains(operation.template, "mov%%ebx,%%esp") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=a" ||
		len(operation.inputs) != 3 || operation.inputs[0].constraint != "a" ||
		operation.inputs[1].constraint != "n" || operation.inputs[2].constraint != "c" ||
		len(operation.clobbers) != 1 || operation.clobbers[0] != "ebx" ||
		t.pointerSize != 4 || t.typeInfo(t.expressionType(operation.inputs[2].expression)).kind != cTypePointer {
		return false
	}
	if !t.asmInterruptStackHelper {
		t.asmInterruptStackHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CInterruptWithStack(number int32,vector uint8,stack uintptr) int32{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CInterruptWithStack(int32(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("),uint8(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("),uintptr(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText(")));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmCPUID(operation cAsm) bool {
	if string(operation.template) != "cpuid" {
		return false
	}
	if len(operation.outputs) == 2 && len(operation.inputs) == 0 &&
		operation.outputs[0].constraint == "+a" && operation.outputs[1].constraint == "=d" {
		if t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 ||
			t.typeSize(t.expressionType(operation.outputs[1].expression)) != 4 {
			return false
		}
		t.ensureAsmCPUIDHelper()
		t.typeSerial++
		ignoredB := "__c_cpuid_b_" + decimalString(t.typeSerial)
		ignoredC := "__c_cpuid_c_" + decimalString(t.typeSerial)
		t.appendText("var ")
		t.appendText(ignoredB)
		t.appendText(" uint32;var ")
		t.appendText(ignoredC)
		t.appendText(" uint32;renvo_runtime_CCPUID(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText(",0,&(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("),&")
		t.appendText(ignoredB)
		t.appendText(",&")
		t.appendText(ignoredC)
		t.appendText(",&(")
		t.emitExpression(operation.outputs[1].expression)
		t.appendText("));")
		return t.ok
	}
	if len(operation.outputs) != 4 || len(operation.inputs) != 2 ||
		operation.outputs[0].constraint != "=a" || operation.outputs[1].constraint != "=b" ||
		operation.outputs[2].constraint != "=c" || operation.outputs[3].constraint != "=d" ||
		operation.inputs[0].constraint != "0" || operation.inputs[1].constraint != "2" {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		if t.typeSize(t.expressionType(operation.outputs[i].expression)) != 4 {
			return false
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 4 {
			return false
		}
	}
	t.ensureAsmCPUIDHelper()
	t.appendText("renvo_runtime_CCPUID(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(",")
	t.emitExpression(operation.inputs[1].expression)
	for i := 0; i < len(operation.outputs); i++ {
		t.appendText(",")
		if pointer, ok := t.asmDereferencePointer(operation.outputs[i].expression); ok {
			t.emitExpression(pointer)
		} else {
			t.appendText("&(")
			t.emitExpression(operation.outputs[i].expression)
			t.appendText(")")
		}
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmXControl(operation cAsm) bool {
	if t.asmTemplateCompactPrefix(operation.template, "xgetbv") {
		if len(operation.outputs) != 2 || len(operation.inputs) != 1 || len(operation.clobbers) != 0 ||
			operation.outputs[0].constraint != "=a" || operation.outputs[1].constraint != "=d" || operation.inputs[0].constraint != "c" {
			return false
		}
		for i := 0; i < len(operation.outputs); i++ {
			if t.typeSize(t.expressionType(operation.outputs[i].expression)) != 4 {
				return false
			}
		}
		if t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 {
			return false
		}
		if t.asmXControl&1 == 0 {
			t.asmXControl |= 1
			t.staticOut = append(t.staticOut, "func renvo_runtime_CReadXControl(index uint32) uint64{return 0}\n"...)
		}
		t.typeSerial++
		value := "__c_xcontrol_" + decimalString(t.typeSerial)
		t.appendText("var ")
		t.appendText(value)
		t.appendText(" uint64=renvo_runtime_CReadXControl(uint32(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=uint32(")
		t.appendText(value)
		t.appendText(");")
		t.emitExpression(operation.outputs[1].expression)
		t.appendText("=uint32(")
		t.appendText(value)
		t.appendText(">>32);")
		return t.ok
	}
	if !t.asmTemplateCompactPrefix(operation.template, "xsetbv") || len(operation.outputs) != 0 || len(operation.inputs) != 3 || len(operation.clobbers) != 0 ||
		operation.inputs[0].constraint != "a" || operation.inputs[1].constraint != "d" || operation.inputs[2].constraint != "c" {
		return false
	}
	for i := 0; i < len(operation.inputs); i++ {
		if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 4 {
			return false
		}
	}
	if t.asmXControl&2 == 0 {
		t.asmXControl |= 2
		t.staticOut = append(t.staticOut, "func renvo_runtime_CWriteXControl(index uint32,value uint64){}\n"...)
	}
	t.appendText("renvo_runtime_CWriteXControl(uint32(")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText("),uint64(uint32(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("))|(uint64(uint32(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("))<<32));")
	return t.ok
}

func (t *translator) emitAsmFPUState(operation cAsm) bool {
	opcode := ""
	switch {
	case t.asmTemplateCompactContains(operation.template, "fnsave%[") &&
		t.asmTemplateCompactContains(operation.template, "];fwait"):
		opcode = "FNSAVE"
	case t.asmTemplateCompactContains(operation.template, "fxsaveq%[fx]"):
		opcode = "FXSAVEQ"
	case t.asmTemplateCompactContains(operation.template, "fxsave%[fx]"):
		opcode = "FXSAVE"
	case t.asmTemplateCompactPrefix(operation.template, "fxsaveq%0"):
		opcode = "FXSAVEQ"
	case t.asmTemplateCompactPrefix(operation.template, "fxsave%0"):
		opcode = "FXSAVE"
	case t.asmTemplateCompactContains(operation.template, "fxrstorq%[fx]"):
		opcode = "FXRSTORQ"
	case t.asmTemplateCompactContains(operation.template, "fxrstor%[fx]"):
		opcode = "FXRSTOR"
	case t.asmTemplateCompactContains(operation.template, "frstor%[fx]"):
		opcode = "FRSTOR"
	case t.asmTemplateCompactPrefix(operation.template, "ldmxcsr%0"):
		opcode = "LDMXCSR"
	}
	if opcode == "" {
		return false
	}
	mode, memory, result := "Plain", cAsmOperand{}, -1
	if opcode == "LDMXCSR" {
		if len(operation.outputs) != 0 || len(operation.inputs) != 1 || operation.inputs[0].constraint != "m" {
			return false
		}
		memory = operation.inputs[0]
	} else if len(operation.outputs) == 1 && len(operation.inputs) == 0 &&
		(operation.outputs[0].constraint == "=m" || operation.outputs[0].constraint == "+m") {
		memory = operation.outputs[0]
	} else if len(operation.outputs) == 1 && len(operation.inputs) == 1 && operation.outputs[0].constraint == "=m" && operation.inputs[0].constraint == "m" {
		memory = operation.inputs[0]
		if t.asmTemplateCompactContains(operation.template, ".long6") {
			mode = "Fault6"
		} else {
			return false
		}
	} else if len(operation.outputs) == 2 && len(operation.inputs) == 2 {
		for i := 0; i < len(operation.outputs); i++ {
			if operation.outputs[i].name == "err" && (operation.outputs[i].constraint == "=a" || operation.outputs[i].constraint == "=r") {
				result = i
			} else if operation.outputs[i].constraint == "=m" {
				memory = operation.outputs[i]
			}
		}
		if result < 0 || memory.expression == nil || operation.inputs[0].constraint != "0" || operation.inputs[1].constraint != "m" ||
			t.typeSize(t.expressionType(operation.outputs[result].expression)) != 4 {
			return false
		}
		if t.asmTemplateCompactContains(operation.template, ".long15") {
			mode = "User15"
		} else if t.asmTemplateCompactContains(operation.template, "extable_type_regreg=%[err],type=(17|((-14)<<16))") {
			mode = "Safe17"
		} else {
			return false
		}
	} else {
		return false
	}
	name := "renvo_runtime_CFPUState_" + opcode + "_" + mode
	found := false
	for i := 0; i < len(t.asmFPUStateHelpers); i++ {
		found = found || t.asmFPUStateHelpers[i] == name
	}
	if !found {
		t.asmFPUStateHelpers = append(t.asmFPUStateHelpers, name)
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr)")...)
		if result >= 0 {
			t.staticOut = append(t.staticOut, " int32{return 0}\n"...)
		} else {
			t.staticOut = append(t.staticOut, "{}\n"...)
		}
	}
	if result >= 0 {
		t.emitExpression(operation.outputs[result].expression)
		t.appendText("=")
	}
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	if pointer, ok := t.asmDereferencePointer(memory.expression); ok {
		t.emitExpression(pointer)
	} else {
		t.appendText("&(")
		t.emitExpression(memory.expression)
		t.appendText(")")
	}
	t.appendText(")));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmExtendedState(operation cAsm) bool {
	opcode := ""
	if t.asmTemplateCompactContains(operation.template, "0x0f,0xae,0x27") {
		opcode = "SAVE"
	} else if t.asmTemplateCompactContains(operation.template, "0x0f,0xae,0x2f") {
		opcode = "RESTORE"
	} else if t.asmTemplateCompactContains(operation.template, "0x0f,0xc7,0x2f") {
		opcode = "SAVES"
	} else if t.asmTemplateCompactContains(operation.template, "0x0f,0xc7,0x1f") {
		opcode = "RESTORES"
	} else {
		return false
	}
	result := -1
	if len(operation.outputs) == 1 && operation.outputs[0].name == "err" &&
		(operation.outputs[0].constraint == "=r" || operation.outputs[0].constraint == "=a") {
		result = 0
	} else if len(operation.outputs) != 0 {
		return false
	}
	if len(operation.inputs) != 4 || operation.inputs[0].constraint != "D" || operation.inputs[1].constraint != "m" ||
		operation.inputs[2].constraint != "a" || operation.inputs[3].constraint != "d" ||
		t.typeInfo(t.expressionType(operation.inputs[0].expression)).kind != cTypePointer ||
		t.typeSize(t.expressionType(operation.inputs[2].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.inputs[3].expression)) != 4 {
		return false
	}
	mode := "Fault6"
	if result >= 0 {
		if t.typeSize(t.expressionType(operation.outputs[result].expression)) != 4 {
			return false
		}
		if t.asmTemplateCompactContains(operation.template, "extable_type_regreg=%[err],type=(17|((-14)<<16))") {
			mode = "Safe17"
		} else if t.asmTemplateCompactContains(operation.template, ".long15") {
			mode = "User15"
		} else {
			return false
		}
	} else if !t.asmTemplateCompactContains(operation.template, ".long6") {
		return false
	}
	name := "renvo_runtime_CExtendedState_" + opcode + "_" + mode
	found := false
	for i := 0; i < len(t.asmFPUStateHelpers); i++ {
		found = found || t.asmFPUStateHelpers[i] == name
	}
	if !found {
		t.asmFPUStateHelpers = append(t.asmFPUStateHelpers, name)
		t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,mask uint64)")...)
		if result >= 0 {
			t.staticOut = append(t.staticOut, " int32{return 0}\n"...)
		} else {
			t.staticOut = append(t.staticOut, "{}\n"...)
		}
	}
	if result >= 0 {
		t.emitExpression(operation.outputs[result].expression)
		t.appendText("=")
	}
	t.appendText(name)
	t.appendText("(uintptr(__c_unsafe.Pointer(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")),uint64(uint32(")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText("))|(uint64(uint32(")
	t.emitExpression(operation.inputs[3].expression)
	t.appendText("))<<32));")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmFPUInit(operation cAsm) bool {
	if t.asmTemplateCompactContains(operation.template, "fwait") &&
		t.asmTemplateCompactContains(operation.template, ".pushsection\"__ex_table\"") &&
		t.asmTemplateCompactContains(operation.template, ".long1") {
		if len(operation.outputs) != 0 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 {
			return false
		}
		if !t.asmFPUWait {
			t.asmFPUWait = true
			t.staticOut = append(t.staticOut, "func renvo_runtime_CFPUWait___ex_table(){}\n"...)
		}
		t.appendText("renvo_runtime_CFPUWait___ex_table();")
		return true
	}
	if t.asmTemplateCompactPrefix(operation.template, "fnclexemmsfildl%[addr]") {
		if len(operation.outputs) != 0 || len(operation.inputs) != 1 || len(operation.clobbers) != 0 ||
			operation.inputs[0].name != "addr" || operation.inputs[0].constraint != "m" {
			return false
		}
		pointer, ok := t.asmDereferencePointer(operation.inputs[0].expression)
		if !ok {
			return false
		}
		if !t.asmFPUPrepare {
			t.asmFPUPrepare = true
			t.staticOut = append(t.staticOut, "func renvo_runtime_CFPUPrepare(address uintptr){}\n"...)
		}
		t.appendText("renvo_runtime_CFPUPrepare(uintptr(__c_unsafe.Pointer(")
		t.emitExpression(pointer)
		t.appendText(")));")
		t.usesUnsafe = true
		return t.ok
	}
	if t.asmTemplateCompactPrefix(operation.template, "fninit") &&
		!t.asmTemplateCompactContains(operation.template, "fnstsw") {
		if len(operation.outputs) != 0 || len(operation.inputs) != 0 || len(operation.clobbers) != 0 {
			return false
		}
		t.ensureAsmFPUHelper()
		t.appendText("_=renvo_runtime_CInitFPU();")
		return true
	}
	if !t.asmTemplateCompactPrefix(operation.template, "fninit;fnstsw%0;fnstcw%1") ||
		len(operation.outputs) != 2 || len(operation.inputs) != 0 ||
		operation.outputs[0].constraint != "+m" || operation.outputs[1].constraint != "+m" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 2 ||
		t.typeSize(t.expressionType(operation.outputs[1].expression)) != 2 {
		return false
	}
	t.ensureAsmFPUHelper()
	t.typeSerial++
	value := "__c_asm_fpu_" + decimalString(t.typeSerial)
	t.appendText("var ")
	t.appendText(value)
	t.appendText(" uint32=renvo_runtime_CInitFPU();")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=uint16(")
	t.appendText(value)
	t.appendText(");")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=uint16(")
	t.appendText(value)
	t.appendText(">>16);")
	return t.ok
}

func (t *translator) emitAsmMSR(operation cAsm) bool {
	if t.asmTemplateCompactContains(operation.template, "rdmsr") && len(operation.outputs) == 3 {
		errorOutput, lowOutput, highOutput := -1, -1, -1
		for i := 0; i < len(operation.outputs); i++ {
			switch operation.outputs[i].constraint {
			case "=r":
				if operation.outputs[i].name == "err" {
					errorOutput = i
				}
			case "=a":
				lowOutput = i
			case "=d":
				highOutput = i
			}
		}
		if errorOutput < 0 || lowOutput < 0 || highOutput < 0 || len(operation.inputs) != 1 ||
			operation.inputs[0].constraint != "c" || t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 ||
			t.typeSize(t.expressionType(operation.outputs[errorOutput].expression)) != 4 {
			return false
		}
		for _, index := range []int{lowOutput, highOutput} {
			size := t.typeSize(t.expressionType(operation.outputs[index].expression))
			if size != 4 && size != 8 {
				return false
			}
		}
		t.ensureAsmMSRHelper()
		t.typeSerial++
		temporary := "__c_msr_safe_word_" + decimalString(t.typeSerial)
		t.appendText("var ")
		t.appendText(temporary)
		t.appendText("_low uint32;var ")
		t.appendText(temporary)
		t.appendText("_high uint32;renvo_runtime_CReadMSRSafe(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(",")
		if pointer, ok := t.asmDereferencePointer(operation.outputs[errorOutput].expression); ok {
			t.emitExpression(pointer)
		} else {
			t.appendText("&(")
			t.emitExpression(operation.outputs[errorOutput].expression)
			t.appendText(")")
		}
		t.appendText(",&")
		t.appendText(temporary)
		t.appendText("_low,&")
		t.appendText(temporary)
		t.appendText("_high);")
		for at, index := range []int{lowOutput, highOutput} {
			t.emitExpression(operation.outputs[index].expression)
			t.appendText("=")
			t.emitType(t.expressionType(operation.outputs[index].expression))
			t.appendText("(")
			t.appendText(temporary)
			if at == 0 {
				t.appendText("_low);")
			} else {
				t.appendText("_high);")
			}
		}
		return t.ok
	}
	if t.asmTemplateCompactContains(operation.template, "rdmsr") && len(operation.outputs) == 2 {
		if len(operation.outputs) != 2 || len(operation.inputs) != 1 ||
			operation.outputs[0].constraint != "=a" || operation.outputs[1].constraint != "=d" ||
			operation.inputs[0].constraint != "c" {
			return false
		}
		wide := false
		for i := 0; i < len(operation.outputs); i++ {
			size := t.typeSize(t.expressionType(operation.outputs[i].expression))
			if size != 4 && size != 8 {
				return false
			}
			wide = wide || size == 8
		}
		if t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 {
			return false
		}
		t.ensureAsmMSRHelper()
		if wide {
			t.typeSerial++
			temporary := "__c_msr_word_" + decimalString(t.typeSerial)
			t.appendText("var ")
			t.appendText(temporary)
			t.appendText("_low uint32;var ")
			t.appendText(temporary)
			t.appendText("_high uint32;")
			t.appendText("renvo_runtime_CReadMSR(")
			t.emitExpression(operation.inputs[0].expression)
			t.appendText(",&")
			t.appendText(temporary)
			t.appendText("_low,&")
			t.appendText(temporary)
			t.appendText("_high);")
			for i := 0; i < len(operation.outputs); i++ {
				t.emitExpression(operation.outputs[i].expression)
				t.appendText("=")
				t.emitType(t.expressionType(operation.outputs[i].expression))
				t.appendText("(")
				t.appendText(temporary)
				if i == 0 {
					t.appendText("_low);")
				} else {
					t.appendText("_high);")
				}
			}
			return t.ok
		}
		t.appendText("renvo_runtime_CReadMSR(")
		t.emitExpression(operation.inputs[0].expression)
		for i := 0; i < len(operation.outputs); i++ {
			t.appendText(",&(")
			t.emitExpression(operation.outputs[i].expression)
			t.appendText(")")
		}
		t.appendText(");")
		return t.ok
	}
	if t.asmTemplateCompactContains(operation.template, "wrmsr") && len(operation.outputs) == 1 {
		if len(operation.inputs) != 3 || operation.outputs[0].name != "err" || operation.outputs[0].constraint != "=a" ||
			operation.inputs[0].constraint != "c" || operation.inputs[1].constraint != "0" || operation.inputs[2].constraint != "d" {
			return false
		}
		for i := 0; i < len(operation.inputs); i++ {
			if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 4 {
				return false
			}
		}
		t.ensureAsmMSRHelper()
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CWriteMSRSafe(")
		for i := 0; i < len(operation.inputs); i++ {
			if i != 0 {
				t.appendText(",")
			}
			t.emitExpression(operation.inputs[i].expression)
		}
		t.appendText(");")
		return t.ok
	}
	if !t.asmTemplateCompactContains(operation.template, "wrmsr") || len(operation.outputs) != 0 || len(operation.inputs) < 3 ||
		operation.inputs[0].constraint != "c" || operation.inputs[1].constraint != "a" || operation.inputs[2].constraint != "d" {
		return false
	}
	for i := 0; i < 3; i++ {
		if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 4 {
			return false
		}
	}
	for i := 3; i < len(operation.inputs); i++ {
		if operation.inputs[i].constraint != "i" {
			return false
		}
	}
	t.ensureAsmMSRHelper()
	t.appendText("renvo_runtime_CWriteMSR(")
	for i := 0; i < 3; i++ {
		if i > 0 {
			t.appendText(",")
		}
		t.emitExpression(operation.inputs[i].expression)
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmCounter(operation cAsm) bool {
	kind := 0
	name := ""
	if t.asmTemplateCompactContains(operation.template, "rdpmc") {
		kind, name = 3, "renvo_runtime_CReadPerformanceCounter"
	} else if t.asmTemplateCompactContains(operation.template, "lfence;rdtsc") ||
		t.asmTemplateCompactContains(operation.template, "rdtscp") {
		kind, name = 2, "renvo_runtime_CReadTimestampCounterOrdered"
	} else if t.asmTemplateCompactPrefix(operation.template, "rdtsc") {
		kind, name = 1, "renvo_runtime_CReadTimestampCounter"
	}
	if kind == 0 || len(operation.outputs) != 2 || operation.outputs[0].constraint != "=a" ||
		operation.outputs[1].constraint != "=d" {
		return false
	}
	if kind == 3 {
		if len(operation.inputs) != 1 || operation.inputs[0].constraint != "c" ||
			t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 {
			return false
		}
	} else if len(operation.inputs) != 0 {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		size := t.typeSize(t.expressionType(operation.outputs[i].expression))
		if size != 4 && size != 8 {
			return false
		}
	}
	if t.asmCounterOps&(1<<kind) == 0 {
		t.asmCounterOps |= 1 << kind
		if kind == 3 {
			t.staticOut = append(t.staticOut, ("func " + name + "(counter uint32,low *uint32,high *uint32){}\n")...)
		} else {
			t.staticOut = append(t.staticOut, ("func " + name + "(low *uint32,high *uint32){}\n")...)
		}
	}
	t.typeSerial++
	temporary := "__c_counter_word_" + decimalString(t.typeSerial)
	t.appendText("var ")
	t.appendText(temporary)
	t.appendText("_low uint32;var ")
	t.appendText(temporary)
	t.appendText("_high uint32;")
	t.appendText(name)
	t.appendText("(")
	if kind == 3 {
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(",")
	}
	t.appendText("&")
	t.appendText(temporary)
	t.appendText("_low,&")
	t.appendText(temporary)
	t.appendText("_high);")
	for i := 0; i < len(operation.outputs); i++ {
		t.emitExpression(operation.outputs[i].expression)
		t.appendText("=")
		t.emitType(t.expressionType(operation.outputs[i].expression))
		t.appendText("(")
		t.appendText(temporary)
		if i == 0 {
			t.appendText("_low);")
		} else {
			t.appendText("_high);")
		}
	}
	return t.ok
}

func (t *translator) emitAsmPause(operation cAsm) bool {
	if len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
		!t.asmTemplateCompactPrefix(operation.template, "pause") && !t.asmTemplateCompactPrefix(operation.template, "rep;nop") {
		return false
	}
	if !t.asmPauseHelper {
		t.asmPauseHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CPause(){}\n"...)
	}
	t.appendText("renvo_runtime_CPause();")
	return true
}

func (t *translator) emitAsmDelayLoop(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "test%0,%0jz3fjmp1f.align161:jmp2f.align162:dec%0jnz2b3:dec%0") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "+a" || len(operation.inputs) != 0 || len(operation.clobbers) != 0 ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != t.pointerSize {
		return false
	}
	if !t.asmDelayLoop {
		t.asmDelayLoop = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CDelayLoop(iterations uint64) uint64{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CDelayLoop(uint64(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmMonitorWait(operation cAsm) bool {
	kind := 0
	name := ""
	if t.asmTemplateCompactPrefix(operation.template, ".byte0x0f,0x01,0xc8") {
		kind, name = 1, "renvo_runtime_CMonitor"
	} else if t.asmTemplateCompactPrefix(operation.template, ".byte0x0f,0x01,0xfa") {
		kind, name = 2, "renvo_runtime_CMonitorExtended"
	} else if t.asmTemplateCompactPrefix(operation.template, ".byte0x0f,0x01,0xc9") {
		kind, name = 3, "renvo_runtime_CWait"
	} else if t.asmTemplateCompactPrefix(operation.template, ".byte0x0f,0x01,0xfb") {
		kind, name = 4, "renvo_runtime_CWaitExtended"
	} else if t.asmTemplateCompactPrefix(operation.template, "sti;.byte0x0f,0x01,0xc9") {
		kind, name = 5, "renvo_runtime_CEnableInterruptsAndWait"
	} else if t.asmTemplateCompactPrefix(operation.template, ".byte0x66,0x0f,0xae,0xf1") ||
		t.asmTemplateCompactPrefix(operation.template, "tpause%%ecx") {
		kind, name = 6, "renvo_runtime_CTimedPause"
	}
	if kind == 0 || len(operation.outputs) != 0 || len(operation.clobbers) != 0 {
		return false
	}
	monitor := kind <= 2
	expectedInputs := 3
	if kind == 3 || kind == 5 {
		expectedInputs = 2
	}
	if len(operation.inputs) != expectedInputs || kind != 6 && operation.inputs[0].constraint != "a" {
		return false
	}
	if monitor {
		if operation.inputs[1].constraint != "c" || operation.inputs[2].constraint != "d" ||
			t.typeInfo(t.expressionType(operation.inputs[0].expression)).kind != cTypePointer {
			return false
		}
	} else if kind == 3 || kind == 5 {
		if operation.inputs[1].constraint != "c" {
			return false
		}
	} else if kind == 6 {
		if operation.inputs[0].constraint != "c" || operation.inputs[1].constraint != "d" || operation.inputs[2].constraint != "a" {
			return false
		}
	} else if operation.inputs[1].constraint != "b" || operation.inputs[2].constraint != "c" {
		return false
	}
	bit := 1 << kind
	if t.asmMonitorWait&bit == 0 {
		t.asmMonitorWait |= bit
		if monitor {
			t.staticOut = append(t.staticOut, ("func " + name + "(address uintptr,extensions uint64,hints uint64){}\n")...)
		} else if kind == 3 || kind == 5 {
			t.staticOut = append(t.staticOut, ("func " + name + "(extensions uint64,hints uint64){}\n")...)
		} else {
			t.staticOut = append(t.staticOut, ("func " + name + "(extensions uint64,counter uint64,hints uint64){}\n")...)
		}
	}
	t.appendText(name)
	t.out = append(t.out, '(')
	for i := 0; i < len(operation.inputs); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		if monitor && i == 0 {
			t.appendText("uintptr(__c_unsafe.Pointer(")
			t.emitExpression(operation.inputs[i].expression)
			t.appendText("))")
			t.usesUnsafe = true
		} else {
			t.appendText("uint64(")
			t.emitExpression(operation.inputs[i].expression)
			t.out = append(t.out, ')')
		}
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmCPUNODE(operation cAsm) bool {
	if len(operation.outputs) != 1 || len(operation.inputs) != 2 ||
		operation.outputs[0].name != "p" || operation.outputs[0].constraint != "=a" ||
		operation.inputs[0].constraint != "i" || operation.inputs[1].name != "seg" || operation.inputs[1].constraint != "r" ||
		!t.asmTemplateCompactContains(operation.template, "lsl%[seg],%[p]") {
		return false
	}
	zero, ok := t.constantExpression(operation.inputs[0].expression)
	if !ok || zero != 0 || t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 {
		return false
	}
	if !t.asmCPUNODEHelper {
		t.asmCPUNODEHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CReadCPUNODE(selector uint32) uint32{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CReadCPUNODE(uint32(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmFarCall16(operation cAsm) bool {
	farCall := t.asmTemplateCompactPrefix(operation.template, "lcallw*%0") && len(operation.inputs) == 1
	farJump := t.asmTemplateCompactPrefix(operation.template, "ljmpl*%0") && len(operation.inputs) == 2
	if len(operation.outputs) != 0 || !farCall && !farJump || operation.inputs[0].constraint != "m" ||
		farJump && operation.inputs[1].constraint != "D" {
		return false
	}
	if farJump {
		name := "renvo_runtime_CFarJump32"
		if !t.asmFarJumpHelper {
			t.asmFarJumpHelper = true
			t.staticOut = append(t.staticOut, "func renvo_runtime_CFarJump32(address uintptr,value uintptr){}\n"...)
		}
		t.appendText(name)
		t.appendText("(uintptr(&(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(")),uintptr(")
		t.emitExpression(operation.inputs[1].expression)
		t.appendText("));")
		return t.ok
	}
	if !t.asmFarCallHelper {
		t.asmFarCallHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CFarCall16(address uintptr){}\n"...)
	}
	t.appendText("renvo_runtime_CFarCall16(uintptr(&(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(")));")
	return t.ok
}

func (t *translator) emitAsmAccessRights(operation cAsm) bool {
	if !t.asmTemplateCompactPrefix(operation.template, "lar%[old_ss],%[ar]jz1fxorl%[ar],%[ar]1:") ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" ||
		len(operation.inputs) != 1 || operation.inputs[0].constraint != "rm" ||
		t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 ||
		t.typeSize(t.expressionType(operation.inputs[0].expression)) != 2 {
		return false
	}
	if !t.asmAccessRightsHelper {
		t.asmAccessRightsHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CLoadAccessRights(selector uint16) uint32{return 0}\n"...)
	}
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CLoadAccessRights(uint16(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText("));")
	return t.ok
}

func (t *translator) emitAsmInterruptFlag(operation cAsm) bool {
	write := -1
	if t.asmTemplateCompactPrefix(operation.template, "cli") {
		write = 0
	} else if t.asmTemplateCompactPrefix(operation.template, "sti") {
		write = 1
	}
	if write < 0 || len(operation.outputs) != 0 || len(operation.inputs) != 0 {
		return false
	}
	bit := 1 << write
	if t.asmInterruptFlags&bit == 0 {
		t.asmInterruptFlags |= bit
		name := "renvo_runtime_CDisableInterrupts"
		if write != 0 {
			name = "renvo_runtime_CEnableInterrupts"
		}
		t.staticOut = append(t.staticOut, ("func " + name + "(){}\n")...)
	}
	if write == 0 {
		t.appendText("renvo_runtime_CDisableInterrupts();")
	} else {
		t.appendText("renvo_runtime_CEnableInterrupts();")
	}
	return true
}

func (t *translator) emitAsmDescriptorTable(operation cAsm) bool {
	kind := 0
	name := ""
	store := false
	directPointer := false
	if t.asmTemplateCompactPrefix(operation.template, "lgdt(%0)") {
		kind, name, directPointer = 1, "renvo_runtime_CLoadGDT", true
	} else if t.asmTemplateCompactPrefix(operation.template, "lidt(%0)") {
		kind, name, directPointer = 2, "renvo_runtime_CLoadIDT", true
	} else if t.asmTemplateCompactPrefix(operation.template, "lgdtl%0") || t.asmTemplateCompactPrefix(operation.template, "lgdt%0") {
		kind, name = 1, "renvo_runtime_CLoadGDT"
	} else if t.asmTemplateCompactPrefix(operation.template, "lidtl%0") || t.asmTemplateCompactPrefix(operation.template, "lidt%0") {
		kind, name = 2, "renvo_runtime_CLoadIDT"
	} else if t.asmTemplateCompactPrefix(operation.template, "lldt%w0") || t.asmTemplateCompactPrefix(operation.template, "lldt%0") {
		kind, name = 4, "renvo_runtime_CLoadLDT"
	} else if t.asmTemplateCompactPrefix(operation.template, "sgdt%0") {
		kind, name, store = 8, "renvo_runtime_CStoreGDT", true
	} else if t.asmTemplateCompactPrefix(operation.template, "sidt%0") {
		kind, name, store = 16, "renvo_runtime_CStoreIDT", true
	} else if t.asmTemplateCompactPrefix(operation.template, "ltr%w0") || t.asmTemplateCompactPrefix(operation.template, "ltr%0") {
		kind, name = 32, "renvo_runtime_CLoadTR"
	} else if t.asmTemplateCompactPrefix(operation.template, "str%0") {
		kind, name, store = 64, "renvo_runtime_CStoreTR", true
	} else if t.asmTemplateCompactPrefix(operation.template, "sldt%0") {
		kind, name, store = 128, "renvo_runtime_CStoreLDT", true
	}
	registerStore := kind == 64 || kind == 128
	registerLoad := kind == 32
	if kind == 0 || store && (len(operation.outputs) != 1 ||
		(registerStore && operation.outputs[0].constraint != "=r" && operation.outputs[0].constraint != "=g" && operation.outputs[0].constraint != "=m" || !registerStore && operation.outputs[0].constraint != "=m") || len(operation.inputs) != 0) ||
		!store && (len(operation.outputs) != 0 || len(operation.inputs) != 1 ||
			(kind == 4 || registerLoad) && operation.inputs[0].constraint != "q" && operation.inputs[0].constraint != "rm" && operation.inputs[0].constraint != "r" ||
			kind != 4 && !registerLoad && !directPointer && operation.inputs[0].constraint != "m" ||
			directPointer && operation.inputs[0].constraint != "r") {
		return false
	}
	if t.asmDescriptorOps&kind == 0 {
		t.asmDescriptorOps |= kind
		declaration := "func " + name + "(address uintptr){}\n"
		if registerStore {
			declaration = "func " + name + "() uint64{return 0}\n"
		} else if kind == 4 || registerLoad {
			declaration = "func " + name + "(value uint16){}\n"
		}
		t.staticOut = append(t.staticOut, declaration...)
	}
	if registerStore {
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=")
		t.emitType(t.expressionType(operation.outputs[0].expression))
		t.appendText("(")
		t.appendText(name)
		t.appendText("());")
	} else if store {
		t.appendText(name)
		t.appendText("(uintptr(&(")
		t.emitExpression(operation.outputs[0].expression)
		t.appendText(")));")
	} else if kind == 4 || registerLoad {
		t.appendText(name)
		t.appendText("(uint16(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText("));")
	} else if directPointer {
		t.appendText(name)
		t.appendText("(uintptr(__c_unsafe.Pointer(")
		pointerType := t.decayedExpressionType(operation.inputs[0].expression)
		t.convertedExpression(pointerType, operation.inputs[0].expression)
		t.appendText(")));")
		t.usesUnsafe = true
	} else {
		t.appendText(name)
		t.appendText("(uintptr(&(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(")));")
	}
	return t.ok
}

func (t *translator) emitAsmEFlagsToggle(operation cAsm) bool {
	if len(operation.outputs) != 2 || len(operation.inputs) != 1 ||
		operation.outputs[0].constraint != "=&r" || operation.outputs[1].constraint != "=&r" ||
		operation.inputs[0].constraint != "ri" ||
		(!t.asmTemplateCompactPrefix(operation.template, "pushflpushflpop%0mov%0,%1xor%2,%1push%1popflpushflpop%1popfl") &&
			!t.asmTemplateCompactPrefix(operation.template, "pushfqpushfqpop%0mov%0,%1xor%2,%1push%1popfqpushfqpop%1popfq")) {
		return false
	}
	typeID := t.expressionType(operation.outputs[0].expression)
	if t.typeSize(typeID) != t.pointerSize || t.typeSize(t.expressionType(operation.outputs[1].expression)) != t.pointerSize {
		return false
	}
	t.ensureAsmFlagsHelper(false, typeID)
	t.ensureAsmFlagsHelper(true, typeID)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CReadFlags();")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("^")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(";renvo_runtime_CWriteFlags(")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(");")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=renvo_runtime_CReadFlags();renvo_runtime_CWriteFlags(")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmControlRegister(operation cAsm) bool {
	register := -1
	for _, candidate := range []int{0, 1, 2, 3, 6, 7} {
		if register < 0 && t.asmTemplateCompactPrefix(operation.template, "mov%%db"+decimalString(candidate)+",%0") {
			register = candidate
		}
	}
	if register >= 0 {
		if len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" || len(operation.inputs) > 1 ||
			len(operation.inputs) == 1 && operation.inputs[0].constraint != "m" {
			return false
		}
		t.ensureAsmDebugHelper(register, false, t.expressionType(operation.outputs[0].expression))
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CReadDebug")
		t.appendDecimal(register)
		t.appendText("();")
		return t.ok
	}
	for _, candidate := range []int{0, 1, 2, 3, 6, 7} {
		if t.asmTemplateCompactPrefix(operation.template, "mov%0,%%db"+decimalString(candidate)) {
			register = candidate
			break
		}
	}
	if register >= 0 {
		if len(operation.outputs) != 0 || len(operation.inputs) < 1 || len(operation.inputs) > 2 ||
			operation.inputs[0].constraint != "r" || len(operation.inputs) == 2 && operation.inputs[1].constraint != "m" {
			return false
		}
		t.ensureAsmDebugHelper(register, true, t.expressionType(operation.inputs[0].expression))
		t.appendText("renvo_runtime_CWriteDebug")
		t.appendDecimal(register)
		t.appendText("(")
		t.emitExpression(operation.inputs[0].expression)
		t.appendText(");")
		return t.ok
	}
	for _, candidate := range []int{0, 2, 3, 4} {
		if register < 0 && (t.asmTemplateCompactPrefix(operation.template, "mov%%cr"+decimalString(candidate)+",%0") ||
			t.asmTemplateCompactPrefix(operation.template, "movl%%cr"+decimalString(candidate)+",%0")) {
			register = candidate
		}
	}
	if register >= 0 {
		if len(operation.outputs) != 1 || operation.outputs[0].constraint != "=r" || len(operation.inputs) > 1 ||
			len(operation.inputs) == 1 && operation.inputs[0].constraint != "m" {
			return false
		}
		t.ensureAsmControlHelper(register, false, t.expressionType(operation.outputs[0].expression))
		t.emitExpression(operation.outputs[0].expression)
		t.appendText("=renvo_runtime_CReadControl")
		t.appendDecimal(register)
		t.appendText("();")
		return t.ok
	}
	for _, candidate := range []int{0, 2, 3, 4} {
		if t.asmTemplateCompactPrefix(operation.template, "mov%0,%%cr"+decimalString(candidate)) ||
			t.asmTemplateCompactPrefix(operation.template, "movl%0,%%cr"+decimalString(candidate)) {
			register = candidate
			break
		}
	}
	var operand []token
	if register >= 0 && len(operation.outputs) == 0 && len(operation.inputs) == 1 && operation.inputs[0].constraint == "r" {
		operand = operation.inputs[0].expression
	} else if register >= 0 && len(operation.outputs) == 1 && operation.outputs[0].constraint == "+r" && len(operation.inputs) == 0 {
		operand = operation.outputs[0].expression
	}
	if register < 0 || len(operand) == 0 {
		return false
	}
	t.ensureAsmControlHelper(register, true, t.expressionType(operand))
	t.appendText("renvo_runtime_CWriteControl")
	t.appendDecimal(register)
	t.appendText("(")
	t.emitExpression(operand)
	t.appendText(");")
	return t.ok
}

func (t *translator) asmTemplateCompactPrefix(value []byte, compact string) bool {
	at := 0
	for i := 0; i < len(value) && at < len(compact); i++ {
		if value[i] == ' ' || value[i] == '\t' || value[i] == '\n' || value[i] == '\r' {
			continue
		}
		if value[i] != compact[at] {
			return false
		}
		at++
	}
	return at == len(compact)
}

func (t *translator) asmTemplateCompactContains(value []byte, compact string) bool {
	if len(compact) == 0 {
		return true
	}
	matched := 0
	for i := 0; i < len(value); i++ {
		if value[i] == ' ' || value[i] == '\t' || value[i] == '\n' || value[i] == '\r' {
			continue
		}
		if value[i] == compact[matched] {
			matched++
			if matched == len(compact) {
				return true
			}
			continue
		}
		if value[i] == compact[0] {
			matched = 1
		} else {
			matched = 0
		}
	}
	return false
}

func (t *translator) emitAsmDivide32(operation cAsm) bool {
	if string(operation.template) != "divl %2" || len(operation.outputs) != 2 || len(operation.inputs) != 3 ||
		operation.outputs[0].constraint != "=a" || operation.outputs[1].constraint != "=d" ||
		operation.inputs[0].constraint != "rm" || operation.inputs[1].constraint != "0" ||
		operation.inputs[2].constraint != "1" {
		return false
	}
	for i := 0; i < len(operation.outputs); i++ {
		if t.typeSize(t.expressionType(operation.outputs[i].expression)) != 4 {
			return false
		}
	}
	for i := 0; i < len(operation.inputs); i++ {
		if t.typeSize(t.expressionType(operation.inputs[i].expression)) != 4 {
			return false
		}
	}
	t.ensureAsmDivideHelper()
	remainderPointer, remainderDeref := t.asmDereferencePointer(operation.outputs[1].expression)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=renvo_runtime_CDivide32(")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(",")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText(",")
	t.emitExpression(operation.inputs[2].expression)
	t.appendText(",")
	if remainderDeref {
		t.emitExpression(remainderPointer)
	} else {
		t.appendText("&(")
		t.emitExpression(operation.outputs[1].expression)
		t.appendText(")")
	}
	t.appendText(");")
	return t.ok
}

func (t *translator) ensureAsmDivideHelper() {
	if t.asmDivideHelper {
		return
	}
	t.asmDivideHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CDivide32(divisor uint32,low uint32,upper uint32,remainder *uint32) uint32{dividend:=(uint64(upper)<<32)|uint64(low);*remainder=uint32(dividend%uint64(divisor));return uint32(dividend/uint64(divisor))}\n"...)
}

func (t *translator) emitAsmBitSet(operation cAsm) bool {
	if string(operation.template) != "btsl %1,%0" || len(operation.outputs) != 1 || len(operation.inputs) != 1 ||
		operation.outputs[0].constraint != "+m" || operation.inputs[0].constraint != "Ir" {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.outputs[0].expression)
	if !ok || t.typeSize(t.expressionType(operation.outputs[0].expression)) != 4 {
		return false
	}
	t.ensureAsmBitSetHelper()
	t.appendText("renvo_runtime_CBitSet32(")
	t.emitExpression(pointer)
	t.appendText(",")
	t.emitExpression(operation.inputs[0].expression)
	t.appendText(");")
	return t.ok
}

func (t *translator) emitAsmCompareBytes(operation cAsm) bool {
	if !t.asmTemplatePrefix(operation.template, "repe; cmpsb") || len(operation.outputs) != 4 ||
		len(operation.inputs) != 0 || operation.outputs[0].constraint != "=@ccnz" ||
		operation.outputs[1].constraint != "+D" || operation.outputs[2].constraint != "+S" ||
		operation.outputs[3].constraint != "+c" {
		return false
	}
	if t.typeInfo(t.expressionType(operation.outputs[0].expression)).kind != cTypeBool ||
		t.typeInfo(t.expressionType(operation.outputs[1].expression)).kind != cTypePointer ||
		t.typeInfo(t.expressionType(operation.outputs[2].expression)).kind != cTypePointer {
		return false
	}
	t.typeSerial++
	suffix := decimalString(t.typeSerial)
	a, b, n := "__c_asm_a_"+suffix, "__c_asm_b_"+suffix, "__c_asm_n_"+suffix
	t.appendText("{")
	t.appendText(a)
	t.appendText(":=")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText(";")
	t.appendText(b)
	t.appendText(":=")
	t.emitExpression(operation.outputs[2].expression)
	t.appendText(";")
	t.appendText(n)
	t.appendText(":=")
	t.emitExpression(operation.outputs[3].expression)
	t.appendText(";")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=0;for ")
	t.appendText(n)
	t.appendText("!=0{__c_asm_diff:=*")
	t.appendText(a)
	t.appendText("!=*")
	t.appendText(b)
	t.appendText(";")
	t.appendText(a)
	t.appendText("=(*byte)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
	t.appendText(a)
	t.appendText("))+1));")
	t.appendText(b)
	t.appendText("=(*byte)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(")
	t.appendText(b)
	t.appendText("))+1));")
	t.appendText(n)
	t.appendText("--;if __c_asm_diff{")
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=1;break}};")
	t.emitExpression(operation.outputs[1].expression)
	t.appendText("=")
	t.appendText(a)
	t.appendText(";")
	t.emitExpression(operation.outputs[2].expression)
	t.appendText("=")
	t.appendText(b)
	t.appendText(";")
	t.emitExpression(operation.outputs[3].expression)
	t.appendText("=")
	t.appendText(n)
	t.appendText(";}")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitAsmBitTest(operation cAsm) bool {
	positional := t.asmTemplatePrefix(operation.template, "btl %2,%1")
	named := t.asmTemplateCompactPrefix(operation.template, "btl%[nr],%[var]")
	if !positional && !named ||
		len(operation.outputs) != 1 || operation.outputs[0].constraint != "=@ccc" ||
		len(operation.inputs) != 2 || operation.inputs[0].constraint != "m" ||
		operation.inputs[1].constraint != "Ir" && operation.inputs[1].constraint != "rI" ||
		named && (operation.inputs[0].name != "var" || operation.inputs[1].name != "nr") {
		return false
	}
	pointer, ok := t.asmDereferencePointer(operation.inputs[0].expression)
	memoryType := t.typeInfo(t.expressionType(operation.inputs[0].expression))
	arrayMemory := memoryType.kind == cTypeArray && t.typeSize(memoryType.base) == 4
	if !ok || t.typeSize(t.expressionType(operation.inputs[0].expression)) != 4 && !arrayMemory ||
		t.typeInfo(t.expressionType(operation.outputs[0].expression)).kind != cTypeBool {
		return false
	}
	t.ensureBoolHelper()
	pointerType := t.expressionType(pointer)
	pointerInfo := t.typeInfo(pointerType)
	if pointerInfo.kind == cTypePointer && t.typeInfo(pointerInfo.base).kind == cTypeArray {
		pointerType = t.pointerType(t.typeInfo(pointerInfo.base).base)
	}
	helper := t.ensurePointerIndexHelper(pointerType)
	t.emitExpression(operation.outputs[0].expression)
	t.appendText("=uint8(__c_bool_int(((*")
	t.appendText(helper)
	t.appendText("(")
	t.convertedExpression(pointerType, pointer)
	t.appendText(",uintptr(int64(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText(")>>5)))&(uint32(1)<<uint32(")
	t.emitExpression(operation.inputs[1].expression)
	t.appendText("&31)))!=0));")
	return t.ok
}

func (t *translator) asmDereferencePointer(tokens []token) ([]token, bool) {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) < 2 || !tokenIs(t.src, tokens[0], "*") {
		return nil, false
	}
	return tokens[1:], true
}

func (t *translator) asmTemplatePrefix(value []byte, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func (t *translator) asmExpressionsEqual(left []token, right []token) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		a, b := tokenText(t.src, left[i]), tokenText(t.src, right[i])
		if len(a) != len(b) {
			return false
		}
		for j := 0; j < len(a); j++ {
			if a[j] != b[j] {
				return false
			}
		}
	}
	return true
}

func (t *translator) ensureAsmMemoryHelper() {
	if t.asmMemoryHelper {
		return
	}
	t.asmMemoryHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CMemoryBarrier(){}\n"...)
}

func (t *translator) ensureAsmCCHelper() {
	if t.asmCCHelper {
		return
	}
	t.asmCCHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CConditionClobber(){}\n"...)
}

func (t *translator) ensureAsmStackHelper(typeID int) {
	if t.asmStackHelper {
		return
	}
	t.asmStackHelper = true
	outer := t.out
	t.out = t.staticOut
	t.appendText("func renvo_runtime_CReadStackPointer() ")
	t.emitType(typeID)
	t.appendText("{return 0}\n")
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) ensureAsmControlHelper(register int, write bool, typeID int) {
	bit := 1 << register
	if write {
		bit <<= 8
	}
	if t.asmControlHelper&bit != 0 {
		return
	}
	t.asmControlHelper |= bit
	outer := t.out
	t.out = t.staticOut
	if write {
		t.appendText("func renvo_runtime_CWriteControl")
		t.appendDecimal(register)
		t.appendText("(value ")
		t.emitType(typeID)
		t.appendText("){}\n")
	} else {
		t.appendText("func renvo_runtime_CReadControl")
		t.appendDecimal(register)
		t.appendText("() ")
		t.emitType(typeID)
		t.appendText("{return 0}\n")
	}
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) ensureAsmDebugHelper(register int, write bool, typeID int) {
	bit := 1 << register
	if write {
		bit <<= 8
	}
	if t.asmDebugHelper&bit != 0 {
		return
	}
	t.asmDebugHelper |= bit
	outer := t.out
	t.out = t.staticOut
	if write {
		t.appendText("func renvo_runtime_CWriteDebug")
		t.appendDecimal(register)
		t.appendText("(value ")
		t.emitType(typeID)
		t.appendText("){}\n")
	} else {
		t.appendText("func renvo_runtime_CReadDebug")
		t.appendDecimal(register)
		t.appendText("() ")
		t.emitType(typeID)
		t.appendText("{return 0}\n")
	}
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) ensureAsmFPUHelper() {
	if t.asmFPUHelper {
		return
	}
	t.asmFPUHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CInitFPU() uint32{return 0}\n"...)
}

func (t *translator) ensureAsmFlagsHelper(write bool, typeID int) {
	bit := 1
	if write {
		bit = 2
	}
	if t.asmFlagsHelper&bit != 0 {
		return
	}
	t.asmFlagsHelper |= bit
	outer := t.out
	t.out = t.staticOut
	if write {
		t.appendText("func renvo_runtime_CWriteFlags(value ")
		t.emitType(typeID)
		t.appendText("){}\n")
	} else {
		t.appendText("func renvo_runtime_CReadFlags() ")
		t.emitType(typeID)
		t.appendText("{return 0}\n")
	}
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) ensureAsmCPUIDHelper() {
	if t.asmCPUIDHelper {
		return
	}
	t.asmCPUIDHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CCPUID(leaf uint32,count uint32,a *uint32,b *uint32,c *uint32,d *uint32){}\n"...)
}

func (t *translator) ensureAsmMSRHelper() {
	if t.asmMSRHelper {
		return
	}
	t.asmMSRHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CReadMSR(register uint32,low *uint32,high *uint32){}\nfunc renvo_runtime_CWriteMSR(register uint32,low uint32,high uint32){}\nfunc renvo_runtime_CReadMSRSafe(register uint32,err *int32,low *uint32,high *uint32){}\nfunc renvo_runtime_CWriteMSRSafe(register uint32,low uint32,high uint32) int32{return 0}\n"...)
}

func (t *translator) ensureAsmIOHelper(size int, output bool) {
	index := 0
	if size == 2 {
		index = 1
	} else if size == 4 {
		index = 2
	}
	if output {
		index += 3
	}
	bit := 1 << index
	if t.asmIOHelpers&bit != 0 {
		return
	}
	t.asmIOHelpers |= bit
	typeName := "uint" + decimalString(size*8)
	if output {
		t.staticOut = append(t.staticOut, ("func renvo_runtime_COut" + decimalString(size*8) + "(value " + typeName + ",port uint16){}\n")...)
	} else {
		t.staticOut = append(t.staticOut, ("func renvo_runtime_CIn" + decimalString(size*8) + "(port uint16) " + typeName + "{return 0}\n")...)
	}
}

func (t *translator) ensureAsmBitSetHelper() {
	if t.asmBitSetHelper {
		return
	}
	t.asmBitSetHelper = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CBitSet32(address *uint32,bit int32){}\n"...)
}

func (t *translator) ensureAsmSegmentHelper(index int, write bool) {
	bit := 1 << index
	if write {
		bit <<= 4
	}
	if t.asmSegmentHelpers&bit != 0 {
		return
	}
	t.asmSegmentHelpers |= bit
	segment := []string{"es", "ds", "ss", "fs", "gs"}[index]
	if write {
		t.staticOut = append(t.staticOut, ("func renvo_runtime_CWriteSegment" + segment + "(value uint16){}\n")...)
	} else {
		t.staticOut = append(t.staticOut, ("func renvo_runtime_CReadSegment" + segment + "() uint16{return 0}\n")...)
	}
}

func (t *translator) ensureAsmSafeSegmentHelper(index int) {
	bit := 1 << index
	if t.asmSafeSegmentHelpers&bit != 0 {
		return
	}
	t.asmSafeSegmentHelpers |= bit
	segment := []string{"es", "ds", "ss"}[index]
	t.staticOut = append(t.staticOut, ("func renvo_runtime_CWriteSegmentSafe" + segment + "(value uint16){}\n")...)
}

func (t *translator) ensureAsmStartupSegmentsHelper() {
	if t.asmStartupSegments {
		return
	}
	t.asmStartupSegments = true
	t.staticOut = append(t.staticOut, "func renvo_runtime_CWriteStartupSegments(value uint16){}\n"...)
}

func (t *translator) ensureAsmSegmentMemoryHelper(segment int, size int, write bool) {
	sizeIndex := 0
	if size == 2 {
		sizeIndex = 1
	} else if size == 4 {
		sizeIndex = 2
	}
	index := segment*3 + sizeIndex
	if write {
		index += 6
	}
	bit := 1 << index
	if t.asmSegmentMemory&bit != 0 {
		return
	}
	t.asmSegmentMemory |= bit
	segmentName := []string{"fs", "gs"}[segment]
	typeName := "uint" + decimalString(size*8)
	if write {
		t.staticOut = append(t.staticOut, ("func renvo_runtime_CWriteSegment" + segmentName + decimalString(size*8) + "(address *" + typeName + ",value " + typeName + "){}\n")...)
	} else {
		t.staticOut = append(t.staticOut, ("func renvo_runtime_CReadSegment" + segmentName + decimalString(size*8) + "(address *" + typeName + ") " + typeName + "{return 0}\n")...)
	}
}

func (t *translator) ensureAsmSegmentCompareHelper(segment int, operation cAsm) {
	bit := 1 << segment
	if t.asmSegmentCompare&bit != 0 {
		return
	}
	t.asmSegmentCompare |= bit
	name := []string{"fs", "gs"}[segment]
	outer := t.out
	t.out = t.staticOut
	t.appendText("func renvo_runtime_CSegmentCompare")
	t.appendText(name)
	for i := 1; i < len(operation.outputs); i++ {
		if i == 1 {
			t.appendText("(p1 *")
		} else {
			t.appendText(",p")
			t.appendDecimal(i)
			t.appendText(" *")
		}
		t.emitType(t.expressionType(operation.outputs[i].expression))
	}
	t.appendText(") uint8{return 0}\n")
	t.staticOut = t.out
	t.out = outer
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
			t.baseAttributes.inline = true
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
				typeID = t.typeofExpressionType(operand)
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
					typeID, known = t.builtinCType(typeName)
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
		if unsigned || t.unsignedChar {
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
		if longCount > 1 || longCount == 1 && t.longSize == 8 {
			if unsigned {
				typeID = cTypeUint64ID
			} else {
				typeID = cTypeInt64ID
			}
		} else if longCount == 1 && unsigned {
			typeID = cTypeUint32ID
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
	if info := t.typeInfo(result.typeID); info.kind == cTypeArray && info.incomplete {
		result.incompleteArray = true
	}
	if result.attributes.requiresObjectMetadata() && !t.checkOnly && !t.object {
		t.fail(TranslateErrUnsupported)
		return result, false
	}
	if result.attributes.callingConvention == 1 {
		result.typeID, ok = t.msABIType(result.typeID)
		if !ok {
			t.fail(TranslateErrUnsupported)
			return result, false
		}
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
			arrayType := t.arrayType(typeID, ops[i].count)
			t.types[arrayType].incomplete = ops[i].incomplete
			typeID = t.qualifiedType(arrayType, ops[i].qualifiers)
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
			operation, valid := t.parseAsmClause()
			if !valid || len(operation.outputs) != 0 || len(operation.inputs) != 0 ||
				len(operation.clobbers) != 0 || len(operation.labels) != 0 || operation.gotoAsm ||
				len(operation.template) == 0 || attributes.asmRegister != "" {
				t.fail(TranslateErrUnsupported)
				return name, ops, attributes, false
			}
			attributes.asmRegister = string(operation.template)
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
		transparentTypeID := 0
		if paramInfo.kind == cTypeUnion && paramInfo.transparentUnion {
			if paramInfo.fieldCount == 0 {
				return params, false, false
			}
			transparentTypeID = paramType
			paramType = t.fields[paramInfo.fieldStart].typeID
			paramInfo = t.typeInfo(paramType)
		}
		if paramInfo.kind == cTypeArray {
			paramType = t.qualifiedType(t.pointerType(paramInfo.base), paramInfo.qualifiers)
		} else if paramInfo.kind == cTypeFunction {
			paramType = t.pointerType(paramType)
		}
		params = append(params, parameter{name: name, typeID: paramType, transparentTypeID: transparentTypeID})
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
	cMain := !t.object && tokenIs(t.src, decl.name, "main") && t.packageName == "main"
	if !t.checkOnly && cMain && !t.validExecutableMain(decl) {
		t.fail(TranslateErrUnsupported)
		return
	}
	linkageStorage := storage
	// Under GNU inline semantics, an extern inline definition is available for
	// inlining in this translation unit but does not provide an out-of-line
	// external definition. Keep Renvo's implementation local so headers can
	// define the same helper in every kernel translation unit.
	if storage == storageExtern && decl.attributes.inline && decl.attributes.gnuInline {
		linkageStorage = storageStatic
	}
	if t.object {
		t.emitObjectMetadata("function", string(tokenText(t.src, decl.name)), decl.attributes, linkageStorage, nil, decl.functionType, decl.variadic)
	}
	if t.object && linkageStorage != storageStatic {
		t.appendText("//export ")
		t.out = append(t.out, tokenText(t.src, decl.name)...)
		t.out = append(t.out, '\n')
	}
	t.appendText("func ")
	name := tokenText(t.src, decl.name)
	if tokenIs(t.src, decl.name, "main") && t.packageName == "main" {
		if t.object || len(decl.params) == 0 {
			name = []byte("appMain")
		} else {
			name = []byte("__c_user_main")
			t.emitExecutableMainWrapper(len(decl.params) == 3)
		}
	} else if t.object && linkageStorage != storageStatic && decl.attributes.weak {
		// A weak definition remains interposable from its own translation unit.
		// Keep the implementation behind the exported C symbol so calls resolve
		// through the linker instead of binding directly to Renvo's local body.
		name = append([]byte("__c_weak_impl_"), name...)
	}
	t.out = append(t.out, t.cFunctionGoName(string(name))...)
	t.out = append(t.out, '(')
	for i := 0; i < len(decl.params); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, t.cSourceGoIdentifier(string(tokenText(t.src, decl.params[i].name)))...)
		t.out = append(t.out, ' ')
		t.emitType(decl.params[i].typeID)
	}
	if decl.variadic {
		if len(decl.params) != 0 {
			t.out = append(t.out, ',')
		}
		t.appendText("__c_va *[3]uintptr")
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
	oldHelperSection := t.helperSection
	t.helperSection = decl.attributes.section
	oldVariadic := t.variadicFunction
	t.variadicFunction = decl.variadic
	for i := 0; i < len(decl.params); i++ {
		t.rememberObject(tokenText(t.src, decl.params[i].name), decl.params[i].typeID)
		if decl.params[i].transparentTypeID > 0 && len(t.objects) > 0 {
			t.objects[len(t.objects)-1].transparentTypeID = decl.params[i].transparentTypeID
		}
	}
	if close := matchingToken(t.src, t.tokens, t.pos, "{", "}"); close > t.pos {
		usesFunctionName := false
		for i := t.pos + 1; i < close; i++ {
			if tokenKind(t.tokens[i]) == tokenIdent && tokenIs(t.src, t.tokens[i], "__func__") {
				usesFunctionName = true
				break
			}
		}
		if usesFunctionName {
			t.typeSerial++
			goName := "__c_func_name_" + decimalString(t.typeSerial)
			functionName := tokenText(t.src, decl.name)
			t.rememberAliasedObject([]byte("__func__"), goName, t.arrayType(cTypeInt8ID, len(functionName)+1))
			outer := t.out
			t.out = t.staticOut
			t.appendText("var ")
			t.appendText(goName)
			t.appendText("=[")
			t.appendDecimal(len(functionName) + 1)
			t.appendText("]int8{")
			for _, value := range functionName {
				t.appendDecimal(int(value))
				t.out = append(t.out, ',')
			}
			t.appendText("0}\n")
			t.staticOut = t.out
			t.out = outer
		}
	}
	t.blockInCurrentScope()
	t.variadicFunction = oldVariadic
	t.resultType = oldResult
	t.helperSection = oldHelperSection
	t.localStart = oldLocalStart
	t.objects = t.objects[:objectMark]
	t.endScope(functionScope)
	t.out = append(t.out, '\n')
}

func (t *translator) validExecutableMain(decl declarator) bool {
	if decl.typeID != cTypeInt32ID {
		return false
	}
	if len(decl.params) == 0 {
		return true
	}
	if len(decl.params) != 2 && len(decl.params) != 3 || decl.params[0].typeID != cTypeInt32ID {
		return false
	}
	for i := 1; i < len(decl.params); i++ {
		outer := t.typeInfo(decl.params[i].typeID)
		if outer.kind != cTypePointer {
			return false
		}
		inner := t.typeInfo(outer.base)
		if inner.kind != cTypePointer || t.typeInfo(inner.base).size != 1 || t.typeInfo(inner.base).kind != cTypeInt {
			return false
		}
	}
	return true
}

func (t *translator) emitExecutableMainWrapper(environment bool) {
	t.usesUnsafe = true
	t.staticOut = append(t.staticOut, "func appMain(__c_args []string,__c_env []string) int32{__c_total:=0;for __c_i:=0;__c_i<len(__c_args);__c_i++{__c_total+=len(__c_args[__c_i])+1};__c_text:=make([]int8,__c_total);__c_argv:=make([]uintptr,len(__c_args)+1);__c_at:=0;for __c_i:=0;__c_i<len(__c_args);__c_i++{__c_argv[__c_i]=uintptr(__c_unsafe.Pointer(&__c_text[__c_at]));for __c_j:=0;__c_j<len(__c_args[__c_i]);__c_j++{__c_text[__c_at+__c_j]=int8(__c_args[__c_i][__c_j])};__c_at+=len(__c_args[__c_i])+1};"...)
	if environment {
		t.staticOut = append(t.staticOut, "__c_env_total:=0;for __c_i:=0;__c_i<len(__c_env);__c_i++{__c_env_total+=len(__c_env[__c_i])+1};__c_env_text:=make([]int8,__c_env_total);__c_envp:=make([]uintptr,len(__c_env)+1);__c_at=0;for __c_i:=0;__c_i<len(__c_env);__c_i++{__c_envp[__c_i]=uintptr(__c_unsafe.Pointer(&__c_env_text[__c_at]));for __c_j:=0;__c_j<len(__c_env[__c_i]);__c_j++{__c_env_text[__c_at+__c_j]=int8(__c_env[__c_i][__c_j])};__c_at+=len(__c_env[__c_i])+1};return __c_user_main(int32(len(__c_args)),(**int8)(__c_unsafe.Pointer(&__c_argv[0])),(**int8)(__c_unsafe.Pointer(&__c_envp[0])))}\n"...)
	} else {
		t.staticOut = append(t.staticOut, "return __c_user_main(int32(len(__c_args)),(**int8)(__c_unsafe.Pointer(&__c_argv[0])))}\n"...)
	}
}

func (t *translator) emitVariable(decl declarator) {
	t.rememberObject(tokenText(t.src, decl.name), decl.typeID)
	name := string(tokenText(t.src, decl.name))
	if t.localStart >= 0 && t.initializerReferencesName(decl.initializer, tokenText(t.src, decl.name)) {
		// In C the declared name is in scope throughout its initializer. Go's
		// VarSpec scope starts after the initializer, so split self-referential
		// local initialization into declaration plus assignment. This also lets
		// statement-expression helpers capture the object by address.
		initializer := decl.initializer
		decl.initializer = nil
		t.emitVariableName(name, decl)
		t.appendText("_=&")
		t.out = append(t.out, t.cSourceGoIdentifier(name)...)
		t.appendText(";")
		oldInitializing := t.initializing
		t.initializing = -1
		t.out = append(t.out, t.cSourceGoIdentifier(name)...)
		decl.initializer = initializer
		t.emitVariableInitializerSuffix(decl)
		t.appendText(";\n")
		t.initializing = oldInitializing
		return
	}
	oldInitializing := t.initializing
	t.initializing = len(t.objects) - 1
	t.emitVariableName(name, decl)
	t.initializing = oldInitializing
	if t.localStart >= 0 {
		t.appendText("_=&")
		t.out = append(t.out, t.cSourceGoIdentifier(name)...)
		t.appendText(";")
	}
}

func (t *translator) emitVariableName(name string, decl declarator) {
	outputType := decl.typeID
	if decl.objectTypeID > 0 {
		outputType = decl.objectTypeID
	}
	if t.localStart < 0 {
		t.emitObjectMetadata("variable", name, decl.attributes, decl.storage, decl.initializer, outputType, false)
	}
	t.appendText("var ")
	t.out = append(t.out, t.cSourceGoIdentifier(name)...)
	t.out = append(t.out, ' ')
	t.emitType(outputType)
	t.emitVariableInitializerSuffix(decl)
	t.out = append(t.out, ';', '\n')
}

func (t *translator) initializerReferencesName(tokens []token, name []byte) bool {
	text := string(name)
	for i := 0; i < len(tokens); i++ {
		if tokenKind(tokens[i]) == tokenIdent && textEquals(tokenText(t.src, tokens[i]), text) {
			return true
		}
	}
	return false
}

func (t *translator) emitVariableInitializerSuffix(decl declarator) {
	if len(decl.initializer) == 0 {
		return
	}
	initializerType := decl.typeID
	if decl.objectTypeID > 0 {
		initializerType = decl.objectTypeID
	}
	info := t.typeInfo(initializerType)
	if t.object && t.localStart >= 0 && info.indirect && t.zeroAggregateInitializer(decl.initializer) {
		return
	}
	t.out = append(t.out, '=')
	if relocationKind, _ := t.objectInitializerRelocation(initializerType, decl.initializer); t.object &&
		(t.localStart < 0 || decl.storage == storageStatic) && relocationKind != 0 {
		if info.kind == cTypePointer {
			t.appendText("nil")
		} else {
			t.appendText("0")
		}
		return
	}
	kind := info.kind
	if t.checkOnly {
		t.checkInitializerValue(initializerType, decl.initializer)
	} else if kind == cTypeArray && t.emitStringArrayInitializer(decl.typeID, decl.initializer) {
	} else if (kind == cTypeArray || kind == cTypeStruct) && tokenIs(t.src, decl.initializer[0], "{") {
		t.emitInitializerValue(initializerType, decl.initializer)
	} else if kind == cTypeUnion && tokenIs(t.src, decl.initializer[0], "{") {
		t.emitInitializerValue(initializerType, decl.initializer)
	} else {
		t.emitInitializerValue(initializerType, decl.initializer)
	}
}

func (t *translator) zeroAggregateInitializer(tokens []token) bool {
	if len(tokens) < 2 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		return false
	}
	items := splitTopLevel(t.src, tokens[1:len(tokens)-1], ",")
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) == 0 {
			continue
		}
		if equal := topLevelToken(t.src, item, "="); equal >= 0 {
			item = item[equal+1:]
		}
		if len(item) > 0 && tokenIs(t.src, item[0], "{") {
			if !t.zeroAggregateInitializer(item) {
				return false
			}
			continue
		}
		value, ok := t.constantExpression(item)
		if !ok || value != 0 {
			return false
		}
	}
	return true
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
	t.emitObjectMetadata(kind, string(tokenText(t.src, decl.name)), decl.attributes, storage, nil, decl.typeID, false)
	t.appendText("var __c_object_alias_")
	t.appendDecimal(t.typeSerial)
	t.appendText(" uint8\n")
}

func (t *translator) emitObjectMetadata(kind string, name string, attributes cAttributes, storage int, initializer []token, typeID int, variadic bool) {
	if !t.object {
		return
	}
	if !t.c11DirectiveEmitted {
		// The object linker roots declarations rather than package trivia. Keep
		// the unit semantic marker adjacent to rooted metadata so it survives
		// compact unit linking and reaches every backend implementation.
		t.appendText("// renvo:c11\n")
		t.c11DirectiveEmitted = true
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
	t.out = append(t.out, ' ')
	if variadic {
		t.out = append(t.out, '1')
	} else {
		t.out = append(t.out, '0')
	}
	t.out = append(t.out, '\n')
}

func (t *translator) objectInitializerRelocation(typeID int, initializer []token) (int, string) {
	if info := t.typeInfo(typeID); info.kind != cTypeArray && info.kind != cTypeStruct && info.kind != cTypeUnion {
		initializer = t.scalarInitializerTokens(initializer)
	}
	if len(initializer) == 1 && tokenKind(initializer[0]) == tokenIdent {
		if object, ok := t.lookupObject(tokenText(t.src, initializer[0])); ok {
			kind := t.typeInfo(object.typeID).kind
			if (kind == cTypeArray || kind == cTypeFunction) && t.typeSize(typeID) == 8 {
				return 1, string(tokenText(t.src, initializer[0]))
			}
		}
	}
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
	decl.storage = storageStatic
	t.typeSerial++
	name := "__c_static_" + decimalString(t.typeSerial) + "_" + string(tokenText(t.src, decl.name))
	t.rememberAliasedObject(tokenText(t.src, decl.name), name, decl.typeID)
	outer := t.out
	// Build the declaration separately: its initializer may create helper
	// declarations which must be appended before the completed variable.
	t.out = nil
	t.emitObjectMetadata("variable", name, decl.attributes, storageStatic, decl.initializer, decl.typeID, false)
	t.emitVariableName(name, decl)
	target := t.staticOut
	if t.capturingDeferred {
		target = t.deferredStaticOut
	}
	if len(target) != 0 && target[len(target)-1] != '\n' {
		target = append(target, '\n')
	}
	target = append(target, t.out...)
	if t.capturingDeferred {
		t.deferredStaticOut = target
	} else {
		t.staticOut = target
	}
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
	labelMark := len(t.localLabels)
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
	t.localLabels = t.localLabels[:labelMark]
	if scoped {
		t.endScope(scope)
	}
}

func (t *translator) statement() {
	if t.skipHoistedGotoGuard() {
		return
	}
	if t.take("__label__") {
		for {
			if t.kind() != tokenIdent {
				t.fail(TranslateErrStatement)
				return
			}
			t.typeSerial++
			name := string(tokenText(t.src, t.tokens[t.pos]))
			t.localLabels = append(t.localLabels, cLocalLabel{name: name, goName: "__c_label_" + decimalString(t.typeSerial)})
			t.pos++
			if !t.take(",") {
				break
			}
		}
		if !t.take(";") {
			t.fail(TranslateErrStatement)
			return
		}
		t.appendText(";")
		return
	}
	if t.isGNUAttributeStatement() {
		if _, ok := t.parseAttributes(); !ok || !t.take(";") {
			t.fail(TranslateErrStatement)
		}
		return
	}
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
		hoistedMark := len(t.hoistedGotos)
		if end := t.findAtDepth(";"); end >= t.pos {
			t.hoistStatementExpressionGoto(t.tokens[t.pos:end])
		}
		t.localDeclaration()
		t.hoistedGotos = t.hoistedGotos[:hoistedMark]
		return
	}
	if t.currentIs("if") {
		hoistedMark := len(t.hoistedGotos)
		if t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], "(") {
			if close := matchingToken(t.src, t.tokens, t.pos+1, "(", ")"); close > t.pos+1 {
				t.hoistStatementExpressionGoto(t.tokens[t.pos+2 : close])
			}
		}
		t.pos++
		if value, prefix, known, close := t.constantParenthesizedConditionWithPrefix(); known {
			t.pos = close + 1
			t.hoistedGotos = t.hoistedGotos[:hoistedMark]
			if len(prefix) != 0 {
				t.emitDiscardedExpression(prefix)
				t.out = append(t.out, ';')
			}
			if value != 0 {
				t.controlledStatement()
				if t.take("else") {
					t.skipControlledStatement()
				}
			} else {
				t.skipControlledStatement()
				if t.take("else") {
					t.controlledStatement()
				} else {
					t.appendText(";")
				}
			}
			t.separateFollowingLoweredBlock()
			return
		}
		if value, known, close := t.constantParenthesizedCondition(); known {
			t.pos = close + 1
			t.hoistedGotos = t.hoistedGotos[:hoistedMark]
			if value != 0 {
				t.controlledStatement()
				if t.take("else") {
					t.skipControlledStatement()
				}
			} else {
				t.skipControlledStatement()
				if t.take("else") {
					t.controlledStatement()
				} else {
					t.appendText(";")
				}
			}
			t.separateFollowingLoweredBlock()
			return
		}
		t.appendText("if ")
		t.parenthesizedCondition()
		t.hoistedGotos = t.hoistedGotos[:hoistedMark]
		t.out = append(t.out, ' ')
		t.controlledStatement()
		if t.take("else") {
			t.appendText(" else ")
			t.controlledStatement()
		}
		t.separateFollowingLoweredBlock()
		return
	}
	if t.currentIs("while") {
		hoistedMark := len(t.hoistedGotos)
		conditionEnd := -1
		if t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], "(") {
			conditionEnd = matchingToken(t.src, t.tokens, t.pos+1, "(", ")")
		}
		if conditionEnd > t.pos+1 {
			t.appendText("for {")
			t.hoistStatementExpressionGoto(t.tokens[t.pos+2 : conditionEnd])
			if len(t.hoistedGotos) > hoistedMark {
				t.pos++
				t.appendText("if !(")
				t.parenthesizedCondition()
				t.appendText("){break};")
				t.hoistedGotos = t.hoistedGotos[:hoistedMark]
				t.controlledStatement()
				t.appendText("}")
				return
			}
			t.out = t.out[:len(t.out)-5]
		}
		t.pos++
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
		name := string(tokenText(t.src, t.tokens[t.pos]))
		t.appendText("goto ")
		t.appendText(t.localLabelGoName(name))
		t.pos++
		if !t.take(";") {
			t.fail(TranslateErrStatement)
			return
		}
		t.out = append(t.out, ';')
		return
	}
	if t.currentIs("asm") || t.currentIs("__asm") || t.currentIs("__asm__") {
		if !t.emitAsmStatement() {
			t.fail(TranslateErrUnsupported)
			return
		}
		return
	}
	if t.currentIs("case") || t.currentIs("default") {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.currentIs("return") {
		hoistedMark := len(t.hoistedGotos)
		if end := t.findAtDepth(";"); end > t.pos+1 {
			t.hoistStatementExpressionGoto(t.tokens[t.pos+1 : end])
		}
		t.pos++
		start := t.pos
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		if end > start && t.resultType == cTypeVoidID && t.expressionType(t.tokens[start:end]) == cTypeVoidID {
			t.expression(t.tokens[start:end])
			t.appendText(";return")
		} else {
			t.appendText("return")
		}
		if end > start && t.resultType != cTypeVoidID {
			t.out = append(t.out, ' ')
			t.convertedExpression(t.resultType, t.tokens[start:end])
		}
		t.out = append(t.out, ';')
		t.pos = end + 1
		t.hoistedGotos = t.hoistedGotos[:hoistedMark]
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
		// An explicit empty statement keeps a label immediately following an
		// opening brace unambiguous to the shared frontend's compact scope scan.
		// It has no C or Go runtime effect, and labels remain function-scoped.
		t.out = append(t.out, ';')
		t.appendText(t.localLabelGoName(string(tokenText(t.src, t.tokens[t.pos]))))
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
	t.emitDiscardedExpression(t.tokens[start:end])
	t.out = append(t.out, ';')
	t.pos = end + 1
}

func (t *translator) constantParenthesizedConditionWithPrefix() (int, []token, bool, int) {
	if !t.currentIs("(") {
		return 0, nil, false, -1
	}
	close := matchingToken(t.src, t.tokens, t.pos, "(", ")")
	if close <= t.pos+1 {
		return 0, nil, false, close
	}
	tokens := t.tokens[t.pos+1 : close]
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
		case paren == 0 && bracket == 0 && brace == 0 &&
			(tokenIs(t.src, tokens[i], "&&") || tokenIs(t.src, tokens[i], "||")):
			// The prefix is emitted as a discarded expression before selecting
			// the fixed branch, which preserves every evaluation required by C's
			// left-to-right short-circuit rules.
			if t.conditionContainsRuntimeObject(tokens[i+1:]) {
				if _, ok := t.inlineRuntimeAnnihilatorCondition(tokens[i+1:], false); !ok {
					continue
				}
			}
			right, known := t.inlinePureConstantCondition(tokens[i+1:])
			if !known || tokenIs(t.src, tokens[i], "&&") && right != 0 || tokenIs(t.src, tokens[i], "||") && right == 0 {
				continue
			}
			value := 0
			if tokenIs(t.src, tokens[i], "||") {
				value = 1
			}
			return value, tokens[:i], true, close
		}
	}
	return 0, nil, false, close
}

func (t *translator) conditionContainsRuntimeObject(tokens []token) bool {
	for i := 0; i < len(tokens); i++ {
		if tokenKind(tokens[i]) != tokenIdent {
			continue
		}
		if _, ok := t.lookupObject(tokenText(t.src, tokens[i])); ok {
			return true
		}
	}
	return false
}

func (t *translator) skipHoistedGotoGuard() bool {
	if len(t.hoistedGotos) == 0 || !t.currentIs("if") || t.pos+1 >= len(t.tokens) ||
		!tokenIs(t.src, t.tokens[t.pos+1], "(") {
		return false
	}
	conditionEnd := matchingToken(t.src, t.tokens, t.pos+1, "(", ")")
	if conditionEnd < 0 || conditionEnd+3 >= len(t.tokens) ||
		!tokenIs(t.src, t.tokens[conditionEnd+1], "goto") ||
		tokenKind(t.tokens[conditionEnd+2]) != tokenIdent ||
		!tokenIs(t.src, t.tokens[conditionEnd+3], ";") {
		return false
	}
	name := string(tokenText(t.src, t.tokens[conditionEnd+2]))
	for i := 0; i < len(t.hoistedGotos); i++ {
		if t.hoistedGotos[i] == name {
			if !t.checkOnly {
				t.hoistedGotos[i] = ""
			}
			t.pos = conditionEnd + 4
			t.out = append(t.out, ';')
			return true
		}
	}
	return false
}

func (t *translator) localLabelGoName(name string) string {
	for i := len(t.localLabels) - 1; i >= 0; i-- {
		if t.localLabels[i].name == name {
			return t.localLabels[i].goName
		}
	}
	return cGoIdentifier(name)
}

func (t *translator) separateFollowingLoweredBlock() {
	if t.currentIs("do") || t.currentIs("{") {
		t.out = append(t.out, ';')
	}
}

func (t *translator) constantParenthesizedCondition() (int, bool, int) {
	if !t.currentIs("(") {
		return 0, false, -1
	}
	close := matchingToken(t.src, t.tokens, t.pos, "(", ")")
	if close <= t.pos+1 {
		return 0, false, close
	}
	value, ok := t.inlinePureConstantCondition(t.tokens[t.pos+1 : close])
	return value, ok, close
}

func (t *translator) constantCondition(tokens []token) (int, bool) {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if value, ok := t.constantExpression(tokens); ok {
		return value, true
	}
	if value, ok := t.inlineConstantCallValue(tokens); ok {
		return value, true
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "!") && t.binaryOperator(tokens) < 0 {
		value, ok := t.constantCondition(tokens[1:])
		if ok {
			if value == 0 {
				return 1, true
			}
			return 0, true
		}
	}
	at := t.binaryOperator(tokens)
	if at <= 0 || at+1 >= len(tokens) {
		return 0, false
	}
	left, leftOK := t.constantCondition(tokens[:at])
	if tokenIs(t.src, tokens[at], "&&") {
		if leftOK && left == 0 {
			return 0, true
		}
		if leftOK {
			right, rightOK := t.constantCondition(tokens[at+1:])
			if rightOK {
				if right != 0 {
					return 1, true
				}
				return 0, true
			}
		}
	}
	if tokenIs(t.src, tokens[at], "||") {
		if leftOK && left != 0 {
			return 1, true
		}
		if leftOK {
			return t.constantCondition(tokens[at+1:])
		}
	}
	return 0, false
}

func (t *translator) inlineConstantCallValue(tokens []token) (int, bool) {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) < 3 || tokenKind(tokens[0]) != tokenIdent || !tokenIs(t.src, tokens[1], "(") ||
		matchingToken(t.src, tokens, 1, "(", ")") != len(tokens)-1 {
		return 0, false
	}
	args := splitTopLevel(t.src, tokens[2:len(tokens)-1], ",")
	if len(tokens) == 3 {
		args = nil
	}
	if tokenIs(t.src, tokens[0], "__builtin_expect") && len(args) == 2 {
		return t.inlinePureConstantCondition(args[0])
	}
	entry := t.lookupNameEntry(tokenText(t.src, tokens[0]), false)
	if entry < 0 || t.names[entry].kind != cNameFunction {
		return 0, false
	}
	fn := t.functions[t.names[entry].index]
	if !fn.constantReturnOK {
		return 0, false
	}
	if len(args) != fn.paramCount || !t.constantCallArgumentsSideEffectFree(args) {
		return 0, false
	}
	return fn.constantReturn, true
}

func (t *translator) skipControlledStatement() {
	end := t.statementTokenEnd(t.pos)
	if end <= t.pos {
		t.fail(TranslateErrStatement)
		return
	}
	t.pos = end
}

func (t *translator) statementTokenEnd(start int) int {
	if start < 0 || start >= len(t.tokens) {
		return -1
	}
	if tokenIs(t.src, t.tokens[start], "{") {
		close := matchingToken(t.src, t.tokens, start, "{", "}")
		if close < 0 {
			return -1
		}
		return close + 1
	}
	if tokenIs(t.src, t.tokens[start], "if") || tokenIs(t.src, t.tokens[start], "while") ||
		tokenIs(t.src, t.tokens[start], "for") || tokenIs(t.src, t.tokens[start], "switch") {
		open := start + 1
		if open >= len(t.tokens) || !tokenIs(t.src, t.tokens[open], "(") {
			return -1
		}
		close := matchingToken(t.src, t.tokens, open, "(", ")")
		if close < 0 {
			return -1
		}
		end := t.statementTokenEnd(close + 1)
		if end < 0 {
			return -1
		}
		if tokenIs(t.src, t.tokens[start], "if") && end < len(t.tokens) && tokenIs(t.src, t.tokens[end], "else") {
			return t.statementTokenEnd(end + 1)
		}
		return end
	}
	if tokenIs(t.src, t.tokens[start], "do") {
		end := t.statementTokenEnd(start + 1)
		if end < 0 || end+1 >= len(t.tokens) || !tokenIs(t.src, t.tokens[end], "while") || !tokenIs(t.src, t.tokens[end+1], "(") {
			return -1
		}
		close := matchingToken(t.src, t.tokens, end+1, "(", ")")
		if close < 0 || close+1 >= len(t.tokens) || !tokenIs(t.src, t.tokens[close+1], ";") {
			return -1
		}
		return close + 2
	}
	if start+1 < len(t.tokens) && tokenKind(t.tokens[start]) == tokenIdent && tokenIs(t.src, t.tokens[start+1], ":") {
		return t.statementTokenEnd(start + 2)
	}
	paren, bracket := 0, 0
	for i := start; i < len(t.tokens); i++ {
		switch {
		case tokenIs(t.src, t.tokens[i], "("):
			paren++
		case tokenIs(t.src, t.tokens[i], ")"):
			paren--
		case tokenIs(t.src, t.tokens[i], "["):
			bracket++
		case tokenIs(t.src, t.tokens[i], "]"):
			bracket--
		case paren == 0 && bracket == 0 && tokenIs(t.src, t.tokens[i], ";"):
			return i + 1
		}
	}
	return -1
}

func (t *translator) emitDiscardedExpression(tokens []token) {
	statementTokens := tokens
	for len(statementTokens) > 2 && tokenIs(t.src, statementTokens[0], "(") &&
		matchingToken(t.src, statementTokens, 0, "(", ")") == len(statementTokens)-1 {
		statementTokens = statementTokens[1 : len(statementTokens)-1]
	}
	if len(statementTokens) >= 2 && tokenIs(t.src, statementTokens[0], "{") &&
		matchingToken(t.src, statementTokens, 0, "{", "}") == len(statementTokens)-1 {
		outerTokens, outerPos := t.tokens, t.pos
		t.tokens, t.pos = statementTokens, 0
		t.block()
		if t.ok && t.pos != len(statementTokens) {
			t.fail(TranslateErrUnsupported)
		}
		t.tokens, t.pos = outerTokens, outerPos
		return
	}
	if len(statementTokens) == 2 {
		name, op := -1, -1
		if tokenKind(statementTokens[0]) == tokenIdent && (tokenIs(t.src, statementTokens[1], "++") || tokenIs(t.src, statementTokens[1], "--")) {
			name, op = 0, 1
		} else if tokenKind(statementTokens[1]) == tokenIdent && (tokenIs(t.src, statementTokens[0], "++") || tokenIs(t.src, statementTokens[0], "--")) {
			name, op = 1, 0
		}
		if name >= 0 {
			if object, ok := t.lookupObject(tokenText(t.src, statementTokens[name])); ok {
				info := t.typeInfo(object.typeID)
				if info.kind == cTypeInt || info.kind == cTypeUint {
					t.out = append(t.out, object.goName...)
					t.out = append(t.out, tokenText(t.src, statementTokens[op])...)
					return
				}
			}
		}
	}
	hoistedMark := len(t.hoistedGotos)
	t.hoistStatementExpressionGoto(statementTokens)
	if t.discardedExpressionNeedsSink(statementTokens) {
		t.appendText("_=")
	}
	t.expression(statementTokens)
	t.hoistedGotos = t.hoistedGotos[:hoistedMark]
}

func (t *translator) hoistStatementExpressionGoto(tokens []token) {
	for i := 0; i+7 < len(tokens); i++ {
		if !tokenIs(t.src, tokens[i], "(") || !tokenIs(t.src, tokens[i+1], "{") || !tokenIs(t.src, tokens[i+2], "if") ||
			!tokenIs(t.src, tokens[i+3], "(") {
			continue
		}
		conditionEnd := matchingToken(t.src, tokens, i+3, "(", ")")
		if conditionEnd < 0 || conditionEnd+3 >= len(tokens) || !tokenIs(t.src, tokens[conditionEnd+1], "goto") ||
			tokenKind(tokens[conditionEnd+2]) != tokenIdent || !tokenIs(t.src, tokens[conditionEnd+3], ";") {
			continue
		}
		name := string(tokenText(t.src, tokens[conditionEnd+2]))
		t.appendText("if ")
		t.condition(tokens[i+4 : conditionEnd])
		t.appendText("{goto ")
		t.appendText(t.localLabelGoName(name))
		t.appendText("};")
		t.hoistedGotos = append(t.hoistedGotos, name)
	}
}

func (t *translator) discardedExpressionNeedsSink(tokens []token) bool {
	if len(tokens) == 0 || t.expressionType(tokens) == cTypeVoidID {
		return false
	}
	if len(tokens) >= 3 && tokenKind(tokens[0]) == tokenIdent && tokenIs(t.src, tokens[1], "(") &&
		matchingToken(t.src, tokens, 1, "(", ")") == len(tokens)-1 {
		name := tokenText(t.src, tokens[0])
		if textEquals(name, "__builtin_va_start") || textEquals(name, "__builtin_va_end") || textEquals(name, "__builtin_va_copy") ||
			textEquals(name, "__builtin_prefetch") ||
			textEquals(name, "__builtin_unreachable") || textEquals(name, "__sync_synchronize") {
			return false
		}
	}
	if tokenIs(t.src, tokens[0], "++") || tokenIs(t.src, tokens[0], "--") ||
		tokenIs(t.src, tokens[len(tokens)-1], "++") || tokenIs(t.src, tokens[len(tokens)-1], "--") {
		return false
	}
	depth := 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") || tokenIs(t.src, tokens[i], "{"):
			depth++
		case tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") || tokenIs(t.src, tokens[i], "}"):
			depth--
		case depth == 0 && isAssignmentToken(t.src, tokens[i]):
			return false
		}
	}
	return true
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
		decl, valid := t.parseDeclarator(base, storage == storageExtern)
		if !valid {
			t.fail(TranslateErrDeclaration)
			return
		}
		decl.attributes.merge(typeAttributes)
		if decl.function {
			if storage != storageExtern || !t.rememberFunction(&decl, false) || !t.take(";") {
				t.fail(TranslateErrDeclaration)
			}
			return
		}
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
			if t.object {
				t.rememberExternalObject(decl)
			}
		} else if storage == storageThread || storage == storageThreadStatic {
			t.emitThreadVariable(decl, false)
		} else if storage == storageThreadExtern {
			t.rememberThreadExtern(decl)
		} else if storage == storageStatic {
			if decl.attributes.cleanup != "" {
				t.fail(TranslateErrDeclaration)
				return
			}
			t.emitStaticLocal(decl)
		} else {
			t.emitVariable(decl)
			if decl.attributes.cleanup != "" {
				t.appendText("defer ")
				t.appendText(cGoIdentifier(decl.attributes.cleanup))
				t.appendText("(&")
				t.appendText(cGoIdentifier(string(tokenText(t.src, decl.name))))
				t.appendText(");")
			}
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
	if t.pos+2 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos], ";") &&
		tokenIs(t.src, t.tokens[t.pos+1], ";") && tokenIs(t.src, t.tokens[t.pos+2], ")") {
		t.pos += 3
		t.appendText("for true ")
		t.controlledStatement()
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
		if end > t.pos {
			t.expression(t.tokens[t.pos:end])
		}
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
	cleanupBody := -1
	if close > t.pos {
		if label, ok := t.cleanupGuardPostLabel(t.tokens[t.pos:close]); ok {
			cleanupBody, _ = t.cleanupGuardBodyStart(close+1, label)
		}
	}
	if close > t.pos && cleanupBody < 0 {
		t.emitDiscardedExpression(t.tokens[t.pos:close])
	}
	t.pos = close + 1
	if cleanupBody >= 0 {
		t.pos = cleanupBody
		t.appendText(" { ")
		t.controlledStatement()
		t.appendText(";break;}")
		if declaration {
			t.out = append(t.out, '}')
		}
		return
	}
	t.out = append(t.out, ' ')
	t.controlledStatement()
	if declaration {
		t.out = append(t.out, '}')
	}
}

func (t *translator) cleanupGuardPostLabel(tokens []token) (string, bool) {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) != 5 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[1], "goto") ||
		tokenKind(tokens[2]) != tokenIdent || !tokenIs(t.src, tokens[3], ";") || !tokenIs(t.src, tokens[4], "}") {
		return "", false
	}
	return string(tokenText(t.src, tokens[2])), true
}

func (t *translator) cleanupGuardBodyStart(at int, label string) (int, bool) {
	if at < 0 || at+1 >= len(t.tokens) || !tokenIs(t.src, t.tokens[at], "if") || !tokenIs(t.src, t.tokens[at+1], "(") {
		return -1, false
	}
	conditionEnd := matchingToken(t.src, t.tokens, at+1, "(", ")")
	if conditionEnd <= at+2 {
		return -1, false
	}
	blockStart := conditionEnd + 1
	if blockStart >= len(t.tokens) || !tokenIs(t.src, t.tokens[blockStart], "{") {
		return -1, false
	}
	blockEnd := matchingToken(t.src, t.tokens, blockStart, "{", "}")
	if blockEnd < 0 || blockEnd+2 >= len(t.tokens) || !tokenIs(t.src, t.tokens[blockEnd+1], "else") {
		return -1, false
	}
	value, constant := t.constantExpression(t.tokens[at+2 : conditionEnd])
	if constant && value == 0 && blockEnd == blockStart+5 &&
		tokenKind(t.tokens[blockStart+1]) == tokenIdent && string(tokenText(t.src, t.tokens[blockStart+1])) == label &&
		tokenIs(t.src, t.tokens[blockStart+2], ":") && tokenIs(t.src, t.tokens[blockStart+3], "break") &&
		tokenIs(t.src, t.tokens[blockStart+4], ";") {
		return blockEnd + 2, true
	}
	// Conditional cleanup guards put the synthetic exit label at the end of
	// their failure arm, often after an early return. Keep the complete if/else
	// inside the generated loop and add the ordinary one-shot break after it.
	for i := blockStart + 1; i+3 < blockEnd; i++ {
		if tokenKind(t.tokens[i]) == tokenIdent && string(tokenText(t.src, t.tokens[i])) == label &&
			tokenIs(t.src, t.tokens[i+1], ":") && tokenIs(t.src, t.tokens[i+2], "break") && tokenIs(t.src, t.tokens[i+3], ";") {
			return at, true
		}
	}
	return -1, false
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
	expressionTokens := t.tokens[t.pos:close]
	switchType := t.integerPromotion(t.expressionType(expressionTokens))
	blockStart := close + 1
	hasNestedCases := t.switchHasNestedCases(blockStart)
	hasLeadingDeclarations := t.switchHasLeadingDeclarations(blockStart)
	if switchValue, constant := t.constantExpression(expressionTokens); constant && !hasNestedCases && !hasLeadingDeclarations {
		t.pos = close + 1
		t.constantSwitchStatementBody(switchValue)
		return
	}
	conditionalCases := hasNestedCases || hasLeadingDeclarations || t.switchUsesConditionalCases(blockStart)
	switchName := ""
	if conditionalCases {
		t.typeSerial++
		switchName = "__c_switch_" + decimalString(t.typeSerial)
		t.appendText("{var ")
		t.appendText(switchName)
		t.out = append(t.out, ' ')
		t.emitType(switchType)
		t.appendText("=")
		t.convertedExpression(switchType, expressionTokens)
		t.out = append(t.out, ';')
	} else {
		t.appendText("switch ")
		t.convertedExpression(switchType, expressionTokens)
	}
	t.pos = close + 1
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	if hasLeadingDeclarations {
		for t.ok && t.isTypeStart() {
			t.statement()
			if t.ok && !t.checkOnly {
				t.out = append(t.out, '\n')
			}
		}
	}
	if conditionalCases {
		t.appendText("switch ")
	}
	t.out = append(t.out, '{')
	haveCase := false
	terminal := false
	caseBodyStarted := false
	var entryValues []int
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
				caseValue, caseConstant := t.constantExpression(caseTokens)
				if caseBodyStarted {
					entryValues = entryValues[:0]
				}
				if caseConstant {
					entryValues = append(entryValues, caseValue)
				}
				caseBodyStarted = false
				if rangeAt := topLevelToken(t.src, caseTokens, "..."); rangeAt >= 0 {
					first, firstOK := t.constantExpression(caseTokens[:rangeAt])
					last, lastOK := t.constantExpression(caseTokens[rangeAt+1:])
					if !firstOK || !lastOK || first > last || !conditionalCases && last-first > 255 {
						t.fail(TranslateErrStatement)
						return
					}
					if !t.checkOnly {
						t.appendText("case ")
						if conditionalCases {
							t.appendText(switchName)
							t.appendText(">=")
							t.appendDecimalSigned(first)
							t.appendText("&&")
							t.appendText(switchName)
							t.appendText("<=")
							t.appendDecimalSigned(last)
						} else {
							for value := first; value <= last; value++ {
								if value > first {
									t.out = append(t.out, ',')
								}
								t.appendDecimalSigned(value)
							}
						}
						t.out = append(t.out, ':')
					}
					t.pos = end + 1
					continue
				}
				t.appendText("case ")
				if conditionalCases {
					t.appendText(switchName)
					t.appendText("==")
				}
				t.convertedExpression(switchType, caseTokens)
				if values, ok := t.directNestedCaseValues(end + 1); ok {
					for i := 0; i < len(values); i++ {
						t.out = append(t.out, ',')
						if conditionalCases {
							t.appendText(switchName)
							t.appendText("==")
						}
						t.appendDecimalSigned(values[i])
					}
				}
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
		if t.currentIs("{") {
			if nested, ok := t.directNestedCaseValues(t.pos); ok {
				if !conditionalCases || len(entryValues) == 0 || !t.nestedSwitchCaseBlock(switchName, entryValues, nested) {
					t.fail(TranslateErrUnsupported)
					return
				}
				caseBodyStarted = true
				if !t.checkOnly {
					t.out = append(t.out, '\n')
				}
				continue
			}
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
		caseBodyStarted = true
		if t.ok && !t.checkOnly {
			t.out = append(t.out, '\n')
		}
	}
	if !t.take("}") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '}')
	if conditionalCases {
		t.out = append(t.out, '}')
	}
}

func (t *translator) switchHasLeadingDeclarations(open int) bool {
	if open < 0 || open+1 >= len(t.tokens) || !tokenIs(t.src, t.tokens[open], "{") {
		return false
	}
	old := t.pos
	t.pos = open + 1
	result := t.isTypeStart()
	t.pos = old
	return result
}

func (t *translator) switchHasNestedCases(open int) bool {
	if open < 0 || open >= len(t.tokens) || !tokenIs(t.src, t.tokens[open], "{") {
		return false
	}
	close := matchingToken(t.src, t.tokens, open, "{", "}")
	depth := 0
	for i := open + 1; i < close; i++ {
		if tokenIs(t.src, t.tokens[i], "{") {
			depth++
		} else if tokenIs(t.src, t.tokens[i], "}") {
			depth--
		} else if depth > 0 && (tokenIs(t.src, t.tokens[i], "case") || tokenIs(t.src, t.tokens[i], "default")) {
			return true
		}
	}
	return false
}

func (t *translator) directNestedCaseValues(open int) ([]int, bool) {
	if open < 0 || open >= len(t.tokens) || !tokenIs(t.src, t.tokens[open], "{") {
		return nil, false
	}
	close := matchingToken(t.src, t.tokens, open, "{", "}")
	if close < 0 {
		return nil, false
	}
	depth := 0
	var values []int
	for i := open + 1; i < close; i++ {
		if tokenIs(t.src, t.tokens[i], "{") {
			depth++
			continue
		}
		if tokenIs(t.src, t.tokens[i], "}") {
			depth--
			continue
		}
		if depth != 0 || !tokenIs(t.src, t.tokens[i], "case") {
			continue
		}
		colon := topLevelToken(t.src, t.tokens[i+1:close], ":")
		if colon < 0 {
			return nil, false
		}
		value, ok := t.constantExpression(t.tokens[i+1 : i+1+colon])
		if !ok {
			return nil, false
		}
		values = append(values, value)
		i += colon + 1
	}
	return values, len(values) > 0
}

func (t *translator) nestedSwitchCaseBlock(switchName string, entryValues []int, nestedValues []int) bool {
	if !t.take("{") {
		return false
	}
	objectMark := len(t.objects)
	labelMark := len(t.localLabels)
	scope := t.beginScope()
	t.out = append(t.out, '{')
	allowed := append([]int{}, entryValues...)
	guardOpen := false
	executable := false
	for t.ok && t.kind() != tokenEOF && !t.currentIs("}") {
		if t.currentIs("case") {
			if guardOpen {
				t.appendText("};")
				guardOpen = false
			}
			t.pos++
			end := t.findAtDepth(":")
			if end < 0 || end == t.pos {
				return false
			}
			value, ok := t.constantExpression(t.tokens[t.pos:end])
			if !ok {
				return false
			}
			allowed = append(allowed, value)
			t.pos = end + 1
			executable = true
			continue
		}
		if !executable && t.isTypeStart() {
			t.statement()
			if t.ok && !t.checkOnly {
				t.out = append(t.out, '\n')
			}
			continue
		}
		executable = true
		if !guardOpen && len(allowed) < len(entryValues)+len(nestedValues) {
			t.appendText("if ")
			for i := 0; i < len(allowed); i++ {
				if i > 0 {
					t.appendText("||")
				}
				t.appendText(switchName)
				t.appendText("==")
				t.appendDecimalSigned(allowed[i])
			}
			t.out = append(t.out, '{')
			guardOpen = true
		}
		t.statement()
		if t.ok && !t.checkOnly {
			t.out = append(t.out, '\n')
		}
	}
	if guardOpen {
		t.appendText("};")
	}
	if !t.take("}") {
		t.objects = t.objects[:objectMark]
		t.localLabels = t.localLabels[:labelMark]
		t.endScope(scope)
		return false
	}
	t.out = append(t.out, '}')
	t.objects = t.objects[:objectMark]
	t.localLabels = t.localLabels[:labelMark]
	t.endScope(scope)
	return t.ok
}

func (t *translator) constantSwitchStatementBody(value int) {
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	open := t.pos - 1
	close := matchingToken(t.src, t.tokens, open, "{", "}")
	selected := t.constantSwitchSelectedCase(open, close, value)
	if close < 0 || selected < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	// Keep a real switch in the generated control-flow tree. Besides making a
	// selected top-level break valid, this preserves C's rule that continue
	// targets an enclosing loop rather than the switch itself.
	t.appendText("switch 0 {default:")
	active, haveCase := false, false
	for t.ok && t.pos < close {
		t.skipDirectives()
		if t.currentIs("case") || t.currentIs("default") {
			label := t.pos
			haveCase = true
			if t.take("case") {
				end := t.findAtDepth(":")
				if end < 0 || end == t.pos {
					t.fail(TranslateErrStatement)
					return
				}
				t.pos = end + 1
			} else {
				t.pos++
				if !t.take(":") {
					t.fail(TranslateErrStatement)
					return
				}
			}
			active = active || label == selected
			continue
		}
		if !haveCase {
			t.fail(TranslateErrUnsupported)
			return
		}
		if active && t.currentIs("break") {
			t.pos++
			if !t.take(";") {
				t.fail(TranslateErrStatement)
				return
			}
			t.appendText("break;")
			active = false
			continue
		}
		if active {
			t.statement()
			if t.ok {
				t.out = append(t.out, '\n')
			}
			continue
		}
		outer := t.out
		oldCheckOnly := t.checkOnly
		t.out = nil
		t.checkOnly = true
		mark := t.beginCheckScratch()
		t.statement()
		t.endCheckScratch(mark)
		t.checkOnly = oldCheckOnly
		t.out = outer
	}
	if !t.take("}") {
		t.fail(TranslateErrStatement)
		return
	}
	t.appendText("}")
}

func (t *translator) constantSwitchSelectedCase(open int, close int, value int) int {
	if open < 0 || close <= open {
		return -1
	}
	selected, fallback := -1, -1
	depth := 0
	for i := open + 1; i < close; i++ {
		if tokenIs(t.src, t.tokens[i], "{") {
			depth++
			continue
		}
		if tokenIs(t.src, t.tokens[i], "}") {
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		if tokenIs(t.src, t.tokens[i], "default") && i+1 < close && tokenIs(t.src, t.tokens[i+1], ":") {
			fallback = i
			continue
		}
		if !tokenIs(t.src, t.tokens[i], "case") {
			continue
		}
		colon := topLevelToken(t.src, t.tokens[i+1:close], ":")
		if colon <= 0 {
			return -1
		}
		caseTokens := t.tokens[i+1 : i+1+colon]
		if rangeAt := topLevelToken(t.src, caseTokens, "..."); rangeAt >= 0 {
			first, firstOK := t.constantExpression(caseTokens[:rangeAt])
			last, lastOK := t.constantExpression(caseTokens[rangeAt+1:])
			if !firstOK || !lastOK || first > last {
				return -1
			}
			if value >= first && value <= last {
				selected = i
			}
		} else if caseValue, ok := t.constantExpression(caseTokens); !ok {
			return -1
		} else if value == caseValue {
			selected = i
		}
		i += colon
	}
	if selected >= 0 {
		return selected
	}
	if fallback >= 0 {
		return fallback
	}
	return close
}

func (t *translator) switchUsesConditionalCases(open int) bool {
	if open < 0 || open >= len(t.tokens) || !tokenIs(t.src, t.tokens[open], "{") {
		return false
	}
	close := matchingToken(t.src, t.tokens, open, "{", "}")
	if close < 0 {
		return false
	}
	depth := 0
	for i := open + 1; i < close; i++ {
		if tokenIs(t.src, t.tokens[i], "{") {
			depth++
			continue
		}
		if tokenIs(t.src, t.tokens[i], "}") {
			depth--
			continue
		}
		if depth != 0 || !tokenIs(t.src, t.tokens[i], "case") {
			continue
		}
		colon := topLevelToken(t.src, t.tokens[i+1:close], ":")
		if colon < 0 {
			return false
		}
		caseTokens := t.tokens[i+1 : i+1+colon]
		rangeAt := topLevelToken(t.src, caseTokens, "...")
		if rangeAt < 0 {
			continue
		}
		first, firstOK := t.constantExpression(caseTokens[:rangeAt])
		last, lastOK := t.constantExpression(caseTokens[rangeAt+1:])
		if firstOK && lastOK && first <= last && last-first > 255 {
			return true
		}
	}
	return false
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
	if value, known := t.inlinePureConstantCondition(tokens); known {
		if value == 0 {
			t.appendText("false")
		} else {
			t.appendText("true")
		}
		return
	}
	if t.logicalConditionOperators(tokens) >= 2 && t.conditionHasSubscript(tokens) {
		t.emitConditionHelper(tokens)
		return
	}
	t.emitConditionExpression(tokens)
}

func (t *translator) conditionHasSubscript(tokens []token) bool {
	for i := 0; i < len(tokens); i++ {
		if tokenIs(t.src, tokens[i], "[") {
			return true
		}
	}
	return false
}

func (t *translator) logicalConditionOperators(tokens []token) int {
	count, paren, bracket := 0, 0, 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		case paren == 0 && bracket == 0 && (tokenIs(t.src, tokens[i], "&&") || tokenIs(t.src, tokens[i], "||")):
			count++
		}
	}
	return count
}

func (t *translator) emitConditionHelper(tokens []token) {
	t.typeSerial++
	name := "__c_condition_" + decimalString(t.typeSerial)
	t.appendText(name)
	t.appendText("(")
	t.emitCaptureArguments(false)
	t.appendText(")")
	outer, names, captureIndices := t.beginHelper()
	t.emitHelperObjectMetadata(name, cTypeBoolID)
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(")
	t.emitCaptureParameters(false)
	t.appendText(") bool{return ")
	t.emitConditionExpression(tokens)
	t.appendText("}\n")
	t.endHelper(outer, names, captureIndices)
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
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") &&
		matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if t.commaOperator(tokens) >= 0 {
		return false
	}
	depth := 0
	for i := 0; i < len(tokens); i++ {
		if tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") || tokenIs(t.src, tokens[i], "{") {
			depth++
		} else if tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") || tokenIs(t.src, tokens[i], "}") {
			depth--
		} else if depth == 0 && isAssignmentToken(t.src, tokens[i]) {
			left := tokens[:i]
			if tokenIs(t.src, tokens[i], "=") && t.emitPrefixDereferenceAssignment(left, tokens[i+1:]) {
				return true
			}
			if tokenIs(t.src, tokens[i], "=") && t.emitPostfixDereferenceAssignment(left, tokens[i+1:]) {
				return true
			}
			leftType := t.lvalueType(left)
			leftInfo := t.typeInfo(leftType)
			if tokenIs(t.src, tokens[i], "=") && leftInfo.size == 0 &&
				(leftInfo.kind == cTypeStruct || leftInfo.kind == cTypeUnion) {
				// GNU empty aggregates occupy no C storage. Their Go carrier is
				// necessarily addressable and therefore one byte wide, so a native
				// Go assignment would overwrite the following C field. Preserve the
				// lvalue and RHS evaluations through the assignment-value helper,
				// whose zero-sized specialization deliberately performs no store.
				return t.emitAssignmentValue(tokens)
			}
			pointerStep := t.typeInfo(leftType).kind == cTypePointer &&
				(tokenIs(t.src, tokens[i], "+=") || tokenIs(t.src, tokens[i], "-="))
			plainIdentifier := len(left) == 1 && tokenKind(left[0]) == tokenIdent
			plainMember := len(left) == 3 && tokenKind(left[0]) == tokenIdent &&
				(tokenIs(t.src, left[1], ".") || tokenIs(t.src, left[1], "->")) && tokenKind(left[2]) == tokenIdent
			if !tokenIs(t.src, tokens[i], "=") && !pointerStep && !plainIdentifier && !plainMember {
				if t.emitCompoundLvalueAssignment(leftType, left, tokens[i], tokens[i+1:]) {
					return true
				}
				return false
			}
			t.emitExpression(left)
			t.out = append(t.out, '=')
			if tokenIs(t.src, tokens[i], "=") {
				t.convertedExpression(t.lvalueType(left), tokens[i+1:])
				return true
			}
			if pointerStep {
				helper := t.ensurePointerStepHelper(leftType, tokenIs(t.src, tokens[i], "-="))
				t.appendText(helper)
				t.appendText("(")
				t.emitExpression(left)
				t.appendText(",uintptr(")
				t.emitExpression(tokens[i+1:])
				t.appendText("))")
				return true
			}
			t.convertedExpression(t.lvalueType(left), left)
			op := tokenText(t.src, tokens[i])
			t.out = append(t.out, op[:len(op)-1]...)
			t.out = append(t.out, '(')
			t.convertedExpression(t.lvalueType(left), tokens[i+1:])
			t.out = append(t.out, ')')
			return true
		}
	}
	return false
}

func (t *translator) emitPrefixDereferenceAssignment(left []token, right []token) bool {
	for len(left) > 2 && tokenIs(t.src, left[0], "(") && matchingToken(t.src, left, 0, "(", ")") == len(left)-1 {
		left = left[1 : len(left)-1]
	}
	if len(left) < 3 || !tokenIs(t.src, left[0], "*") {
		return false
	}
	mutation := left[1:]
	if tokenIs(t.src, left[1], "(") {
		if len(left) < 5 || matchingToken(t.src, left, 1, "(", ")") != len(left)-1 {
			return false
		}
		mutation = left[2 : len(left)-1]
	}
	if len(mutation) < 2 || !tokenIs(t.src, mutation[0], "++") && !tokenIs(t.src, mutation[0], "--") {
		return false
	}
	target := mutation[1:]
	pointerType := t.expressionType(target)
	info := t.typeInfo(pointerType)
	if info.kind != cTypePointer {
		return false
	}
	decrement := tokenIs(t.src, mutation[0], "--")
	key := pointerType * 2
	name := "__c_pre_deref_assign_inc_" + decimalString(pointerType)
	if decrement {
		key++
		name = "__c_pre_deref_assign_dec_" + decimalString(pointerType)
	}
	found := false
	for i := 0; i < len(t.prefixDerefAssignTypes); i++ {
		found = found || t.prefixDerefAssignTypes[i] == key
	}
	if !found {
		t.prefixDerefAssignTypes = append(t.prefixDerefAssignTypes, key)
		step := t.ensurePointerStepHelper(pointerType, decrement)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(target **")
		t.emitType(info.base)
		t.appendText(",value ")
		t.emitType(info.base)
		t.appendText("){*target=")
		t.appendText(step)
		t.appendText("(*target,1);**target=value}\n")
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(target)
	t.appendText("),")
	t.convertedExpression(info.base, right)
	t.appendText(")")
	return true
}

func (t *translator) emitCompoundLvalueAssignment(typeID int, left []token, operator token, right []token) bool {
	info := t.typeInfo(typeID)
	if info.kind != cTypeInt && info.kind != cTypeUint && info.kind != cTypeFloat && info.kind != cTypeBool {
		return false
	}
	op := string(tokenText(t.src, operator))
	if len(op) < 2 || op[len(op)-1] != '=' {
		return false
	}
	op = op[:len(op)-1]
	opIndex := 0
	switch op {
	case "+":
		opIndex = 1
	case "-":
		opIndex = 2
	case "*":
		opIndex = 3
	case "/":
		opIndex = 4
	case "%":
		opIndex = 5
	case "<<":
		opIndex = 6
	case ">>":
		opIndex = 7
	case "&":
		opIndex = 8
	case "^":
		opIndex = 9
	case "|":
		opIndex = 10
	default:
		return false
	}
	if info.kind == cTypeBool {
		rightType := t.expressionType(right)
		valueType := rightType
		operationType := t.usualArithmeticType(typeID, rightType)
		if op == "<<" || op == ">>" {
			operationType = t.integerPromotion(typeID)
			valueType = t.integerPromotion(rightType)
		}
		key := (typeID*16+opIndex)*65536 + operationType
		name := "__c_compound_bool_" + decimalString(opIndex) + "_" + decimalString(operationType)
		found := false
		for i := 0; i < len(t.compoundAssignKeys); i++ {
			found = found || t.compoundAssignKeys[i] == key
		}
		if !found {
			t.compoundAssignKeys = append(t.compoundAssignKeys, key)
			t.ensureBoolHelper()
			outer := t.out
			t.out = nil
			t.appendText("func ")
			t.appendText(name)
			t.appendText("(target *uint8,value ")
			t.emitType(valueType)
			t.appendText(") uint8{*target=uint8(__c_bool_int((")
			t.emitType(operationType)
			t.appendText("(*target)")
			t.appendText(op)
			t.appendText("value)!=0));return *target}\n")
			t.staticOut = append(t.staticOut, t.out...)
			t.out = outer
		}
		t.appendText(name)
		t.appendText("(&(")
		t.emitExpression(left)
		t.appendText("),")
		t.convertedExpression(valueType, right)
		t.appendText(")")
		return true
	}
	key := typeID*16 + opIndex
	name := "__c_compound_" + decimalString(opIndex) + "_" + decimalString(typeID)
	found := false
	for i := 0; i < len(t.compoundAssignKeys); i++ {
		found = found || t.compoundAssignKeys[i] == key
	}
	if !found {
		t.compoundAssignKeys = append(t.compoundAssignKeys, key)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(target *")
		t.emitType(typeID)
		t.appendText(",value ")
		t.emitType(typeID)
		t.appendText(") ")
		t.emitType(typeID)
		t.appendText("{*target=*target")
		t.appendText(op)
		t.appendText("value;return *target}\n")
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(left)
	t.appendText("),")
	t.convertedExpression(typeID, right)
	t.appendText(")")
	return true
}

func (t *translator) emitPostfixDereferenceAssignment(left []token, right []token) bool {
	for len(left) > 2 && tokenIs(t.src, left[0], "(") &&
		matchingToken(t.src, left, 0, "(", ")") == len(left)-1 {
		left = left[1 : len(left)-1]
	}
	// WRITE_ONCE-style expansions dereference a qualified pointer cast of the
	// lvalue address: *(volatile typeof(value) *)&value. Once the cast has
	// been checked as a pointer to the lvalue type, assignment can target the
	// underlying value directly.
	if candidate, ok := t.qualifiedAddressLvalue(left); ok {
		left = candidate
	}
	// C macro expansions commonly form *&*cursor++ while forcing an lvalue
	// through a volatile cast.  The leading *& cancels without evaluating the
	// operand, leaving the same postfix-dereference assignment as *cursor++.
	for len(left) > 2 && tokenIs(t.src, left[0], "*") && tokenIs(t.src, left[1], "&") {
		left = left[2:]
		for len(left) > 2 && tokenIs(t.src, left[0], "(") &&
			matchingToken(t.src, left, 0, "(", ")") == len(left)-1 {
			left = left[1 : len(left)-1]
		}
	}
	if len(left) < 3 || !tokenIs(t.src, left[0], "*") {
		return false
	}
	mutation := left[1:]
	for len(mutation) > 2 && tokenIs(t.src, mutation[0], "(") &&
		matchingToken(t.src, mutation, 0, "(", ")") == len(mutation)-1 {
		mutation = mutation[1 : len(mutation)-1]
	}
	if len(mutation) < 2 ||
		!tokenIs(t.src, mutation[len(mutation)-1], "++") && !tokenIs(t.src, mutation[len(mutation)-1], "--") {
		return false
	}
	operand := mutation[:len(mutation)-1]
	pointerType := t.lvalueType(operand)
	info := t.typeInfo(pointerType)
	if info.kind != cTypePointer {
		return false
	}
	decrement := tokenIs(t.src, mutation[len(mutation)-1], "--")
	key := pointerType + 1
	if decrement {
		key = -key
	}
	name := "__c_post_assign_"
	if decrement {
		name += "dec_"
	} else {
		name += "inc_"
	}
	name += decimalString(pointerType)
	found := false
	for i := 0; i < len(t.postfixAssignTypes); i++ {
		found = found || t.postfixAssignTypes[i] == key
	}
	if !found {
		t.postfixAssignTypes = append(t.postfixAssignTypes, key)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(p **")
		t.emitType(info.base)
		t.appendText(",value ")
		t.emitType(info.base)
		t.appendText(") ")
		t.emitType(info.base)
		t.appendText("{target:=*p;*p=(*")
		t.emitType(info.base)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(*p))")
		if decrement {
			t.out = append(t.out, '-')
		} else {
			t.out = append(t.out, '+')
		}
		t.appendText("uintptr(")
		t.appendDecimal(t.typeSize(info.base))
		t.appendText(")));*target=value;return value}\n")
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
		t.usesUnsafe = true
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(operand)
	t.appendText("),")
	t.convertedExpression(info.base, right)
	t.appendText(")")
	return true
}

func (t *translator) qualifiedAddressLvalue(tokens []token) ([]token, bool) {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") &&
		matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) <= 5 || !tokenIs(t.src, tokens[0], "*") || !tokenIs(t.src, tokens[1], "(") {
		return nil, false
	}
	close := matchingToken(t.src, tokens, 1, "(", ")")
	if close <= 2 || close+2 >= len(tokens) || !tokenIs(t.src, tokens[close+1], "&") {
		return nil, false
	}
	castType, castOK := t.typeFromTokens(tokens[2:close])
	candidate := tokens[close+2:]
	for len(candidate) > 2 && tokenIs(t.src, candidate[0], "(") &&
		matchingToken(t.src, candidate, 0, "(", ")") == len(candidate)-1 {
		candidate = candidate[1 : len(candidate)-1]
	}
	candidateType := t.lvalueType(candidate)
	cast := t.typeInfo(castType)
	if !castOK || cast.kind != cTypePointer || candidateType == cTypeVoidID ||
		!t.compatibleParameterType(cast.base, candidateType) {
		return nil, false
	}
	return candidate, true
}

func (t *translator) convertedExpression(typeID int, tokens []token) {
	if t.checkOnly {
		t.checkExpression(tokens)
		return
	}
	targetInfo := t.typeInfo(typeID)
	if targetInfo.kind == cTypePointer {
		if t.isNullPointerExpression(tokens) {
			t.appendText("nil")
			return
		}
	}
	if (targetInfo.kind == cTypeInt || targetInfo.kind == cTypeUint) && t.isNullPointerExpression(tokens) {
		// Raw union carriers represent pointer members with an integer word.
		// Preserve a C null-pointer initializer as an all-zero carrier without
		// leaking the invalid shared-Go spelling uint64(nil).
		t.appendText("0")
		return
	}
	initializerTokens := tokens
	for len(initializerTokens) > 2 && tokenIs(t.src, initializerTokens[0], "(") &&
		matchingToken(t.src, initializerTokens, 0, "(", ")") == len(initializerTokens)-1 {
		initializerTokens = initializerTokens[1 : len(initializerTokens)-1]
	}
	if t.emitObjectScalarUnionCast(typeID, initializerTokens) {
		return
	}
	if _, _, ok := t.statementExpressionValue(initializerTokens); ok {
		t.emitStatementExpression(initializerTokens, typeID)
		return
	}
	if len(initializerTokens) > 3 && tokenIs(t.src, initializerTokens[0], "(") {
		close := matchingToken(t.src, initializerTokens, 0, "(", ")")
		if close > 1 && close+1 < len(initializerTokens) && tokenIs(t.src, initializerTokens[close+1], "{") {
			end := matchingToken(t.src, initializerTokens, close+1, "{", "}")
			if end == len(initializerTokens)-1 {
				t.emitInitializerValue(typeID, initializerTokens[close+1:])
				return
			}
		}
	}
	if info := t.typeInfo(typeID); info.kind == cTypePointer {
		if _, ok := t.cStringBytes(initializerTokens); ok {
			t.ensureStringPointerHelper()
			t.appendText("renvo_runtime_CStringPointer(")
			t.emitCStringValues(initializerTokens)
			t.appendText(")")
			return
		}
	}
	if assignment := t.simpleMutationAssignment(initializerTokens); t.commaOperator(initializerTokens) < 0 && assignment > 0 {
		assignmentType := t.lvalueType(initializerTokens[:assignment])
		if assignmentType == typeID && t.emitAssignmentValue(initializerTokens) {
			return
		}
		assignmentInfo, targetInfo := t.typeInfo(assignmentType), t.typeInfo(typeID)
		if assignmentInfo.kind == cTypePointer && targetInfo.kind == cTypePointer {
			t.out = append(t.out, '(')
			t.emitType(typeID)
			t.appendText(")(")
			if t.emitAssignmentValue(initializerTokens) {
				t.out = append(t.out, ')')
				return
			}
			t.fail(TranslateErrUnsupported)
			return
		}
		if assignmentType != cTypeVoidID &&
			(assignmentInfo.kind == cTypeInt || assignmentInfo.kind == cTypeUint || assignmentInfo.kind == cTypeBool) &&
			(targetInfo.kind == cTypeInt || targetInfo.kind == cTypeUint || targetInfo.kind == cTypeBool) {
			t.emitType(typeID)
			t.out = append(t.out, '(')
			t.emitAssignmentValue(initializerTokens)
			t.out = append(t.out, ')')
			return
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
	info := targetInfo
	if t.object && info.kind == cTypePointer && t.typeInfo(info.base).kind == cTypeFunction {
		// Function pointers in relocatable objects use the C ABI's raw address
		// representation. Keep both implicit assignment conversion and explicit
		// casts as word-preserving expressions at the shared frontend boundary.
		t.expression(tokens)
		return
	}
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
	if sourceInfo.kind == cTypeArray && (info.kind == cTypeInt || info.kind == cTypeUint) {
		// Array-to-pointer conversion happens before an explicit pointer-to-
		// integer conversion. Emitting the array expression directly would load
		// its first word, which is observably wrong for member arrays such as the
		// kernel's gdt_page.gdt bootstrap address.
		t.emitType(typeID)
		t.appendText("(__c_unsafe.Pointer(&(")
		t.expression(tokens)
		t.appendText(")))")
		t.usesUnsafe = true
		return
	}
	if t.typeInfo(typeID).kind == cTypePointer && sourceInfo.kind == cTypeArray {
		if t.emitSubscriptMemberArrayDecay(typeID, tokens) {
			return
		}
		if sourceInfo.count == 0 || !t.compatibleParameterType(t.typeInfo(typeID).base, sourceInfo.base) {
			t.usesUnsafe = true
			t.out = append(t.out, '(')
			t.emitType(typeID)
			t.appendText(")(__c_unsafe.Pointer(&(")
			t.expression(tokens)
			t.appendText(")))")
			return
		}
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
	kernelLinkAddress := t.kernelCodeModel && t.initializing < 0 && t.kernelLinkAddressExpression(tokens) &&
		sourceInfo.kind == cTypePointer &&
		(info.kind == cTypeInt || info.kind == cTypeUint)
	if kernelLinkAddress {
		t.ensureKernelLinkAddressHelper()
		t.appendText("renvo_runtime_CKernelLinkAddress(")
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
	if kernelLinkAddress {
		t.out = append(t.out, ')')
	}
}

func (t *translator) kernelLinkAddressExpression(tokens []token) bool {
	tokens = t.trimExpressionParens(tokens)
	binary := t.binaryOperator(tokens)
	return binary > 0 && binary < len(tokens) &&
		(tokenIs(t.src, tokens[binary], "+") || tokenIs(t.src, tokens[binary], "-"))
}

func (t *translator) ensureKernelLinkAddressHelper() {
	if t.kernelLinkAddressHelper {
		return
	}
	t.staticOut = append(t.staticOut, "func renvo_runtime_CKernelLinkAddress(value uint64) uint64{return value}\n"...)
	t.kernelLinkAddressHelper = true
}

func (t *translator) ensureStringPointerHelper() {
	if t.stringPointerHelper {
		return
	}
	t.staticOut = append(t.staticOut, "func renvo_runtime_CStringPointer(value string) *int8{return nil}\n"...)
	t.stringPointerHelper = true
}

func (t *translator) ensureWideStringPointerHelper() string {
	bit := 1
	typeID := cTypeUint16ID
	if t.wcharSize == 4 {
		bit = 2
		typeID = cTypeUint32ID
	}
	name := "renvo_runtime_CWideStringPointer" + decimalString(t.wcharSize)
	if t.wideStringHelpers&bit != 0 {
		return name
	}
	t.wideStringHelpers |= bit
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(value string) *")
	t.emitType(typeID)
	t.appendText("{return nil}\n")
	t.staticOut = t.out
	t.out = outer
	return name
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
		case tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") || tokenIs(t.src, tokens[i], "{"):
			depth++
		case tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") || tokenIs(t.src, tokens[i], "}"):
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
		// Unary * observes the usual array-to-pointer conversion on its
		// operand.  This matters for expressions such as *pointer_array:
		// the result is one pointer element, not an untyped scalar.
		pointer := t.typeInfo(t.decayedExpressionType(tokens[1:]))
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
	if typeID, ok := t.statementExpressionType(tokens); ok {
		t.emitStatementExpression(tokens, typeID)
		return
	}
	if t.emitGenericSelection(tokens) {
		return
	}
	if t.emitAssignmentValue(tokens) {
		return
	}
	if t.emitBuiltinExpression(tokens) {
		return
	}
	// Split the operators with the weakest remaining precedence before any
	// unary or postfix special form can consume only one side of the expression.
	if at := t.commaOperator(tokens); at >= 0 {
		t.emitCommaExpression(tokens[:at], tokens[at+1:])
		return
	}
	if question, colon := t.conditionalOperator(tokens); question >= 0 {
		if t.emitConstantConditionalExpression(tokens, question, colon) {
			return
		}
		t.emitConditionalExpression(tokens, question, colon)
		return
	}
	if t.emitUint128MultiplyAddShift(tokens) {
		return
	}
	if at := t.binaryOperator(tokens); at >= 0 {
		t.emitBinaryExpression(tokens, at)
		return
	}
	if value, ok := t.qualifiedAddressLvalue(tokens); ok {
		t.emitExpression(value)
		return
	}
	if t.emitPostfixDereference(tokens) {
		return
	}
	if t.emitPointerMutation(tokens) {
		return
	}
	if t.emitScalarPostfixMutation(tokens) {
		return
	}
	if t.emitScalarPrefixMutation(tokens) {
		return
	}
	if t.emitFunctionPointerCall(tokens) {
		return
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "&") && t.arraySubscriptAddressNeedsPointerArithmetic(tokens[1:]) &&
		t.emitOnePastArraySubscriptAddress(tokens[1:]) {
		return
	}
	if t.emitSubscriptMemberCall(tokens) {
		return
	}
	if t.emitSubscriptMemberExpression(tokens) {
		return
	}
	if t.emitPointerSubscript(tokens) {
		return
	}
	if t.emitParenthesizedDirectMember(tokens) {
		return
	}
	if t.emitDirectMemberExpression(tokens) {
		return
	}
	if len(tokens) > 5 && tokenIs(t.src, tokens[0], "&") && tokenIs(t.src, tokens[1], "(") {
		close := matchingToken(t.src, tokens, 1, "(", ")")
		if close > 2 && close+1 < len(tokens) && tokenIs(t.src, tokens[close+1], "{") &&
			matchingToken(t.src, tokens, close+1, "{", "}") == len(tokens)-1 {
			if typeID, ok := t.typeFromTokens(tokens[2:close]); ok {
				info := t.typeInfo(typeID)
				if info.kind == cTypeStruct || info.kind == cTypeUnion || info.kind == cTypeArray {
					t.appendText("&(")
					t.emitInitializerValue(typeID, tokens[close+1:])
					t.appendText(")")
					return
				}
			}
		}
	}
	if len(tokens) > 4 && (tokenIs(t.src, tokens[0], "*") || tokenIs(t.src, tokens[0], "&")) && tokenIs(t.src, tokens[1], "(") {
		close := matchingToken(t.src, tokens, 1, "(", ")")
		if close > 2 && close < len(tokens)-1 {
			if typeID, ok := t.typeFromTokens(tokens[2:close]); ok {
				t.out = append(t.out, tokenText(t.src, tokens[0])...)
				t.out = append(t.out, '(')
				if t.typeInfo(typeID).kind == cTypePointer {
					t.out = append(t.out, '(')
				}
				t.emitType(typeID)
				if t.typeInfo(typeID).kind == cTypePointer {
					t.out = append(t.out, ')')
				}
				t.out = append(t.out, '(')
				t.emitExpression(tokens[close+1:])
				t.appendText("))")
				return
			}
		}
	}
	if len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		binary := t.binaryOperator(tokens)
		if binary == close+1 && binary < len(tokens) && t.unaryOperator(tokens[binary]) {
			binary = -1
		}
		if close > 1 && close < len(tokens)-1 && binary < 0 && !tokenIs(t.src, tokens[close+1], "{") {
			if typeID, ok := t.typeFromTokens(tokens[1:close]); ok {
				operand := t.trimExpressionParens(tokens[close+1:])
				if typeID != cTypeVoidID && t.simpleMutationAssignment(operand) >= 0 {
					pointer := t.typeInfo(typeID).kind == cTypePointer
					if pointer {
						t.out = append(t.out, '(')
					}
					t.emitType(typeID)
					if pointer {
						t.out = append(t.out, ')')
					}
					t.out = append(t.out, '(')
					if t.emitAssignmentValue(operand) {
						t.out = append(t.out, ')')
						return
					}
					return
				}
				if typeID == cTypeVoidID {
					operandType := t.decayedExpressionType(operand)
					if operandType == cTypeVoidID {
						t.emitExpression(operand)
					} else {
						t.appendText(t.ensureDiscardHelper(operandType))
						t.appendText("(")
						t.convertedExpression(operandType, operand)
						t.appendText(")")
					}
					return
				}
				pointer := t.typeInfo(typeID).kind == cTypePointer
				functionPointer := pointer && t.typeInfo(t.typeInfo(typeID).base).kind == cTypeFunction
				if t.object && functionPointer {
					// A C function pointer is one raw machine word in an ELF object.
					// Emitting an inline Go func-type conversion is unnecessary and
					// cannot be parsed as an expression by the compact shared backend.
					t.emitExpression(tokens[close+1:])
					return
				}
				operandType := t.expressionType(tokens[close+1:])
				operandInfo := t.typeInfo(operandType)
				castInfo := t.typeInfo(typeID)
				if (operandInfo.kind == cTypeArray || operandInfo.kind == cTypePointer) &&
					(castInfo.kind == cTypeInt || castInfo.kind == cTypeUint) {
					t.convertedExpression(typeID, tokens[close+1:])
					return
				}
				arrayOperandType := operandType
				arrayOperandKind := t.typeInfo(arrayOperandType).kind
				if arrayOperandKind != cTypeArray && arrayOperandKind != cTypePointer {
					// Parenthesized pointer-member expressions can retain their
					// scalar value type through the general expression inference
					// path even though the selected lvalue is an array. Consult the
					// lvalue type before applying C's array-to-pointer conversion.
					arrayOperandType = t.lvalueType(tokens[close+1:])
				}
				if pointer && t.typeInfo(arrayOperandType).kind == cTypeArray {
					t.usesUnsafe = true
					t.out = append(t.out, '(')
					t.emitType(typeID)
					t.appendText(")(__c_unsafe.Pointer(&(")
					t.emitExpression(tokens[close+1:])
					t.appendText(")))")
					return
				}
				nullPointer := false
				if value, constant := t.constantExpression(tokens[close+1:]); constant && value == 0 {
					nullPointer = true
				}
				if pointer && (nullPointer || t.nullPointerPrefix(tokens[close+1:]) == len(tokens)-close-1 ||
					t.pointerConversionCompatible(typeID, t.expressionType(tokens[close+1:]))) {
					if nullPointer {
						t.appendText("nil")
					} else {
						t.emitExpression(tokens[close+1:])
					}
					return
				}
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
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "!") {
		t.appendText("!(")
		t.emitConditionExpression(tokens[1:])
		t.out = append(t.out, ')')
		return
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "*") {
		operand := t.typeInfo(t.expressionType(tokens[1:]))
		if operand.kind == cTypeArray {
			// C's usual array-to-pointer conversion happens before unary *.
			// Dereferencing that decayed pointer selects the array's first
			// element rather than treating its storage as a pointer word.
			t.appendText("(")
			t.emitExpression(tokens[1:])
			t.appendText(")[0]")
			return
		}
		pointer := t.typeInfo(t.decayedExpressionType(tokens[1:]))
		if pointer.kind == cTypePointer && t.typeInfo(pointer.base).kind == cTypeFunction {
			// In C, applying unary * to a function pointer yields a function
			// designator. Its ordinary value conversion is the original pointer;
			// unlike a data-pointer dereference, it performs no memory load.
			t.emitExpression(tokens[1:])
			return
		}
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[0], "*") || tokenIs(t.src, tokens[0], "&")) {
		t.out = append(t.out, tokenText(t.src, tokens[0])...)
		t.out = append(t.out, '(')
		t.emitExpression(tokens[1:])
		t.out = append(t.out, ')')
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
			t.out = append(t.out, '(')
		}
		t.convertedExpression(typeID, tokens[1:])
		t.appendText("))")
		return
	}
	if len(tokens) > 1 && t.sizeOperator(tokens[0]) {
		t.emitSizeof(tokens)
		return
	}
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tokenIs(t.src, tok, "_Generic") && i+1 < len(tokens) && tokenIs(t.src, tokens[i+1], "(") {
			end := matchingToken(t.src, tokens, i+1, "(", ")")
			if end > i+1 && t.emitGenericSelection(tokens[i:end+1]) {
				i = end
				continue
			}
		}
		if tokenKind(tok) == tokenIdent && i+1 < len(tokens) && tokenIs(t.src, tokens[i+1], "(") {
			close := matchingToken(t.src, tokens, i+1, "(", ")")
			if close >= i+1 && t.diagnosticErrorName(tokenText(t.src, tok)) {
				t.ensureUndefinedInstructionHelper()
				t.appendText("renvo_runtime_CUndefinedInstruction()")
				i = close
				continue
			}
			if function, ok := t.lookupFunction(tokenText(t.src, tok)); ok {
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
						if t.emitDynamicOffsetof(typeID, parts[1]) {
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
			if value, ok := t.cWideStringBytes(tokens[i:end]); ok {
				t.appendText(t.ensureWideStringPointerHelper())
				t.out = append(t.out, '(')
				t.emitByteString(value)
				t.out = append(t.out, ')')
			} else {
				t.emitCStringValues(tokens[i:end])
			}
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
			} else {
				if t.isolateGoBuiltins && !t.object && t.cIdentifierIsGoPredeclared(string(text)) {
					text = []byte("__c_name_" + string(text))
				}
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

func (t *translator) emitConstantConditionalExpression(tokens []token, question int, colon int) bool {
	condition, known := t.constantCondition(tokens[:question])
	if !known {
		return false
	}
	leftTokens := tokens[question+1 : colon]
	if len(leftTokens) == 0 {
		leftTokens = tokens[:question]
	}
	rightTokens := tokens[colon+1:]
	left, right := t.decayedExpressionType(leftTokens), t.decayedExpressionType(rightTokens)
	typeID := t.usualArithmeticType(left, right)
	if t.typeInfo(left).kind == cTypePointer {
		typeID = left
	} else if t.typeInfo(right).kind == cTypePointer {
		typeID = right
	}
	selected := rightTokens
	if condition != 0 {
		selected = leftTokens
	}
	t.convertedExpression(typeID, selected)
	return true
}

func (t *translator) ensureDiscardHelper(typeID int) string {
	name := "__c_discard_" + decimalString(typeID)
	for i := 0; i < len(t.discardTypes); i++ {
		if t.discardTypes[i] == typeID {
			return name
		}
	}
	t.discardTypes = append(t.discardTypes, typeID)
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(value ")
	t.emitType(typeID)
	t.appendText("){_=value}\n")
	t.staticOut = t.out
	t.out = outer
	return name
}

func (t *translator) trimExpressionParens(tokens []token) []token {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	return tokens
}

func (t *translator) uint128ConversionOperand(tokens []token) []token {
	tokens = t.trimExpressionParens(tokens)
	if len(tokens) < 4 || !tokenIs(t.src, tokens[0], "(") {
		return tokens
	}
	close := matchingToken(t.src, tokens, 0, "(", ")")
	if close <= 1 || close >= len(tokens)-1 {
		return tokens
	}
	typeID, ok := t.typeFromTokens(tokens[1:close])
	if !ok || t.typeSize(typeID) != 16 {
		return tokens
	}
	return t.trimExpressionParens(tokens[close+1:])
}

func (t *translator) emitUint128MultiplyAddShift(tokens []token) bool {
	tokens = t.trimExpressionParens(tokens)
	shift := t.binaryOperator(tokens)
	if shift <= 0 || !tokenIs(t.src, tokens[shift], ">>") || t.typeSize(t.expressionType(tokens[:shift])) != 16 {
		return false
	}
	left := t.trimExpressionParens(tokens[:shift])
	add := t.binaryOperator(left)
	first := left
	var addend []token
	if add > 0 && tokenIs(t.src, left[add], "+") {
		first, addend = t.trimExpressionParens(left[:add]), t.trimExpressionParens(left[add+1:])
	}
	multiply := t.binaryOperator(first)
	if (multiply <= 0 || !tokenIs(t.src, first[multiply], "*")) && len(addend) != 0 {
		first, addend = addend, first
		multiply = t.binaryOperator(first)
	}
	if multiply <= 0 || !tokenIs(t.src, first[multiply], "*") {
		return false
	}
	if !t.uint128MultiplyShift {
		t.uint128MultiplyShift = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CMultiplyAddShift64(value uint64,multiplier uint64,addend uint64,shift uint32) uint64{return 0}\n"...)
	}
	t.appendText("renvo_runtime_CMultiplyAddShift64(uint64(")
	t.emitExpression(t.uint128ConversionOperand(first[:multiply]))
	t.appendText("),uint64(")
	t.emitExpression(t.uint128ConversionOperand(first[multiply+1:]))
	t.appendText("),uint64(")
	if len(addend) == 0 {
		t.out = append(t.out, '0')
	} else {
		t.emitExpression(t.uint128ConversionOperand(addend))
	}
	t.appendText("),uint32(")
	t.emitExpression(tokens[shift+1:])
	t.appendText("))")
	return t.ok
}

func (t *translator) emitGenericSelection(tokens []token) bool {
	if len(tokens) < 4 || !tokenIs(t.src, tokens[0], "_Generic") || !tokenIs(t.src, tokens[1], "(") {
		return false
	}
	end := matchingToken(t.src, tokens, 1, "(", ")")
	if end != len(tokens)-1 {
		return false
	}
	items := splitTopLevel(t.src, tokens[2:end], ",")
	if len(items) < 2 {
		t.fail(TranslateErrUnsupported)
		return true
	}
	controlType := t.expressionType(items[0])
	var selected []token
	var fallback []token
	for i := 1; i < len(items); i++ {
		colon := topLevelToken(t.src, items[i], ":")
		if colon <= 0 || colon+1 >= len(items[i]) {
			t.fail(TranslateErrUnsupported)
			return true
		}
		if colon == 1 && tokenIs(t.src, items[i][0], "default") {
			fallback = items[i][colon+1:]
			continue
		}
		associationType, ok := t.typeFromTokens(items[i][:colon])
		if !ok {
			t.fail(TranslateErrUnsupported)
			return true
		}
		if selected == nil && t.compatibleParameterType(controlType, associationType) {
			selected = items[i][colon+1:]
		}
	}
	if selected == nil {
		selected = fallback
	}
	if selected == nil {
		t.fail(TranslateErrUnsupported)
		return true
	}
	// The controlling expression and unselected associations are not evaluated.
	t.emitExpression(selected)
	return true
}

func (t *translator) emitParenthesizedDirectMember(tokens []token) bool {
	if len(tokens) < 5 || !tokenIs(t.src, tokens[0], "(") {
		return false
	}
	close := matchingToken(t.src, tokens, 0, "(", ")")
	if close < 2 || close+2 >= len(tokens) || close+3 != len(tokens) ||
		!tokenIs(t.src, tokens[close+1], "->") && !tokenIs(t.src, tokens[close+1], ".") ||
		tokenKind(tokens[close+2]) != tokenIdent {
		return false
	}
	aggregateType := t.expressionType(tokens[1:close])
	arrayArrow := false
	if tokenIs(t.src, tokens[close+1], "->") {
		pointer := t.typeInfo(aggregateType)
		if pointer.kind == cTypeArray {
			aggregateType = pointer.base
			arrayArrow = true
		} else if pointer.kind != cTypePointer {
			return false
		} else {
			aggregateType = pointer.base
		}
	}
	aggregate := t.typeInfo(aggregateType)
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[close+2]))
	if !ok || aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion || field.bitWidth != 0 {
		return false
	}
	needsAccessor := aggregate.kind == cTypeUnion || aggregate.indirect || !field.emit
	if needsAccessor && !tokenIs(t.src, tokens[close+1], "->") {
		if !tokenIs(t.src, tokens[1], "*") {
			// The generated pointer accessor requires an addressable receiver. A
			// parenthesized value (not pointer) can be a non-addressable expression.
			return false
		}
	}
	if needsAccessor {
		t.appendText("(*(")
		receiver := tokens[1:close]
		if tokenIs(t.src, tokens[close+1], ".") {
			receiver = tokens[2:close]
		}
		t.emitExpression(receiver)
		if arrayArrow {
			t.appendText(")[0].")
		} else {
			t.appendText(").")
		}
		tappend := cPointerAccessorName(field)
		t.appendText(tappend)
		t.appendText("())")
		return t.ok
	}
	t.appendText("(")
	t.emitExpression(tokens[1:close])
	if arrayArrow {
		t.appendText(")[0].")
	} else {
		t.appendText(").")
	}
	t.appendText(field.goName)
	return t.ok
}

func (t *translator) emitDirectMemberExpression(tokens []token) bool {
	if t.emitTransparentUnionParameterMember(tokens) {
		return true
	}
	if len(tokens) > 3 && tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close < len(tokens)-1 {
			if _, cast := t.typeFromTokens(tokens[1:close]); cast {
				// The postfix selector belongs to the cast operand. A selector on
				// the cast result is parenthesized as `((T *)value)->field` and is
				// handled after the outer grouping is emitted.
				return false
			}
		}
	}
	if len(tokens) > 1 && (t.unaryArithmetic(tokens[0]) || tokenIs(t.src, tokens[0], "!") ||
		tokenIs(t.src, tokens[0], "&") || tokenIs(t.src, tokens[0], "*")) {
		// Postfix member access binds more tightly than every unary operator.
		// Let the unary emitter consume the complete selector operand rather than
		// treating the unary prefix as the selector's aggregate receiver.
		return false
	}
	if len(tokens) > 0 && tokenKind(tokens[0]) == tokenIdent {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			hasSubscript := false
			for i := 1; i < len(tokens); i++ {
				hasSubscript = hasSubscript || tokenIs(t.src, tokens[i], "[")
			}
			if !hasSubscript && t.typeInfo(object.typeID).kind != cTypeArray {
				return false
			}
		}
	}
	member := t.finalDirectMemberToken(tokens)
	if member <= 0 || member+2 != len(tokens) || tokenKind(tokens[member+1]) != tokenIdent {
		return false
	}
	baseTokens := tokens[:member]
	baseType := t.expressionType(baseTokens)
	arrow := tokenIs(t.src, tokens[member], "->")
	aggregateType := baseType
	arrayArrow := false
	if arrow {
		pointer := t.typeInfo(baseType)
		if pointer.kind == cTypeArray {
			aggregateType = pointer.base
			arrayArrow = true
		} else if pointer.kind != cTypePointer {
			return false
		} else {
			aggregateType = pointer.base
		}
	}
	aggregate := t.typeInfo(aggregateType)
	if aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion {
		return false
	}
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[member+1]))
	if !ok {
		return false
	}
	if field.bitWidth > 0 {
		t.out = append(t.out, '(')
		t.emitExpression(baseTokens)
		if arrayArrow {
			t.appendText(")[0].__c_get_")
		} else {
			t.appendText(").__c_get_")
		}
		t.out = append(t.out, field.name...)
		t.appendText("()")
		return t.ok
	}
	if arrow || t.finalDirectMemberToken(baseTokens) > 0 {
		before := len(t.out)
		t.appendText("(*(")
		if t.emitDirectMemberPointer(tokens) {
			t.appendText("))")
			return t.ok
		}
		t.out = t.out[:before]
	}
	if aggregate.kind == cTypeUnion || aggregate.indirect || !field.emit {
		t.appendText("(*(")
		t.emitExpression(baseTokens)
		t.appendText(").")
		t.appendText(cPointerAccessorName(field))
		t.appendText("())")
		return t.ok
	}
	t.out = append(t.out, '(')
	t.emitExpression(baseTokens)
	if arrayArrow {
		t.appendText(")[0].")
	} else {
		t.appendText(").")
	}
	t.out = append(t.out, field.goName...)
	return t.ok
}

func (t *translator) emitTransparentUnionParameterMember(tokens []token) bool {
	if len(tokens) != 3 || tokenKind(tokens[0]) != tokenIdent || !tokenIs(t.src, tokens[1], ".") ||
		tokenKind(tokens[2]) != tokenIdent {
		return false
	}
	object, ok := t.lookupObject(tokenText(t.src, tokens[0]))
	if !ok || object.transparentTypeID <= 0 {
		return false
	}
	field, ok := t.lookupField(object.transparentTypeID, tokenText(t.src, tokens[2]))
	if !ok {
		return false
	}
	if field.typeID != object.typeID {
		t.out = append(t.out, '(')
		t.emitType(field.typeID)
		t.out = append(t.out, ')', '(')
	}
	t.out = append(t.out, object.goName...)
	if field.typeID != object.typeID {
		t.out = append(t.out, ')')
	}
	return true
}

func (t *translator) finalDirectMemberToken(tokens []token) int {
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
	return member
}

func (t *translator) emitDirectMemberPointer(tokens []token) bool {
	mark := len(t.out)
	member := t.finalDirectMemberToken(tokens)
	if member <= 0 || member+2 != len(tokens) || tokenKind(tokens[member+1]) != tokenIdent {
		return false
	}
	baseTokens := tokens[:member]
	baseType := t.expressionType(baseTokens)
	arrow := tokenIs(t.src, tokens[member], "->")
	aggregateType := baseType
	arrayArrow := false
	if arrow {
		pointer := t.typeInfo(baseType)
		if pointer.kind == cTypeArray {
			aggregateType = pointer.base
			arrayArrow = true
		} else if pointer.kind != cTypePointer {
			return false
		} else {
			aggregateType = pointer.base
		}
	}
	aggregate := t.typeInfo(aggregateType)
	if aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion {
		return false
	}
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[member+1]))
	if !ok || field.bitWidth != 0 {
		return false
	}
	if arrow {
		t.out = append(t.out, '(')
		t.emitExpression(baseTokens)
		if arrayArrow {
			t.appendText(")[0].")
		} else {
			t.appendText(").")
		}
	} else {
		t.out = append(t.out, '(')
		if !t.emitDirectMemberPointer(baseTokens) {
			t.out = t.out[:mark]
			return false
		}
		t.appendText(").")
	}
	if aggregate.kind == cTypeUnion || aggregate.indirect || !field.emit {
		t.appendText(cPointerAccessorName(field))
	} else {
		t.appendText(t.ensureNestedFieldAccessor(aggregateType, field))
	}
	t.appendText("()")
	return t.ok
}

func (t *translator) emitAssignmentValue(tokens []token) bool {
	if t.commaOperator(tokens) >= 0 {
		return false
	}
	assignment := t.simpleMutationAssignment(tokens)
	if assignment <= 0 || assignment+1 >= len(tokens) {
		return false
	}
	left, right := tokens[:assignment], tokens[assignment+1:]
	if tokenIs(t.src, tokens[assignment], "=") && t.emitPrefixDereferenceAssignment(left, right) {
		return true
	}
	if tokenIs(t.src, tokens[assignment], "=") && t.emitPostfixDereferenceAssignment(left, right) {
		return true
	}
	typeID := t.lvalueType(left)
	if typeID == cTypeVoidID {
		return false
	}
	if !tokenIs(t.src, tokens[assignment], "=") {
		info := t.typeInfo(typeID)
		increment := tokenIs(t.src, tokens[assignment], "+=")
		decrement := tokenIs(t.src, tokens[assignment], "-=")
		if info.kind == cTypePointer && (increment || decrement) {
			key := typeID * 2
			name := "__c_pointer_assignment_inc_value_" + decimalString(typeID)
			if decrement {
				key++
				name = "__c_pointer_assignment_dec_value_" + decimalString(typeID)
			}
			found := false
			for i := 0; i < len(t.pointerAssignValueKey); i++ {
				found = found || t.pointerAssignValueKey[i] == key
			}
			if !found {
				t.pointerAssignValueKey = append(t.pointerAssignValueKey, key)
				step := t.ensurePointerStepHelper(typeID, decrement)
				outer := t.out
				t.out = nil
				t.appendText("func ")
				t.appendText(name)
				t.appendText("(target *")
				t.emitType(typeID)
				t.appendText(",count uintptr) ")
				t.emitType(typeID)
				t.appendText("{*target=")
				t.appendText(step)
				t.appendText("(*target,count);return *target}\n")
				t.staticOut = append(t.staticOut, t.out...)
				t.out = outer
			}
			t.appendText(name)
			t.appendText("(&(")
			t.emitExpression(left)
			t.appendText("),uintptr(")
			t.emitExpression(right)
			t.appendText("))")
			return true
		}
		return t.emitCompoundLvalueAssignment(typeID, left, tokens[assignment], right)
	}
	name := "__c_assignment_value_" + decimalString(typeID)
	found := false
	for i := 0; i < len(t.assignmentValueTypes); i++ {
		found = found || t.assignmentValueTypes[i] == typeID
	}
	if !found {
		t.assignmentValueTypes = append(t.assignmentValueTypes, typeID)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(target *")
		t.emitType(typeID)
		t.appendText(",value ")
		t.emitType(typeID)
		t.appendText(") ")
		t.emitType(typeID)
		info := t.typeInfo(typeID)
		if info.size == 0 && (info.kind == cTypeStruct || info.kind == cTypeUnion) {
			t.appendText("{return value}\n")
		} else {
			t.appendText("{*target=value;return *target}\n")
		}
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(left)
	t.appendText("),")
	t.convertedExpression(typeID, right)
	t.appendText(")")
	return true
}

func (t *translator) simpleMutationAssignment(tokens []token) int {
	depth := 0
	assignment := -1
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") || tokenIs(t.src, tokens[i], "{"):
			depth++
		case tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") || tokenIs(t.src, tokens[i], "}"):
			depth--
		case depth == 0 && assignment < 0 && isAssignmentToken(t.src, tokens[i]):
			assignment = i
		}
	}
	return assignment
}

func (t *translator) simpleAssignment(tokens []token) int {
	depth := 0
	assignment := -1
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "(") || tokenIs(t.src, tokens[i], "[") || tokenIs(t.src, tokens[i], "{"):
			depth++
		case tokenIs(t.src, tokens[i], ")") || tokenIs(t.src, tokens[i], "]") || tokenIs(t.src, tokens[i], "}"):
			depth--
		case depth == 0 && assignment < 0 && tokenIs(t.src, tokens[i], "="):
			assignment = i
		}
	}
	return assignment
}

func (t *translator) emitStatementExpression(tokens []token, typeID int) {
	lastStart, lastEnd, ok := t.statementExpressionValue(tokens)
	if !ok && typeID == cTypeVoidID && t.statementExpressionWithoutValue(tokens) {
		lastStart, lastEnd, ok = len(tokens)-1, len(tokens)-1, true
	}
	if !ok {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.typeSerial++
	name := "__c_statement_expr_" + decimalString(t.typeSerial)
	captureMark := t.limitCapturesToTokens(tokens)
	t.appendText(name)
	t.appendText("(")
	t.emitCaptureArguments(false)
	t.appendText(")")

	outer, names, captureIndices := t.beginHelper()
	t.emitHelperObjectMetadata(name, typeID)
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(")
	t.emitCaptureParameters(false)
	t.appendText(")")
	if typeID != cTypeVoidID {
		t.appendText(" ")
		t.emitType(typeID)
	}
	t.appendText("{")

	outerTokens, outerPos := t.tokens, t.pos
	outerResultType := t.resultType
	objectMark := len(t.objects)
	labelMark := len(t.localLabels)
	scope := t.beginScope()
	t.tokens, t.pos, t.resultType = tokens, 1, typeID
	stop := lastStart
	if typeID == cTypeVoidID {
		stop = len(tokens) - 1
	}
	for t.ok && t.pos < stop {
		t.statement()
		if t.ok {
			t.out = append(t.out, '\n')
		}
	}
	if t.ok && t.pos != stop {
		t.fail(TranslateErrUnsupported)
	}
	if t.ok && typeID != cTypeVoidID {
		value := tokens[lastStart:lastEnd]
		for len(value) >= 2 && tokenKind(value[0]) == tokenIdent && tokenIs(t.src, value[1], ":") {
			t.appendText(t.localLabelGoName(string(tokenText(t.src, value[0]))))
			t.appendText(":")
			value = value[2:]
		}
		t.appendText("return ")
		t.convertedExpression(typeID, value)
		t.appendText("}")
	} else if t.ok {
		t.appendText("}")
	}
	t.tokens, t.pos, t.resultType = outerTokens, outerPos, outerResultType
	t.objects = t.objects[:objectMark]
	t.localLabels = t.localLabels[:labelMark]
	t.endScope(scope)
	t.endHelper(outer, names, captureIndices)
	t.captureExclusions = t.captureExclusions[:captureMark]
}

func (t *translator) emitPointerMutation(tokens []token) bool {
	if len(tokens) < 2 {
		return false
	}
	var value []token
	decrement := false
	postfix := false
	if tokenIs(t.src, tokens[len(tokens)-1], "++") || tokenIs(t.src, tokens[len(tokens)-1], "--") {
		value = tokens[:len(tokens)-1]
		// Postfix mutation binds more tightly than comma and conditional
		// expressions.  It also binds more tightly than an unparenthesized
		// unary prefix: *p++ mutates p, whereas (*p)++ mutates *p.
		if t.commaOperator(value) >= 0 {
			return false
		}
		if question, _ := t.conditionalOperator(value); question >= 0 {
			return false
		}
		if len(value) > 1 && !tokenIs(t.src, value[0], "(") && t.unaryOperator(value[0]) {
			return false
		}
		decrement = tokenIs(t.src, tokens[len(tokens)-1], "--")
		postfix = true
	} else if tokenIs(t.src, tokens[0], "++") || tokenIs(t.src, tokens[0], "--") {
		value = tokens[1:]
		decrement = tokenIs(t.src, tokens[0], "--")
	} else {
		return false
	}
	typeID := t.lvalueType(value)
	if t.typeInfo(typeID).kind != cTypePointer {
		return false
	}
	key := (typeID + 1) * 4
	if decrement {
		key++
	}
	if postfix {
		key += 2
	}
	name := "__c_pointer_"
	if postfix {
		name += "post_"
	} else {
		name += "pre_"
	}
	if decrement {
		name += "dec_"
	} else {
		name += "inc_"
	}
	name += decimalString(typeID)
	found := false
	for i := 0; i < len(t.pointerMutationTypes); i++ {
		found = found || t.pointerMutationTypes[i] == key
	}
	if !found {
		t.pointerMutationTypes = append(t.pointerMutationTypes, key)
		base := t.typeInfo(typeID).base
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(p **")
		t.emitType(base)
		t.appendText(") *")
		t.emitType(base)
		t.appendText("{")
		if postfix {
			t.appendText("value:=*p;")
		}
		t.appendText("*p=(*")
		t.emitType(base)
		t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(*p))")
		if decrement {
			t.out = append(t.out, '-')
		} else {
			t.out = append(t.out, '+')
		}
		t.appendText("uintptr(")
		t.appendDecimal(t.typeSize(base))
		t.appendText(")));")
		if postfix {
			t.appendText("return value}")
		} else {
			t.appendText("return *p}")
		}
		t.out = append(t.out, '\n')
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
		t.usesUnsafe = true
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(value)
	t.appendText("))")
	return true
}

func (t *translator) emitScalarPostfixMutation(tokens []token) bool {
	if len(tokens) < 2 || !tokenIs(t.src, tokens[len(tokens)-1], "++") && !tokenIs(t.src, tokens[len(tokens)-1], "--") {
		return false
	}
	value := tokens[:len(tokens)-1]
	// A trailing postfix operator can belong to the right operand of a
	// lower-precedence expression.  In particular, a for post clause such as
	// "left++, right++" is a comma expression, not an increment of the comma
	// expression's result.
	if t.commaOperator(value) >= 0 {
		return false
	}
	if question, _ := t.conditionalOperator(value); question >= 0 {
		return false
	}
	typeID := t.lvalueType(value)
	info := t.typeInfo(typeID)
	if info.kind != cTypeInt && info.kind != cTypeUint {
		return false
	}
	decrement := tokenIs(t.src, tokens[len(tokens)-1], "--")
	key := typeID + 1
	if decrement {
		key = -key
	}
	name := "__c_scalar_post_"
	if decrement {
		name += "dec_"
	} else {
		name += "inc_"
	}
	name += decimalString(typeID)
	found := false
	for i := 0; i < len(t.postfixScalarTypes); i++ {
		found = found || t.postfixScalarTypes[i] == key
	}
	if !found {
		t.postfixScalarTypes = append(t.postfixScalarTypes, key)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(p *")
		t.emitType(typeID)
		t.appendText(") ")
		t.emitType(typeID)
		t.appendText("{value:=*p;*p=*p")
		if decrement {
			t.out = append(t.out, '-')
		} else {
			t.out = append(t.out, '+')
		}
		t.appendText("1;return value}\n")
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(value)
	t.appendText("))")
	return true
}

func (t *translator) emitScalarPrefixMutation(tokens []token) bool {
	if len(tokens) < 2 || !tokenIs(t.src, tokens[0], "++") && !tokenIs(t.src, tokens[0], "--") {
		return false
	}
	value := tokens[1:]
	typeID := t.lvalueType(value)
	info := t.typeInfo(typeID)
	if info.kind != cTypeInt && info.kind != cTypeUint {
		return false
	}
	decrement := tokenIs(t.src, tokens[0], "--")
	key := typeID + 1
	if decrement {
		key = -key
	}
	name := "__c_scalar_pre_"
	if decrement {
		name += "dec_"
	} else {
		name += "inc_"
	}
	name += decimalString(typeID)
	found := false
	for i := 0; i < len(t.prefixScalarTypes); i++ {
		found = found || t.prefixScalarTypes[i] == key
	}
	if !found {
		t.prefixScalarTypes = append(t.prefixScalarTypes, key)
		outer := t.out
		t.out = nil
		t.appendText("func ")
		t.appendText(name)
		t.appendText("(p *")
		t.emitType(typeID)
		t.appendText(") ")
		t.emitType(typeID)
		t.appendText("{*p=*p")
		if decrement {
			t.out = append(t.out, '-')
		} else {
			t.out = append(t.out, '+')
		}
		t.appendText("1;return *p}\n")
		t.staticOut = append(t.staticOut, t.out...)
		t.out = outer
	}
	t.appendText(name)
	t.appendText("(&(")
	t.emitExpression(value)
	t.appendText("))")
	return true
}

func (t *translator) emitPostfixDereference(tokens []token) bool {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[0], "*") {
		return false
	}
	operand := tokens[1:]
	for len(operand) > 2 && tokenIs(t.src, operand[0], "(") &&
		matchingToken(t.src, operand, 0, "(", ")") == len(operand)-1 {
		operand = operand[1 : len(operand)-1]
	}
	if len(operand) < 2 || !tokenIs(t.src, operand[len(operand)-1], "++") &&
		!tokenIs(t.src, operand[len(operand)-1], "--") {
		return false
	}
	value := operand[:len(operand)-1]
	if len(value) > 3 && tokenIs(t.src, value[0], "(") {
		close := matchingToken(t.src, value, 0, "(", ")")
		if close > 1 && close+2 == len(value) && tokenKind(value[close+1]) == tokenIdent {
			castType, castOK := t.typeFromTokens(value[1:close])
			sourceType := t.expressionType(value[close+1:])
			cast, source := t.typeInfo(castType), t.typeInfo(sourceType)
			if castOK && cast.kind == cTypePointer && source.kind == cTypePointer {
				decrement := tokenIs(t.src, operand[len(operand)-1], "--")
				helper := t.ensureCastedPostfixDereferenceHelper(sourceType, cast.base, decrement)
				t.appendText(helper)
				t.appendText("(&(")
				t.emitExpression(value[close+1:])
				t.appendText("))")
				return true
			}
		}
	}
	pointerType := t.typeInfo(t.expressionType(value))
	if pointerType.kind != cTypePointer {
		return false
	}
	decrement := tokenIs(t.src, operand[len(operand)-1], "--")
	helper := t.ensurePostfixDereferenceHelper(pointerType.base, decrement)
	t.appendText(helper)
	t.appendText("(&(")
	t.emitExpression(value)
	t.appendText("))")
	return true
}

func (t *translator) ensureCastedPostfixDereferenceHelper(pointerType int, resultType int, decrement bool) string {
	key := decimalString(pointerType) + "_" + decimalString(resultType)
	name := "__c_post_cast_deref_"
	if decrement {
		key += "_dec"
		name += "dec_"
	} else {
		key += "_inc"
		name += "inc_"
	}
	name += decimalString(pointerType) + "_" + decimalString(resultType)
	for i := 0; i < len(t.postfixCastedDerefKeys); i++ {
		if t.postfixCastedDerefKeys[i] == key {
			return name
		}
	}
	t.postfixCastedDerefKeys = append(t.postfixCastedDerefKeys, key)
	sourceBase := t.typeInfo(pointerType).base
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(p **")
	t.emitType(sourceBase)
	t.appendText(") ")
	t.emitType(resultType)
	t.appendText("{value:=*((*")
	t.emitType(resultType)
	t.appendText(")(__c_unsafe.Pointer(*p)));*p=(*")
	t.emitType(sourceBase)
	t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(*p))")
	if decrement {
		t.appendText("-")
	} else {
		t.appendText("+")
	}
	t.appendText("uintptr(")
	t.appendDecimal(t.typeSize(sourceBase))
	t.appendText(")));return value}\n")
	t.staticOut = t.out
	t.out = outer
	t.usesUnsafe = true
	return name
}

func (t *translator) ensurePostfixDereferenceHelper(typeID int, decrement bool) string {
	key := typeID + 1
	if decrement {
		key = -key
	}
	name := "__c_post_deref_"
	if decrement {
		name += "dec_"
	} else {
		name += "inc_"
	}
	name += decimalString(typeID)
	for i := 0; i < len(t.postfixDerefTypes); i++ {
		if t.postfixDerefTypes[i] == key {
			return name
		}
	}
	t.postfixDerefTypes = append(t.postfixDerefTypes, key)
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(p **")
	t.emitType(typeID)
	t.appendText(") ")
	t.emitType(typeID)
	t.appendText("{value:=**p;*p=(*")
	t.emitType(typeID)
	t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(*p))")
	if decrement {
		t.appendText("-")
	} else {
		t.appendText("+")
	}
	t.appendText("uintptr(")
	t.appendDecimal(t.typeSize(typeID))
	t.appendText(")));return value}\n")
	t.staticOut = t.out
	t.out = outer
	t.usesUnsafe = true
	return name
}

func (t *translator) ensurePointerStepHelper(pointerType int, decrement bool) string {
	key := pointerType + 1
	if decrement {
		key = -key
	}
	name := "__c_pointer_step_"
	if decrement {
		name += "dec_"
	} else {
		name += "inc_"
	}
	name += decimalString(pointerType)
	for i := 0; i < len(t.pointerStepTypes); i++ {
		if t.pointerStepTypes[i] == key {
			return name
		}
	}
	t.pointerStepTypes = append(t.pointerStepTypes, key)
	base := t.typeInfo(pointerType).base
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(pointer ")
	t.emitType(pointerType)
	t.appendText(",count uintptr) ")
	t.emitType(pointerType)
	t.appendText("{return (")
	t.emitType(pointerType)
	t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(pointer))")
	if decrement {
		t.appendText("-")
	} else {
		t.appendText("+")
	}
	t.appendText("count*uintptr(")
	t.appendDecimal(t.typeSize(base))
	t.appendText(")))}\n")
	t.staticOut = t.out
	t.out = outer
	t.usesUnsafe = true
	return name
}

func (t *translator) emitPointerSubscript(tokens []token) bool {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[len(tokens)-1], "]") {
		return false
	}
	if tokenIs(t.src, tokens[0], "(") {
		close := matchingToken(t.src, tokens, 0, "(", ")")
		if close > 1 && close+1 < len(tokens) {
			if t.unaryOperator(tokens[close+1]) {
				return false
			}
			if _, cast := t.typeFromTokens(tokens[1:close]); cast {
				// In (T *)rows[i], postfix subscripting binds before the
				// leading cast. Treating the cast as part of the subscript base
				// changes a pointer-to-row stride into sizeof(T).
				return false
			}
		}
	}
	// Postfix subscripting binds more tightly than an unparenthesized unary
	// prefix: ~values[i] is ~(values[i]), not (~values)[i].
	if t.unaryOperator(tokens[0]) {
		return false
	}
	paren, bracket, open := 0, 0, -1
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
		return false
	}
	if t.binaryOperator(tokens[:open]) >= 0 {
		return false
	}
	pointerType := t.expressionType(tokens[:open])
	pointerInfo := t.typeInfo(pointerType)
	if pointerInfo.kind == cTypeArray {
		if index, constant := t.constantExpression(tokens[open+1 : len(tokens)-1]); t.initializing >= 0 && constant && index >= 0 && index < pointerInfo.count {
			// Keep in-bounds static-initializer subscripts as Go constant
			// addresses so the object backend can emit a relocation. A helper
			// call would turn constructs such as &nodes[0].next into a dynamic
			// initializer, which C does not require.
			t.emitExpression(tokens[:open])
			t.out = append(t.out, '[')
			t.emitExpression(tokens[open+1 : len(tokens)-1])
			t.out = append(t.out, ']')
			return true
		}
		// C array subscripting is defined in terms of pointer arithmetic. Do not
		// lower it to a Go array index: that would add Go bounds checks and panic
		// state to C code, and would reject the unchecked accesses C permits.
		elementPointer := t.pointerType(pointerInfo.base)
		t.appendText("(*")
		t.appendText(t.ensureArrayIndexHelper(elementPointer))
		t.appendText("(__c_unsafe.Pointer(&(")
		t.emitExpression(tokens[:open])
		t.appendText(")),")
		t.appendText("uintptr(")
		t.emitExpression(tokens[open+1 : len(tokens)-1])
		t.appendText(")))")
		t.usesUnsafe = true
		return t.ok
	}
	if t.typeInfo(pointerType).kind != cTypePointer {
		return false
	}
	helper := t.ensurePointerIndexHelper(pointerType)
	t.appendText("(*")
	t.appendText(helper)
	t.appendText("(")
	t.convertedExpression(pointerType, tokens[:open])
	t.appendText(",uintptr(")
	t.emitExpression(tokens[open+1 : len(tokens)-1])
	t.appendText(")))")
	return true
}

func (t *translator) arraySubscriptAddressNeedsPointerArithmetic(tokens []token) bool {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[len(tokens)-1], "]") {
		return false
	}
	paren, bracket, open := 0, 0, -1
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
		return false
	}
	array := t.typeInfo(t.expressionType(tokens[:open]))
	if array.kind != cTypeArray {
		return false
	}
	index, constant := t.constantExpression(tokens[open+1 : len(tokens)-1])
	return array.count == 0 || constant && index >= array.count
}

func (t *translator) emitOnePastArraySubscriptAddress(tokens []token) bool {
	if t.emitPointerSubscriptAddress(tokens) {
		return true
	}
	if len(tokens) < 3 || !tokenIs(t.src, tokens[len(tokens)-1], "]") {
		return false
	}
	paren, bracket, open := 0, 0, -1
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
		return false
	}
	array := t.typeInfo(t.expressionType(tokens[:open]))
	if array.kind != cTypeArray {
		return false
	}
	elementPointer := t.pointerType(array.base)
	t.appendText(t.ensurePointerIndexHelper(elementPointer))
	t.appendText("((")
	t.emitType(elementPointer)
	t.appendText(")(__c_unsafe.Pointer(&(")
	t.emitExpression(tokens[:open])
	t.appendText(")[0])),uintptr(")
	t.emitExpression(tokens[open+1 : len(tokens)-1])
	t.appendText("))")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitSubscriptMemberExpression(tokens []token) bool {
	if len(tokens) > 1 && (t.unaryArithmetic(tokens[0]) || tokenIs(t.src, tokens[0], "!") ||
		tokenIs(t.src, tokens[0], "&") || tokenIs(t.src, tokens[0], "*")) {
		// The subscript and following member access both bind before a leading
		// unary operator. Leave the complete postfix expression to the unary
		// emitter instead of folding the prefix into the pointer base.
		return false
	}
	paren, bracket, member := 0, 0, -1
	hasSubscript := false
	for i := 0; i+1 < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
			hasSubscript = true
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		case paren == 0 && bracket == 0 && (tokenIs(t.src, tokens[i], ".") || tokenIs(t.src, tokens[i], "->")):
			member = i
		}
	}
	if !hasSubscript || member <= 0 || member+2 != len(tokens) || tokenKind(tokens[member+1]) != tokenIdent {
		return false
	}
	aggregateType := t.expressionType(tokens[:member])
	arrayArrow := false
	if tokenIs(t.src, tokens[member], "->") {
		pointer := t.typeInfo(aggregateType)
		if pointer.kind == cTypeArray {
			aggregateType = pointer.base
			arrayArrow = true
		} else if pointer.kind != cTypePointer {
			return false
		} else {
			aggregateType = pointer.base
		}
	}
	aggregate := t.typeInfo(aggregateType)
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[member+1]))
	if !ok || aggregate.kind != cTypeStruct || aggregate.indirect || !field.emit || field.bitWidth != 0 {
		return false
	}
	start := len(t.out)
	t.appendText("(*")
	if !t.emitPointerSubscriptAddress(tokens[:member]) {
		t.out = t.out[:start]
		t.out = append(t.out, '(')
		t.emitExpression(tokens[:member])
	}
	if arrayArrow {
		t.appendText(")[0].")
	} else {
		t.appendText(").")
	}
	t.appendText(field.goName)
	return t.ok
}

func (t *translator) emitPointerSubscriptAddress(tokens []token) bool {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[len(tokens)-1], "]") {
		return false
	}
	paren, bracket, open := 0, 0, -1
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
	if open <= 0 || matchingToken(t.src, tokens, open, "[", "]") != len(tokens)-1 || t.binaryOperator(tokens[:open]) >= 0 {
		return false
	}
	pointerType := t.expressionType(tokens[:open])
	pointerInfo := t.typeInfo(pointerType)
	if pointerInfo.kind == cTypeArray && pointerInfo.count == 0 {
		elementPointer := t.pointerType(pointerInfo.base)
		t.appendText(t.ensurePointerIndexHelper(elementPointer))
		t.appendText("((")
		t.emitType(elementPointer)
		t.appendText(")(__c_unsafe.Pointer(&(")
		t.emitExpression(tokens[:open])
		t.appendText("))),uintptr(")
		t.emitExpression(tokens[open+1 : len(tokens)-1])
		t.appendText("))")
		t.usesUnsafe = true
		return t.ok
	}
	if pointerInfo.kind == cTypeArray {
		elementPointer := t.pointerType(pointerInfo.base)
		start := len(t.out)
		t.appendText(t.ensurePointerIndexHelper(elementPointer))
		t.appendText("((")
		t.emitType(elementPointer)
		t.appendText(")(__c_unsafe.Pointer(")
		if !t.emitDirectMemberPointer(tokens[:open]) {
			t.out = t.out[:start]
			return false
		}
		t.appendText(")),uintptr(")
		t.emitExpression(tokens[open+1 : len(tokens)-1])
		t.appendText("))")
		t.usesUnsafe = true
		return t.ok
	}
	if pointerInfo.kind != cTypePointer {
		return false
	}
	t.appendText(t.ensurePointerIndexHelper(pointerType))
	t.out = append(t.out, '(')
	t.emitExpression(tokens[:open])
	t.appendText(",uintptr(")
	t.emitExpression(tokens[open+1 : len(tokens)-1])
	t.appendText("))")
	return true
}

func (t *translator) emitSubscriptMemberArrayDecay(pointerType int, tokens []token) bool {
	member := t.finalDirectMemberToken(tokens)
	if member <= 0 || member+2 != len(tokens) || tokenKind(tokens[member+1]) != tokenIdent {
		return false
	}
	baseTokens := tokens[:member]
	hasSubscript := false
	for i := 0; i < len(baseTokens); i++ {
		hasSubscript = hasSubscript || tokenIs(t.src, baseTokens[i], "[")
	}
	if !hasSubscript || tokenIs(t.src, tokens[member], "->") {
		return false
	}
	aggregateType := t.expressionType(baseTokens)
	aggregate := t.typeInfo(aggregateType)
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[member+1]))
	fieldType := t.typeInfo(field.typeID)
	target := t.typeInfo(pointerType)
	if !ok || aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion || field.bitWidth != 0 ||
		fieldType.kind != cTypeArray || target.kind != cTypePointer ||
		!t.compatibleParameterType(target.base, fieldType.base) {
		return false
	}
	start := len(t.out)
	t.out = append(t.out, '(')
	t.emitType(pointerType)
	t.appendText(")(__c_unsafe.Pointer((")
	if !t.emitPointerSubscriptAddress(baseTokens) {
		t.out = t.out[:start]
		return false
	}
	t.appendText(").")
	if aggregate.kind == cTypeUnion || aggregate.indirect || !field.emit {
		t.appendText(cPointerAccessorName(field))
	} else {
		t.appendText(t.ensureNestedFieldAccessor(aggregateType, field))
	}
	t.appendText("()))")
	t.usesUnsafe = true
	return t.ok
}

func (t *translator) emitSubscriptMemberCall(tokens []token) bool {
	// A subscript in an argument or conditional branch does not make the final
	// call a call through a subscripted member. Respect the operators which bind
	// more weakly than the trailing call before inspecting its callee.
	if len(tokens) > 0 && t.unaryOperator(tokens[0]) {
		return false
	}
	open := -1
	paren, bracket := 0, 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			if paren == 0 && bracket == 0 && matchingToken(t.src, tokens, i, "(", ")") == len(tokens)-1 {
				open = i
			}
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		}
	}
	if open <= 0 {
		return false
	}
	if t.binaryOperator(tokens[:open]) >= 0 || t.commaOperator(tokens[:open]) >= 0 {
		return false
	}
	if question, _ := t.conditionalOperator(tokens[:open]); question >= 0 {
		return false
	}
	hasSubscript := false
	for i := 0; i < open; i++ {
		hasSubscript = hasSubscript || tokenIs(t.src, tokens[i], "[")
	}
	if !hasSubscript {
		return false
	}
	functionType := t.expressionType(tokens[:open])
	function := t.typeInfo(functionType)
	if function.kind == cTypePointer {
		functionType = function.base
		function = t.typeInfo(functionType)
	}
	if function.kind != cTypeFunction || function.variadic {
		return false
	}
	args := splitTopLevel(t.src, tokens[open+1:len(tokens)-1], ",")
	if open+1 == len(tokens)-1 {
		args = nil
	}
	if len(args) != function.paramCount {
		t.fail(TranslateErrUnsupported)
		return true
	}
	callee := t.normalizedFunctionPointerCallee(tokens[:open])
	if function.msABI {
		return t.emitMSABICall(functionType, callee, args)
	}
	t.emitExpression(callee)
	t.out = append(t.out, '(')
	for i := 0; i < len(args); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.convertedExpression(t.functionParams[function.paramStart+i], args[i])
	}
	t.out = append(t.out, ')')
	return t.ok
}

func (t *translator) emitFunctionPointerCall(tokens []token) bool {
	// Postfix calls bind before an unparenthesized unary prefix: -source() is
	// -(source()), not a call through the negated function designator.
	if len(tokens) > 0 && t.unaryOperator(tokens[0]) {
		return false
	}
	open := -1
	paren, bracket := 0, 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			if paren == 0 && bracket == 0 && matchingToken(t.src, tokens, i, "(", ")") == len(tokens)-1 {
				open = i
			}
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		}
	}
	if open <= 0 {
		return false
	}
	if t.binaryOperator(tokens[:open]) >= 0 || t.commaOperator(tokens[:open]) >= 0 {
		return false
	}
	if question, _ := t.conditionalOperator(tokens[:open]); question >= 0 {
		return false
	}
	if open == 1 && tokenKind(tokens[0]) == tokenIdent {
		if _, direct := t.lookupFunction(tokenText(t.src, tokens[0])); direct {
			return false
		}
	}
	functionType := t.expressionType(tokens[:open])
	function := t.typeInfo(functionType)
	if function.kind == cTypePointer {
		functionType = function.base
		function = t.typeInfo(functionType)
	}
	if function.kind != cTypeFunction || function.variadic {
		return false
	}
	args := splitTopLevel(t.src, tokens[open+1:len(tokens)-1], ",")
	if open+1 == len(tokens)-1 {
		args = nil
	}
	if len(args) != function.paramCount {
		t.fail(TranslateErrUnsupported)
		return true
	}
	callee := t.normalizedFunctionPointerCallee(tokens[:open])
	if function.msABI {
		return t.emitMSABICall(functionType, callee, args)
	}
	t.emitExpression(callee)
	t.out = append(t.out, '(')
	for i := 0; i < len(args); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.convertedExpression(t.functionParams[function.paramStart+i], args[i])
	}
	t.out = append(t.out, ')')
	return t.ok
}

func (t *translator) normalizedFunctionPointerCallee(tokens []token) []token {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "*") {
		operand := tokens[1:]
		info := t.typeInfo(t.expressionType(operand))
		if info.kind == cTypePointer && t.typeInfo(info.base).kind == cTypeFunction {
			tokens = operand
			for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
				tokens = tokens[1 : len(tokens)-1]
			}
		}
	}
	return tokens
}

func (t *translator) emitMSABICall(functionType int, callee []token, args [][]token) bool {
	function := t.typeInfo(functionType)
	if function.kind == cTypePointer {
		functionType = function.base
		function = t.typeInfo(functionType)
	}
	if function.kind != cTypeFunction || !function.msABI || function.variadic || len(args) != function.paramCount || len(args) > 8 ||
		!t.msABIScalarType(function.base, true) {
		t.fail(TranslateErrUnsupported)
		return true
	}
	for i := 0; i < function.paramCount; i++ {
		if !t.msABIScalarType(t.functionParams[function.paramStart+i], false) {
			t.fail(TranslateErrUnsupported)
			return true
		}
	}
	helper := t.ensureMSABICallHelper(functionType)
	t.appendText(helper)
	t.out = append(t.out, '(')
	t.emitExpression(callee)
	for i := 0; i < len(args); i++ {
		t.out = append(t.out, ',')
		t.convertedExpression(t.functionParams[function.paramStart+i], args[i])
	}
	t.out = append(t.out, ')')
	return t.ok
}

func (t *translator) msABIScalarType(typeID int, allowVoid bool) bool {
	info := t.typeInfo(typeID)
	return allowVoid && info.kind == cTypeVoid || info.kind == cTypeBool || info.kind == cTypeInt ||
		info.kind == cTypeUint || info.kind == cTypePointer
}

func (t *translator) ensureMSABICallHelper(functionType int) string {
	name := "renvo_runtime_CMSABICall_" + decimalString(functionType)
	for i := 0; i < len(t.msABICallTypes); i++ {
		if t.msABICallTypes[i] == functionType {
			return name
		}
	}
	t.msABICallTypes = append(t.msABICallTypes, functionType)
	function := t.typeInfo(functionType)
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(function ")
	t.emitType(functionType)
	for i := 0; i < function.paramCount; i++ {
		t.appendText(",p")
		t.appendDecimal(i)
		t.out = append(t.out, ' ')
		t.emitType(t.functionParams[function.paramStart+i])
	}
	t.out = append(t.out, ')')
	if function.base != cTypeVoidID {
		t.out = append(t.out, ' ')
		t.emitType(function.base)
		t.appendText("{return ")
		if t.typeInfo(function.base).kind == cTypePointer {
			t.appendText("nil")
		} else {
			t.appendText("0")
		}
		t.appendText("}\n")
	} else {
		t.appendText("{}\n")
	}
	t.staticOut = t.out
	t.out = outer
	return name
}

func (t *translator) ensurePointerIndexHelper(pointerType int) string {
	name := "__c_pointer_index_" + decimalString(pointerType)
	for i := 0; i < len(t.pointerIndexTypes); i++ {
		if t.pointerIndexTypes[i] == pointerType {
			return name
		}
	}
	t.pointerIndexTypes = append(t.pointerIndexTypes, pointerType)
	elementType := t.typeInfo(pointerType).base
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(pointer ")
	t.emitType(pointerType)
	t.appendText(",index uintptr) *")
	t.emitType(elementType)
	t.appendText("{return (*")
	t.emitType(elementType)
	t.appendText(")(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(pointer))+index*uintptr(")
	t.appendDecimal(t.typeSize(elementType))
	t.appendText(")))}\n")
	t.staticOut = t.out
	t.out = outer
	t.usesUnsafe = true
	return name
}

func (t *translator) ensureArrayIndexHelper(pointerType int) string {
	name := "__c_array_index_" + decimalString(pointerType)
	for i := 0; i < len(t.arrayIndexTypes); i++ {
		if t.arrayIndexTypes[i] == pointerType {
			return name
		}
	}
	t.arrayIndexTypes = append(t.arrayIndexTypes, pointerType)
	elementType := t.typeInfo(pointerType).base
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(pointer __c_unsafe.Pointer,index uintptr) *")
	t.emitType(elementType)
	t.appendText("{return (*")
	t.emitType(elementType)
	t.appendText(")(__c_unsafe.Pointer(uintptr(pointer)+index*uintptr(")
	t.appendDecimal(t.typeSize(elementType))
	t.appendText(")))}\n")
	t.staticOut = t.out
	t.out = outer
	t.usesUnsafe = true
	return name
}

func (t *translator) emitBuiltinExpression(tokens []token) bool {
	if len(tokens) < 3 || tokenKind(tokens[0]) != tokenIdent || !tokenIs(t.src, tokens[1], "(") ||
		matchingToken(t.src, tokens, 1, "(", ")") != len(tokens)-1 {
		return false
	}
	name := tokenText(t.src, tokens[0])
	args := splitTopLevel(t.src, tokens[2:len(tokens)-1], ",")
	if textEquals(name, "__sync_synchronize") && len(tokens) == 3 {
		t.ensureAsmMemoryHelper()
		t.appendText("renvo_runtime_CMemoryBarrier()")
		return true
	}
	if textEquals(name, "__builtin_unreachable") && len(tokens) == 3 {
		t.ensureUndefinedInstructionHelper()
		t.appendText("renvo_runtime_CUndefinedInstruction()")
		return true
	}
	if textEquals(name, "__builtin_va_start") && len(args) == 2 {
		if !t.variadicFunction || t.typeInfo(t.expressionType(args[0])).kind != cTypeArray {
			t.fail(TranslateErrUnsupported)
			return true
		}
		t.emitExpression(args[0])
		t.appendText("=*__c_va")
		return true
	}
	if textEquals(name, "__builtin_va_end") && len(args) == 1 {
		t.appendText("_=")
		t.emitExpression(args[0])
		return true
	}
	if textEquals(name, "__builtin_va_copy") && len(args) == 2 {
		t.emitBuiltinVACopy(args[0], args[1])
		return true
	}
	if textEquals(name, "__builtin_va_arg") && len(args) == 2 {
		t.emitBuiltinVAArg(args[0], args[1])
		return true
	}
	if textEquals(name, "__builtin_prefetch") && len(args) >= 1 && len(args) <= 3 {
		if t.typeInfo(t.expressionType(args[0])).kind != cTypePointer {
			t.fail(TranslateErrUnsupported)
			return true
		}
		if len(args) >= 2 {
			write, constant := t.constantExpression(args[1])
			if !constant || write < 0 || write > 1 {
				t.fail(TranslateErrUnsupported)
				return true
			}
		}
		if len(args) == 3 {
			locality, constant := t.constantExpression(args[2])
			if !constant || locality < 0 || locality > 3 {
				t.fail(TranslateErrUnsupported)
				return true
			}
		}
		if !t.asmPrefetch {
			t.asmPrefetch = true
			t.staticOut = append(t.staticOut, "func renvo_runtime_CPrefetch(address uintptr){}\n"...)
		}
		t.appendText("renvo_runtime_CPrefetch(uintptr(__c_unsafe.Pointer(")
		t.emitExpression(args[0])
		t.appendText(")))")
		t.usesUnsafe = true
		return true
	}
	if (textEquals(name, "__builtin_expect") && len(args) == 2) ||
		(textEquals(name, "__builtin_expect_with_probability") && len(args) == 3) {
		t.emitExpression(args[0])
		return true
	}
	if textEquals(name, "__builtin_constant_p") && len(args) == 1 {
		if _, ok := t.constantExpression(args[0]); ok {
			t.out = append(t.out, '1')
		} else {
			t.out = append(t.out, '0')
		}
		return true
	}
	if textEquals(name, "__builtin_strlen") && len(args) == 1 {
		t.emitStringLength(args[0])
		return true
	}
	if textEquals(name, "__builtin_isdigit") && len(args) == 1 {
		if !t.builtinIsDigitHelper {
			t.builtinIsDigitHelper = true
			t.staticOut = append(t.staticOut, "func __c_builtin_isdigit(value int32) int32{if value>=48&&value<=57{return 1};return 0}\n"...)
		}
		t.appendText("__c_builtin_isdigit(")
		t.convertedExpression(cTypeInt32ID, args[0])
		t.appendText(")")
		return true
	}
	if (textEquals(name, "__builtin_frame_address") || textEquals(name, "__builtin_return_address")) && len(args) == 1 {
		level, constant := t.constantExpression(args[0])
		frameCaller := textEquals(name, "__builtin_frame_address") && level == 1
		if !constant || level != 0 && !frameCaller {
			t.fail(TranslateErrUnsupported)
			return true
		}
		kind, helper := 1, "renvo_runtime_CFrameAddress"
		if frameCaller {
			kind, helper = 4, "renvo_runtime_CCallerFrameAddress"
		} else if textEquals(name, "__builtin_return_address") {
			kind, helper = 2, "renvo_runtime_CReturnAddress"
		}
		if t.builtinAddressHelpers&kind == 0 {
			t.builtinAddressHelpers |= kind
			t.staticOut = append(t.staticOut, ("func " + helper + "() uintptr{return 0}\n")...)
		}
		t.appendText(helper)
		t.appendText("()")
		return true
	}
	if (textEquals(name, "__builtin_extract_return_addr") || textEquals(name, "__builtin_frob_return_addr")) && len(args) == 1 {
		t.emitExpression(args[0])
		return true
	}
	if (textEquals(name, "__builtin_object_size") || textEquals(name, "__builtin_dynamic_object_size")) && len(args) == 2 {
		mode, ok := t.constantExpression(args[1])
		if !ok || mode < 0 || mode > 3 {
			t.fail(TranslateErrUnsupported)
			return true
		}
		if size, known := t.builtinObjectSize(args[0]); known {
			t.appendDecimal(size)
		} else if mode&2 != 0 {
			t.out = append(t.out, '0')
		} else if t.pointerSize == 8 {
			t.appendText("^uint64(0)")
		} else {
			t.appendText("^uint32(0)")
		}
		return true
	}
	if textEquals(name, "__builtin_choose_expr") && len(args) == 3 {
		value, ok := t.constantExpression(args[0])
		if !ok {
			t.fail(TranslateErrUnsupported)
			return true
		}
		if value != 0 {
			t.emitExpression(args[1])
		} else {
			t.emitExpression(args[2])
		}
		return true
	}
	if textEquals(name, "__builtin_types_compatible_p") && len(args) == 2 {
		left, leftOK := t.typeFromTokens(args[0])
		right, rightOK := t.typeFromTokens(args[1])
		if !leftOK || !rightOK {
			t.fail(TranslateErrUnsupported)
			return true
		}
		if t.compatibleParameterType(left, right) {
			t.out = append(t.out, '1')
		} else {
			t.out = append(t.out, '0')
		}
		return true
	}
	bitOperation, bitWidth := 0, 0
	for operation, prefix := range []string{"__builtin_clz", "__builtin_ctz", "__builtin_popcount", "__builtin_ffs"} {
		if textEquals(name, prefix) {
			bitOperation, bitWidth = operation+1, 4
		} else if textEquals(name, prefix+"l") {
			bitOperation, bitWidth = operation+1, t.typeSize(cTypeUintptrID)
		} else if textEquals(name, prefix+"ll") {
			bitOperation, bitWidth = operation+1, 8
		}
	}
	if bitOperation != 0 && len(args) == 1 {
		name := t.ensureBuiltinBitHelper(bitOperation, bitWidth)
		t.appendText(name)
		t.appendText("(")
		if bitWidth == 8 {
			t.appendText("uint64(")
		} else {
			t.appendText("uint32(")
		}
		t.emitExpression(args[0])
		t.appendText("))")
		return true
	}
	byteSwapWidth := 0
	if textEquals(name, "__builtin_bswap16") {
		byteSwapWidth = 2
	} else if textEquals(name, "__builtin_bswap32") {
		byteSwapWidth = 4
	} else if textEquals(name, "__builtin_bswap64") {
		byteSwapWidth = 8
	}
	if byteSwapWidth != 0 && len(args) == 1 {
		t.appendText(t.ensureByteSwapHelper(byteSwapWidth))
		t.appendText("(")
		t.appendText("uint")
		t.appendDecimal(byteSwapWidth * 8)
		t.appendText("(")
		t.emitExpression(args[0])
		t.appendText("))")
		return true
	}
	memoryFunction := ""
	if textEquals(name, "__builtin_memcpy") {
		memoryFunction = "memcpy"
	} else if textEquals(name, "__builtin_memmove") {
		memoryFunction = "memmove"
	} else if textEquals(name, "__builtin_memset") {
		memoryFunction = "memset"
	} else if textEquals(name, "__builtin_memcmp") {
		memoryFunction = "memcmp"
	}
	if memoryFunction != "" {
		function, ok := t.lookupFunction([]byte(memoryFunction))
		if !ok || !t.emitFixedCall(function, tokens[2:len(tokens)-1]) {
			t.fail(TranslateErrUnsupported)
		}
		return true
	}
	operation := ""
	if textEquals(name, "__builtin_add_overflow") {
		operation = "add"
	} else if textEquals(name, "__builtin_sub_overflow") {
		operation = "sub"
	} else if textEquals(name, "__builtin_mul_overflow") {
		operation = "mul"
	}
	if operation != "" && len(args) == 3 {
		pointerType := t.typeInfo(t.expressionType(args[2]))
		if pointerType.kind != cTypePointer {
			t.fail(TranslateErrUnsupported)
			return true
		}
		valueType := pointerType.base
		info := t.typeInfo(valueType)
		if (info.kind != cTypeInt && info.kind != cTypeUint) || info.size < 1 || info.size > 8 {
			t.fail(TranslateErrUnsupported)
			return true
		}
		helper := t.ensureOverflowHelper(operation, valueType)
		t.appendText(helper)
		t.appendText("(")
		t.convertedExpression(valueType, args[0])
		t.appendText(",")
		t.convertedExpression(valueType, args[1])
		t.appendText(",")
		t.emitExpression(args[2])
		t.appendText(")")
		return true
	}
	return false
}

func (t *translator) builtinObjectSize(tokens []token) (int, bool) {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if value, ok := t.cStringBytes(tokens); ok {
		return len(value), true
	}
	if len(tokens) == 1 && tokenKind(tokens[0]) == tokenIdent {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok {
			info := t.typeInfo(object.typeID)
			if info.kind == cTypeArray {
				return info.size, true
			}
		}
	}
	if len(tokens) == 2 && tokenIs(t.src, tokens[0], "&") && tokenKind(tokens[1]) == tokenIdent {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[1])); ok {
			return t.typeSize(object.typeID), true
		}
	}
	if len(tokens) >= 5 && tokenIs(t.src, tokens[0], "&") && tokenKind(tokens[1]) == tokenIdent &&
		tokenIs(t.src, tokens[2], "[") && tokenIs(t.src, tokens[len(tokens)-1], "]") {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[1])); ok {
			info := t.typeInfo(object.typeID)
			index, constant := t.constantExpression(tokens[3 : len(tokens)-1])
			if info.kind == cTypeArray && constant && index >= 0 && index <= info.count {
				return (info.count - index) * t.typeSize(info.base), true
			}
		}
	}
	return 0, false
}

func (t *translator) ensureBuiltinBitHelper(operation int, width int) string {
	key := operation
	if width == 8 {
		key += 4
	}
	names := []string{"", "leading", "trailing", "population", "first"}
	name := "__c_builtin_" + names[operation] + "_" + decimalString(width*8)
	if t.builtinBitHelpers&(1<<key) != 0 {
		return name
	}
	t.builtinBitHelpers |= 1 << key
	typeName := "uint32"
	if width == 8 {
		typeName = "uint64"
	}
	body := ""
	switch operation {
	case 1:
		body = "if value==0{return " + decimalString(width*8) + "};var count int32;mask:=" + typeName + "(1)<<" + decimalString(width*8-1) + ";for value&mask==0{count++;mask>>=1};return count"
	case 2:
		body = "if value==0{return " + decimalString(width*8) + "};var count int32;for value&1==0{count++;value>>=1};return count"
	case 3:
		body = "var count int32;for value!=0{count+=int32(value&1);value>>=1};return count"
	case 4:
		trailing := t.ensureBuiltinBitHelper(2, width)
		body = "if value==0{return 0};return " + trailing + "(value)+1"
	}
	t.staticOut = append(t.staticOut, ("func " + name + "(value " + typeName + ") int32{" + body + "}\n")...)
	return name
}

func (t *translator) emitBuiltinVAArg(state []token, typeTokens []token) {
	typeID, ok := t.typeFromTokens(typeTokens)
	info := t.typeInfo(typeID)
	if !ok || info.kind != cTypePointer && info.kind != cTypeInt && info.kind != cTypeUint && info.kind != cTypeBool && info.kind != cTypeFloat ||
		info.kind != cTypePointer && (info.size < 1 || info.size > 8) {
		t.fail(TranslateErrUnsupported)
		return
	}
	if !t.vaListHelper {
		t.vaListHelper = true
		t.usesUnsafe = true
		if t.pointerSize == 4 {
			t.staticOut = append(t.staticOut, "func renvo_runtime_CVAArg32(state *uintptr) uintptr{pointer:=*state;value:=*(*uintptr)(__c_unsafe.Pointer(pointer));*state=pointer+4;return value}\n"...)
			t.staticOut = append(t.staticOut, "func renvo_runtime_CVAArg64(state *uintptr) uint64{low:=uint64(renvo_runtime_CVAArg32(state));high:=uint64(renvo_runtime_CVAArg32(state));return low|high<<32}\n"...)
			t.staticOut = append(t.staticOut, "func __c_va_float32(state *uintptr) float32{bits:=uint32(renvo_runtime_CVAArg32(state));return *(*float32)(__c_unsafe.Pointer(&bits))}\nfunc __c_va_float64(state *uintptr) float64{bits:=renvo_runtime_CVAArg64(state);return *(*float64)(__c_unsafe.Pointer(&bits))}\n"...)
		} else {
			t.staticOut = append(t.staticOut, "func renvo_runtime_CVAArg(state *uintptr) uint64{next:=(*uintptr)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(state))+8));index:=*next;value:=*(*uint64)(__c_unsafe.Pointer(*state+index*8));*next=index+1;return value}\n"...)
			t.staticOut = append(t.staticOut, "func __c_va_float32(state *uintptr) float32{bits:=uint32(renvo_runtime_CVAArg(state));return *(*float32)(__c_unsafe.Pointer(&bits))}\nfunc __c_va_float64(state *uintptr) float64{bits:=renvo_runtime_CVAArg(state);return *(*float64)(__c_unsafe.Pointer(&bits))}\n"...)
		}
	}
	if info.kind == cTypeFloat {
		t.usesUnsafe = true
		if info.size == 4 {
			t.appendText("__c_va_float32(")
		} else {
			t.appendText("__c_va_float64(")
		}
		t.emitVAListPointer(state)
		t.appendText(")")
		return
	}
	if info.kind == cTypePointer {
		t.usesUnsafe = true
		t.out = append(t.out, '(')
		t.emitType(typeID)
		t.appendText(")(__c_unsafe.Pointer(uintptr(")
		if t.pointerSize == 4 {
			t.appendText("renvo_runtime_CVAArg32(")
		} else {
			t.appendText("renvo_runtime_CVAArg(")
		}
		t.emitVAListPointer(state)
		t.appendText("))))")
		return
	}
	t.emitType(typeID)
	if t.pointerSize == 4 && info.size > 4 {
		t.appendText("(renvo_runtime_CVAArg64(")
		t.emitVAListPointer(state)
		t.appendText("))")
	} else {
		t.appendText("(")
		if t.pointerSize == 4 {
			t.appendText("renvo_runtime_CVAArg32(")
		} else {
			t.appendText("renvo_runtime_CVAArg(")
		}
		t.emitVAListPointer(state)
		t.appendText("))")
	}
}

func (t *translator) emitBuiltinVACopy(destination []token, source []token) {
	if !t.vaCopyHelper {
		t.vaCopyHelper = true
		index := t.ensurePointerIndexHelper(t.pointerType(cTypeUintptrID))
		t.staticOut = append(t.staticOut, ("func __c_va_copy(destination *uintptr,source *uintptr){*destination=*source;*" + index + "(destination,1)=*" + index + "(source,1);*" + index + "(destination,2)=*" + index + "(source,2)}\n")...)
	}
	t.appendText("__c_va_copy(")
	t.emitVAListPointer(destination)
	t.appendText(",")
	t.emitVAListPointer(source)
	t.appendText(")")
}

func (t *translator) emitVAListPointer(tokens []token) {
	info := t.typeInfo(t.expressionType(tokens))
	if info.kind == cTypeArray && info.base == cTypeUintptrID {
		t.appendText("&(")
		t.emitExpression(tokens)
		t.appendText(")[0]")
		return
	}
	if info.kind == cTypePointer && info.base == cTypeUintptrID {
		t.emitExpression(tokens)
		return
	}
	t.fail(TranslateErrUnsupported)
}

func (t *translator) ensureOverflowHelper(operation string, typeID int) string {
	info := t.typeInfo(typeID)
	opIndex := 0
	if operation == "sub" {
		opIndex = 1
	} else if operation == "mul" {
		opIndex = 2
	}
	sizeIndex := 0
	if info.size == 2 {
		sizeIndex = 1
	} else if info.size == 4 {
		sizeIndex = 2
	} else if info.size == 8 {
		sizeIndex = 3
	}
	unsigned := info.kind == cTypeUint
	key := opIndex*8 + sizeIndex
	letter := "i"
	if unsigned {
		key += 4
		letter = "u"
	}
	name := "__c_" + operation + "_overflow_" + letter + decimalString(info.size*8)
	bit := 1 << key
	if t.overflowHelpers&bit != 0 {
		return name
	}
	t.overflowHelpers |= bit
	outer := t.out
	t.out = t.staticOut
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(a ")
	t.emitType(typeID)
	t.appendText(",b ")
	t.emitType(typeID)
	t.appendText(",out *")
	t.emitType(typeID)
	t.appendText(") uint8{")
	if operation == "add" {
		t.appendText("result:=a+b;*out=result;if ")
		if unsigned {
			t.appendText("result<a")
		} else {
			t.appendText("((a^result)&(b^result))<0")
		}
		t.appendText("{return 1};return 0}")
	} else if operation == "sub" {
		t.appendText("result:=a-b;*out=result;if ")
		if unsigned {
			t.appendText("a<b")
		} else {
			t.appendText("((a^b)&(a^result))<0")
		}
		t.appendText("{return 1};return 0}")
	} else {
		t.appendText("overflow:=false;")
		if unsigned {
			t.appendText("if b!=0{overflow=a>")
			t.appendText(cIntegerMaximum(info.size, true))
			t.appendText("/b};")
		} else {
			minimum, maximum := cIntegerMinimum(info.size), cIntegerMaximum(info.size, false)
			t.appendText("if a==-1{overflow=b==")
			t.appendText(minimum)
			t.appendText("}else if b==-1{overflow=a==")
			t.appendText(minimum)
			t.appendText("}else if a>0{if b>0{overflow=a>")
			t.appendText(maximum)
			t.appendText("/b}else if b<0{overflow=b < ")
			t.appendText(minimum)
			t.appendText("/a}}else if a<0{if b>0{overflow=a < ")
			t.appendText(minimum)
			t.appendText("/b}else if b<0{overflow=b < ")
			t.appendText(maximum)
			t.appendText("/a}};")
		}
		t.appendText("*out=a*b;if overflow{return 1};return 0}")
	}
	t.appendText("\n")
	t.staticOut = t.out
	t.out = outer
	return name
}

func cIntegerMinimum(size int) string {
	if size == 1 {
		return "-128"
	}
	if size == 2 {
		return "-32768"
	}
	if size == 4 {
		return "-2147483648"
	}
	return "-9223372036854775808"
}

func cIntegerMaximum(size int, unsigned bool) string {
	if unsigned {
		if size == 1 {
			return "255"
		}
		if size == 2 {
			return "65535"
		}
		if size == 4 {
			return "4294967295"
		}
		return "18446744073709551615"
	}
	if size == 1 {
		return "127"
	}
	if size == 2 {
		return "32767"
	}
	if size == 4 {
		return "2147483647"
	}
	return "9223372036854775807"
}

func (t *translator) emitSizeof(tokens []token) {
	operand := tokens[1:]
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
		t.appendDecimal(t.typeAlign(t.expressionType(operand)))
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
	if fn.attributes.diagnosticError {
		t.ensureUndefinedInstructionHelper()
		t.appendText("renvo_runtime_CUndefinedInstruction()")
		return true
	}
	if fn.constantReturnOK && t.constantCallArgumentsSideEffectFree(args) {
		t.emitType(fn.resultType)
		t.out = append(t.out, '(')
		t.appendDecimalSigned(fn.constantReturn)
		t.out = append(t.out, ')')
		return true
	}
	if t.object && fn.name == "strlen" && len(args) == 1 && fn.paramCount == 1 {
		param := t.typeInfo(t.functionParams[fn.paramStart])
		base := t.typeInfo(param.base)
		if param.kind == cTypePointer && base.kind == cTypeInt && base.size == 1 {
			t.emitStringLength(args[0])
			return true
		}
	}
	t.out = append(t.out, t.cFunctionGoName(fn.name)...)
	t.out = append(t.out, '(')
	for i := 0; i < len(args); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		paramType := t.functionParams[fn.paramStart+i]
		paramInfo := t.typeInfo(paramType)
		if t.object && !fn.defined && !t.translationUnitFunctionDefined(fn.name) && paramInfo.kind == cTypePointer {
			if _, ok := t.cStringBytes(args[i]); ok {
				t.emitCStringValues(args[i])
				continue
			}
		}
		t.convertedExpression(paramType, args[i])
	}
	t.out = append(t.out, ')')
	return true
}

func (t *translator) constantCallArgumentsSideEffectFree(args [][]token) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		for j := 0; j < len(arg); j++ {
			if tokenIs(t.src, arg[j], "++") || tokenIs(t.src, arg[j], "--") ||
				tokenIs(t.src, arg[j], ",") || isAssignmentToken(t.src, arg[j]) {
				return false
			}
			if j+1 < len(arg) && tokenKind(arg[j]) == tokenIdent && tokenIs(t.src, arg[j+1], "(") {
				return false
			}
		}
	}
	return true
}

func (t *translator) ensureUndefinedInstructionHelper() {
	if t.asmDirectOps&16 != 0 {
		return
	}
	t.asmDirectOps |= 16
	t.staticOut = append(t.staticOut, "func renvo_runtime_CUndefinedInstruction(){}\n"...)
}

func (t *translator) emitStringLength(tokens []token) {
	if !t.stringLengthHelper {
		t.stringLengthHelper = true
		t.staticOut = append(t.staticOut, "func renvo_runtime_CStringLength(value *int8) uintptr{count:=uintptr(0);for *(*int8)(__c_unsafe.Pointer(uintptr(__c_unsafe.Pointer(value))+count))!=0{count++};return count}\n"...)
	}
	t.appendText("renvo_runtime_CStringLength(")
	// strlen receives its operand after the ordinary array-to-pointer
	// conversion.  In particular, a struct's inline char array must decay to
	// its first element instead of being cast as though its bytes were an
	// already-loaded pointer value.
	t.convertedExpression(t.pointerType(cTypeInt8ID), tokens)
	t.appendText(")")
	t.usesUnsafe = true
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
		t.emitOmittedConditionalExpression(tokens, question, colon)
		return
	}
	left, right := t.decayedExpressionType(tokens[question+1:colon]), t.decayedExpressionType(tokens[colon+1:])
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
	outer, names, captureIndices := t.beginHelper()
	t.emitHelperObjectMetadata(name, typeID)
	t.appendText("func ")
	t.out = append(t.out, name...)
	t.appendText("(c bool")
	t.emitCaptureParameters(true)
	t.out = append(t.out, ')')
	if typeID == cTypeVoidID {
		t.appendText("{if c{")
		t.expression(tokens[question+1 : colon])
		t.appendText(";return};")
		t.expression(tokens[colon+1:])
		t.appendText("}\n")
		t.endHelper(outer, names, captureIndices)
		return
	}
	t.out = append(t.out, ' ')
	t.emitType(typeID)
	t.appendText("{if c{return ")
	t.convertedExpression(typeID, tokens[question+1:colon])
	t.appendText("};return ")
	t.convertedExpression(typeID, tokens[colon+1:])
	t.appendText("}\n")
	t.endHelper(outer, names, captureIndices)
}

func (t *translator) emitOmittedConditionalExpression(tokens []token, question int, colon int) {
	valueTokens := tokens[:question]
	valueType := t.decayedExpressionType(valueTokens)
	rightType := t.decayedExpressionType(tokens[colon+1:])
	typeID := t.usualArithmeticType(valueType, rightType)
	if t.typeInfo(valueType).kind == cTypePointer {
		typeID = valueType
	} else if t.typeInfo(rightType).kind == cTypePointer {
		typeID = rightType
	}
	valueKind := t.typeInfo(valueType).kind
	if valueKind != cTypePointer && valueKind != cTypeInt && valueKind != cTypeUint && valueKind != cTypeBool {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.typeSerial++
	name := "__c_conditional_value_" + decimalString(t.typeSerial)
	t.appendText(name)
	t.out = append(t.out, '(')
	t.convertedExpression(valueType, valueTokens)
	t.emitCaptureArguments(true)
	t.out = append(t.out, ')')
	outer, names, captureIndices := t.beginHelper()
	t.emitHelperObjectMetadata(name, typeID)
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(value ")
	t.emitType(valueType)
	t.emitCaptureParameters(true)
	t.appendText(") ")
	t.emitType(typeID)
	t.appendText("{if value!=")
	if valueKind == cTypePointer {
		t.appendText("nil")
	} else {
		t.appendText("0")
	}
	t.appendText("{return ")
	if typeID == valueType {
		t.appendText("value")
	} else {
		t.emitType(typeID)
		t.appendText("(value)")
	}
	t.appendText("};return ")
	t.convertedExpression(typeID, tokens[colon+1:])
	t.appendText("}\n")
	t.endHelper(outer, names, captureIndices)
}

func (t *translator) commaOperator(tokens []token) int {
	paren, bracket, brace, conditional := 0, 0, 0, 0
	for i := len(tokens) - 1; i >= 0; i-- {
		switch {
		case tokenIs(t.src, tokens[i], ")"):
			if match := i + tokens[i].match; tokens[i].match < 0 && match >= 0 && tokenIs(t.src, tokens[match], "(") {
				i = match
				continue
			}
			paren++
		case tokenIs(t.src, tokens[i], "("):
			paren--
		case tokenIs(t.src, tokens[i], "]"):
			if match := i + tokens[i].match; tokens[i].match < 0 && match >= 0 && tokenIs(t.src, tokens[match], "[") {
				i = match
				continue
			}
			bracket++
		case tokenIs(t.src, tokens[i], "["):
			bracket--
		case tokenIs(t.src, tokens[i], "}"):
			if match := i + tokens[i].match; tokens[i].match < 0 && match >= 0 && tokenIs(t.src, tokens[match], "{") {
				i = match
				continue
			}
			brace++
		case tokenIs(t.src, tokens[i], "{"):
			brace--
		case paren == 0 && bracket == 0 && brace == 0 && tokenIs(t.src, tokens[i], ":"):
			conditional++
		case paren == 0 && bracket == 0 && brace == 0 && conditional > 0 && tokenIs(t.src, tokens[i], "?"):
			conditional--
		case paren == 0 && bracket == 0 && brace == 0 && conditional == 0 && tokenIs(t.src, tokens[i], ","):
			return i
		}
	}
	return -1
}

func (t *translator) emitCommaExpression(left []token, right []token) {
	typeID := t.expressionType(right)
	t.typeSerial++
	name := "__c_comma_" + decimalString(t.typeSerial)
	captureMark := t.limitCapturesToTokenPair(left, right)
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	t.emitCaptureArguments(false)
	t.out = append(t.out, ')')
	outer, names, captureIndices := t.beginHelper()
	t.emitHelperObjectMetadata(name, typeID)
	t.appendText("func ")
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	t.emitCaptureParameters(false)
	t.appendText(") ")
	t.emitType(typeID)
	t.out = append(t.out, '{')
	if t.emptyStatementExpression(left) {
		t.appendText("_=0")
	} else {
		t.expression(left)
	}
	t.appendText(";return ")
	t.convertedExpression(typeID, right)
	t.appendText("}\n")
	t.endHelper(outer, names, captureIndices)
	t.captureExclusions = t.captureExclusions[:captureMark]
}

func (t *translator) emptyStatementExpression(tokens []token) bool {
	for len(tokens) > 2 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) < 2 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		return false
	}
	for i := 1; i < len(tokens)-1; i++ {
		if !tokenIs(t.src, tokens[i], ";") {
			return false
		}
	}
	return true
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

func (t *translator) beginHelper() ([]byte, []string, []int) {
	outer := t.out
	t.out = nil
	t.captureExclusions = append(t.captureExclusions, t.initializing)
	var names []string
	var indices []int
	parameter := 0
	for i := t.localStart; i >= 0 && i < len(t.objects); i++ {
		if !t.captureObject(i) {
			continue
		}
		names = append(names, t.objects[i].goName)
		indices = append(indices, i)
		t.objects[i].goName = "(*p" + decimalString(parameter) + ")"
		parameter++
	}
	return outer, names, indices
}

func (t *translator) emitHelperObjectMetadata(name string, resultType int) {
	if !t.object || t.helperSection == "" {
		return
	}
	t.emitObjectMetadata("function", name, cAttributes{section: t.helperSection}, storageStatic, nil, resultType, false)
}

func (t *translator) endHelper(outer []byte, names []string, indices []int) {
	body := t.out
	if len(body) > 0 && body[len(body)-1] != '\n' {
		body = append(body, '\n')
	}
	for i := 0; i < len(names) && i < len(indices); i++ {
		index := indices[i]
		if index >= 0 && index < len(t.objects) {
			t.objects[index].goName = names[i]
		}
	}
	t.staticOut = append(t.staticOut, body...)
	t.out = outer
	t.captureExclusions = t.captureExclusions[:len(t.captureExclusions)-1]
}

func (t *translator) captureObject(index int) bool {
	if index < 0 || index >= len(t.objects) || index == t.initializing || !t.objects[index].auto {
		return false
	}
	for i := len(t.captureExclusions) - 1; i >= 0; i-- {
		if t.captureExclusions[i] == index {
			return false
		}
	}
	return true
}

func (t *translator) limitCapturesToTokens(tokens []token) int {
	mark := len(t.captureExclusions)
	for index := t.localStart; index >= 0 && index < len(t.objects); index++ {
		if !t.captureObject(index) || t.tokensReferenceObject(tokens, t.objects[index].name) {
			continue
		}
		t.captureExclusions = append(t.captureExclusions, index)
	}
	return mark
}

func (t *translator) limitCapturesToTokenPair(left []token, right []token) int {
	mark := len(t.captureExclusions)
	for index := t.localStart; index >= 0 && index < len(t.objects); index++ {
		if !t.captureObject(index) || t.tokensReferenceObject(left, t.objects[index].name) ||
			t.tokensReferenceObject(right, t.objects[index].name) {
			continue
		}
		t.captureExclusions = append(t.captureExclusions, index)
	}
	return mark
}

func (t *translator) tokensReferenceObject(tokens []token, name string) bool {
	for i := 0; i < len(tokens); i++ {
		if tokenKind(tokens[i]) == tokenIdent && string(tokenText(t.src, tokens[i])) == name {
			return true
		}
	}
	return false
}

func (t *translator) binaryOperator(tokens []token) int {
	best, precedence := -1, 100
	paren, bracket, brace := 0, 0, 0
	topOpen := -1
	operand := true
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			if match := i + tokens[i].match; tokens[i].match > 0 && match < len(tokens) && tokenIs(t.src, tokens[match], ")") {
				operand = (i == 0 || tokenKind(tokens[i-1]) != tokenIdent) && t.parenthesizedTypeStart(tokens[i+1:match])
				i = match
				continue
			}
			if paren == 0 {
				topOpen = i
			}
			paren++
			operand = true
			continue
		case tokenIs(t.src, tokens[i], ")"):
			paren--
			operand = false
			if paren == 0 && topOpen >= 0 &&
				(topOpen == 0 || tokenKind(tokens[topOpen-1]) != tokenIdent) &&
				t.parenthesizedTypeStart(tokens[topOpen+1:i]) {
				// A closing cast parenthesis still expects its unary operand.
				// This distinguishes (T)*p from the binary multiplication (x)*p.
				operand = true
			}
			continue
		case tokenIs(t.src, tokens[i], "["):
			if match := i + tokens[i].match; tokens[i].match > 0 && match < len(tokens) && tokenIs(t.src, tokens[match], "]") {
				i = match
				continue
			}
			bracket++
			continue
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
			continue
		case tokenIs(t.src, tokens[i], "{"):
			if match := i + tokens[i].match; tokens[i].match > 0 && match < len(tokens) && tokenIs(t.src, tokens[match], "}") {
				operand = false
				i = match
				continue
			}
			brace++
			operand = true
			continue
		case tokenIs(t.src, tokens[i], "}"):
			brace--
			operand = false
			continue
		}
		if paren != 0 || bracket != 0 || brace != 0 {
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

func (t *translator) parenthesizedTypeStart(tokens []token) bool {
	if len(tokens) == 0 || tokenKind(tokens[0]) != tokenIdent {
		return false
	}
	text := tokenText(t.src, tokens[0])
	if _, ok := t.lookupTypedef(text); ok {
		return true
	}
	if _, ok := t.builtinCType(text); ok {
		return true
	}
	return textEquals(text, "void") || textEquals(text, "_Bool") || textEquals(text, "bool") ||
		textEquals(text, "char") || textEquals(text, "short") || textEquals(text, "int") ||
		textEquals(text, "long") || textEquals(text, "float") || textEquals(text, "double") ||
		textEquals(text, "signed") || textEquals(text, "unsigned") || textEquals(text, "struct") ||
		textEquals(text, "union") || textEquals(text, "enum") || textEquals(text, "const") ||
		textEquals(text, "volatile") || textEquals(text, "restrict") || textEquals(text, "_Atomic") ||
		textEquals(text, "typeof") || textEquals(text, "__typeof") || textEquals(text, "__typeof__") ||
		textEquals(text, "__attribute") || textEquals(text, "__attribute__")
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
	leftType, rightType := t.decayedExpressionType(left), t.decayedExpressionType(right)
	leftInfo, rightInfo := t.typeInfo(leftType), t.typeInfo(rightType)
	if (tokenIs(t.src, op, "+") || tokenIs(t.src, op, "-")) && leftInfo.kind == cTypePointer && rightInfo.kind != cTypePointer {
		helper := t.ensurePointerStepHelper(leftType, tokenIs(t.src, op, "-"))
		t.appendText(helper)
		t.appendText("(")
		t.convertedExpression(leftType, left)
		t.appendText(",uintptr(")
		t.convertedExpression(rightType, right)
		t.appendText("))")
		return
	}
	if tokenIs(t.src, op, "+") && rightInfo.kind == cTypePointer && leftInfo.kind != cTypePointer {
		helper := t.ensurePointerStepHelper(rightType, false)
		t.appendText(helper)
		t.appendText("(")
		t.emitExpression(right)
		t.appendText(",uintptr(")
		t.convertedExpression(leftType, left)
		t.appendText("))")
		return
	}
	if tokenIs(t.src, op, "-") && leftInfo.kind == cTypePointer && rightInfo.kind == cTypePointer {
		t.appendText(t.ensurePointerDifferenceHelper(leftType))
		t.appendText("(")
		t.convertedExpression(leftType, left)
		t.appendText(",")
		t.convertedExpression(leftType, right)
		t.appendText(")")
		return
	}
	shift := tokenIs(t.src, op, "<<") || tokenIs(t.src, op, ">>")
	typeID := t.usualArithmeticType(leftType, rightType)
	if shift {
		typeID = t.integerPromotion(leftType)
	}
	t.convertedExpression(typeID, left)
	t.out = append(t.out, tokenText(t.src, op)...)
	if shift {
		// C gives multiplication higher precedence than shifts, while Go puts
		// them in the same left-associative group. Preserve the C tree for
		// expressions such as 1UL << 8 * align.
		nested := t.binaryOperator(right) >= 0
		if nested {
			t.out = append(t.out, '(')
		}
		t.convertedExpression(t.integerPromotion(rightType), right)
		if nested {
			t.out = append(t.out, ')')
		}
	} else {
		outer := t.out
		t.out = nil
		t.convertedExpression(typeID, right)
		emitted := t.out
		t.out = outer
		if len(emitted) > 0 && emitted[0] == '-' {
			// Keep a folded negative right operand from merging with the binary
			// operator into a different Go token such as <- or --.
			t.out = append(t.out, '(')
			t.out = append(t.out, emitted...)
			t.out = append(t.out, ')')
		} else {
			t.out = append(t.out, emitted...)
		}
	}
}

func (t *translator) decayedExpressionType(tokens []token) int {
	typeID := t.expressionType(tokens)
	info := t.typeInfo(typeID)
	if info.kind == cTypeArray && t.binaryOperator(tokens) < 0 {
		return t.pointerType(info.base)
	}
	if info.kind == cTypeFunction && t.binaryOperator(tokens) < 0 {
		return t.pointerType(typeID)
	}
	return typeID
}

func (t *translator) ensurePointerDifferenceHelper(pointerType int) string {
	name := "__c_pointer_diff_" + decimalString(pointerType)
	for i := 0; i < len(t.pointerDiffTypes); i++ {
		if t.pointerDiffTypes[i] == pointerType {
			return name
		}
	}
	t.pointerDiffTypes = append(t.pointerDiffTypes, pointerType)
	outer := t.out
	t.out = nil
	t.appendText("func ")
	t.appendText(name)
	t.appendText("(left ")
	t.emitType(pointerType)
	t.appendText(",right ")
	t.emitType(pointerType)
	t.appendText(") ")
	resultType := cTypeInt64ID
	if t.pointerSize == 4 {
		resultType = cTypeInt32ID
	}
	t.emitType(resultType)
	t.appendText("{return 0}\n")
	t.staticOut = append(t.staticOut, t.out...)
	t.out = outer
	return name
}

func (t *translator) emitConditionExpression(tokens []token) {
	if t.booleanExpression(tokens) {
		t.emitExpression(tokens)
	} else {
		t.out = append(t.out, '(')
		t.emitExpression(tokens)
		if t.typeInfo(t.expressionType(tokens)).kind == cTypePointer {
			t.appendText(")!=nil")
		} else {
			t.appendText(")!=0")
		}
	}
}

func (t *translator) booleanExpression(tokens []token) bool {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if len(tokens) >= 4 && tokenKind(tokens[0]) == tokenIdent && tokenIs(t.src, tokens[1], "(") &&
		matchingToken(t.src, tokens, 1, "(", ")") == len(tokens)-1 &&
		(tokenIs(t.src, tokens[0], "__builtin_expect") || tokenIs(t.src, tokens[0], "__builtin_expect_with_probability")) {
		args := splitTopLevel(t.src, tokens[2:len(tokens)-1], ",")
		return len(args) >= 1 && t.booleanExpression(args[0])
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
		if i < fn.paramCount {
			info := t.typeInfo(typeID)
			if t.object && !fn.defined && !t.translationUnitFunctionDefined(fn.name) && info.kind == cTypePointer {
				if _, ok := t.cStringBytes(args[i]); ok {
					t.emitCStringValues(args[i])
					continue
				}
			}
			t.convertedExpression(typeID, args[i])
		} else if t.typeInfo(typeID).kind == cTypePointer && t.typeInfo(original).kind == cTypeArray {
			t.convertedExpression(typeID, args[i])
		} else if typeID != original && t.typeInfo(typeID).kind != cTypePointer {
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
	if at := t.commaOperator(tokens); at >= 0 {
		return t.expressionType(tokens[at+1:])
	}
	if assignment := t.simpleAssignment(tokens); assignment > 0 {
		if typeID := t.lvalueType(tokens[:assignment]); typeID != cTypeVoidID {
			return typeID
		}
	}
	if len(tokens) >= 6 && tokenIs(t.src, tokens[0], "__builtin_choose_expr") && tokenIs(t.src, tokens[1], "(") &&
		matchingToken(t.src, tokens, 1, "(", ")") == len(tokens)-1 {
		args := splitTopLevel(t.src, tokens[2:len(tokens)-1], ",")
		if len(args) == 3 {
			if value, ok := t.constantExpression(args[0]); ok {
				if value != 0 {
					return t.expressionType(args[1])
				}
				return t.expressionType(args[2])
			}
		}
	}
	if open := t.trailingCallOpen(tokens); open > 0 {
		calleeType := t.expressionType(tokens[:open])
		callee := t.typeInfo(calleeType)
		if callee.kind == cTypePointer {
			callee = t.typeInfo(callee.base)
		}
		if callee.kind == cTypeFunction {
			return callee.base
		}
	}
	if len(tokens) >= 6 && tokenIs(t.src, tokens[0], "__builtin_va_arg") && tokenIs(t.src, tokens[1], "(") &&
		matchingToken(t.src, tokens, 1, "(", ")") == len(tokens)-1 {
		args := splitTopLevel(t.src, tokens[2:len(tokens)-1], ",")
		if len(args) == 2 {
			if typeID, ok := t.typeFromTokens(args[1]); ok {
				return typeID
			}
		}
	}
	if len(tokens) >= 5 && tokenIs(t.src, tokens[0], "__builtin_offsetof") && tokenIs(t.src, tokens[1], "(") &&
		matchingToken(t.src, tokens, 1, "(", ")") == len(tokens)-1 {
		if t.pointerSize == 4 {
			return cTypeUint32ID
		}
		return cTypeUint64ID
	}
	if len(tokens) >= 3 && tokenKind(tokens[0]) == tokenIdent && tokenIs(t.src, tokens[1], "(") &&
		matchingToken(t.src, tokens, 1, "(", ")") == len(tokens)-1 {
		name := tokenText(t.src, tokens[0])
		if textEquals(name, "__sync_synchronize") && len(tokens) == 3 {
			return cTypeVoidID
		}
		if textEquals(name, "__builtin_frame_address") || textEquals(name, "__builtin_return_address") ||
			textEquals(name, "__builtin_extract_return_addr") || textEquals(name, "__builtin_frob_return_addr") {
			return t.pointerType(cTypeVoidID)
		}
	}
	if typeID, ok := t.statementExpressionType(tokens); ok {
		return typeID
	}
	if typeID, ok := t.genericExpressionType(tokens); ok {
		return typeID
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "!") {
		return cTypeInt32ID
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[0], "++") || tokenIs(t.src, tokens[0], "--")) {
		return t.lvalueType(tokens[1:])
	}
	if len(tokens) > 1 && t.unaryArithmetic(tokens[0]) {
		return t.integerPromotion(t.expressionType(tokens[1:]))
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "*") {
		info := t.typeInfo(t.decayedExpressionType(tokens[1:]))
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
		leftType, rightType := t.decayedExpressionType(tokens[:at]), t.decayedExpressionType(tokens[at+1:])
		leftInfo, rightInfo := t.typeInfo(leftType), t.typeInfo(rightType)
		if (tokenIs(t.src, tokens[at], "+") || tokenIs(t.src, tokens[at], "-")) && leftInfo.kind == cTypePointer {
			if tokenIs(t.src, tokens[at], "-") && rightInfo.kind == cTypePointer {
				if t.pointerSize == 4 {
					return cTypeInt32ID
				}
				return cTypeInt64ID
			}
			return leftType
		}
		if tokenIs(t.src, tokens[at], "+") && rightInfo.kind == cTypePointer {
			return rightType
		}
		if tokenIs(t.src, tokens[at], "<<") || tokenIs(t.src, tokens[at], ">>") {
			return t.integerPromotion(leftType)
		}
		return t.usualArithmeticType(leftType, rightType)
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[len(tokens)-1], "++") || tokenIs(t.src, tokens[len(tokens)-1], "--")) {
		if typeID := t.postfixMutationType(tokens[:len(tokens)-1]); typeID != cTypeVoidID {
			return typeID
		}
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
		text := tokenText(t.src, tokens[0])
		if len(text) > 1 && text[0] == 'L' {
			if t.wcharSize == 2 {
				return t.pointerType(cTypeUint16ID)
			}
			return t.pointerType(cTypeUint32ID)
		}
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

// Unlike most expression contexts, typeof does not apply array-to-pointer
// conversion to a string literal.  Preserve its complete array type so kernel
// note payloads such as typeof("") occupy one byte rather than one pointer.
func (t *translator) typeofExpressionType(tokens []token) int {
	for len(tokens) > 1 && tokenIs(t.src, tokens[0], "(") && matchingToken(t.src, tokens, 0, "(", ")") == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}
	if value, ok := t.cStringBytes(tokens); ok {
		return t.arrayType(cTypeInt8ID, len(value))
	}
	if value, ok := t.cWideStringBytes(tokens); ok {
		base := cTypeUint32ID
		if t.wcharSize == 2 {
			base = cTypeUint16ID
		}
		return t.arrayType(base, len(value)/t.wcharSize)
	}
	if len(tokens) > 1 && tokenIs(t.src, tokens[0], "&") {
		// The address-of operator changes the type; it is not itself an
		// lvalue. Asking lvalueType first made typeof(&array[0]) inherit the
		// array type, defeating Linux's compile-time ARRAY_SIZE check.
		return t.expressionType(tokens)
	}
	if typeID := t.lvalueType(tokens); typeID != cTypeVoidID {
		return typeID
	}
	return t.expressionType(tokens)
}

func (t *translator) tokensContainSubscript(tokens []token) bool {
	for i := 0; i < len(tokens); i++ {
		if tokenIs(t.src, tokens[i], "[") {
			return true
		}
	}
	return false
}

func (t *translator) trailingCallOpen(tokens []token) int {
	if len(tokens) > 1 && tokenIs(t.src, tokens[len(tokens)-1], ")") {
		open := len(tokens) - 1 + tokens[len(tokens)-1].match
		if tokens[len(tokens)-1].match < 0 && open > 0 && tokenIs(t.src, tokens[open], "(") && t.postfixCallCalleeEnd(tokens[open-1]) {
			return open
		}
	}
	paren, bracket := 0, 0
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			if paren == 0 && bracket == 0 && i > 0 && t.postfixCallCalleeEnd(tokens[i-1]) &&
				matchingToken(t.src, tokens, i, "(", ")") == len(tokens)-1 {
				return i
			}
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		}
	}
	return -1
}

func (t *translator) postfixCallCalleeEnd(tok token) bool {
	return tokenKind(tok) == tokenIdent || tokenIs(t.src, tok, ")") || tokenIs(t.src, tok, "]")
}

func (t *translator) statementExpressionValue(tokens []token) (int, int, bool) {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		return -1, -1, false
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
			if paren == 0 && bracket == 0 && brace == 0 {
				start = i + 1
			}
		case paren == 0 && bracket == 0 && brace == 0 && tokenIs(t.src, tokens[i], ";"):
			if i > start {
				lastStart, lastEnd = start, i
			}
			start = i + 1
		}
	}
	if lastStart < 0 {
		return -1, -1, false
	}
	return lastStart, lastEnd, true
}

func (t *translator) statementExpressionWithoutValue(tokens []token) bool {
	if len(tokens) < 3 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		return false
	}
	first := tokens[1]
	return tokenIs(t.src, first, "{") || tokenIs(t.src, first, "if") || tokenIs(t.src, first, "switch") ||
		tokenIs(t.src, first, "while") || tokenIs(t.src, first, "do") || tokenIs(t.src, first, "for") ||
		tokenIs(t.src, first, "goto") || tokenIs(t.src, first, "continue") || tokenIs(t.src, first, "break") ||
		tokenIs(t.src, first, "return") || tokenIs(t.src, first, "asm") || tokenIs(t.src, first, "__asm") ||
		tokenIs(t.src, first, "__asm__")
}

func (t *translator) statementExpressionType(tokens []token) (int, bool) {
	lastStart, lastEnd, ok := t.statementExpressionValue(tokens)
	if !ok && t.statementExpressionWithoutValue(tokens) {
		lastStart, lastEnd, ok = len(tokens)-1, len(tokens)-1, true
	}
	if !ok {
		return cTypeVoidID, false
	}
	value := tokens[lastStart:lastEnd]
	for len(value) >= 2 && tokenKind(value[0]) == tokenIdent && tokenIs(t.src, value[1], ":") {
		value = value[2:]
	}
	voidValue := len(value) == 0 || tokenIs(t.src, value[0], "{") || tokenIs(t.src, value[0], "if") ||
		tokenIs(t.src, value[0], "switch") || tokenIs(t.src, value[0], "while") || tokenIs(t.src, value[0], "do") ||
		tokenIs(t.src, value[0], "for") || tokenIs(t.src, value[0], "goto") || tokenIs(t.src, value[0], "continue") ||
		tokenIs(t.src, value[0], "break") || tokenIs(t.src, value[0], "return") || tokenIs(t.src, value[0], "asm") ||
		tokenIs(t.src, value[0], "__asm") || tokenIs(t.src, value[0], "__asm__")
	outerTokens, outerPos := t.tokens, t.pos
	outerOut, outerStaticOut := t.out, t.staticOut
	outerCheckOnly := t.checkOnly
	objectMark := len(t.objects)
	labelMark := len(t.localLabels)
	scope := t.beginScope()
	t.tokens, t.pos, t.out, t.checkOnly = tokens, 1, nil, true
	for t.ok && t.pos < lastStart {
		t.statement()
	}
	typeID := cTypeVoidID
	if t.ok && t.pos == lastStart {
		if !voidValue {
			typeID = t.expressionType(value)
		}
	}
	t.tokens, t.pos, t.out, t.staticOut, t.checkOnly = outerTokens, outerPos, outerOut, outerStaticOut, outerCheckOnly
	t.objects = t.objects[:objectMark]
	t.localLabels = t.localLabels[:labelMark]
	t.endScope(scope)
	return typeID, t.ok
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
	if member == 1 && tokenKind(tokens[0]) == tokenIdent && tokenIs(t.src, tokens[member], ".") {
		if object, ok := t.lookupObject(tokenText(t.src, tokens[0])); ok && object.transparentTypeID > 0 {
			if field, ok := t.lookupField(object.transparentTypeID, tokenText(t.src, tokens[member+1])); ok {
				return field.typeID, true
			}
		}
	}
	base := t.expressionType(tokens[:member])
	if tokenIs(t.src, tokens[member], "->") {
		pointer := t.typeInfo(base)
		if pointer.kind == cTypeArray {
			// An array operand decays to a pointer to its first element before
			// -> selects the member. This is common for one-element carrier
			// typedefs such as the kernel's cpumask_var_t.
			base = pointer.base
		} else if pointer.kind == cTypePointer {
			base = pointer.base
		} else {
			return cTypeVoidID, false
		}
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
			for fieldIndex < info.fieldCount && t.skipPositionalInitializerField(t.fields[info.fieldStart+fieldIndex]) {
				fieldIndex++
			}
			if fieldIndex >= info.fieldCount {
				t.fail(TranslateErrUnsupported)
				return
			}
			field := t.fields[info.fieldStart+fieldIndex]
			valueType = field.typeID
			fieldIndex++
			t.out = append(t.out, field.goName...)
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
			if t.unionInitializerNeedsMutation(value) && t.emitMutationInitializer(typeID, value) {
				return
			}
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
	if info.kind != cTypeStruct && info.kind != cTypeArray && info.kind != cTypeUnion {
		value = t.scalarInitializerTokens(value)
	}
	t.convertedExpression(typeID, value)
}

func (t *translator) skipPositionalInitializerField(field cField) bool {
	if field.promoted {
		return true
	}
	if !field.synthetic {
		return false
	}
	kind := t.typeInfo(field.typeID).kind
	return kind != cTypeStruct && kind != cTypeUnion
}

func (t *translator) unionInitializerNeedsMutation(tokens []token) bool {
	if len(tokens) < 2 || !tokenIs(t.src, tokens[0], "{") || !tokenIs(t.src, tokens[len(tokens)-1], "}") {
		return false
	}
	items := splitTopLevel(t.src, tokens[1:len(tokens)-1], ",")
	count := 0
	for i := 0; i < len(items); i++ {
		if len(items[i]) != 0 {
			count++
		}
	}
	return count > 1
}

func (t *translator) scalarInitializerTokens(value []token) []token {
	for len(value) >= 2 && tokenIs(t.src, value[0], "{") && tokenIs(t.src, value[len(value)-1], "}") {
		items := splitTopLevel(t.src, value[1:len(value)-1], ",")
		var selected []token
		count := 0
		for _, item := range items {
			if len(item) == 0 {
				continue
			}
			selected = item
			count++
		}
		if count != 1 || tokenIs(t.src, selected[0], ".") || tokenIs(t.src, selected[0], "[") {
			break
		}
		value = selected
	}
	return value
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
		// GNU C's obsolete designated-initializer spelling permits `[index]
		// value` without the equals token. xv6 intentionally uses that dialect.
		if at < len(item) && tokenIs(t.src, item[at], "=") {
			at++
		}
	} else if info.kind == cTypeArray {
		if info.count > 0 && next >= info.count {
			t.fail(TranslateErrDeclaration)
			return next
		}
		targetType = info.base
		next++
	} else {
		for next < info.fieldCount && t.skipPositionalInitializerField(t.fields[info.fieldStart+next]) {
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
	// Indirect structs use byte carriers rather than Go fields.  Even a purely
	// positional initializer must therefore be expressed through offset-aware
	// assignments (or packed directly when every value is constant).
	mutation := info.kind == cTypeStruct && info.indirect
	for i := 0; i < len(items); i++ {
		item := items[i]
		if len(item) == 0 {
			continue
		}
		path := initializerPath{start: len(steps), typeID: typeID}
		at := 0
		rangeStep, rangeFirst, rangeLast := -1, 0, 0
		if tokenIs(t.src, item[0], ".") || tokenIs(t.src, item[0], "[") {
			mutation = true
			firstDesignator := true
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
					if firstDesignator && info.kind == cTypeStruct {
						next = fieldIndex - info.fieldStart + 1
					}
					at += 2
				} else {
					close := matchingToken(t.src, item, at, "[", "]")
					if close <= at+1 || current.kind != cTypeArray {
						t.fail(TranslateErrUnsupported)
						return true
					}
					designator := item[at+1 : close]
					rangeAt := topLevelToken(t.src, designator, "...")
					firstTokens := designator
					if rangeAt >= 0 {
						firstTokens = designator[:rangeAt]
					}
					index, ok := t.constantExpression(firstTokens)
					last := index
					if rangeAt >= 0 {
						last, ok = t.constantExpression(designator[rangeAt+1:])
					}
					if !ok || index < 0 || last < index || last >= current.count || rangeAt >= 0 && rangeStep >= 0 {
						t.fail(TranslateErrUnsupported)
						return true
					}
					if rangeAt >= 0 {
						rangeStep, rangeFirst, rangeLast = len(steps), index, last
					}
					steps = append(steps, initializerStep{field: -1, index: index})
					path.typeID = current.base
					if firstDesignator && info.kind == cTypeArray {
						next = last + 1
					}
					at = close + 1
				}
				firstDesignator = false
			}
			if at < len(item) && tokenIs(t.src, item[at], "=") {
				at++
			}
		} else if info.kind == cTypeArray {
			if next >= info.count {
				t.fail(TranslateErrUnsupported)
				return true
			}
			steps = append(steps, initializerStep{field: -1, index: next})
			path.typeID = info.base
			next++
		} else {
			for next < info.fieldCount && t.skipPositionalInitializerField(t.fields[info.fieldStart+next]) {
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
		if rangeStep >= 0 {
			rangeOffset := rangeStep - path.start
			for index := rangeFirst + 1; index <= rangeLast; index++ {
				start := len(steps)
				steps = append(steps, steps[path.start:path.start+path.count]...)
				steps[start+rangeOffset].index = index
				paths = append(paths, initializerPath{start: start, count: path.count, typeID: path.typeID})
				values = append(values, item[at:])
			}
		}
	}
	if !mutation {
		return false
	}
	if info.kind == cTypeArray && t.emitIndexedArrayInitializer(typeID, paths, steps, values) {
		return true
	}
	if info.kind == cTypeStruct && t.emitObjectNestedBitfieldInitializer(typeID, paths, steps, values) {
		return true
	}
	if info.kind == cTypeStruct && t.emitObjectIndirectConstantInitializer(typeID, paths, steps, values) {
		return true
	}
	if info.kind == cTypeStruct && t.emitObjectKeyedBitfieldInitializer(typeID, paths, steps, values) {
		return true
	}
	if info.kind == cTypeStruct && !info.indirect && t.emitKeyedStructInitializer(typeID, paths, steps, values) {
		return true
	}
	t.typeSerial++
	name := "__c_aggregate_init_" + decimalString(t.typeSerial)
	if t.object {
		encoded := "__c_object_aggregate_"
		encodable := true
		for i := 0; i < len(paths); i++ {
			offset := 0
			currentType := typeID
			path := paths[i]
			for j := 0; j < path.count; j++ {
				step := steps[path.start+j]
				if step.field >= 0 {
					field := t.fields[step.field]
					if field.bitWidth > 0 {
						encodable = false
						break
					}
					offset += field.offset
					currentType = field.typeID
				} else {
					current := t.typeInfo(currentType)
					if current.kind != cTypeArray || step.index < 0 || step.index >= current.count {
						encodable = false
						break
					}
					offset += step.index * t.typeSize(current.base)
					currentType = current.base
				}
			}
			if !encodable {
				break
			}
			encoded += decimalString(offset) + "_"
		}
		if encodable {
			name = encoded + "x" + decimalString(t.typeSerial)
		}
	}
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
	// Aggregate initializers can themselves be emitted while another static
	// declaration is being built. Construct the helper in an independent
	// buffer so appending it cannot overwrite the enclosing static output.
	t.out = nil
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
	t.out = append(t.out, '=')
	t.emitType(typeID)
	t.appendText("{}")
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
	t.staticOut = append(t.staticOut, t.out...)
	t.out = outer
	return true
}

func (t *translator) emitObjectNestedBitfieldInitializer(typeID int, paths []initializerPath, steps []initializerStep, values [][]token) bool {
	if !t.object {
		return false
	}
	packed := make([]uint64, len(paths))
	nested := make([]bool, len(paths))
	hasNested := false
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		if path.count == 1 {
			step := steps[path.start]
			if step.field < 0 || !t.fields[step.field].emit || t.fields[step.field].bitWidth != 0 {
				return false
			}
			continue
		}
		if path.count != 2 {
			return false
		}
		outerStep, innerStep := steps[path.start], steps[path.start+1]
		if outerStep.field < 0 || innerStep.field < 0 {
			return false
		}
		outer, inner := t.fields[outerStep.field], t.fields[innerStep.field]
		nestedInfo := t.typeInfo(outer.typeID)
		value, constant := t.constantExpression(values[i])
		if !outer.emit || outer.bitWidth != 0 || inner.bitWidth <= 0 || !nestedInfo.indirect ||
			nestedInfo.kind != cTypeStruct || nestedInfo.size < 1 || nestedInfo.size > 8 || !constant {
			return false
		}
		mask := uint64(^uint64(0))
		if inner.bitWidth < 64 {
			mask = (uint64(1) << inner.bitWidth) - 1
		}
		packed[i] = (uint64(value) & mask) << inner.bitOffset
		nested[i] = true
		hasNested = true
	}
	if !hasNested {
		return false
	}
	for i := 0; i < len(paths); i++ {
		outer := steps[paths[i].start].field
		for j := 0; j < i; j++ {
			if steps[paths[j].start].field == outer && (!nested[i] || !nested[j]) {
				return false
			}
		}
	}
	t.emitType(typeID)
	t.out = append(t.out, '{')
	written := 0
	for i := 0; i < len(paths); i++ {
		outerIndex := steps[paths[i].start].field
		first := true
		for j := 0; j < i; j++ {
			first = first && steps[paths[j].start].field != outerIndex
		}
		if !first {
			continue
		}
		if written > 0 {
			t.out = append(t.out, ',')
		}
		outer := t.fields[outerIndex]
		t.appendText(outer.goName)
		t.out = append(t.out, ':')
		if !nested[i] {
			t.emitInitializerValue(paths[i].typeID, values[i])
			written++
			continue
		}
		value := uint64(0)
		for j := i; j < len(paths); j++ {
			if nested[j] && steps[paths[j].start].field == outerIndex {
				value |= packed[j]
			}
		}
		info := t.typeInfo(outer.typeID)
		t.emitType(outer.typeID)
		t.appendText("{__c_align:")
		alignBits := info.align * 8
		alignMask := uint64(^uint64(0))
		if alignBits < 64 {
			alignMask = (uint64(1) << alignBits) - 1
		}
		t.appendDecimal(int(value & alignMask))
		if tail := info.size - info.align; tail > 0 {
			t.appendText(",__c_tail:[")
			t.appendDecimal(tail)
			t.appendText("]byte{")
			for at := 0; at < tail; at++ {
				if at > 0 {
					t.out = append(t.out, ',')
				}
				t.appendDecimal(int(byte(value >> ((info.align + at) * 8))))
			}
			t.out = append(t.out, '}')
		}
		t.out = append(t.out, '}')
		written++
	}
	t.out = append(t.out, '}')
	return true
}

// Indirect structs use a compact byte carrier because their C layout cannot be
// represented faithfully by ordinary Go fields.  Object initializers still
// need the exact C bytes, including mixtures of scalar fields and bitfields.
func (t *translator) emitObjectIndirectConstantInitializer(typeID int, paths []initializerPath, steps []initializerStep, values [][]token) bool {
	info := t.typeInfo(typeID)
	if !t.object || info.kind != cTypeStruct || !info.indirect || info.size < 1 || info.align < 1 || info.align > 8 || info.align > info.size {
		return false
	}
	data := make([]byte, info.size)
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		if path.count != 1 || steps[path.start].field < 0 {
			return false
		}
		field := t.fields[steps[path.start].field]
		value, constant := t.constantExpression(values[i])
		fieldInfo := t.typeInfo(field.typeID)
		if !constant || field.offset < 0 || fieldInfo.size < 1 || fieldInfo.size > 8 || field.offset+fieldInfo.size > len(data) {
			return false
		}
		bits, bitOffset := fieldInfo.size*8, 0
		if field.bitWidth > 0 {
			bits, bitOffset = field.bitWidth, field.bitOffset
			if bits > fieldInfo.size*8 || bitOffset < 0 || bitOffset+bits > fieldInfo.size*8 {
				return false
			}
		}
		mask := uint64(^uint64(0))
		if bits < 64 {
			mask = (uint64(1) << bits) - 1
		}
		encoded := (uint64(value) & mask) << bitOffset
		for at := 0; at < fieldInfo.size; at++ {
			data[field.offset+at] |= byte(encoded >> (at * 8))
		}
	}
	t.emitType(typeID)
	t.appendText("{__c_align:")
	alignValue := uint64(0)
	for at := 0; at < info.align; at++ {
		alignValue |= uint64(data[at]) << (at * 8)
	}
	t.appendDecimal(int(alignValue))
	if tail := info.size - info.align; tail > 0 {
		t.appendText(",__c_tail:[")
		t.appendDecimal(tail)
		t.appendText("]byte{")
		for at := info.align; at < info.size; at++ {
			if at > info.align {
				t.out = append(t.out, ',')
			}
			t.appendDecimal(int(data[at]))
		}
		t.out = append(t.out, '}')
	}
	t.out = append(t.out, '}')
	return true
}

func (t *translator) emitObjectKeyedBitfieldInitializer(typeID int, paths []initializerPath, steps []initializerStep, values [][]token) bool {
	if !t.object {
		return false
	}
	hasBitfield := false
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		if path.count != 1 || steps[path.start].field < 0 {
			return false
		}
		field := t.fields[steps[path.start].field]
		if field.bitWidth > 0 {
			if _, ok := t.objectBitfieldCarrier(typeID, field); !ok {
				return false
			}
			hasBitfield = true
		} else {
			if _, _, _, ok := t.keyedStructInitializerFields(typeID, path, steps); !ok {
				return false
			}
		}
		for j := 0; j < i; j++ {
			previous := t.fields[steps[paths[j].start].field]
			if field.bitWidth > 0 && previous.bitWidth > 0 && previous.name == field.name {
				return false
			}
			currentKey := ""
			if field.bitWidth > 0 {
				carrier, _ := t.objectBitfieldCarrier(typeID, field)
				currentKey = carrier.goName
			} else {
				outer, _, _, _ := t.keyedStructInitializerFields(typeID, path, steps)
				currentKey = outer.goName
			}
			previousKey := ""
			if previous.bitWidth > 0 {
				carrier, _ := t.objectBitfieldCarrier(typeID, previous)
				previousKey = carrier.goName
			} else {
				outer, _, _, _ := t.keyedStructInitializerFields(typeID, paths[j], steps)
				previousKey = outer.goName
			}
			if currentKey == previousKey && (field.bitWidth == 0 || previous.bitWidth == 0) {
				return false
			}
		}
	}
	if !hasBitfield {
		return false
	}
	t.emitType(typeID)
	t.out = append(t.out, '{')
	written := 0
	for i := 0; i < len(paths); i++ {
		field := t.fields[steps[paths[i].start].field]
		if field.bitWidth == 0 {
			outer, inner, nested, _ := t.keyedStructInitializerFields(typeID, paths[i], steps)
			if written > 0 {
				t.out = append(t.out, ',')
			}
			t.appendText(outer.goName)
			t.out = append(t.out, ':')
			if !nested {
				t.emitInitializerValue(paths[i].typeID, values[i])
			} else {
				t.emitObjectScalarUnionValue(outer.typeID, inner, values[i])
			}
			written++
			continue
		}
		firstCarrier := true
		for j := 0; j < i; j++ {
			previous := t.fields[steps[paths[j].start].field]
			if previous.bitWidth > 0 && previous.carrier == field.carrier {
				firstCarrier = false
				break
			}
		}
		if !firstCarrier {
			continue
		}
		carrier, _ := t.objectBitfieldCarrier(typeID, field)
		carrierType := carrier.typeID
		if written > 0 {
			t.out = append(t.out, ',')
		}
		t.appendText(carrier.goName)
		t.out = append(t.out, ':')
		carrierInfo := t.typeInfo(carrierType)
		term := 0
		for j := i; j < len(paths); j++ {
			component := t.fields[steps[paths[j].start].field]
			if component.bitWidth == 0 || component.carrier != field.carrier {
				continue
			}
			if term > 0 {
				t.out = append(t.out, '|')
			}
			t.out = append(t.out, '(')
			t.convertedExpression(carrierType, values[j])
			t.appendText("&")
			t.emitBitfieldMask(carrierType, component.bitWidth, carrierInfo.size*8)
			t.out = append(t.out, ')')
			if component.bitOffset > 0 {
				t.appendText("<<")
				t.appendDecimal(component.bitOffset)
			}
			term++
		}
		written++
	}
	t.out = append(t.out, '}')
	return true
}

func (t *translator) objectBitfieldCarrier(typeID int, field cField) (cField, bool) {
	if field.bitWidth == 0 || field.carrier == "" {
		return cField{}, false
	}
	info := t.typeInfo(typeID)
	for i := 0; i < info.fieldCount; i++ {
		candidate := t.fields[info.fieldStart+i]
		if candidate.emit && candidate.synthetic && candidate.name == field.carrier {
			return candidate, true
		}
	}
	return cField{}, false
}

func (t *translator) emitKeyedStructInitializer(typeID int, paths []initializerPath, steps []initializerStep, values [][]token) bool {
	for i := 0; i < len(paths); i++ {
		outer, _, _, ok := t.keyedStructInitializerFields(typeID, paths[i], steps)
		if !ok {
			return false
		}
		for j := 0; j < i; j++ {
			previous, _, _, previousOK := t.keyedStructInitializerFields(typeID, paths[j], steps)
			if !previousOK || previous.goName == outer.goName {
				return false
			}
		}
	}
	t.emitType(typeID)
	t.out = append(t.out, '{')
	for i := 0; i < len(paths); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		outer, inner, nested, _ := t.keyedStructInitializerFields(typeID, paths[i], steps)
		t.appendText(outer.goName)
		t.out = append(t.out, ':')
		if !nested {
			t.emitInitializerValue(paths[i].typeID, values[i])
		} else {
			t.emitObjectScalarUnionValue(outer.typeID, inner, values[i])
		}
	}
	t.out = append(t.out, '}')
	return true
}

func (t *translator) keyedStructInitializerFields(typeID int, path initializerPath, steps []initializerStep) (cField, cField, bool, bool) {
	if path.count < 1 || path.count > 2 {
		return cField{}, cField{}, false, false
	}
	step := steps[path.start]
	if step.field < 0 {
		return cField{}, cField{}, false, false
	}
	outer := t.fields[step.field]
	if path.count == 2 {
		innerStep := steps[path.start+1]
		if !outer.emit || outer.bitWidth != 0 || innerStep.field < 0 {
			return cField{}, cField{}, false, false
		}
		inner := t.fields[innerStep.field]
		if t.objectScalarUnionStorageType(outer.typeID, inner) == cTypeVoidID {
			return cField{}, cField{}, false, false
		}
		return outer, inner, true, true
	}
	if outer.emit && outer.bitWidth == 0 {
		return outer, cField{}, false, true
	}
	if !t.object || outer.bitWidth != 0 {
		return cField{}, cField{}, false, false
	}
	info := t.typeInfo(typeID)
	for i := 0; i < info.fieldCount; i++ {
		candidate := t.fields[info.fieldStart+i]
		union := t.typeInfo(candidate.typeID)
		if !candidate.emit || candidate.synthetic || union.kind != cTypeUnion || candidate.offset != outer.offset {
			continue
		}
		for j := 0; j < union.fieldCount; j++ {
			inner := t.fields[union.fieldStart+j]
			if inner.name == outer.name && t.objectScalarUnionStorageType(candidate.typeID, inner) != cTypeVoidID {
				return candidate, inner, true, true
			}
		}
	}
	return cField{}, cField{}, false, false
}

func (t *translator) emitIndexedArrayInitializer(typeID int, paths []initializerPath, steps []initializerStep, values [][]token) bool {
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		if path.count != 1 || steps[path.start].field >= 0 {
			return false
		}
		for j := 0; j < i; j++ {
			if steps[paths[j].start].index == steps[path.start].index {
				return false
			}
		}
	}
	t.emitType(typeID)
	t.out = append(t.out, '{')
	for i := 0; i < len(paths); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.appendDecimal(steps[paths[i].start].index)
		t.out = append(t.out, ':')
		t.emitInitializerValue(paths[i].typeID, values[i])
	}
	t.out = append(t.out, '}')
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
		if info := t.typeInfo(typeID); info.kind == cTypeUnion || info.indirect || field.promoted {
			receiver := t.out
			t.out = append([]byte("(*"), receiver...)
			t.out = append(t.out, '.')
			t.appendText(cPointerAccessorName(field))
			t.appendText("())")
		} else {
			t.out = append(t.out, '.')
			t.out = append(t.out, field.goName...)
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
		// GNU C accepts an empty aggregate initializer as an all-zero value.
		// The zero Go carrier has the same representation for every union layout.
		t.emitType(typeID)
		t.appendText("{}")
		return
	}
	field := cField{}
	value := item
	nestedDesignator := item
	if len(item) >= 2 && tokenIs(t.src, item[0], "{") && tokenIs(t.src, item[len(item)-1], "}") {
		items := splitTopLevel(t.src, item[1:len(item)-1], ",")
		if len(items) == 1 {
			nestedDesignator = items[0]
		}
	}
	if len(nestedDesignator) >= 4 && tokenIs(t.src, nestedDesignator[0], ".") && tokenKind(nestedDesignator[1]) == tokenIdent &&
		tokenIs(t.src, nestedDesignator[2], "=") {
		var ok bool
		field, ok = t.lookupField(typeID, tokenText(t.src, nestedDesignator[1]))
		if !ok || !field.emit && !field.promoted {
			t.fail(TranslateErrUnsupported)
			return
		}
		value = nestedDesignator[3:]
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
	if t.emitObjectScalarUnionValue(typeID, field, value) {
		return
	}
	t.typeSerial++
	name := "__c_union_init_" + decimalString(t.typeSerial)
	if t.object && field.bitWidth == 0 {
		// The relocatable backend can serialize this helper as a raw write of
		// its sole member, including promoted members and nested aggregates.
		name = "__c_object_union_init_" + decimalString(field.offset) + "_x" + decimalString(t.typeSerial)
	}
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
	t.out = append(t.out, '=')
	t.emitType(typeID)
	t.appendText("{}")
	t.out = append(t.out, ';')
	if field.bitWidth > 0 {
		t.appendText("p.__c_set_")
		t.out = append(t.out, field.name...)
		t.appendText("(v);")
	} else {
		t.appendText("(*p.")
		t.appendText(cPointerAccessorName(field))
		t.appendText("())=v;")
	}
	t.appendText("return p}\n")
	t.staticOut = t.out
	t.out = outer
}

func (t *translator) objectScalarUnionStorageType(typeID int, field cField) int {
	info := t.typeInfo(typeID)
	if !t.object || info.kind != cTypeUnion || field.bitWidth != 0 || field.offset != 0 ||
		info.size != info.align || t.typeSize(field.typeID) != info.size {
		return cTypeVoidID
	}
	fieldInfo := t.typeInfo(field.typeID)
	if fieldInfo.kind != cTypeInt && fieldInfo.kind != cTypeUint && fieldInfo.kind != cTypeBool && fieldInfo.kind != cTypePointer {
		return cTypeVoidID
	}
	switch info.size {
	case 1:
		return cTypeUint8ID
	case 2:
		return cTypeUint16ID
	case 4:
		return cTypeUint32ID
	case 8:
		return cTypeUint64ID
	}
	return cTypeVoidID
}

func (t *translator) emitObjectScalarUnionValue(typeID int, field cField, value []token) bool {
	storageType := t.objectScalarUnionStorageType(typeID, field)
	if storageType == cTypeVoidID {
		return false
	}
	t.emitType(typeID)
	t.appendText("{__c_align:")
	t.convertedExpression(storageType, value)
	t.out = append(t.out, '}')
	return true
}

func (t *translator) emitObjectScalarUnionCast(typeID int, tokens []token) bool {
	info := t.typeInfo(typeID)
	if !t.object || info.kind != cTypeUnion || len(tokens) < 4 || !tokenIs(t.src, tokens[0], "(") {
		return false
	}
	close := matchingToken(t.src, tokens, 0, "(", ")")
	if close <= 1 || close >= len(tokens)-1 || tokenIs(t.src, tokens[close+1], "{") {
		return false
	}
	castType, ok := t.typeFromTokens(tokens[1:close])
	if !ok || castType != typeID {
		return false
	}
	for i := 0; i < info.fieldCount; i++ {
		field := t.fields[info.fieldStart+i]
		if t.objectScalarUnionStorageType(typeID, field) != cTypeVoidID {
			return t.emitObjectScalarUnionValue(typeID, field, tokens[close+1:])
		}
	}
	return false
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
			if indexed, indexedOK := t.subscriptBitfieldAccess(tokens[:i]); indexedOK {
				access, ok = indexed, true
			}
		}
		if !ok || !access.bitfield || access.end != i {
			if expression, expressionOK := t.expressionBitfieldAccess(tokens[:i]); expressionOK {
				access, ok = expression, true
			}
		}
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
		if !ok || !access.bitfield || access.end != len(tokens)-1 {
			if indexed, indexedOK := t.subscriptBitfieldAccess(tokens[:len(tokens)-1]); indexedOK {
				access, ok = indexed, true
			}
		}
		if ok && access.bitfield && access.end == len(tokens)-1 {
			t.emitBitfieldStep(access, tokens[len(tokens)-1])
			return true
		}
	}
	if len(tokens) > 1 && (tokenIs(t.src, tokens[0], "++") || tokenIs(t.src, tokens[0], "--")) {
		access, ok := t.memberAccess(tokens[1:], 0)
		if !ok || !access.bitfield || access.end != len(tokens)-1 {
			if indexed, indexedOK := t.subscriptBitfieldAccess(tokens[1:]); indexedOK {
				access, ok = indexed, true
			}
		}
		if ok && access.bitfield && access.end == len(tokens)-1 {
			t.emitBitfieldStep(access, tokens[0])
			return true
		}
	}
	return false
}

func (t *translator) expressionBitfieldAccess(tokens []token) (cMemberAccess, bool) {
	member := t.finalDirectMemberToken(tokens)
	if member <= 0 || member+2 != len(tokens) ||
		(!tokenIs(t.src, tokens[member], "->") && !tokenIs(t.src, tokens[member], ".")) ||
		tokenKind(tokens[member+1]) != tokenIdent {
		return cMemberAccess{}, false
	}
	aggregateType := t.expressionType(tokens[:member])
	if tokenIs(t.src, tokens[member], "->") {
		pointer := t.typeInfo(aggregateType)
		if pointer.kind != cTypePointer {
			return cMemberAccess{}, false
		}
		aggregateType = pointer.base
	}
	aggregate := t.typeInfo(aggregateType)
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[member+1]))
	if !ok || (aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion) || field.bitWidth == 0 {
		return cMemberAccess{}, false
	}
	outer := t.out
	t.out = nil
	t.emitExpression(tokens[:member])
	receiver := append([]byte{}, t.out...)
	t.out = outer
	expr := append([]byte{}, receiver...)
	expr = append(expr, ".__c_get_"...)
	expr = append(expr, field.name...)
	expr = append(expr, '(', ')')
	return cMemberAccess{expr: expr, receiver: receiver, field: field, typeID: field.typeID,
		end: len(tokens), bitfield: true}, true
}

func (t *translator) subscriptBitfieldAccess(tokens []token) (cMemberAccess, bool) {
	paren, bracket, member := 0, 0, -1
	hasSubscript := false
	for i := 0; i+1 < len(tokens); i++ {
		switch {
		case tokenIs(t.src, tokens[i], "("):
			paren++
		case tokenIs(t.src, tokens[i], ")"):
			paren--
		case tokenIs(t.src, tokens[i], "["):
			bracket++
			hasSubscript = true
		case tokenIs(t.src, tokens[i], "]"):
			bracket--
		case paren == 0 && bracket == 0 && (tokenIs(t.src, tokens[i], ".") || tokenIs(t.src, tokens[i], "->")):
			member = i
		}
	}
	if !hasSubscript || member <= 0 || member+2 != len(tokens) || tokenKind(tokens[member+1]) != tokenIdent {
		return cMemberAccess{}, false
	}
	aggregateType := t.expressionType(tokens[:member])
	if tokenIs(t.src, tokens[member], "->") {
		pointer := t.typeInfo(aggregateType)
		if pointer.kind != cTypePointer {
			return cMemberAccess{}, false
		}
		aggregateType = pointer.base
	}
	aggregate := t.typeInfo(aggregateType)
	field, ok := t.lookupField(aggregateType, tokenText(t.src, tokens[member+1]))
	if !ok || (aggregate.kind != cTypeStruct && aggregate.kind != cTypeUnion) || field.bitWidth == 0 {
		return cMemberAccess{}, false
	}
	outer := t.out
	t.out = nil
	if !t.emitPointerSubscript(tokens[:member]) {
		t.out = outer
		return cMemberAccess{}, false
	}
	receiver := append([]byte{}, t.out...)
	t.out = outer
	expr := append([]byte{}, receiver...)
	expr = append(expr, ".__c_get_"...)
	expr = append(expr, field.name...)
	expr = append(expr, '(', ')')
	return cMemberAccess{expr: expr, receiver: receiver, field: field, typeID: field.typeID,
		end: len(tokens), bitfield: true}, true
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
	return tokenIs(t.src, tok, "++") || tokenIs(t.src, tok, "--") || t.sizeOperator(tok)
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
	var address []byte
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
		fieldInfo := t.typeInfo(fieldType)
		functionPointerField := fieldInfo.kind == cTypePointer && t.typeInfo(fieldInfo.base).kind == cTypeFunction
		if arrow && len(address) == 0 && !cSimpleGoIdentifier(expr) && !functionPointerField && !t.checkOnly {
			address = append([]byte{}, expr...)
		}
		receiver := append([]byte{}, result.expr...)
		if len(address) != 0 {
			receiver = append(receiver[:0], address...)
		}
		if field.bitWidth > 0 {
			expr = append(receiver, ".__c_get_"...)
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
		needsAccessor := aggregate.kind == cTypeUnion || aggregate.indirect || !field.emit
		if len(address) != 0 && !needsAccessor && !t.checkOnly {
			name := t.ensureNestedFieldAccessor(aggregateType, field)
			call := make([]byte, 0, len(name)+len(address)+3)
			call = append(call, address...)
			call = append(call, '.')
			call = append(call, name...)
			call = append(call, '(', ')')
			wrapped := make([]byte, 0, len(call)+3)
			wrapped = append(wrapped, '(', '*')
			wrapped = append(wrapped, call...)
			wrapped = append(wrapped, ')')
			expr = wrapped
			address = call
		} else if needsAccessor {
			accessorReceiver := expr
			if len(address) != 0 {
				accessorReceiver = address
			}
			call := make([]byte, 0, len(accessorReceiver)+len(field.name)+13)
			call = append(call, accessorReceiver...)
			call = append(call, '.')
			call = append(call, cPointerAccessorName(field)...)
			call = append(call, '(', ')')
			wrapped := make([]byte, 0, len(call)+3)
			wrapped = append(wrapped, '(', '*')
			wrapped = append(wrapped, call...)
			wrapped = append(wrapped, ')')
			expr = wrapped
			address = call
		} else if field.emit {
			expr = append(expr, '.')
			expr = append(expr, field.goName...)
			address = nil
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

func cSimpleGoIdentifier(expr []byte) bool {
	if len(expr) == 0 || !((expr[0] >= 'a' && expr[0] <= 'z') || (expr[0] >= 'A' && expr[0] <= 'Z') || expr[0] == '_') {
		return false
	}
	for i := 1; i < len(expr); i++ {
		if !((expr[i] >= 'a' && expr[i] <= 'z') || (expr[i] >= 'A' && expr[i] <= 'Z') ||
			(expr[i] >= '0' && expr[i] <= '9') || expr[i] == '_') {
			return false
		}
	}
	return true
}

func (t *translator) ensureNestedFieldAccessor(typeID int, field cField) string {
	name := "__c_nested_ptr_" + decimalString(typeID) + "_" + decimalString(field.offset) + "_" + field.goName
	for i := 0; i < len(t.nestedAccessorKeys); i++ {
		if t.nestedAccessorKeys[i] == name {
			return name
		}
	}
	t.nestedAccessorKeys = append(t.nestedAccessorKeys, name)
	outer := t.out
	t.out = nil
	t.out = append(t.out, '\n')
	t.appendText("func(p *")
	t.emitType(typeID)
	t.appendText(")")
	t.appendText(name)
	t.appendText("() *")
	t.emitType(field.typeID)
	t.appendText("{return &p.")
	t.appendText(field.goName)
	t.appendText("}\n")
	t.staticOut = append(t.staticOut, t.out...)
	t.out = outer
	return name
}

func cPointerAccessorName(field cField) string {
	return "__c_ptr_" + decimalString(field.offset) + "_" + field.goName
}

func matchingToken(src []byte, tokens []token, at int, open string, close string) int {
	if at < 0 || at >= len(tokens) || !tokenIs(src, tokens[at], open) {
		return -1
	}
	if match := at + tokens[at].match; tokens[at].match > 0 && match < len(tokens) && tokenIs(src, tokens[match], close) {
		return match
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

type cOffsetofIndex struct {
	tokens   []token
	elemSize int
}

func (t *translator) emitDynamicOffsetof(typeID int, tokens []token) bool {
	offset := 0
	var indices []cOffsetofIndex
	for pos := 0; pos < len(tokens); {
		info := t.typeInfo(typeID)
		if (info.kind != cTypeStruct && info.kind != cTypeUnion) || tokenKind(tokens[pos]) != tokenIdent {
			return false
		}
		field, found := t.lookupField(typeID, tokenText(t.src, tokens[pos]))
		if !found || field.bitWidth != 0 {
			return false
		}
		offset += field.offset
		typeID = field.typeID
		pos++
		for pos < len(tokens) && tokenIs(t.src, tokens[pos], "[") {
			end := matchingToken(t.src, tokens, pos, "[", "]")
			info = t.typeInfo(typeID)
			if end < 0 || info.kind != cTypeArray || end == pos+1 {
				return false
			}
			if index, ok := t.constantExpression(tokens[pos+1 : end]); ok && index >= 0 {
				offset += index * t.typeSize(info.base)
			} else {
				indices = append(indices, cOffsetofIndex{tokens: tokens[pos+1 : end], elemSize: t.typeSize(info.base)})
			}
			typeID = info.base
			pos = end + 1
		}
		if pos == len(tokens) {
			break
		}
		if !tokenIs(t.src, tokens[pos], ".") {
			return false
		}
		pos++
	}
	if len(indices) == 0 {
		return false
	}
	sizeType := cTypeUint64ID
	if t.pointerSize == 4 {
		sizeType = cTypeUint32ID
	}
	t.emitType(sizeType)
	t.out = append(t.out, '(')
	t.appendDecimal(offset)
	for i := 0; i < len(indices); i++ {
		t.appendText("+")
		t.emitType(sizeType)
		t.out = append(t.out, '(')
		t.convertedExpression(sizeType, indices[i].tokens)
		t.appendText(")*")
		t.appendDecimal(indices[i].elemSize)
	}
	t.out = append(t.out, ')')
	return t.ok
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

func (t *translator) isNullPointerExpression(tokens []token) bool {
	if value, ok := t.constantExpression(tokens); ok && value == 0 {
		return true
	}
	if t.nullPointerPrefix(tokens) == len(tokens) {
		return true
	}
	tokens = t.trimExpressionParens(tokens)
	if len(tokens) < 4 || !tokenIs(t.src, tokens[0], "(") {
		return false
	}
	close := matchingToken(t.src, tokens, 0, "(", ")")
	if close <= 1 || close >= len(tokens)-1 {
		return false
	}
	typeID, ok := t.typeFromTokens(tokens[1:close])
	if !ok || t.typeInfo(typeID).kind != cTypePointer {
		return false
	}
	value, constant := t.constantExpression(t.trimExpressionParens(tokens[close+1:]))
	return constant && value == 0
}

func (t *translator) emitCStringValues(tokens []token) {
	value, ok := t.cStringBytes(tokens)
	if !ok {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.emitByteString(value)
}

func (t *translator) emitByteString(value []byte) {
	t.out = append(t.out, '"')
	hex := "0123456789abcdef"
	for i := 0; i < len(value); i++ {
		t.out = append(t.out, '\\', 'x', hex[value[i]>>4], hex[value[i]&15])
	}
	t.out = append(t.out, '"')
}

func (t *translator) cWideStringBytes(tokens []token) ([]byte, bool) {
	var out []byte
	if len(tokens) == 0 || t.wcharSize != 2 && t.wcharSize != 4 {
		return nil, false
	}
	for i := 0; i < len(tokens); i++ {
		if tokenKind(tokens[i]) != tokenString {
			return nil, false
		}
		text := tokenText(t.src, tokens[i])
		if len(text) < 3 || text[0] != 'L' || text[1] != '"' {
			return nil, false
		}
		value, ok := decodeCString(text)
		if !ok {
			return nil, false
		}
		for j := 0; j < len(value); j++ {
			if value[j] >= 128 {
				return nil, false
			}
			out = append(out, value[j])
			for width := 1; width < t.wcharSize; width++ {
				out = append(out, 0)
			}
		}
	}
	for width := 0; width < t.wcharSize; width++ {
		out = append(out, 0)
	}
	return out, true
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
	if !ok && len(tokens) > 0 && tokenKind(tokens[0]) == tokenString {
		t.fail(TranslateErrDeclaration)
		return true
	}
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
	if t.isGNUAttribute() {
		return true
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
	if _, known := t.builtinCType(name); known {
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
