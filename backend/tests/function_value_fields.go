package main

type transformFunc func(int) int

type functionHolder struct {
	transform transformFunc
}

func addOne(value int) int {
	return value + 1
}

func setTransform(holder *functionHolder, transform transformFunc) {
	holder.transform = transform
}

func appMain() int {
	direct := &functionHolder{transform: addOne}
	assigned := &functionHolder{}
	assigned.transform = addOne
	setTransform(assigned, addOne)
	if direct.transform(40) == 41 && assigned.transform(41) == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
