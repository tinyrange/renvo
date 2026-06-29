package main

type rtg_j5_nz_rtg_rtgProgram struct {
	src []byte
	ok  bool
}

func appMain() int {
	var prog rtg_j5_nz_rtg_rtgProgram
	prog = rtg_j5_nz_rtg_rtgProgram{src: []byte{1}, ok: true}
	progOK := prog.ok
	if !progOK {
		print("long_type_bool_field_selector failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
