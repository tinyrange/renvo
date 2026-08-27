package unit

// BindEntrypoint adds the selected function index to an otherwise complete
// unit. It must run before target binding because compact backend readers keep
// the target identity as the final child.
func BindEntrypoint(data []byte, function int) ([]byte, bool) {
	if function < 0 {
		return nil, false
	}
	payload := appendVarint(nil, function)
	return appendOptionalChild(data, TagEntrypoint, payload)
}

// BindForeignPrograms adds unresolved or resolved foreign-program records to
// an otherwise complete unit. Existing records are deliberately rejected so
// orchestration cannot accidentally retain stale artifacts.
func BindForeignPrograms(data []byte, programs []ForeignProgram) ([]byte, bool) {
	if len(programs) == 0 {
		return data, true
	}
	return appendOptionalChild(data, TagForeignPrograms, encodeForeignProgramsCore(programs))
}

// ReadForeignPrograms decodes the optional multi-target payload without
// decoding the ordinary source tables. Absence is a successful empty result.
func ReadForeignPrograms(data []byte) ([]ForeignProgram, bool) {
	_, payload, found, ok := findOptionalChild(data, TagForeignPrograms)
	if !ok || !found {
		return nil, ok
	}
	r := foreignReader{data: payload, ok: true}
	count := r.varint()
	programs := make([]ForeignProgram, 0, count)
	for i := 0; i < count && r.ok; i++ {
		program := ForeignProgram{}
		program.Global = r.varint()
		state := r.varint()
		payload := r.bytes()
		program.EntryOffset = r.varint()
		program.Kind = state
		if state > 2 {
			program.Kind -= 2
			program.Artifact = payload
		} else {
			program.Unit = payload
			binding, bound := ReadTargetBinding(payload)
			if bound {
				program.Target = binding.Target
			}
		}
		program.InPlace = program.Kind == ForeignProgramEntrypoint
		if program.Global < 0 || state < 1 || state > 4 || len(payload) == 0 ||
			program.Kind < ForeignProgramBytes || program.Kind > ForeignProgramEntrypoint ||
			(len(program.Unit) > 0 && program.Target == "") || program.EntryOffset < 0 ||
			(program.Kind == ForeignProgramBytes && program.EntryOffset != 0) ||
			(state == 4 && program.EntryOffset >= len(program.Artifact)) {
			r.ok = false
		}
		programs = append(programs, program)
	}
	if !r.ok || r.pos != len(r.data) {
		return nil, false
	}
	return programs, true
}

// ResolveForeignPrograms replaces the optional table while preserving child
// order, including the final target-binding nodes expected by compact readers.
func ResolveForeignPrograms(data []byte, programs []ForeignProgram) ([]byte, bool) {
	at, _, found, ok := findOptionalChild(data, TagForeignPrograms)
	if !ok || !found || len(programs) == 0 {
		return nil, false
	}
	payload := encodeForeignProgramsCore(programs)
	oldLength := readUint32Foreign(data, at+2)
	oldEnd := at + 6 + oldLength
	out := make([]byte, 0, len(data)-(oldEnd-at)+6+len(payload))
	out = append(out, data[:at]...)
	out = appendNode(out, TagForeignPrograms, payload)
	out = append(out, data[oldEnd:]...)
	patchUint32Core(out, 10, len(out)-14)
	return out, true
}

func appendOptionalChild(data []byte, tag int, payload []byte) ([]byte, bool) {
	_, _, found, ok := findOptionalChild(data, tag)
	if !ok || found {
		return nil, false
	}
	out := make([]byte, len(data), len(data)+6+len(payload))
	copy(out, data)
	out = appendNode(out, tag, payload)
	patchUint32Core(out, 10, len(out)-14)
	return out, true
}

func findOptionalChild(data []byte, wanted int) (int, []byte, bool, bool) {
	if !validUnitRoot(data) {
		return 0, nil, false, false
	}
	for pos := 14; pos < len(data); {
		if pos+6 > len(data) {
			return 0, nil, false, false
		}
		tag := int(data[pos]) | int(data[pos+1])<<8
		length := readUint32Foreign(data, pos+2)
		start := pos + 6
		next := start + length
		if length < 0 || next < start || next > len(data) {
			return 0, nil, false, false
		}
		if tag == wanted {
			return pos, data[start:next], true, true
		}
		pos = next
	}
	return 0, nil, false, true
}

type foreignReader struct {
	data []byte
	pos  int
	ok   bool
}

func (r *foreignReader) varint() int {
	if !r.ok || r.pos >= len(r.data) {
		r.ok = false
		return -1
	}
	value := 0
	shift := 0
	for r.pos < len(r.data) && shift <= 28 {
		part := r.data[r.pos]
		r.pos++
		if shift == 28 && part >= 16 {
			r.ok = false
			return -1
		}
		value |= int(part&127) << shift
		if part < 128 {
			if shift > 0 && part == 0 {
				r.ok = false
			}
			return value
		}
		shift += 7
	}
	r.ok = false
	return -1
}

func (r *foreignReader) bytes() []byte {
	length := r.varint()
	if !r.ok || length < 0 || r.pos+length < r.pos || r.pos+length > len(r.data) {
		r.ok = false
		return nil
	}
	value := r.data[r.pos : r.pos+length]
	r.pos += length
	return value
}

func readUint32Foreign(data []byte, at int) int {
	if at < 0 || at+4 > len(data) {
		return -1
	}
	return int(data[at]) | int(data[at+1])<<8 | int(data[at+2])<<16 | int(data[at+3])<<24
}
