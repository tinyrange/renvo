package main

import (
	"bufio"
	"bytes"
)

func main() {
	out := &bytes.Buffer{}
	writer := bufio.NewWriterSize(out, 8)
	if n, err := writer.WriteString("hello"); n != 5 || err != nil {
		print("FAIL string\n")
		return
	}
	if writer.Buffered() != 5 || writer.Available() != 3 || out.Len() != 0 {
		print("FAIL counts\n")
		return
	}
	if writer.WriteByte('!') != nil {
		print("FAIL byte\n")
		return
	}
	if n, err := writer.WriteRune('€'); n != 3 || err != nil {
		print("FAIL rune\n")
		return
	}
	if writer.Flush() != nil || out.String() != "hello!€" {
		print("FAIL flush\n")
		return
	}

	second := &bytes.Buffer{}
	writer.Reset(second)
	if _, err := writer.ReadFrom(bytes.NewBufferString("reader data")); err != nil {
		print("FAIL readfrom\n")
		return
	}
	if writer.Flush() != nil || second.String() != "reader data" {
		print("FAIL reset\n")
		return
	}
	print("PASS\n")
}
