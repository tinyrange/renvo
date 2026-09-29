package elflink

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// ReadArchive decodes ordinary Unix ar, including GNU and BSD long names.
// Symbol-index members are metadata, never linker inputs.
func ReadArchive(data []byte) ([]Input, Error) {
	if !bytes.HasPrefix(data, []byte("!<arch>\n")) {
		return nil, Error{Message: "invalid archive magic"}
	}
	var members []Input
	var names []byte
	for at := 8; at < len(data); {
		if len(data)-at < 60 || string(data[at+58:at+60]) != "`\n" {
			return nil, Error{Message: "invalid archive member header"}
		}
		h := data[at : at+60]
		size, e := strconv.ParseUint(strings.TrimSpace(string(h[48:58])), 10, 64)
		at += 60
		if e != nil || size > uint64(len(data)-at) {
			return nil, Error{Message: "invalid archive member size"}
		}
		body := data[at : at+int(size)]
		at += int(size)
		if size&1 != 0 {
			if at >= len(data) {
				return nil, Error{Message: "missing archive padding"}
			}
			at++
		}
		name := strings.TrimSpace(string(h[:16]))
		if name == "//" {
			names = body
			continue
		}
		if name == "/" || name == "/SYM64/" {
			continue
		}
		if strings.HasPrefix(name, "#1/") {
			n, e := strconv.Atoi(name[3:])
			if e != nil || n < 0 || n > len(body) {
				return nil, Error{Message: "invalid BSD archive name"}
			}
			name = strings.TrimRight(string(body[:n]), "\x00")
			body = body[n:]
		} else if strings.HasPrefix(name, "/") {
			n, e := strconv.Atoi(name[1:])
			if e != nil || n < 0 || n >= len(names) {
				return nil, Error{Message: "invalid GNU archive name"}
			}
			end := bytes.IndexByte(names[n:], '\n')
			if end < 0 {
				return nil, Error{Message: "unterminated GNU archive name"}
			}
			name = strings.TrimSuffix(string(names[n:n+end]), "/")
		} else {
			name = strings.TrimSuffix(name, "/")
		}
		if strings.HasPrefix(name, "__.SYMDEF") {
			continue
		}
		if name == "" {
			return nil, Error{Message: "empty archive member name"}
		}
		members = append(members, Input{Name: name, Data: body})
	}
	return members, Error{}
}
func DefinedSymbols(input Input) ([]string, Error) {
	obj, e := parseObject(input)
	if e.Message != "" {
		return nil, e
	}
	var names []string
	for _, table := range obj.symbols {
		for _, s := range table {
			binding := s.info >> 4
			if s.name != "" && s.section != shnUndefined && (binding == stbGlobal || binding == stbWeak) {
				names = appendUniqueSymbol(names, s.name)
			}
		}
	}
	return names, Error{}
}
func archiveHeader(name string, size int) []byte {
	return []byte(fmt.Sprintf("%-16s%-12d%-6d%-6d%-8s%-10d`\n", name, 0, 0, 0, "100644", size))
}
func appendArchiveMember(dst []byte, name string, data []byte) []byte {
	dst = append(dst, archiveHeader(name, len(data))...)
	dst = append(dst, data...)
	if len(data)&1 != 0 {
		dst = append(dst, '\n')
	}
	return dst
}
func appendBig32(dst []byte, v uint32) []byte {
	return append(dst, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// WriteArchive emits deterministic ar with a real GNU symbol index.
func WriteArchive(members []Input) ([]byte, Error) {
	var indexNames []string
	var indexMembers []int
	encoded := make([]Input, len(members))
	for i, m := range members {
		if m.Name == "" || strings.ContainsAny(m.Name, "\n\x00") {
			return nil, Error{Input: m.Name, Message: "invalid archive name"}
		}
		encoded[i] = m
		if len(m.Name) > 15 || strings.Contains(m.Name, " ") {
			encoded[i] = Input{Name: fmt.Sprintf("#1/%d", len(m.Name)), Data: append(append([]byte(nil), m.Name...), m.Data...)}
		} else {
			encoded[i].Name = m.Name + "/"
		}
		if bytes.HasPrefix(m.Data, []byte("\x7fELF")) {
			symbols, e := DefinedSymbols(m)
			if e.Message != "" {
				return nil, e
			}
			for _, s := range symbols {
				indexNames = append(indexNames, s)
				indexMembers = append(indexMembers, i)
			}
		}
	}
	size := 4 + 4*len(indexNames)
	for _, n := range indexNames {
		size += len(n) + 1
	}
	offsets := make([]uint32, len(members))
	at := uint64(8 + 60 + size + size%2)
	for i, m := range encoded {
		if at > 0xffffffff {
			return nil, Error{Message: "archive exceeds 32-bit symbol index"}
		}
		offsets[i] = uint32(at)
		at += uint64(60 + len(m.Data) + len(m.Data)%2)
	}
	index := appendBig32(nil, uint32(len(indexNames)))
	for _, i := range indexMembers {
		index = appendBig32(index, offsets[i])
	}
	for _, n := range indexNames {
		index = append(index, n...)
		index = append(index, 0)
	}
	out := appendArchiveMember([]byte("!<arch>\n"), "/", index)
	for _, m := range encoded {
		out = appendArchiveMember(out, m.Name, m.Data)
	}
	return out, Error{}
}

// ExtractArchive performs a fixed-point search at this archive's position in
// the command line. Members unrelated to unresolved symbols stay unlinked.
func ExtractArchive(inputs []Input, archive Input) ([]Input, Error) {
	members, e := ReadArchive(archive.Data)
	if e.Message != "" {
		e.Input = archive.Name
		return nil, e
	}
	defs := make([][]string, len(members))
	used := make([]bool, len(members))
	for i, m := range members {
		defs[i], e = DefinedSymbols(m)
		if e.Message != "" {
			return nil, e
		}
	}
	for {
		needed, e := Unresolved(inputs)
		if e.Message != "" {
			return nil, e
		}
		if len(inputs) == 0 {
			needed = append(needed, "main")
		}
		progress := false
		for i, m := range members {
			if used[i] {
				continue
			}
			wanted := false
			for _, n := range needed {
				for _, d := range defs[i] {
					if n == d {
						wanted = true
					}
				}
			}
			if !wanted {
				continue
			}
			m.Name = archive.Name + "(" + m.Name + ")"
			inputs = append(inputs, m)
			used[i] = true
			progress = true
			needed, e = Unresolved(inputs)
			if e.Message != "" {
				return nil, e
			}
		}
		if !progress {
			return inputs, Error{}
		}
	}
}
