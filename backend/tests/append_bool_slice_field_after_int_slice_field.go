package main

type renvoABSFReloc struct {
	at    int
	label int
}

type renvoABSFSymbol struct {
	name  []byte
	label int
}

type renvoABSFAsm struct {
	code      []byte
	labelPos  []int
	labelSet  []bool
	relocs    []renvoABSFReloc
	absRelocs []renvoABSFReloc
	symbols   []renvoABSFSymbol
	data      []byte
}

func renvoABSFInit(a *renvoABSFAsm) {
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []renvoABSFReloc
	var absRelocs []renvoABSFReloc
	var symbols []renvoABSFSymbol
	var data []byte
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.absRelocs = absRelocs
	a.symbols = symbols
	a.data = data
}

func renvoABSFNewLabel(a *renvoABSFAsm) int {
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	label := len(a.labelPos) - 1
	return label
}

func appMain(args []string) int {
	var a renvoABSFAsm
	renvoABSFInit(&a)

	first := renvoABSFNewLabel(&a)
	second := renvoABSFNewLabel(&a)

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
