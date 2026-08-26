package backendjit

import "renvo.dev/internal/unit"

func attachRTGAssemblyCode(data []byte, code [][]byte) ([]byte, bool) {
	sources, bindings, ok := readRTGAssembly(data)
	if !ok || len(code) != len(bindings) {
		return nil, false
	}
	payload := appendRTGASMVarint(nil, len(sources))
	for i := 0; i < len(sources); i++ {
		payload = appendRTGASMBytes(payload, []byte(sources[i].Path))
		payload = appendRTGASMBytes(payload, sources[i].Source)
	}
	payload = appendRTGASMVarint(payload, len(bindings))
	for i := 0; i < len(bindings); i++ {
		if len(code[i]) == 0 {
			return nil, false
		}
		payload = appendRTGASMVarint(payload, bindings[i].Func)
		payload = appendRTGASMVarint(payload, bindings[i].Source)
		payload = appendRTGASMVarint(payload, bindings[i].Entry)
		payload = appendRTGASMBytes(payload, code[i])
	}
	out := append([]byte(nil), data[:14]...)
	found := false
	for at := 14; at+6 <= len(data); {
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		next := at + 6 + size
		if size < 0 || next < at || next > len(data) || tag == unit.TagRTGAssembly && found {
			return nil, false
		}
		if tag == unit.TagRTGAssembly {
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

func readRTGAssembly(data []byte) ([]unit.RTGAssemblySource, []unit.RTGAssemblyBinding, bool) {
	if len(data) < 14 || string(data[:4]) != unit.Magic {
		return nil, nil, false
	}
	for at := 14; at+6 <= len(data); {
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		at += 6
		if size < 0 || at+size < at || at+size > len(data) {
			return nil, nil, false
		}
		if tag == unit.TagRTGAssembly {
			r := rtgasmUnitReader{data: data[at : at+size], ok: true}
			sources := make([]unit.RTGAssemblySource, r.varint())
			for i := 0; i < len(sources); i++ {
				sources[i] = unit.RTGAssemblySource{Path: string(r.bytes()), Source: r.bytes()}
			}
			bindings := make([]unit.RTGAssemblyBinding, r.varint())
			for i := 0; i < len(bindings); i++ {
				bindings[i] = unit.RTGAssemblyBinding{Func: r.varint(), Source: r.varint(), Entry: r.varint(), Code: r.bytes()}
			}
			return sources, bindings, r.ok && r.at == len(r.data)
		}
		at += size
	}
	return nil, nil, false
}

type rtgasmUnitReader struct {
	data []byte
	at   int
	ok   bool
}

func (r *rtgasmUnitReader) varint() int {
	value := 0
	for shift := 0; r.at < len(r.data) && shift < 63; shift += 7 {
		part := int(r.data[r.at])
		r.at++
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
