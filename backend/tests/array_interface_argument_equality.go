package main

func sameArray(a interface{}, b interface{}) bool { return a == b }
func appMain(args []string) int {
	a := [2]int{1, 2}
	var boxed interface{} = a
	if !sameArray(boxed, boxed) || !sameArray(a, [2]int{1, 2}) {
		return 1
	}
	print("PASS\n")
	return 0
}
