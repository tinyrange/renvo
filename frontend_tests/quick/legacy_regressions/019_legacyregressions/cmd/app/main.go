package main

type issue19T struct{ N int }

func (t issue19T) Name() string { return "ok" }

func main() {
	v := issue19T{N: 1}
	if v.Name() == "ok" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
