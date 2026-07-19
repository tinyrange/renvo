package main

type renvoStructReturnWrapperCallBox struct {
	value int
	ok    bool
}

func renvoStructReturnWrapperCallInner() renvoStructReturnWrapperCallBox {
	return renvoStructReturnWrapperCallBox{value: 11, ok: true}
}

func renvoStructReturnWrapperCallOuter() renvoStructReturnWrapperCallBox {
	return renvoStructReturnWrapperCallInner()
}

func appMain(args []string) int {
	box := renvoStructReturnWrapperCallOuter()
	if !box.ok || box.value != 11 {
		print("struct return wrapper call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
