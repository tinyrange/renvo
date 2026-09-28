package main

const capacity = 8192

var buffer [capacity]byte

type Diagnostic struct{ Message string }

func finish(buf []byte, diagnostic Diagnostic) (int, string) {
	copy(buf, diagnostic.Message)
	return 1, string(buf[:len(diagnostic.Message)])
}
func run(r rune) (int, string) {
	return finish(buffer[:], Diagnostic{Message: string(r)})
}
func main() {
	n, s := run(65)
	if n != 1 || s != "A" {
		panic("result")
	}
	print("PASS\n")
}
