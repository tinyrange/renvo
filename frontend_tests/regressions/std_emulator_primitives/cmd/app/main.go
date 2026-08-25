package main

import (
	"errors"
	"fmt"
	"io"
	"math/bits"
	"os"
)

func main() {
	err := fmt.Errorf("emulator %s: %d", "fault", 7)
	if err.Error() != "emulator fault: 7" {
		print("FAIL fmt.Errorf\n")
		return
	}
	if io.SeekStart != 0 || io.SeekCurrent != 1 || io.SeekEnd != 2 {
		print("FAIL seek constants\n")
		return
	}
	if bits.OnesCount8(0xa5) != 4 {
		print("FAIL bits.OnesCount8\n")
		return
	}
	if os.Stdin == nil || os.Stdout == nil || os.Stderr == nil {
		print("FAIL standard files\n")
		return
	}
	if _, writeErr := os.Stdout.Write([]byte("PASS\n")); writeErr != nil {
		print("FAIL stdout write\n")
		return
	}
	file, fileErr := os.Create("seek.tmp")
	if fileErr != nil {
		print("FAIL create\n")
		return
	}
	if _, fileErr = file.Write([]byte("abcdef")); fileErr != nil {
		print("FAIL write\n")
		return
	}
	if offset, seekErr := file.Seek(-2, io.SeekCurrent); seekErr != nil || offset != 4 {
		print("FAIL seek current\n")
		return
	}
	if _, fileErr = file.Write([]byte("XY")); fileErr != nil {
		print("FAIL positioned write\n")
		return
	}
	if file.Close() != nil {
		print("FAIL close write\n")
		return
	}
	file, fileErr = os.Open("seek.tmp")
	if fileErr != nil {
		print("FAIL reopen\n")
		return
	}
	if offset, seekErr := file.Seek(-2, io.SeekEnd); seekErr != nil || offset != 4 {
		print("FAIL seek end\n")
		return
	}
	buf := make([]byte, 2)
	if n, readErr := file.Read(buf); readErr != nil || n != 2 || string(buf) != "XY" {
		print("FAIL positioned read\n")
		return
	}
	if _, readErr := file.Read(buf[:1]); !errors.Is(readErr, io.EOF) {
		print("FAIL EOF identity\n")
		return
	}
	if file.Close() != nil {
		print("FAIL close read\n")
		return
	}
}
