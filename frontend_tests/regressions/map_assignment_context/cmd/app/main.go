package main

type Value struct{ N int }
type Number int

func main() {
	values := []Value{{N: 7}}
	mapping := make(map[string]Value)
	for _, index := range []int{0} {
		v := values[index]
		mapping["value"] = v
	}
	numbers := make(map[string]Number)
	numbers["a"], numbers["b"] = 1, 2
	numbers["a"], numbers["b"] = numbers["b"], numbers["a"]
	pointers := make(map[string]*Value)
	pointers["nil"] = nil
	anonymous := struct{ N int }{N: 8}
	mapping["anonymous"] = anonymous
	nested := make(map[string]map[string]Number)
	nested["numbers"] = numbers
	if mapping["anonymous"].N != 8 || nested["numbers"]["a"] != 2 || mapping["value"].N != 7 || numbers["a"] != 2 || numbers["b"] != 1 || pointers["nil"] != nil {
		panic("assignment context")
	}
	println("PASS")
}
