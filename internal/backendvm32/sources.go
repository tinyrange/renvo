//go:build !renvo

package backendvm32

func CompilerSource(index int) (string, string, bool) {
	name := compilerSourceName(index)
	if name == "" {
		return "", "", false
	}
	text, ok := decompressCompilerSource(index, compilerSourceSize(index))
	return name, text, ok
}

func decompressCompilerSource(index int, size int) (string, bool) {
	if size < 0 {
		return "", false
	}
	source := make([]byte, 0, size)
	for chunkIndex := 0; chunkIndex < compilerSourceChunkCount(index); chunkIndex++ {
		compressed, decoded := decodeCompilerSourceChunk(compilerSourceChunk(index, chunkIndex))
		if !decoded {
			return "", false
		}
		for at := 0; at < len(compressed); {
			control := int(compressed[at])
			at++
			if control < 128 {
				count := control + 1
				if at+count > len(compressed) || len(source)+count > size {
					return "", false
				}
				source = append(source, compressed[at:at+count]...)
				at += count
				continue
			}
			if at+2 > len(compressed) {
				return "", false
			}
			count := control&127 + 3
			distance := int(compressed[at]) | int(compressed[at+1])<<8
			at += 2
			if distance <= 0 || distance > len(source) || len(source)+count > size {
				return "", false
			}
			for i := 0; i < count; i++ {
				source = append(source, source[len(source)-distance])
			}
		}
	}
	if len(source) != size {
		return "", false
	}
	return string(source), true
}

func decodeCompilerSourceChunk(encoded string) ([]byte, bool) {
	if len(encoded)%4 == 1 {
		return nil, false
	}
	out := make([]byte, 0, len(encoded)*3/4)
	value := 0
	bits := 0
	for i := 0; i < len(encoded); i++ {
		ch := encoded[i]
		digit := -1
		if ch >= 'A' && ch <= 'Z' {
			digit = int(ch - 'A')
		} else if ch >= 'a' && ch <= 'z' {
			digit = int(ch-'a') + 26
		} else if ch >= '0' && ch <= '9' {
			digit = int(ch-'0') + 52
		} else if ch == '+' {
			digit = 62
		} else if ch == '/' {
			digit = 63
		} else {
			return nil, false
		}
		value = value<<6 | digit
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(value>>bits))
			if bits == 0 {
				value = 0
			} else {
				value &= 1<<bits - 1
			}
		}
	}
	return out, value == 0
}
