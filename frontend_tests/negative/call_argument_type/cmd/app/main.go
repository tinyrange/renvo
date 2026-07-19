package main

type Program struct {
	Package string
	Tokens  []int
}

func needsPointer(program *Program) bool {
	return len(program.Tokens) != 0
}

func main() {
	var program Program
	if needsPointer(program) {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
