package main

type renvo1049Node struct {
	value int
}

func renvo1049Build() (renvo1049Node, []byte) {
	var data []byte
	data = append(data, 'x')
	return renvo1049Node{value: 8}, data
}

func renvo1049Wrap() (renvo1049Node, []byte) {
	return renvo1049Build()
}

func appMain(args []string) int {
	node, data := renvo1049Wrap()
	if node.value != 8 || len(data) != 1 || data[0] != 'x' {
		print("RENVO-1049 return struct slice wrapper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
