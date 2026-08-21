package main

type namedReturnMap map[string]int

func makeNamedReturnMap() namedReturnMap {
	values := make(namedReturnMap)
	values["answer"] = 42
	return values
}

func main() {
	values := makeNamedReturnMap()
	if values["answer"] == 42 {
		print("PASS\n")
	}
}
