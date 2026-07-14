//go:build rtg && windows

package driver

import "j5.nz/rtg/rtg/internal/load"

const rtgWindowsFindDataSize = 320
const rtgWindowsFindNameOffset = 44
const rtgWindowsFindNameLimit = 304
const rtgWindowsFileAttributeDirectory = 16

// rtg:linkstatic kernel32.dll,FindFirstFileA
func rtgWindowsFindFirstFile(path *byte, data *byte) int { return -1 }

// rtg:linkstatic kernel32.dll,FindNextFileA
func rtgWindowsFindNextFile(handle int, data *byte) int { return 0 }

// rtg:linkstatic kernel32.dll,FindClose
func rtgWindowsFindClose(handle int) int { return 0 }

func rtgReadDirNative(path string) ([]DirEntry, bool) {
	pattern := rtgWindowsPathBytes(load.JoinPath(path, "*"))
	data := make([]byte, rtgWindowsFindDataSize)
	handle := rtgWindowsFindFirstFile(&pattern[0], &data[0])
	if handle == -1 {
		return nil, false
	}
	out := make([]DirEntry, 0, 32)
	for {
		nameEnd := rtgWindowsFindNameOffset
		for nameEnd < rtgWindowsFindNameLimit && data[nameEnd] != 0 {
			nameEnd++
		}
		if nameEnd > rtgWindowsFindNameOffset && !rtgDirNameIsDot(data, rtgWindowsFindNameOffset, nameEnd) {
			attributes := int(data[0]) | int(data[1])<<8 | int(data[2])<<16 | int(data[3])<<24
			out = append(out, DirEntry{
				Name:  string(data[rtgWindowsFindNameOffset:nameEnd]),
				IsDir: attributes&rtgWindowsFileAttributeDirectory != 0,
			})
		}
		if rtgWindowsFindNextFile(handle, &data[0]) == 0 {
			break
		}
	}
	rtgWindowsFindClose(handle)
	sortDirEntries(out)
	return out, true
}

func rtgWindowsPathBytes(path string) []byte {
	out := make([]byte, 0, len(path)+1)
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' {
			c = '\\'
		}
		out = append(out, c)
	}
	out = append(out, 0)
	return out
}
