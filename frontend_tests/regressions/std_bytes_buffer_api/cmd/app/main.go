package main

import "bytes"

func main() {
	var buffer bytes.Buffer
	count, err := buffer.WriteString("hello")
	if count != 5 || err != nil || buffer.Len() != 5 || buffer.String() != "hello" {
		print("FAIL\n")
		return
	}
	data := make([]byte, 2)
	count, err = buffer.Read(data)
	if count != 2 || err != nil || string(data) != "he" || buffer.String() != "llo" {
		print("FAIL\n")
		return
	}
	data = make([]byte, 3)
	count, err = buffer.Read(data)
	if count != 3 || err != nil || string(data) != "llo" {
		print("FAIL\n")
		return
	}
	count, err = buffer.Read(data)
	if count != 0 || err == nil {
		print("FAIL\n")
		return
	}
	buffer.Reset()
	if buffer.Len() != 0 || buffer.String() != "" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
