package main

func assignEmptyInterface(target any, value any) bool {
	switch p := target.(type) {
	case *int:
		return false
	case *[]any:
		return false
	case *any:
		if p == nil {
			return false
		}
		*p = value
		return true
	}
	return false
}

func appMain(args []string) int {
	var value any = "old"
	if !assignEmptyInterface(&value, nil) {
		panic("assign nil false")
	}
	if value != nil {
		panic("assign nil value")
	}
	if !assignEmptyInterface(&value, 42) {
		panic("assign int")
	}
	n, ok := value.(int)
	if !ok || n != 42 {
		panic("read int")
	}
	var alternate interface{} = "other"
	if !assignEmptyInterface(&alternate, nil) || alternate != nil {
		panic("empty interface alias")
	}
	print("PASS\n")
	return 0
}
