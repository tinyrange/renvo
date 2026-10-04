package main

type pointerValueReceiver struct{ padding, value int }

func (v pointerValueReceiver) Get() int                     { return v.value }
func readPointerValueReceiver(v interface{ Get() int }) int { return v.Get() }
func appMain() int {
	v := pointerValueReceiver{17, 42}
	if readPointerValueReceiver(&v) != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
