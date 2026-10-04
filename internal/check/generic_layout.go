package check

// Storage follows the concrete lowering ABI, including normalized descriptor
// slots. Native aggregates use their fields' native alignment. Language
// alignment is separate: complex64 has float32 alignment even where storage
// reserves an eight-byte field. Ordinary callable descriptors have fixed tag
// and an interface environment; object callables retain native code pointers.
type genericValueLayout struct {
	size         uint64
	align        int
	storageAlign int
	known        bool
	offsets      []uint64
	offsetKnown  []bool
}

func (e *genericEnvironment) scalarAlignment(size uint64) int {
	maximum := e.graph.Layout.ScalarAlign
	if maximum == 0 {
		maximum = 4
		if e.wordBits == 64 {
			maximum = 8
		}
	}
	align := 1
	if size >= 8 {
		align = 8
	} else if size >= 4 {
		align = 4
	} else if size >= 2 {
		align = 2
	}
	if align > maximum {
		align = maximum
	}
	return align
}

func genericLayoutAlign(value uint64, align int) (uint64, bool) {
	padding := uint64(align - 1)
	if align < 1 || value > ^uint64(0)-padding {
		return 0, false
	}
	return (value + padding) &^ padding, true
}

func genericLayoutQuerySize(layout genericValueLayout) uint64 {
	return layout.size
}

func (e *genericEnvironment) valueLayout(typ int, native bool, depth int) genericValueLayout {
	if typ == 0 || depth > len(e.types.items) {
		return genericValueLayout{}
	}
	v := e.types.get(e.types.underlying(typ))
	size := uint64(e.unsafeScalarSize(typ))
	if v.kind == genericBasic && v.name == "unsafe.Pointer" || v.kind == genericPointer {
		size = 8
		if native {
			size = uint64(e.wordBits / 8)
		}
	} else if v.kind == genericSlice || v.kind == genericMap {
		size = 24
	} else if v.kind == genericInterface {
		size = 16
	} else if v.kind == genericChan {
		size = uint64(e.wordBits / 8)
	} else if v.kind == genericFunc {
		if e.graph.Layout.Object {
			size = uint64(e.wordBits / 8)
		} else {
			size = 24
		}
	}
	if size > 0 {
		align := e.scalarAlignment(size)
		languageAlign := align
		if v.kind == genericBasic && v.name == "complex64" {
			languageAlign = e.scalarAlignment(4)
		}
		return genericValueLayout{size: size, align: languageAlign, storageAlign: align, known: true}
	}
	if v.kind == genericArray {
		element := e.valueLayout(v.elem, native, depth+1)
		result := genericValueLayout{align: element.align, storageAlign: element.storageAlign}
		if v.length == ^uint64(0) {
			return result
		}
		if v.length == 0 {
			result.known = true
			return result
		}
		if !element.known {
			return result
		}
		stride := element.size
		if !native {
			stride = genericLayoutQuerySize(element)
		}
		if stride != 0 && v.length > ^uint64(0)/stride {
			return result
		}
		result.size, result.known = v.length*stride, true
		return result
	}
	if v.kind != genericStruct {
		return genericValueLayout{}
	}
	packed := native || e.graph.Layout.Object
	if !packed && len(v.fields) > 0 {
		packed = true
		for _, field := range v.fields {
			base := e.types.get(e.types.underlying(field.typ))
			if base.kind != genericBasic || base.name == "int" || base.name == "uint" || base.name == "uintptr" || base.name == "string" || base.name == "unsafe.Pointer" || base.name == "complex64" || base.name == "complex128" {
				packed = false
			}
		}
	}
	result := genericValueLayout{align: 1, storageAlign: 1, known: true, offsets: make([]uint64, len(v.fields)), offsetKnown: make([]bool, len(v.fields))}
	position := uint64(0)
	for i, field := range v.fields {
		layout := e.valueLayout(field.typ, packed, depth+1)
		if layout.align == 0 {
			result.align = 0
		} else if result.align != 0 && layout.align > result.align {
			result.align = layout.align
		}
		alignment := 8
		fieldSize := genericLayoutQuerySize(layout)
		if packed {
			alignment, fieldSize = layout.storageAlign, layout.size
			if alignment > result.storageAlign {
				result.storageAlign = alignment
			}
		}
		if result.known {
			var ok bool
			position, ok = genericLayoutAlign(position, alignment)
			result.known = ok
			result.offsets[i], result.offsetKnown[i] = position, ok
		}
		if !layout.known || fieldSize > ^uint64(0)-position {
			result.known = false
		} else if result.known {
			position += fieldSize
		}
	}
	if result.known {
		alignment := 8
		if packed {
			alignment = result.storageAlign
		}
		result.size, result.known = genericLayoutAlign(position, alignment)
	}
	return result
}

func (e *genericEnvironment) layoutOffset(typ int, path []int) (uint64, bool) {
	position := uint64(0)
	native := e.graph.Layout.Object
	for _, field := range path {
		layout := e.valueLayout(typ, native, 0)
		if field < 0 || field >= len(layout.offsets) || !layout.offsetKnown[field] || layout.offsets[field] > ^uint64(0)-position {
			return 0, false
		}
		position += layout.offsets[field]
		base := e.types.get(e.types.underlying(typ))
		// Native aggregate layout propagates into embedded aggregates.
		if e.graph.Layout.Object {
			native = true
		}
		typ = base.fields[field].typ
	}
	return position, true
}
