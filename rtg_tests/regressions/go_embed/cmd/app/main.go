package main

import "embed"

//go:embed assets/message.txt
var message string

//go:embed assets/data.txt
var data []byte

//go:embed assets
var assets embed.FS

func main() {
	if message != "hello embed\n" || string(data) != "0123456789\n" {
		print("FAIL\n")
		return
	}
	fromFS, err := assets.ReadFile("assets/message.txt")
	if err != nil || string(fromFS) != message {
		print("FAIL\n")
		return
	}
	entries, err := assets.ReadDir("assets")
	if err != nil || len(entries) != 2 || entries[0].Name() != "data.txt" || entries[0].IsDir() || entries[1].Name() != "message.txt" || entries[1].IsDir() {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
