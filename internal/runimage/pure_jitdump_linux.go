//go:build !renvo && linux

package runimage

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

// JITDump is an opt-in perf jitdump v1 stream. Each load contains its real native
// bytes, absolute load address and host-compatible timestamp. The executable
// mapping is perf's discovery marker only; no byte of this data file is called.
// One process-owned stream can name several serialized, append-only arenas.
type JITDump struct {
	mu           sync.Mutex
	file         *os.File
	marker       []byte
	index, bytes uint64
	failed       bool
}

const jitDumpLimit = 32 << 20

func NewJITDump(path string) (*JITDump, error) {
	machine := uint32(62) // EM_X86_64
	if runtime.GOARCH == "arm64" {
		machine = 183
	} else if runtime.GOARCH != "amd64" {
		return nil, fmt.Errorf("unsupported jitdump host")
	}
	timestamp, err := jitTimestamp()
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	var header [40]byte
	binary.LittleEndian.PutUint32(header[0:4], 0x4a695444)
	binary.LittleEndian.PutUint32(header[4:8], 1)
	binary.LittleEndian.PutUint32(header[8:12], 40)
	binary.LittleEndian.PutUint32(header[12:16], machine)
	binary.LittleEndian.PutUint32(header[20:24], uint32(os.Getpid()))
	binary.LittleEndian.PutUint64(header[24:32], timestamp)
	// Intel PT uses raw TSC on amd64; other hosts use CLOCK_MONOTONIC.
	binary.LittleEndian.PutUint64(header[32:40], jitTimestampFlags)
	if _, err = file.Write(header[:]); err != nil {
		file.Close()
		return nil, err
	}
	marker, err := syscall.Mmap(int(file.Fd()), 0, os.Getpagesize(), syscall.PROT_READ|syscall.PROT_EXEC, syscall.MAP_PRIVATE)
	if err != nil {
		file.Close()
		return nil, err
	}
	return &JITDump{file: file, marker: marker, bytes: 40}, nil
}

func (j *JITDump) WriteCode(address uintptr, code []byte, name string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	size := uint64(56 + len(name) + 1 + len(code))
	if j.file == nil || j.failed || address == 0 || len(code) == 0 || len(name) == 0 || len(name) > 160 || strings.ContainsAny(name, "\r\n\x00") || size > jitDumpLimit-j.bytes {
		return fmt.Errorf("invalid or over-budget jitdump load")
	}
	timestamp, err := jitTimestamp()
	if err != nil {
		return err
	}
	j.index++
	var record [56]byte
	// JIT_CODE_LOAD (id zero), followed by name NUL and exact code bytes.
	binary.LittleEndian.PutUint32(record[4:8], uint32(size))
	binary.LittleEndian.PutUint64(record[8:16], timestamp)
	binary.LittleEndian.PutUint32(record[16:20], uint32(os.Getpid()))
	binary.LittleEndian.PutUint32(record[20:24], uint32(syscall.Gettid()))
	binary.LittleEndian.PutUint64(record[24:32], uint64(address))
	binary.LittleEndian.PutUint64(record[32:40], uint64(address))
	binary.LittleEndian.PutUint64(record[40:48], uint64(len(code)))
	binary.LittleEndian.PutUint64(record[48:56], j.index)
	for _, part := range [][]byte{record[:], []byte(name + "\x00"), code} {
		written, writeErr := j.file.Write(part)
		if writeErr != nil || written != len(part) {
			j.failed = true
			if writeErr != nil {
				return writeErr
			}
			return io.ErrShortWrite
		}
	}
	j.bytes += size
	return nil
}

func (j *JITDump) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	var result error
	if !j.failed {
		stamp, err := jitTimestamp()
		if err == nil {
			var closeRecord [16]byte
			binary.LittleEndian.PutUint32(closeRecord[0:4], 3) // JIT_CODE_CLOSE
			binary.LittleEndian.PutUint32(closeRecord[4:8], 16)
			binary.LittleEndian.PutUint64(closeRecord[8:16], stamp)
			_, err = j.file.Write(closeRecord[:])
		}
		result = err
	}
	if err := syscall.Munmap(j.marker); result == nil {
		result = err
	}
	if err := j.file.Close(); result == nil {
		result = err
	}
	j.file, j.marker = nil, nil
	return result
}
