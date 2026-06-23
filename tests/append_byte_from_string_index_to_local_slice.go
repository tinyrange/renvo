package main

func rtgABFSBuild(s string) []byte {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func appMain(args []string) int {
	out := rtgABFSBuild("PWD=/tmp")
	if len(out) != 8 {
		print("append_byte_from_string_index_to_local_slice length failed\n")
		return 1
	}
	if out[0] != 'P' {
		print("append_byte_from_string_index_to_local_slice first byte failed\n")
		return 1
	}
	if out[4] != '/' {
		print("append_byte_from_string_index_to_local_slice slash byte failed\n")
		return 1
	}
	if out[7] != 'p' {
		print("append_byte_from_string_index_to_local_slice last byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
