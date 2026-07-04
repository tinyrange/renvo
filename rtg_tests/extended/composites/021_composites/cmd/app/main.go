package main

type inner struct {
	a int
}

type outer struct {
	name string
	list []inner
}

func main() {
	v := outer{name: "ok", list: []inner{{a: 4}, {a: 2}}}
	if v.name == "ok" && v.list[0].a+v.list[1].a == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
