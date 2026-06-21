package main

type rtgMD42Box struct {
	value int
}

func rtgMD42New(v int) rtgMD42Box {
	return rtgMD42Box{value: v}
}

func (b rtgMD42Box) Double() int {
	return b.value * 2
}

func appMain(args []string) int {
	if rtgMD42New(6).Double() != 12 {
		print("methods_method_call_on_helper_result failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
