package main

func rtgasmAnswer() int

func main() {
	if rtgasmAnswer() != 42 {
		print("RTGASM failure\n")
		return
	}
	print("Hello from Renvo on PDP-11 Unix V7!\n")
}
