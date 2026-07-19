package main

type renvoMD42Box struct {
	value int
}

func renvoMD42New(v int) renvoMD42Box {
	return renvoMD42Box{value: v}
}

func (b renvoMD42Box) Double() int {
	return b.value * 2
}

func appMain(args []string) int {
	if renvoMD42New(6).Double() != 12 {
		print("methods_method_call_on_helper_result failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
