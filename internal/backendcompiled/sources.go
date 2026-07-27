//go:build !renvo

package backendcompiled

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
		compressed := []byte(compilerSourceChunk(index, chunkIndex))
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
