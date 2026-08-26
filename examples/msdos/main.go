package main

func rtgasmAnswer() int

func main() {
	if rtgasmAnswer() != 42 {
		print("RTGASM failure\r\n")
		return
	}
	print("Hello from Renvo on MS-DOS!\r\n")
}
