//go:build renvo && windows

package os

const renvoWindowsFindDataSize = 320
const renvoWindowsFindNameOffset = 44
const renvoWindowsFindNameLimit = 304
const renvoWindowsFileAttributeDirectory = 16

// renvo:linkstatic kernel32.dll,FindFirstFileA
func renvoWindowsFindFirstFile(path *byte, data *byte) int { return -1 }

// renvo:linkstatic kernel32.dll,FindNextFileA
func renvoWindowsFindNextFile(handle int, data *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,FindClose
func renvoWindowsFindClose(handle int) int { return 0 }

func ReadDir(path string) ([]DirEntry, error) {
	pattern := renvoWindowsPathBytes(renvoWindowsJoinGlob(path))
	data := make([]byte, renvoWindowsFindDataSize)
	handle := renvoWindowsFindFirstFile(&pattern[0], &data[0])
	if handle == -1 {
		return nil, errIO()
	}
	var out []DirEntry
	for {
		nameEnd := renvoWindowsFindNameOffset
		for nameEnd < renvoWindowsFindNameLimit && data[nameEnd] != 0 {
			nameEnd++
		}
		if nameEnd > renvoWindowsFindNameOffset && !dirNameIsDot(data, renvoWindowsFindNameOffset, nameEnd) {
			attributes := int(data[0]) | int(data[1])<<8 | int(data[2])<<16 | int(data[3])<<24
			out = append(out, DirEntry{
				name:  string(data[renvoWindowsFindNameOffset:nameEnd]),
				isDir: attributes&renvoWindowsFileAttributeDirectory != 0,
			})
		}
		if renvoWindowsFindNextFile(handle, &data[0]) == 0 {
			break
		}
	}
	renvoWindowsFindClose(handle)
	sortDirEntries(out)
	return out, nil
}

func renvoWindowsJoinGlob(path string) string {
	if path == "" || path == "." {
		return "*"
	}
	last := path[len(path)-1]
	if last == '/' || last == '\\' {
		return path + "*"
	}
	return path + "/*"
}

func renvoWindowsPathBytes(path string) []byte {
	out := make([]byte, 0, len(path)+1)
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' {
			c = '\\'
		}
		out = append(out, c)
	}
	return append(out, 0)
}
