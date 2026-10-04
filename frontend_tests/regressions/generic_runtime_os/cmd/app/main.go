package main

import "os"

func Identity[T any](v T) T { return v }

func main() {
	args := Identity(os.Args)
	if len(args) == 0 {
		panic("process arguments")
	}
	name := args[0] + ".data"
	if err := os.WriteFile(name, Identity([]byte("value")), 0600); err != nil {
		panic(err)
	}
	read := Identity(os.ReadFile)
	data, err := read(name)
	if err != nil || string(data) != "value" {
		panic("read file callback")
	}
	open := Identity(os.Open)
	file, err := open(name)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 5)
	n, err := file.Read(buf)
	if err != nil || n != 5 || string(buf) != "value" {
		panic("descriptor read")
	}
	if file.Close() != nil {
		panic("descriptor close")
	}
	print("PASS\n")
}
