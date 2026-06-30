//go:build rtg

package os

import "runtime"

const O_RDONLY = 0
const O_WRONLY = 1
const O_RDWR = 2
const O_CREATE = 64
const O_TRUNC = 512

const Stdin = 0
const Stdout = 1
const Stderr = 2

var Args []string

type FileInfo struct {
	name  string
	isDir bool
}

func (info FileInfo) IsDir() bool {
	return info.isDir
}

type DirEntry struct {
	name  string
	isDir bool
}

func (entry DirEntry) Name() string {
	return entry.name
}

func (entry DirEntry) IsDir() bool {
	return entry.isDir
}

func Open(path string, flags int) int {
	return open(cString(path), flags)
}

func Close(fd int) int {
	return close(fd)
}

func Read(fd int, buf []byte, off int64) int {
	return read(fd, buf, off)
}

func Write(fd int, buf []byte, off int64) int {
	return write(fd, buf, off)
}

func Chmod(fd int, mode int) int {
	return chmod(fd, mode)
}

func Exit(code int) {
}

func Getenv(name string) string {
	return ""
}

func Getwd() (string, error) {
	return ".", nil
}

func Stat(path string) (FileInfo, error) {
	fd := Open(path, O_RDONLY)
	if fd < 0 {
		return FileInfo{}, statError("open failed: " + path)
	}
	buf := make([]byte, 256)
	n := getdents64(fd, buf)
	Close(fd)
	return FileInfo{name: path, isDir: n >= 0}, nil
}

func IsNotExist(err error) bool {
	return err != nil
}

func ReadFile(path string) ([]byte, error) {
	size, err := fileSize(path)
	if err != nil {
		return nil, err
	}
	fd := Open(path, O_RDONLY)
	if fd < 0 {
		return nil, statError("open failed: " + path)
	}
	out := make([]byte, size)
	off := 0
	for off < size {
		n := Read(fd, out[off:], int64(off))
		if n < 0 {
			Close(fd)
			return nil, statError("read failed")
		}
		if n == 0 {
			Close(fd)
			return nil, statError("short read")
		}
		off = off + n
	}
	Close(fd)
	return out, nil
}

func fileSize(path string) (int, error) {
	fd := Open(path, O_RDONLY)
	if fd < 0 {
		return 0, statError("open failed: " + path)
	}
	buf := make([]byte, 4096)
	size := 0
	for {
		n := Read(fd, buf, int64(size))
		if n < 0 {
			Close(fd)
			return 0, statError("read failed")
		}
		if n == 0 {
			break
		}
		size = size + n
		if size > 4194304 {
			Close(fd)
			return 0, statError("file too large")
		}
		if n < len(buf) {
			break
		}
	}
	Close(fd)
	return size, nil
}

func WriteFile(path string, data []byte, mode int) error {
	fd := Open(path, O_WRONLY|O_CREATE|O_TRUNC)
	if fd < 0 {
		return statError("open failed: " + path)
	}
	written := Write(fd, data, 0)
	if written != len(data) {
		Close(fd)
		return statError("write failed")
	}
	if mode != 0 {
		Chmod(fd, mode)
	}
	if Close(fd) != 0 {
		return statError("close failed")
	}
	return nil
}

func ReadDir(path string) ([]DirEntry, error) {
	fd := Open(path, O_RDONLY)
	if fd < 0 {
		return nil, statError("open failed: " + path)
	}
	var out []DirEntry
	for {
		buf := make([]byte, 8192)
		n := getdents64(fd, buf)
		if n < 0 {
			Close(fd)
			return nil, statError("readdir failed: " + path)
		}
		if n == 0 {
			break
		}
		pos := 0
		for pos+19 < n {
			lenLow := buf[pos+16]
			lenHigh := buf[pos+17]
			reclen := int(lenLow) + int(lenHigh)*256
			if reclen <= 19 {
				Close(fd)
				return nil, statError("invalid directory entry: " + path)
			}
			if pos+reclen > n {
				Close(fd)
				return nil, statError("invalid directory entry: " + path)
			}
			nameStart := pos + 19
			nameEnd := nameStart
			for nameEnd < pos+reclen && buf[nameEnd] != 0 {
				nameEnd = nameEnd + 1
			}
			name := stringFromBytes(buf, nameStart, nameEnd)
			if name != "." {
				if name != ".." {
					if name != "" {
						entryType := buf[pos+18]
						out = append(out, DirEntry{name: name, isDir: entryType == 4})
					}
				}
			}
			pos = pos + reclen
		}
	}
	Close(fd)
	return out, nil
}

func MkdirAll(path string, mode int) error {
	if path == "" {
		return nil
	}
	start := 0
	if path[0] == '/' {
		start = 1
	}
	for i := start; i <= len(path); i++ {
		if i < len(path) && path[i] != '/' {
			continue
		}
		if i == 0 {
			continue
		}
		part := path[:i]
		if part == "" {
			continue
		}
		mkdir(part, mode)
	}
	return nil
}

type statError string

func (err statError) Error() string {
	return string(err)
}

func getdents64(fd int, buf []byte) int {
	if runtime.GOOS != "linux" {
		return -1
	}
	num := 217
	if runtime.GOARCH == "386" {
		num = 220
	}
	if runtime.GOARCH == "arm" {
		num = 217
	}
	if runtime.GOARCH == "arm64" {
		num = 61
	}
	return syscall(num, fd, buf, len(buf))
}

func mkdir(path string, mode int) int {
	if runtime.GOOS != "linux" {
		return -1
	}
	return syscall(83, cString(path), mode)
}

func stringFromBytes(buf []byte, start int, end int) string {
	var out []byte
	for i := start; i < end; i++ {
		out = append(out, buf[i])
	}
	return string(out)
}

func cString(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return string(out)
}
