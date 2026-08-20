package main

import (
	"bufio"
	"bytes"
)

func main() {
	r := bufio.NewReaderSize(bytes.NewBufferString("alpha\r\nbeta\ngamma"), 32)
	peek, err := r.Peek(5)
	if err != nil || string(peek) != "alpha" {
		print("FAIL peek\n")
		return
	}
	if n, err := r.Discard(2); n != 2 || err != nil {
		print("FAIL discard\n")
		return
	}
	line, prefix, err := r.ReadLine()
	if string(line) != "pha" || prefix || err != nil {
		print("FAIL line one\n")
		return
	}
	print("PASS\n")
}
