package main

type strings21Box struct{ name string }

func appMain(args []string) int {
	x := strings21Box{name: "ptr"}
	p := &x
	if p.name != "ptr" {
		print("strings_21 ptr\n")
		return 1
	}
	print("PASS\n")
	return 0
}
