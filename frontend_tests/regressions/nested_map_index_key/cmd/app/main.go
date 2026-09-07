package main

type cell struct { id int }
var calls int
func key() string { calls++; return "x" }

func main() {
	cells := map[string]cell{"x": {id: 7}}
	objects := map[int]string{7: "found"}
	if objects[cells[key()].id] != "found" || calls != 1 { panic("nested read") }
	value, ok := objects[cells[key()].id]
	if !ok || value != "found" || calls != 2 { panic("nested lookup") }
	objects[cells[key()].id] = "changed"
	if objects[7] != "changed" || calls != 3 { panic("nested write") }
	outer := map[string]map[int]string{"x": objects}
	if outer[key()][cells[key()].id] != "changed" || calls != 5 { panic("chained read") }
	print("PASS\n")
}
