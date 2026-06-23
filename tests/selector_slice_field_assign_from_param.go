package main

type rtgSSFHolder struct {
	src []byte
	ok  bool
}

func rtgSSFBuild(src []byte) rtgSSFHolder {
	var h rtgSSFHolder
	h.src = src
	h.ok = true
	return h
}

func appMain(args []string) int {
	var src []byte
	src = append(src, 'A')
	src = append(src, 'B')

	h := rtgSSFBuild(src)
	if !h.ok {
		print("selector_slice_field_assign_from_param flag failed\n")
		return 1
	}
	if len(h.src) != 2 {
		print("selector_slice_field_assign_from_param length failed\n")
		return 1
	}
	if h.src[0] != 'A' {
		print("selector_slice_field_assign_from_param first byte failed\n")
		return 1
	}
	if h.src[1] != 'B' {
		print("selector_slice_field_assign_from_param second byte failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
