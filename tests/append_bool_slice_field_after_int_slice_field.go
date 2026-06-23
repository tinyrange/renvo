package main

type rtgABSFReloc struct {
	at    int
	label int
}

type rtgABSFSymbol struct {
	name  []byte
	label int
}

type rtgABSFAsm struct {
	code      []byte
	labelPos  []int
	labelSet  []bool
	relocs    []rtgABSFReloc
	absRelocs []rtgABSFReloc
	symbols   []rtgABSFSymbol
	data      []byte
}

func rtgABSFInit(a *rtgABSFAsm) {
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []rtgABSFReloc
	var absRelocs []rtgABSFReloc
	var symbols []rtgABSFSymbol
	var data []byte
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.absRelocs = absRelocs
	a.symbols = symbols
	a.data = data
}

func rtgABSFNewLabel(a *rtgABSFAsm) int {
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	label := len(a.labelPos) - 1
	return label
}

func appMain(args []string) int {
	var a rtgABSFAsm
	rtgABSFInit(&a)

	first := rtgABSFNewLabel(&a)
	second := rtgABSFNewLabel(&a)

	if first != 0 {
		print("append_bool_slice_field_after_int_slice_field first label failed\n")
		return 1
	}
	if second != 1 {
		print("append_bool_slice_field_after_int_slice_field second label failed\n")
		return 1
	}
	if len(a.labelPos) != 2 {
		print("append_bool_slice_field_after_int_slice_field labelPos length failed\n")
		return 1
	}
	if len(a.labelSet) != 2 {
		print("append_bool_slice_field_after_int_slice_field labelSet length failed\n")
		return 1
	}
	if a.labelSet[0] {
		print("append_bool_slice_field_after_int_slice_field labelSet zero failed\n")
		return 1
	}
	if a.labelSet[1] {
		print("append_bool_slice_field_after_int_slice_field labelSet one failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
