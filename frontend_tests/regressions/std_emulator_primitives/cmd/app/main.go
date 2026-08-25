package main

import (
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
	file, fileErr := os.Create("seek.tmp")
	if fileErr != nil {
		print("FAIL create\n")
		return
	}
	if _, fileErr = file.Write([]byte("abcdef")); fileErr != nil {
		print("FAIL write\n")
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
	buf := make([]byte, 6)
	if n, readErr := file.Read(buf); readErr != nil || n != 6 || string(buf) != "abcdef" {
		print("FAIL read\n")
		return
	}
	if _, readErr := file.Read(buf[:1]); readErr == nil {
		print("FAIL EOF\n")
		return
	}
	if file.Close() != nil {
		print("FAIL close read\n")
		return
	}
	if _, writeErr := os.Stdout.Write([]byte("PASS\n")); writeErr != nil {
		print("FAIL stdout write\n")
	}
}
