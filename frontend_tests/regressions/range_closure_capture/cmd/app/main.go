package main

type Node struct{ N int }
type Callback func() int

func invoke(callback Callback) int { return callback() }

func sum(nodes []*Node) int {
	total := 0
	for _, node := range nodes {
		total += invoke(func() int { return node.N })
	}
	return total
}

func main() {
	if sum([]*Node{&Node{N: 3}, &Node{N: 7}}) != 10 {
		panic("range capture")
	}
	var callbacks []Callback
	for _, node := range []*Node{&Node{N: 11}, &Node{N: 19}} {
		callback := Callback(func() int { return node.N })
		callbacks = append(callbacks, callback)
	}
	if callbacks[0]() != 11 || callbacks[1]() != 19 {
		panic("per-iteration capture")
	}
	println("PASS")
}
