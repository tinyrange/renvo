package main

type returnStructSliceCallFieldBox struct {
	data []byte
	ok   bool
}

func returnStructSliceCallFieldBytes() []byte {
	var out []byte
	out = append(out, 'P')
	out = append(out, 'A')
	return out
}

func returnStructSliceCallFieldMake() returnStructSliceCallFieldBox {
	return returnStructSliceCallFieldBox{data: returnStructSliceCallFieldBytes(), ok: true}
}

func appMain(args []string) int {
	box := returnStructSliceCallFieldMake()
	if !box.ok || len(box.data) != 2 || box.data[0] != 'P' || box.data[1] != 'A' {
		print("return struct slice call field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
