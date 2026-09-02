//go:build renvo && msdos

package os

import "io"

const O_RDONLY = 0
const O_WRONLY = 1
const O_RDWR = 2

var Args []string
var Stdin = &File{fd: 0}
var Stdout = &File{fd: 1}
var Stderr = &File{fd: 2}

var processEnv []string

type File struct {
	fd int
}

type filePosition struct {
	offset    int64
	size      int64
	active    bool
	sizeKnown bool
}

var filePositions []filePosition

func beginFilePosition(fd int, sizeKnown bool) {
	if fd < 3 {
		return
	}
	if len(filePositions) == 0 {
		filePositions = make([]filePosition, 16)
	}
	if fd >= len(filePositions) {
		length := len(filePositions)
		for length <= fd {
			length *= 2
		}
		grown := make([]filePosition, length)
		copy(grown, filePositions)
		filePositions = grown
	}
	filePositions[fd] = filePosition{active: true, sizeKnown: sizeKnown}
}

func positionForFile(fd int) *filePosition {
	if fd < 3 || fd >= len(filePositions) || !filePositions[fd].active {
		return nil
	}
	return &filePositions[fd]
}

type DirEntry struct {
	name  string
	isDir bool
}

type osError struct {
	text string
}

var ioErrorValue = osError{text: "I/O error"}
var invalidErrorValue = osError{text: "invalid argument"}

func (e *osError) Error() string {
	if e == nil {
		return ""
	}
	return e.text
}

func errIO() *osError {
	return &ioErrorValue
}

func errEOF() error {
	return io.EOF
}

func errInvalid() *osError {
	return &invalidErrorValue
}

func Environ() []string {
	out := make([]string, len(processEnv))
	for i := 0; i < len(processEnv); i++ {
		out[i] = processEnv[i]
	}
	return out
}

func renvo_runtime_SetProcess(args []string, env []string) {
	Args = args
	processEnv = env
}

func renvo_runtime_Exit(code int) {}

func Exit(code int) { renvo_runtime_Exit(code) }

func ReadFile(name string) ([]byte, error) {
	fd := open(renvoPathCString(name), O_RDONLY)
	if fd < 0 {
		return nil, errIO()
	}
	out := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n := read(fd, buf, -1)
		if n < 0 {
			close(fd)
			return nil, errIO()
		}
		if n == 0 {
			break
		}
		out = append(out, buf[:n]...)
	}
	close(fd)
	return out, nil
}

func WriteFile(name string, data []byte, perm FileMode) error {
	fd := open(renvoPathCString(name), O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		return errIO()
	}
	written := 0
	for written < len(data) {
		n := write(fd, data[written:], -1)
		if n <= 0 {
			close(fd)
			return errIO()
		}
		written += n
	}
	if chmod(fd, int(perm)) != 0 {
		close(fd)
		return errIO()
	}
	if close(fd) != 0 {
		return errIO()
	}
	return nil
}

func Open(name string) (*File, error) {
	return OpenFile(name, O_RDONLY, 0)
}

func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	fd := open(renvoPathCString(name), flag)
	if fd < 0 {
		return nil, errIO()
	}
	if flag&O_CREATE != 0 && chmod(fd, int(perm)) != 0 {
		close(fd)
		return nil, errIO()
	}
	beginFilePosition(fd, flag&O_TRUNC != 0)
	return &File{fd: fd}, nil
}

func Create(name string) (*File, error) {
	return OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)
}

func (f *File) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if f.fd <= 2 {
		n := read(f.fd, p, -1)
		if n < 0 {
			return 0, errIO()
		}
		if n == 0 {
			return 0, errEOF()
		}
		return n, nil
	}
	offset := -1
	position := positionForFile(f.fd)
	if position != nil {
		var ok bool
		offset, ok = fileOffsetInt(position.offset)
		if !ok {
			return 0, errInvalid()
		}
	}
	n := read(f.fd, p, offset)
	if n < 0 {
		return 0, errIO()
	}
	if position != nil {
		position.offset += int64(n)
	}
	if n == 0 {
		return 0, errEOF()
	}
	return n, nil
}

func (f *File) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if f.fd <= 2 {
		n := write(f.fd, p, -1)
		if n < 0 || n != len(p) {
			return n, errIO()
		}
		return n, nil
	}
	offset := -1
	position := positionForFile(f.fd)
	if position != nil {
		var ok bool
		offset, ok = fileOffsetInt(position.offset)
		if !ok {
			return 0, errInvalid()
		}
	}
	n := write(f.fd, p, offset)
	if n < 0 {
		return 0, errIO()
	}
	if position != nil {
		position.offset += int64(n)
		if !position.sizeKnown || position.offset > position.size {
			position.size = position.offset
			position.sizeKnown = true
		}
	}
	if n != len(p) {
		return n, errIO()
	}
	return n, nil
}

func fileOffsetInt(offset int64) (int, bool) {
	value := int(offset)
	return value, int64(value) == offset
}

func (f *File) discoverSize(position *filePosition) (int64, error) {
	if position.sizeKnown {
		return position.size, nil
	}
	buf := make([]byte, 4096)
	var offset int64
	for {
		at, ok := fileOffsetInt(offset)
		if !ok {
			return 0, errInvalid()
		}
		n := read(f.fd, buf, at)
		if n < 0 {
			return 0, errIO()
		}
		offset += int64(n)
		if n < len(buf) {
			position.size = offset
			position.sizeKnown = true
			return offset, nil
		}
	}
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	position := positionForFile(f.fd)
	if position == nil {
		return 0, errInvalid()
	}
	base := int64(0)
	if whence == 1 {
		base = position.offset
	} else if whence == 2 {
		size, err := f.discoverSize(position)
		if err != nil {
			return 0, err
		}
		base = size
	} else if whence != 0 {
		return 0, errInvalid()
	}
	next := base + offset
	if next < 0 {
		return 0, errInvalid()
	}
	if _, ok := fileOffsetInt(next); !ok {
		return 0, errInvalid()
	}
	position.offset = next
	return next, nil
}

func (f *File) Close() error {
	if close(f.fd) != 0 {
		return errIO()
	}
	if position := positionForFile(f.fd); position != nil {
		*position = filePosition{}
	}
	return nil
}

func (d DirEntry) Name() string {
	return d.name
}

func (d DirEntry) IsDir() bool {
	return d.isDir
}

func dirNameIsDot(buf []byte, start int, end int) bool {
	if end-start == 1 && buf[start] == '.' {
		return true
	}
	return end-start == 2 && buf[start] == '.' && buf[start+1] == '.'
}

func sortDirEntries(entries []DirEntry) {
	for i := 1; i < len(entries); i++ {
		item := entries[i]
		j := i - 1
		for j >= 0 && entries[j].name > item.name {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = item
	}
}
