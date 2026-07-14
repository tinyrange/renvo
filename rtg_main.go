//go:build !rtg

package main

import (
	"io"
	"log/slog"
	"os"
	"reflect"
)

type file interface {
	io.Reader
	io.Writer
	io.Closer

	Chmod(mode os.FileMode) error
}

type offsetFile interface {
	file
	io.ReaderAt
	io.WriterAt
}

var files = make(map[int]file)

const (
	O_RDWR   = os.O_RDWR
	O_RDONLY = os.O_RDONLY
	O_WRONLY = os.O_WRONLY
	O_CREATE = os.O_CREATE
	O_TRUNC  = os.O_TRUNC
)

func open(path string, flags int) int {
	f, err := os.OpenFile(path, flags, 0666)
	if err != nil {
		slog.Error("failed to open file", "path", path, "error", err)
		return -1
	}
	fd := 3
	for {
		if _, occupied := files[fd]; !occupied {
			break
		}
		fd++
	}
	files[fd] = f
	return fd
}

func close(fd int) int {
	file, ok := files[fd]
	if !ok {
		slog.Error("invalid file descriptor", "fd", fd)
		return -1
	}
	err := file.Close()
	if err != nil {
		slog.Error("failed to close file", "fd", fd, "error", err)
		return -1
	}
	delete(files, fd)
	return 0
}

func read(fd int, buf []byte, off int64) int {
	file, ok := files[fd]
	if !ok {
		slog.Error("invalid file descriptor", "fd", fd)
		return -1
	}

	if off < 0 {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return n
			}
			slog.Error("read error", "fd", fd, "error", err)
			return -1
		}
		return n
	}

	oFile, ok := file.(offsetFile)
	if !ok {
		slog.Error("file does not support offset operations", "fd", fd)
		return -1
	}
	n, err := oFile.ReadAt(buf, off)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return n
		}
		slog.Error("read error", "fd", fd, "error", err)
		return -1
	}
	return n
}

func write(fd int, buf []byte, off int64) int {
	file, ok := files[fd]
	if !ok {
		slog.Error("invalid file descriptor", "fd", fd)
		return -1
	}

	if off < 0 {
		n, err := file.Write(buf)
		if err != nil {
			slog.Error("write error", "fd", fd, "error", err)
			return -1
		}
		return n
	}

	oFile, ok := file.(offsetFile)
	if !ok {
		slog.Error("file does not support offset operations", "fd", fd)
		return -1
	}
	n, err := oFile.WriteAt(buf, off)
	if err != nil {
		slog.Error("write error", "fd", fd, "error", err)
		return -1
	}
	return n
}

func chmod(fd int, mode int) int {
	file, ok := files[fd]
	if !ok {
		slog.Error("invalid file descriptor", "fd", fd)
		return -1
	}
	err := file.Chmod(os.FileMode(mode))
	if err != nil {
		slog.Error("chmod error", "fd", fd, "error", err)
		return -1
	}
	return 0
}

func print(s string) {
	write(1, []byte(s), -1)
}

func runAppMain() int {
	fn := reflect.ValueOf(appMain)
	fnType := fn.Type()
	if fnType.NumOut() != 1 || fnType.Out(0).Kind() != reflect.Int {
		return 1
	}

	var result []reflect.Value
	args := reflect.ValueOf(os.Args)
	env := reflect.ValueOf(os.Environ())
	if fnType.NumIn() == 0 {
		result = fn.Call(nil)
	} else if fnType.NumIn() == 1 {
		if fnType.In(0) != args.Type() {
			return 1
		}
		result = fn.Call([]reflect.Value{args})
	} else if fnType.NumIn() == 2 {
		if fnType.In(0) != args.Type() || fnType.In(1) != env.Type() {
			return 1
		}
		result = fn.Call([]reflect.Value{args, env})
	} else {
		return 1
	}
	return int(result[0].Int())
}

func main() {
	files[0] = os.Stdin
	files[1] = os.Stdout
	files[2] = os.Stderr

	exit := runAppMain()

	for fd, file := range files {
		if fd >= 3 {
			err := file.Close()
			if err != nil {
				slog.Error("failed to close file", "fd", fd, "error", err)
			}
		}
	}

	os.Exit(exit)
}
