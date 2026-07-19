package main

type Renvo0664Box struct{ value int }

func appMain(args []string) int {
	wide := int64(14)
	box := Renvo0664Box{value: int(wide)}
	if box.value != 14 {
		print("RENVO-0664 int64 int conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
