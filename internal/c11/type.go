package c11

const (
	DataModelInvalid = iota
	DataModelLP64
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
	goName     string
}

type cField struct {
	name   string
	typeID int
	offset int
	emit   bool
}

type cTypeName struct {
	name   string
	typeID int
}

type cValueName struct {
	name  string
	value int
}

func (t *translator) initTypes(dataModel int) {
	// Object mode is currently gated to Linux/x86_64. Keeping the model as an
	// explicit ID prevents host sizeof assumptions from leaking into C types
	// and leaves the declaration arena ready for later target descriptors.
	t.dataModel = dataModel
	if dataModel != DataModelLP64 {
		t.ok = false
		t.err = TranslateErrUnsupported
		return
	}
	t.pointerSize = 8
	t.types = append(t.types,
		cTypeInfo{kind: cTypeVoid, size: 1, align: 1},
		cTypeInfo{kind: cTypeBool, size: 1, align: 1, goName: "bool"},
		cTypeInfo{kind: cTypeInt, size: 1, align: 1, goName: "int8"},
		cTypeInfo{kind: cTypeUint, size: 1, align: 1, goName: "uint8"},
		cTypeInfo{kind: cTypeInt, size: 2, align: 2, goName: "int16"},
		cTypeInfo{kind: cTypeUint, size: 2, align: 2, goName: "uint16"},
		cTypeInfo{kind: cTypeInt, size: 4, align: 4, goName: "int32"},
		cTypeInfo{kind: cTypeUint, size: 4, align: 4, goName: "uint32"},
		cTypeInfo{kind: cTypeInt, size: 8, align: 8, goName: "int64"},
		cTypeInfo{kind: cTypeUint, size: 8, align: 8, goName: "uint64"},
		cTypeInfo{kind: cTypeFloat, size: 4, align: 4, goName: "float32"},
		cTypeInfo{kind: cTypeFloat, size: 8, align: 8, goName: "float64"},
		cTypeInfo{kind: cTypeUint, size: 8, align: 8, goName: "uintptr"},
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

func (t *translator) arrayType(base int, count int) int {
	info := t.typeInfo(base)
	t.types = append(t.types, cTypeInfo{kind: cTypeArray, base: base, count: count, size: info.size * count, align: info.align})
	return len(t.types) - 1
}

func (t *translator) functionType(result int, params []parameter) int {
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
	for i := len(t.typedefs) - 1; i >= 0; i-- {
		if t.typedefs[i].name == name {
			t.typedefs[i].typeID = typeID
			return
		}
	}
	t.typedefs = append(t.typedefs, cTypeName{name: name, typeID: typeID})
}

func (t *translator) lookupTypedef(name []byte) (int, bool) {
	for i := len(t.typedefs) - 1; i >= 0; i-- {
		if textEquals(name, t.typedefs[i].name) {
			return t.typedefs[i].typeID, true
		}
	}
	return cTypeVoidID, false
}

func (t *translator) lookupTag(name []byte) (int, bool) {
	for i := len(t.tags) - 1; i >= 0; i-- {
		if textEquals(name, t.tags[i].name) {
			return t.tags[i].typeID, true
		}
	}
	return cTypeVoidID, false
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
		t.out = append(t.out, "func("...)
		for i := 0; i < info.paramCount; i++ {
			if i > 0 {
				t.out = append(t.out, ',')
			}
			t.emitType(t.functionParams[info.paramStart+i])
		}
		t.out = append(t.out, ')')
		if info.base != cTypeVoidID {
			t.out = append(t.out, ' ')
			t.emitType(info.base)
		}
	case cTypeVoid:
		t.out = append(t.out, "byte"...)
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
