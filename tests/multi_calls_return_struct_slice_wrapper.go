package main

type rtg1049Node struct {
	value int
}

func rtg1049Build() (rtg1049Node, []byte) {
	var data []byte
	data = append(data, 'x')
	return rtg1049Node{value: 8}, data
}

func rtg1049Wrap() (rtg1049Node, []byte) {
	return rtg1049Build()
}

func appMain(args []string) int {
	node, data := rtg1049Wrap()
	if node.value != 8 || len(data) != 1 || data[0] != 'x' {
		print("RTG-1049 return struct slice wrapper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
