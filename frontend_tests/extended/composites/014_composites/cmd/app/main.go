package main

type inner struct {
	a int
}

type outer struct {
	name string
	list []inner
}

func main() {
	v := outer{name: "ok", list: []inner{{a: 14}, {a: 14}}}
	if v.name == "ok" && v.list[0].a+v.list[1].a == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
