package main

var mutableGlobal int = 3

func incMutable() {
	mutableGlobal = mutableGlobal + 1
}
func appMain(args []string) int {
	incMutable()
	if !(mutableGlobal == 4) {
		print("RENVO-0300 global_var_changed_by_helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
