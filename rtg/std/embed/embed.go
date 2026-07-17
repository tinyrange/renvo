package embed

type FS struct {
	archive string
}

type Entry struct {
	name string
	dir  bool
}

type embedError struct {
	name string
}

func (e embedError) Error() string {
	return "file not found: " + e.name
}

// NewFS is used by the RTG frontend when lowering an embed.FS variable. Go's
// frontend initializes its own embed.FS representation directly instead.
func NewFS(compressed string, size int) FS {
	archive, ok := decompressArchive(compressed, size)
	if !ok {
		return FS{}
	}
	return FS{archive: string(archive)}
}

func (f FS) ReadFile(name string) ([]byte, error) {
	data, ok := f.ReadFileOK(name)
	if !ok {
		return nil, embedError{name: name}
	}
	return data, nil
}

func (f FS) ReadDir(name string) ([]Entry, error) {
	entries, ok := f.ReadDirOK(name)
	if !ok {
		return nil, embedError{name: name}
	}
	return entries, nil
}

func (f FS) ReadFileOK(name string) ([]byte, bool) {
	if !validPath(name) || name == "." {
		return nil, false
	}
	count, pos, ok := archiveHeader(f.archive)
	if !ok {
		return nil, false
	}
	for i := 0; i < count; i++ {
		fileName, dataStart, dataEnd, next, entryOK := archiveEntry(f.archive, pos)
		if !entryOK {
			return nil, false
		}
		if fileName == name {
			out := make([]byte, 0, dataEnd-dataStart)
			for j := dataStart; j < dataEnd; j++ {
				out = append(out, f.archive[j])
			}
			return out, true
		}
		pos = next
	}
	return nil, false
}

func (f FS) ReadDirOK(name string) ([]Entry, bool) {
	if !validPath(name) {
		return nil, false
	}
	prefix := ""
	if name != "." {
		prefix = name + "/"
	}
	count, pos, ok := archiveHeader(f.archive)
	if !ok {
		return nil, false
	}
	var out []Entry
	found := name == "."
	for i := 0; i < count; i++ {
		fileName, _, _, next, entryOK := archiveEntry(f.archive, pos)
		if !entryOK {
			return nil, false
		}
		pos = next
		if !hasPrefix(fileName, prefix) {
			continue
		}
		rest := fileName[len(prefix):]
		if rest == "" {
			continue
		}
		found = true
		end := 0
		for end < len(rest) && rest[end] != '/' {
			end++
		}
		entryName := rest[:end]
		isDir := end < len(rest)
		if !hasEntry(out, entryName) {
			out = append(out, Entry{name: entryName, dir: isDir})
		}
	}
	if !found {
		return nil, false
	}
	sortEntries(out)
	return out, true
}

func (e Entry) Name() string {
	return e.name
}

func (e Entry) IsDir() bool {
	return e.dir
}

func archiveHeader(archive string) (int, int, bool) {
	if len(archive) < 4 {
		return 0, 0, false
	}
	return archiveUint32(archive, 0), 4, true
}

func archiveEntry(archive string, pos int) (string, int, int, int, bool) {
	if pos < 0 || pos+8 > len(archive) {
		return "", 0, 0, pos, false
	}
	nameSize := archiveUint32(archive, pos)
	dataSize := archiveUint32(archive, pos+4)
	nameStart := pos + 8
	dataStart := nameStart + nameSize
	dataEnd := dataStart + dataSize
	if nameSize < 0 || dataSize < 0 || dataStart < nameStart || dataEnd < dataStart || dataEnd > len(archive) {
		return "", 0, 0, pos, false
	}
	return archive[nameStart:dataStart], dataStart, dataEnd, dataEnd, true
}

func archiveUint32(data string, pos int) int {
	return int(data[pos]) | int(data[pos+1])<<8 | int(data[pos+2])<<16 | int(data[pos+3])<<24
}

func validPath(name string) bool {
	if name == "." {
		return true
	}
	if name == "" || name[0] == '/' || name[len(name)-1] == '/' {
		return false
	}
	start := 0
	for i := 0; i <= len(name); i++ {
		if i < len(name) && name[i] != '/' {
			continue
		}
		if i == start || i-start == 1 && name[start] == '.' || i-start == 2 && name[start] == '.' && name[start+1] == '.' {
			return false
		}
		start = i + 1
	}
	return true
}

func hasPrefix(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func hasEntry(entries []Entry, name string) bool {
	for i := 0; i < len(entries); i++ {
		if entries[i].name == name {
			return true
		}
	}
	return false
}

func sortEntries(entries []Entry) {
	for i := 1; i < len(entries); i++ {
		item := entries[i]
		j := i - 1
		for j >= 0 && entryNameAfter(entries[j].name, item.name) {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = item
	}
}

func entryNameAfter(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] > right[i] {
			return true
		}
		if left[i] < right[i] {
			return false
		}
	}
	return len(left) > len(right)
}

func decompressArchive(compressed string, size int) ([]byte, bool) {
	if size < 0 {
		return nil, false
	}
	out := make([]byte, 0, size)
	pos := 0
	for pos < len(compressed) && len(out) < size {
		flags := compressed[pos]
		pos++
		for bit := 0; bit < 8 && len(out) < size; bit++ {
			if flags&(1<<bit) != 0 {
				if pos >= len(compressed) {
					return nil, false
				}
				out = append(out, compressed[pos])
				pos++
				continue
			}
			if pos+1 >= len(compressed) {
				return nil, false
			}
			pair := int(compressed[pos])<<8 | int(compressed[pos+1])
			pos += 2
			distance := (pair >> 4) + 1
			length := pair&15 + 3
			if distance > len(out) || len(out)+length > size {
				return nil, false
			}
			for i := 0; i < length; i++ {
				out = append(out, out[len(out)-distance])
			}
		}
	}
	if len(out) != size || pos != len(compressed) {
		return nil, false
	}
	return out, true
}
