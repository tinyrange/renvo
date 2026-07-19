package main

import (
	"bytes"
	"io"
)

func main() {
	buffer := bytes.NewBufferString("hello")
	data, err := io.ReadAll(buffer)
	if string(data) != "hello" || err != nil {
		print("FAIL\n")
		return
	}
	p := make([]byte, 1)
	n, err := buffer.Read(p)
	if n != 0 || err != io.EOF || err.Error() != "EOF" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
