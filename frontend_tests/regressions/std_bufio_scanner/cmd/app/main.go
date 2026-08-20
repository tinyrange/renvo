package main

import (
	"bufio"
	"bytes"
)

func main() {
	words := bufio.NewScanner(bytes.NewBufferString("one two"))
	words.Split(bufio.ScanWords)
	if words.Scan() && words.Text() == "one" && words.Scan() && words.Text() == "two" && !words.Scan() && words.Err() == nil {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
