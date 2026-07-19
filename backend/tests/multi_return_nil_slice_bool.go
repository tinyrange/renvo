package main

func maybeBytes(ok bool) ([]byte, bool) {
	if !ok {
		return nil, false
	}
	return []byte{80, 65, 83, 83, 10}, true
}

func appMain() int {
	empty, ok := maybeBytes(false)
	if ok {
		print("multi_return_nil_slice_bool ok failed\n")
		return 1
	}
	if len(empty) != 0 {
		print("multi_return_nil_slice_bool nil length failed\n")
		return 1
	}
	data, ok := maybeBytes(true)
	if !ok {
		print("multi_return_nil_slice_bool true failed\n")
		return 1
	}
	print(string(data))
	return 0
}
