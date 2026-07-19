package main

type renvoANSPHolder struct {
	data []byte
}

func renvoANSPAppend(out []byte) []byte {
	out = append(out, 'O')
	out = append(out, 'K')
	return out
}

func appMain(args []string) int {
	var h renvoANSPHolder
	h.data = renvoANSPAppend(h.data)

	if len(h.data) != 2 {
		print("append_nil_slice_parameter_from_struct_field length failed\n")
		return 1
	}
	if h.data[0] != 'O' {
		print("append_nil_slice_parameter_from_struct_field first byte failed\n")
		return 1
	}
	if h.data[1] != 'K' {
		print("append_nil_slice_parameter_from_struct_field second byte failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
