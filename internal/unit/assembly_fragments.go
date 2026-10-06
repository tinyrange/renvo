package unit

// AttachRTGAssemblyFragments replaces only the validated optional assembly
// child. Source bytes and function/source/entry identity are retained.

func AttachRTGAssemblyFragments(data []byte, bindings []RTGAssemblyBinding, code [][]byte) ([]byte, bool) {
	sources, original, ok := ReadRTGAssemblyFragments(data)
	if !ok || len(bindings) != len(original) || len(code) != len(bindings) {
		return nil, false
	}
	payload := appendRTGASMVarint(nil, 0)
	payload = appendRTGASMVarint(payload, 2)
	payload = appendRTGASMVarint(payload, len(sources))
	for i := 0; i < len(sources); i++ {
		payload = appendRTGASMBytes(payload, []byte(sources[i].Path))
		payload = appendRTGASMBytes(payload, sources[i].Source)
	}
	payload = appendRTGASMVarint(payload, len(bindings))
	for i := 0; i < len(bindings); i++ {
		if bindings[i].Func != original[i].Func || bindings[i].Source != original[i].Source || bindings[i].Entry != original[i].Entry || bindings[i].Mode < 0 || bindings[i].Mode > 1 || bindings[i].Inputs < 0 || bindings[i].Inputs > 6 || bindings[i].Outputs < 0 || bindings[i].Outputs > 1 {
			return nil, false
		}
		if bindings[i].Mode == 0 && (bindings[i].Inputs != 0 || bindings[i].Outputs != 0) {
			return nil, false
		}
		if len(code[i]) == 0 || len(code[i]) > 16*1024*1024 {
			return nil, false
		}
		payload = appendRTGASMVarint(payload, bindings[i].Func)
		payload = appendRTGASMVarint(payload, bindings[i].Source)
		payload = appendRTGASMVarint(payload, bindings[i].Entry)
		payload = appendRTGASMVarint(payload, bindings[i].Mode)
		payload = appendRTGASMVarint(payload, bindings[i].Inputs)
		payload = appendRTGASMVarint(payload, bindings[i].Outputs)
		payload = appendRTGASMBytes(payload, code[i])
	}
	out := append([]byte(nil), data[:14]...)
	found := false
	for at := 14; at+6 <= len(data); {
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		next := at + 6 + size
		if size < 0 || next < at || next > len(data) || tag == TagRTGAssembly && found {
			return nil, false
		}
		if tag == TagRTGAssembly {
			out = appendRTGASMNode(out, tag, payload)
			found = true
		} else {
			out = append(out, data[at:next]...)
		}
		at = next
	}
	if !found {
		return nil, false
	}
	writeRTGASMUint32(out, 10, len(out)-14)
	return out, true
}

// ReadRTGAssemblyFragments validates the complete child stream. A valid unit
// without an assembly child returns empty tables and true.
func ReadRTGAssemblyFragments(data []byte) ([]RTGAssemblySource, []RTGAssemblyBinding, bool) {
	if !validUnitRoot(data) {
		return nil, nil, false
	}
	var payload []byte
	for at := 14; at < len(data); {
		if at+6 > len(data) {
			return nil, nil, false
		}
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		at += 6
		if size < 0 || size > len(data)-at {
			return nil, nil, false
		}
		if tag == TagRTGAssembly {
			if payload != nil {
				return nil, nil, false
			}
			payload = data[at : at+size]
		}
		at += size
	}
	if payload == nil {
		return nil, nil, true
	}

	r := rtgasmUnitReader{data: payload, ok: true}
	count := r.varint()
	version := 1
	if count == 0 {
		version = r.varint()
		count = r.varint()
		if version != 2 {
			return nil, nil, false
		}
	}
	if !r.ok || count < 0 || count > 8192 || count > len(r.data) {
		return nil, nil, false
	}
	sources := make([]RTGAssemblySource, count)
	for i := 0; i < len(sources); i++ {
		sources[i] = RTGAssemblySource{Path: string(r.bytes()), Source: r.bytes()}
		if len(sources[i].Source) > 16*1024*1024 {
			return nil, nil, false
		}
	}
	count = r.varint()
	if !r.ok || count < 0 || count > 8192 || count > len(r.data) {
		return nil, nil, false
	}
	bindings := make([]RTGAssemblyBinding, count)
	for i := 0; i < len(bindings); i++ {
		binding := RTGAssemblyBinding{Func: r.varint(), Source: r.varint(), Entry: r.varint()}
		if version == 2 {
			binding.Mode, binding.Inputs, binding.Outputs = r.varint(), r.varint(), r.varint()
		}
		binding.Code = r.bytes()
		if binding.Func < 0 || binding.Source < 0 || binding.Source >= len(sources) || binding.Entry < 0 || binding.Mode == 0 && (binding.Inputs != 0 || binding.Outputs != 0) || binding.Mode < 0 || binding.Mode > 1 || binding.Inputs < 0 || binding.Inputs > 6 || binding.Outputs < 0 || binding.Outputs > 1 {
			return nil, nil, false
		}
		bindings[i] = binding
	}
	return sources, bindings, r.ok && r.at == len(r.data)
}

type rtgasmUnitReader struct {
	data []byte
	at   int
	ok   bool
}

func (r *rtgasmUnitReader) varint() int {
	value := 0
	for shift := 0; r.at < len(r.data) && shift <= 28; shift += 7 {
		part := int(r.data[r.at])
		r.at++
		if shift == 28 && part > 7 {
			r.ok = false
			return 0
		}
		value |= (part & 127) << shift
		if part < 128 {
			return value
		}
	}
	r.ok = false
	return 0
}

func (r *rtgasmUnitReader) bytes() []byte {
	size := r.varint()
	if !r.ok || size < 0 || r.at+size < r.at || r.at+size > len(r.data) {
		r.ok = false
		return nil
	}
	value := r.data[r.at : r.at+size]
	r.at += size
	return value
}

func appendRTGASMBytes(out []byte, value []byte) []byte {
	out = appendRTGASMVarint(out, len(value))
	return append(out, value...)
}

func appendRTGASMVarint(out []byte, value int) []byte {
	for value >= 128 {
		out = append(out, byte(value)|128)
		value >>= 7
	}
	return append(out, byte(value))
}

func appendRTGASMNode(out []byte, tag int, payload []byte) []byte {
	start := len(out)
	out = append(out, byte(tag), byte(tag>>8), 0, 0, 0, 0)
	out = append(out, payload...)
	writeRTGASMUint32(out, start+2, len(payload))
	return out
}

func writeRTGASMUint32(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
	out[at+2] = byte(value >> 16)
	out[at+3] = byte(value >> 24)
}
