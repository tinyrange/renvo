package main

type RealValue float32

func (v RealValue) value() float32 { return float32(v) }

func appMain() int {
	var value float64 = 1
	if !(value+0i < 2 && 0 < 0i+value && value != 0i && 0i != value) {
		return 1
	}
	var zero float32
	if zero != 0i || 0i != zero {
		return 2
	}
	var named RealValue = 42
	if (named+0i).value() != 42 || (0i+named).value() != 42 {
		return 3
	}
	var integer int = 42
	if integer+0i != 42 || 0i+integer != 42 {
		return 4
	}
	print("PASS\n")
	return 0
}
