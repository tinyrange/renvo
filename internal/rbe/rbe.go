// Package rbe parses Renvo Backend Enablement source files.
//
// An RBE is an RTG definition followed by zero or more standard-library
// additions.  Library paths are relative to Renvo's standard-library root:
//
//	@stdlib "syscall/syscall_unixv7_pdp11_renvo.go"
//	package syscall
//	...
//	@endstdlib
//
// Keeping the RTG portion byte-for-byte intact lets the existing definition
// parser, import resolver, and backend generator remain the authority for the
// machine definition language.
package rbe

import "strings"

const (
	sectionPrefix = "@stdlib"
	sectionEnd    = "@endstdlib"
	maxFiles      = 4096
	maxSource     = 32 << 20
)

type File struct {
	Path   string
	Source []byte
}

type Bundle struct {
	Definition []byte
	Files      []File
	Message    string
	Offset     int
	Ok         bool
}

// Parse accepts both plain RTG and RBE input. Plain RTG is returned unchanged,
// which keeps callers from needing a filename-based format switch.
func Parse(source []byte) Bundle {
	result := Bundle{Ok: true}
	if len(source) > maxSource {
		return fail(result, 0, "backend enablement exceeds the 32 MiB source limit")
	}
	definition := make([]byte, 0, len(source))
	at := 0
	for at < len(source) {
		lineStart := at
		lineEnd, next := line(source, at)
		text := strings.TrimSpace(string(source[lineStart:lineEnd]))
		if !strings.HasPrefix(text, sectionPrefix) {
			definition = append(definition, source[lineStart:next]...)
			at = next
			continue
		}
		path, ok := parseSectionHeader(text)
		if !ok {
			return fail(result, lineStart, `expected @stdlib "package/file"`)
		}
		if !ValidLibraryPath(path) {
			return fail(result, lineStart, "standard-library path must be a clean relative slash path")
		}
		for i := range result.Files {
			if result.Files[i].Path == path {
				return fail(result, lineStart, "duplicate standard-library path "+path)
			}
		}
		if len(result.Files) >= maxFiles {
			return fail(result, lineStart, "backend enablement exceeds the library file limit")
		}
		contentStart := next
		at = next
		foundEnd := false
		for at < len(source) {
			endStart := at
			endLine, endNext := line(source, at)
			if strings.TrimSpace(string(source[endStart:endLine])) == sectionEnd {
				content := append([]byte(nil), source[contentStart:endStart]...)
				result.Files = append(result.Files, File{Path: path, Source: content})
				at = endNext
				foundEnd = true
				break
			}
			at = endNext
		}
		if !foundEnd {
			return fail(result, lineStart, "unterminated standard-library section "+path)
		}
	}
	result.Definition = definition
	return result
}

func parseSectionHeader(text string) (string, bool) {
	rest := strings.TrimSpace(text[len(sectionPrefix):])
	if len(rest) < 2 || rest[0] != '"' || rest[len(rest)-1] != '"' {
		return "", false
	}
	path := rest[1 : len(rest)-1]
	return path, path != "" && !strings.Contains(path, "\"")
}

// ValidLibraryPath reports whether path is safe to overlay below the Renvo
// standard-library root. Prepared artifacts repeat this validation when they
// are decoded, since an RTGB may arrive without its original RBE source.
func ValidLibraryPath(path string) bool {
	if path == "" || path[0] == '/' || strings.Contains(path, "\\") || strings.Contains(path, "//") {
		return false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func line(source []byte, at int) (int, int) {
	end := at
	for end < len(source) && source[end] != '\n' {
		end++
	}
	next := end
	if next < len(source) {
		next++
	}
	return end, next
}

func fail(result Bundle, offset int, message string) Bundle {
	result.Ok = false
	result.Message = message
	result.Offset = offset
	return result
}
