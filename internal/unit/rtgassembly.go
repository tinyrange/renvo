package unit

// ReadRTGAssembly returns the optional source-preserving RTGASM table from a
// canonical unit. Older units simply report no table.
func ReadRTGAssembly(data []byte) ([]RTGAssemblySource, []RTGAssemblyBinding, bool) {
	if !validUnitRoot(data) {
		return nil, nil, false
	}
	for at := 14; at < len(data); {
		if at+6 > len(data) {
			return nil, nil, false
		}
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		at += 6
		if size < 0 || at+size < at || at+size > len(data) {
			return nil, nil, false
		}
		if tag == TagRTGAssembly {
			return decodeRTGAssemblyCore(data[at : at+size])
		}
		at += size
	}
	return nil, nil, false
}

type rtgAssemblyReader struct {
	data []byte
	at   int
	ok   bool
}

func decodeRTGAssemblyCore(data []byte) ([]RTGAssemblySource, []RTGAssemblyBinding, bool) {
	r := rtgAssemblyReader{data: data, ok: true}
	sourceCount := r.varint()
	if sourceCount < 0 || sourceCount > len(data) {
		return nil, nil, false
	}
	sources := make([]RTGAssemblySource, sourceCount)
	for i := 0; i < len(sources); i++ {
		sources[i].Path = string(r.bytes())
		sources[i].Source = r.bytes()
		if sources[i].Path == "" {
			r.ok = false
		}
	}
	bindingCount := r.varint()
	if bindingCount < 0 || bindingCount > len(data) {
		return nil, nil, false
	}
	bindings := make([]RTGAssemblyBinding, bindingCount)
	for i := 0; i < len(bindings); i++ {
		bindings[i] = RTGAssemblyBinding{Func: r.varint(), Source: r.varint(), Entry: r.varint()}
		bindings[i].Code = r.bytes()
		if bindings[i].Func < 0 || bindings[i].Source < 0 || bindings[i].Source >= len(sources) || bindings[i].Entry < 0 {
			r.ok = false
		}
	}
	return sources, bindings, r.ok && r.at == len(data)
}

// AttachRTGAssemblyCode replaces only the optional RTGASM table, retaining all
// other canonical-unit children and the original source bytes.
func AttachRTGAssemblyCode(data []byte, code [][]byte) ([]byte, bool) {
	sources, bindings, ok := ReadRTGAssembly(data)
	if !ok || len(code) != len(bindings) {
		return nil, false
	}
	for i := 0; i < len(bindings); i++ {
		if len(code[i]) == 0 {
			return nil, false
		}
		bindings[i].Code = append([]byte(nil), code[i]...)
	}
	payload := encodeRTGAssemblyCore(sources, bindings)
	out := make([]byte, 0, len(data)+len(payload))
	out = append(out, data[:14]...)
	found := false
	for at := 14; at < len(data); {
		if at+6 > len(data) {
			return nil, false
		}
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		next := at + 6 + size
		if size < 0 || next < at || next > len(data) {
			return nil, false
		}
		if tag == TagRTGAssembly {
			if found {
				return nil, false
			}
			out = appendNode(out, TagRTGAssembly, payload)
			found = true
		} else {
			out = append(out, data[at:next]...)
		}
		at = next
	}
	if !found {
		return nil, false
	}
	patchUint32Core(out, 10, len(out)-14)
	return out, true
}

func (r *rtgAssemblyReader) varint() int {
	if !r.ok {
		return -1
	}
	value := 0
	shift := 0
	for r.at < len(r.data) && shift < 63 {
		part := int(r.data[r.at])
		r.at++
		value |= (part & 127) << shift
		if part < 128 {
			return value
		}
		shift += 7
	}
	r.ok = false
	return -1
}

func (r *rtgAssemblyReader) bytes() []byte {
	size := r.varint()
	if size < 0 || r.at+size < r.at || r.at+size > len(r.data) {
		r.ok = false
		return nil
	}
	out := r.data[r.at : r.at+size]
	r.at += size
	return out
}
