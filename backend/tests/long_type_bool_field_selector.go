package main

type renvo_j5_nz_renvo_renvoProgram struct {
	src []byte
	ok  bool
}

func appMain() int {
	var prog renvo_j5_nz_renvo_renvoProgram
	prog = renvo_j5_nz_renvo_renvoProgram{src: []byte{1}, ok: true}
	progOK := prog.ok
	if !progOK {
		print("long_type_bool_field_selector failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
