package main

func choose(value any) int {
	switch value.(type) {
	case map[string]int:
		mapping, ok := value.(map[string]int)
		if !ok {
			panic("assertion")
		}
		out := mapping
		return out["key"]
	case [2]int:
		var out [2]int
		out[0] = 7
		return out[0]
	default:
		out := []int{9}
		return out[0]
	}
}

func main() {
	if choose(map[string]int{"key": 3}) != 3 || choose([2]int{}) != 7 || choose(true) != 9 {
		panic("case scope")
	}
	print("PASS\n")
}
