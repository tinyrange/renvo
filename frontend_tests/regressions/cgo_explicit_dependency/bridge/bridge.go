package bridge

/* int add_go_double(int value, int extra); */
import "C"

//export go_double
func go_double(value int) int { return value * 2 }

func Value(value int) int { return int(C.add_go_double(value, 2)) }
