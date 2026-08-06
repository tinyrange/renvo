package main

type key struct {
	name string
	code int
}

type value struct {
	left  int
	right string
}

type named map[key]value

type holder struct {
	values map[string]int
}

func (values named) count() int { return len(values) }

func mutate(values named, k key) {
	values[k] = value{left: values[k].left + 1, right: "changed"}
}

func mapComparePanics(left interface{}, right interface{}) (ok bool) {
	defer func() { ok = recover() != nil }()
	return left == right
}

func unhashableKeyPanics(values map[interface{}]string) (ok bool) {
	defer func() { ok = recover() != nil }()
	values[[]int{1}] = "bad"
	return false
}

func main() {
	k := key{name: "alpha", code: 7}
	values := named{k: {left: 2, right: "start"}}
	alias := values
	mutate(alias, k)
	got, ok := values[k]
	if !ok || got.left != 3 || got.right != "changed" {
		print("FAIL: composite lookup\n")
		return
	}
	delete(alias, k)
	if _, ok := values[k]; ok || len(values) != 0 {
		print("FAIL: alias delete\n")
		return
	}
	madeNamed := make(named, 1)
	madeNamed[k] = value{left: 9}
	if madeNamed.count() != 1 || madeNamed[k].left != 9 {
		print("FAIL: named make\n")
		return
	}
	arrays := make(map[[2]int]string, 2)
	arrays[[2]int{4, 5}] = "array"
	if arrays[[2]int{4, 5}] != "array" {
		print("FAIL: array key\n")
		return
	}
	arrayValues := map[int][2]int{1: {2, 3}}
	arrayValue := arrayValues[1]
	if arrayValue[0] != 2 || arrayValue[1] != 3 {
		print("FAIL: array map value\n")
		return
	}
	var empty map[int]string
	if len(empty) != 0 || empty[9] != "" {
		print("FAIL: nil lookup\n")
		return
	}
	ranged := map[int]int{1: 1, 2: 2, 3: 3}
	visits := 0
	for key := range ranged {
		visits++
		delete(ranged, 1)
		delete(ranged, 2)
		delete(ranged, 3)
		_ = key
	}
	if visits != 1 {
		print("FAIL: deleted range entries\n")
		return
	}
	interfaces := map[interface{}]string{7: "integer", "seven": "string"}
	if interfaces[7] != "integer" || interfaces["seven"] != "string" {
		print("FAIL: interface keys\n")
		return
	}
	if !mapComparePanics(arrays, arrays) {
		print("FAIL: map interface comparison\n")
		return
	}
	if !unhashableKeyPanics(interfaces) {
		print("FAIL: unhashable interface key\n")
		return
	}
	dynamic := map[string]interface{}{"value": value{left: 12, right: "dynamic"}}
	dynamicValue, dynamicOK := dynamic["value"].(value)
	if !dynamicOK || dynamicValue.left != 12 || dynamicValue.right != "dynamic" {
		print("FAIL: interface map values\n")
		return
	}
	nested := map[string]map[int]string{"first": {1: "one"}}
	nested["first"][2] = "two"
	nested["second"] = map[int]string{3: "three"}
	if nested["first"][1] != "one" || nested["first"][2] != "two" || nested["second"][3] != "three" {
		print("FAIL: nested maps\n")
		return
	}
	first := map[string]int{"a": 1}
	box := holder{values: first}
	box.values["c"] = 3
	if box.values["a"] != 1 || first["c"] != 3 {
		print("FAIL: map struct field\n")
		return
	}
	rows := []map[string]int{first, {"b": 2}}
	rows[1]["d"] = 4
	if rows[0]["a"] != 1 || rows[1]["b"] != 2 || rows[1]["d"] != 4 {
		print("FAIL: map slice elements\n")
		return
	}
	order := map[string]int{"old": 3}
	orderKey := "old"
	orderKey, order[orderKey] = "new", 7
	if orderKey != "new" || order["old"] != 7 || order["new"] != 0 {
		print("FAIL: map assignment order\n")
		return
	}
	firstLookup, firstPresent := order["old"]
	secondLookup, secondPresent := order["old"]
	if !firstPresent || !secondPresent || firstLookup+secondLookup != 14 {
		print("FAIL: consecutive comma-ok lookups\n")
		return
	}
	length := -1
	order["inserted"], length = 8, len(order)
	if length != 1 || len(order) != 2 {
		print("FAIL: map insertion timing\n")
		return
	}
	timing := make(map[string]int)
	observed := -1
	timing["new"], observed = 7, len(timing)
	if observed != 0 || len(timing) != 1 {
		print("FAIL: empty map insertion timing\n")
		return
	}
	timing["multiline"] = len(
		timing,
	)
	if timing["multiline"] != 1 || len(timing) != 2 {
		print("FAIL: multiline map assignment\n")
		return
	}
	compound := make(map[string]int)
	compound["new"] += len(compound) + 1
	if compound["new"] != 1 || len(compound) != 1 {
		print("FAIL: compound insertion timing\n")
		return
	}
	firstMake, secondMake := make(map[int]int), make(map[int]int)
	firstMake[1], secondMake[1] = 10, 20
	if firstMake[1] != 10 || secondMake[1] != 20 {
		print("FAIL: fresh map allocation\n")
		return
	}
	grown := make(map[int]int, 1)
	for index := 0; index < 32; index++ {
		grown[index] = index * index
	}
	sum := 0
	for index, value := range grown {
		if value != index*index {
			print("FAIL: ranged map value\n")
			return
		}
		sum += value
	}
	if len(grown) != 32 || grown[31] != 961 || sum != 10416 {
		print("FAIL: map growth\n")
		return
	}
	print("PASS\n")
}
