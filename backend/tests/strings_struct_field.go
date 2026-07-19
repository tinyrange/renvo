package main

type strings20Box struct{ name string }

func appMain(args []string) int {
	x := strings20Box{name: "box"}
	if x.name != "box" {
		print("strings_20 struct\n")
		return 1
	}
	print("PASS\n")
	return 0
}
