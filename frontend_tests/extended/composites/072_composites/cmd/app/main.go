package main

type inner struct {
	a int
}

type outer struct {
	name string
	list []inner
}

func main() {
	v := outer{name: "ok", list: []inner{{a: 4}, {a: 15}}}
	if v.name == "ok" && v.list[0].a+v.list[1].a == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
