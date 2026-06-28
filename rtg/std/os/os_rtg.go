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
	fd := Open(path, O_RDONLY)
	if fd < 0 {
		return nil, statError("open failed: " + path)
	}
	out := make([]byte, 0, 4194304)
	off := int64(0)
	for {
		buf := make([]byte, 4096)
		n := Read(fd, buf, off)
		if n < 0 {
			Close(fd)
			return nil, statError("read failed")
		}
		if n == 0 {
			break
		}
		if len(out)+n > 4194304 {
			Close(fd)
			return nil, statError("file too large")
		}
		for i := 0; i < n; i++ {
			out = append(out, buf[i])
		}
		off = off + int64(n)
		if n < len(buf) {
			break
		}
	}
	Close(fd)
	return out, nil
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
			reclen := int(buf[pos+16]) + int(buf[pos+17])*256
			if reclen <= 19 || pos+reclen > n {
				Close(fd)
				return nil, statError("invalid directory entry: " + path)
			}
			nameStart := pos + 19
			nameEnd := nameStart
			for nameEnd < pos+reclen && buf[nameEnd] != 0 {
				nameEnd = nameEnd + 1
			}
			name := stringFromBytes(buf, nameStart, nameEnd)
			if name != "." && name != ".." && name != "" {
				out = append(out, DirEntry{name: name, isDir: buf[pos+18] == 4})
			}
			pos = pos + reclen
		}
	}
	Close(fd)
	return out, nil
}

func MkdirAll(path string, mode int) error {
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
