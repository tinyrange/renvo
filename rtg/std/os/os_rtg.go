//go:build rtg

package os

const O_RDONLY = 0
const O_RDWR = 2
const O_CREATE = 64
const O_TRUNC = 512

type FileMode int

var Args []string

type File struct {
	fd int
}

type DirEntry struct {
	name  string
	isDir bool
}

type osError struct {
	text string
}

func (e *osError) Error() string {
	if e == nil {
		return ""
	}
	return e.text
}

func errIO() *osError {
	return nil
}

func Environ() []string {
	return nil
}

func Exit(code int) {}

func Getwd() (string, *osError) {
	return ".", nil
}

func ReadFile(name string) ([]byte, *osError) {
	fd := open(rtgPathCString(name), O_RDONLY)
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

func WriteFile(name string, data []byte, perm FileMode) *osError {
	fd := open(rtgPathCString(name), O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		return errIO()
	}
	if write(fd, data, -1) != len(data) {
		close(fd)
		return errIO()
	}
	if chmod(fd, int(perm)) != 0 {
		close(fd)
		return errIO()
	}
	close(fd)
	return nil
}

func Open(name string) (File, *osError) {
	fd := open(rtgPathCString(name), O_RDONLY)
	if fd < 0 {
		return File{}, errIO()
	}
	return File{fd: fd}, nil
}

func Create(name string) (File, *osError) {
	fd := open(rtgPathCString(name), O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		return File{}, errIO()
	}
	return File{fd: fd}, nil
}

func (f File) Read(p []byte) (int, *osError) {
	n := read(f.fd, p, -1)
	if n < 0 {
		return 0, errIO()
	}
	return n, nil
}

func (f File) Write(p []byte) (int, *osError) {
	n := write(f.fd, p, -1)
	if n < 0 {
		return 0, errIO()
	}
	return n, nil
}

func (f File) Close() *osError {
	if close(f.fd) != 0 {
		return errIO()
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
