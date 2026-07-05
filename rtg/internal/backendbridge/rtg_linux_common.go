//go:build rtg && linux && (aarch64 || arm64)

package backendbridge

func backendPath(env []string) string {
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) >= 12 &&
			item[0] == 'R' && item[1] == 'T' && item[2] == 'G' && item[3] == '_' &&
			item[4] == 'B' && item[5] == 'A' && item[6] == 'C' && item[7] == 'K' &&
			item[8] == 'E' && item[9] == 'N' && item[10] == 'D' && item[11] == '=' {
			return item[12:]
		}
	}
	return ""
}

func tempPath(prefix string, pid int, suffix string) string {
	out := []byte(prefix)
	out = appendInt(out, pid)
	for i := 0; i < len(suffix); i++ {
		out = append(out, suffix[i])
	}
	return string(out)
}

func appendInt(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
	}
	if value < 0 {
		out = append(out, '-')
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value = value / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return out
}

func writeFile(path string, data []byte, mode int) bool {
	fd := open(cstring(path), O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		return false
	}
	if write(fd, data, -1) != len(data) {
		close(fd)
		return false
	}
	if chmod(fd, mode) != 0 {
		close(fd)
		return false
	}
	close(fd)
	return true
}

func chmodOutput(path string) bool {
	if path == "-" {
		return true
	}
	fd := open(cstring(path), O_RDWR)
	if fd < 0 {
		return false
	}
	ok := chmod(fd, 493) == 0
	close(fd)
	return ok
}

func backendScript(backend string, targetName string, outputPath string, unitPath string, stripSymbols bool) []byte {
	out := []byte("#!/bin/sh\nexec ")
	out = appendString(out, backend)
	out = appendString(out, " -t ")
	out = appendString(out, targetName)
	if stripSymbols {
		out = appendString(out, " -s")
	}
	out = appendString(out, " -o ")
	out = appendString(out, outputPath)
	out = append(out, ' ')
	out = appendString(out, unitPath)
	out = append(out, '\n')
	return out
}

func appendString(out []byte, value string) []byte {
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return out
}

func cstring(value string) string {
	var out []byte
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	out = append(out, 0)
	return string(out)
}
