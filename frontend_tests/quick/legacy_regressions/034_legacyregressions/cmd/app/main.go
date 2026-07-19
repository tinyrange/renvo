package main

type issue34T struct{ s string }

func main() {
	t := issue34T{s: "a"}
	t.s = "b" + "c"
	if t.s == "bc" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
