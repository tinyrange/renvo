package bridge

import "C"

//export go_double
func double(value int) int { return value * 2 }

func Value(value int) int { return int(C.add_go_double(value, 2)) }
