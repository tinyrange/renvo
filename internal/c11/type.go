package c11

const (
	DataModelInvalid = iota
	DataModelLP64
)

type cTargetDescriptor struct {
	dataModel   int
	boolSize    int
	charSize    int
	shortSize   int
	intSize     int
	longSize    int
	floatSize   int
	doubleSize  int
	pointerSize int
}

// cX8664SysV is the single source of target sizes and natural alignments for
// the first C object target. Aggregate and bitfield layout consume the scalar
// type entries initialized from this descriptor.
var cX8664SysV = cTargetDescriptor{
	dataModel: DataModelLP64, boolSize: 1, charSize: 1, shortSize: 2,
	intSize: 4, longSize: 8, floatSize: 4, doubleSize: 8, pointerSize: 8,
}

const (
	cQualifierConst = 1 << iota
	cQualifierVolatile
	cQualifierRestrict
	cQualifierAtomic
)

const (
	cTypeInvalid = iota
	cTypeVoid
	cTypeBool
	cTypeInt
	cTypeUint
	cTypeFloat
	cTypePointer
	cTypeArray
	cTypeStruct
	cTypeUnion
	cTypeFunction
	cTypeOpaque
)

const (
	cTypeVoidID = iota
	cTypeBoolID
	cTypeInt8ID
	cTypeUint8ID
	cTypeInt16ID
	cTypeUint16ID
	cTypeInt32ID
	cTypeUint32ID
	cTypeInt64ID
	cTypeUint64ID
	cTypeFloat32ID
	cTypeFloat64ID
	cTypeUintptrID
)

// cTypeInfo is deliberately pointer-free. C types are integer IDs into one
// flat arena so declarations remain cheap even for generated and kernel-sized
// translation units.
type cTypeInfo struct {
	kind       int
	base       int
	count      int
	size       int
	align      int
	fieldStart int
	fieldCount int
	paramStart int
	paramCount int
	variadic   bool
	qualifiers int
	goName     string
}

type cField struct {
	name      string
	typeID    int
	offset    int
	bitOffset int
	bitWidth  int
	carrier   string
	emit      bool
	synthetic bool
}

type cTypeName struct {
	name   string
	typeID int
}

const (
	cNameTypedef = iota
	cNameValue
	cNameObject
	cNameFunction
	cNameTag
)

type cNamespaceName struct {
	name  string
	index int
	kind  int
}

type cScopeMark struct {
	names, nameStart int
}

type cObjectName struct {
	name   string
	goName string
	typeID int
	auto   bool
}

type cFunctionName struct {
	name       string
	typeID     int
	resultType int
	paramStart int
	paramCount int
	variadic   bool
	defined    bool
}

type cVariadicCall struct {
	function   int
	paramStart int
	paramCount int
	goName     string
}

type cMemberAccess struct {
	expr     []byte
	receiver []byte
	field    cField
	typeID   int
	end      int
	bitfield bool
}

func (t *translator) initTypes(dataModel int) {
	// Object mode is currently gated to Linux/x86_64. The explicit descriptor
	// prevents host sizeof assumptions from leaking into C types.
	t.dataModel = dataModel
	target := cX8664SysV
	if dataModel != target.dataModel {
		t.ok = false
		t.err = TranslateErrUnsupported
		return
	}
	t.pointerSize = target.pointerSize
	t.types = append(t.types,
		cTypeInfo{kind: cTypeVoid, size: target.charSize, align: target.charSize},
		cTypeInfo{kind: cTypeBool, size: target.boolSize, align: target.boolSize, goName: "uint8"},
		cTypeInfo{kind: cTypeInt, size: target.charSize, align: target.charSize, goName: "int8"},
		cTypeInfo{kind: cTypeUint, size: target.charSize, align: target.charSize, goName: "uint8"},
		cTypeInfo{kind: cTypeInt, size: target.shortSize, align: target.shortSize, goName: "int16"},
		cTypeInfo{kind: cTypeUint, size: target.shortSize, align: target.shortSize, goName: "uint16"},
		cTypeInfo{kind: cTypeInt, size: target.intSize, align: target.intSize, goName: "int32"},
		cTypeInfo{kind: cTypeUint, size: target.intSize, align: target.intSize, goName: "uint32"},
		cTypeInfo{kind: cTypeInt, size: target.longSize, align: target.longSize, goName: "int64"},
		cTypeInfo{kind: cTypeUint, size: target.longSize, align: target.longSize, goName: "uint64"},
		cTypeInfo{kind: cTypeFloat, size: target.floatSize, align: target.floatSize, goName: "float32"},
		cTypeInfo{kind: cTypeFloat, size: target.doubleSize, align: target.doubleSize, goName: "float64"},
		cTypeInfo{kind: cTypeUint, size: target.pointerSize, align: target.pointerSize, goName: "uintptr"},
	)
}

func (t *translator) typeInfo(typeID int) cTypeInfo {
	if typeID < 0 || typeID >= len(t.types) {
		return cTypeInfo{}
	}
	return t.types[typeID]
}

func (t *translator) pointerType(base int) int {
	for i := cTypeUintptrID + 1; i < len(t.types); i++ {
		if t.types[i].kind == cTypePointer && t.types[i].base == base {
			return i
		}
	}
	t.types = append(t.types, cTypeInfo{kind: cTypePointer, base: base, size: t.pointerSize, align: t.pointerSize})
	return len(t.types) - 1
}

func (t *translator) qualifiedType(typeID int, qualifiers int) int {
	if qualifiers == 0 || t.typeInfo(typeID).qualifiers&qualifiers == qualifiers {
		return typeID
	}
	want := t.typeInfo(typeID)
	want.qualifiers |= qualifiers
	for i := 0; i < len(t.types); i++ {
		info := t.types[i]
		if info.kind == want.kind && info.base == want.base && info.count == want.count &&
			info.fieldStart == want.fieldStart && info.fieldCount == want.fieldCount &&
			info.paramStart == want.paramStart && info.paramCount == want.paramCount &&
			info.variadic == want.variadic && info.qualifiers == want.qualifiers {
			return i
		}
	}
	t.types = append(t.types, want)
	return len(t.types) - 1
}

func (t *translator) arrayType(base int, count int) int {
	info := t.typeInfo(base)
	t.types = append(t.types, cTypeInfo{kind: cTypeArray, base: base, count: count, size: info.size * count, align: info.align})
	return len(t.types) - 1
}

func (t *translator) functionType(result int, params []parameter, variadic bool) int {
	start := len(t.functionParams)
	for i := 0; i < len(params); i++ {
		t.functionParams = append(t.functionParams, params[i].typeID)
	}
	t.types = append(t.types, cTypeInfo{
		kind:       cTypeFunction,
		base:       result,
		size:       t.pointerSize,
		align:      t.pointerSize,
		paramStart: start,
		paramCount: len(params),
		variadic:   variadic,
	})
	return len(t.types) - 1
}

func (t *translator) opaqueType(name string) int {
	for i := 0; i < len(t.opaqueTypes); i++ {
		if t.opaqueTypes[i].name == name {
			return t.opaqueTypes[i].typeID
		}
	}
	t.types = append(t.types, cTypeInfo{kind: cTypeOpaque, size: 1, align: 1, goName: "byte"})
	typeID := len(t.types) - 1
	t.opaqueTypes = append(t.opaqueTypes, cTypeName{name: name, typeID: typeID})
	return typeID
}

func (t *translator) rememberTypedef(name string, typeID int) {
	t.rememberName(name, typeID, cNameTypedef)
}

func (t *translator) lookupTypedef(name []byte) (int, bool) {
	return t.lookupName(name, cNameTypedef)
}

func (t *translator) lookupTag(name []byte) (int, bool) {
	return t.lookupName(name, cNameTag)
}

func (t *translator) emitType(typeID int) {
	info := t.typeInfo(typeID)
	switch info.kind {
	case cTypePointer:
		if t.typeInfo(info.base).kind == cTypeFunction {
			t.emitType(info.base)
			break
		}
		t.out = append(t.out, '*')
		t.emitType(info.base)
	case cTypeArray:
		t.out = append(t.out, '[')
		t.appendDecimal(info.count)
		t.out = append(t.out, ']')
		t.emitType(info.base)
	case cTypeStruct, cTypeUnion:
		t.out = append(t.out, info.goName...)
	case cTypeFunction:
		t.appendText("func(")
		for i := 0; i < info.paramCount; i++ {
			if i > 0 {
				t.out = append(t.out, ',')
			}
			t.emitType(t.functionParams[info.paramStart+i])
		}
		if info.variadic {
			if info.paramCount > 0 {
				t.out = append(t.out, ',')
			}
			t.appendText("...uintptr")
		}
		t.out = append(t.out, ')')
		if info.base != cTypeVoidID {
			t.out = append(t.out, ' ')
			t.emitType(info.base)
		}
	case cTypeVoid:
		t.appendText("byte")
	default:
		t.out = append(t.out, info.goName...)
	}
}

func (t *translator) typeSize(typeID int) int {
	return t.typeInfo(typeID).size
}

func (t *translator) typeAlign(typeID int) int {
	align := t.typeInfo(typeID).align
	if align < 1 {
		return 1
	}
	return align
}

func alignC(value int, alignment int) int {
	return (value + alignment - 1) &^ (alignment - 1)
}

func builtinCType(name []byte) (int, bool) {
	if textEquals(name, "size_t") || textEquals(name, "uintptr_t") {
		return cTypeUintptrID, true
	}
	if textEquals(name, "intptr_t") || textEquals(name, "ptrdiff_t") {
		return cTypeInt64ID, true
	}
	if textEquals(name, "int8_t") {
		return cTypeInt8ID, true
	}
	if textEquals(name, "uint8_t") {
		return cTypeUint8ID, true
	}
	if textEquals(name, "int16_t") {
		return cTypeInt16ID, true
	}
	if textEquals(name, "uint16_t") {
		return cTypeUint16ID, true
	}
	if textEquals(name, "int32_t") {
		return cTypeInt32ID, true
	}
	if textEquals(name, "uint32_t") {
		return cTypeUint32ID, true
	}
	if textEquals(name, "int64_t") {
		return cTypeInt64ID, true
	}
	if textEquals(name, "uint64_t") {
		return cTypeUint64ID, true
	}
	return cTypeVoidID, false
}
