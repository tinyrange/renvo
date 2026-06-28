package main

type savedFile struct {
	source []byte
}

func buildSource(tag byte) []byte {
	var out []byte
	out = append(out, tag)
	out = append(out, 'A')
	out = append(out, 'S')
	return out
}

func buildFile(tag byte) savedFile {
	data := buildSource(tag)
	var file savedFile
	file.source = data
	return file
}

func appMain(args []string, env []string) int {
	first := buildFile('P')
	second := buildFile('Q')
	if len(first.source) != 3 {
		print("FAIL first len\n")
		return 1
	}
	if len(second.source) != 3 {
		print("FAIL second len\n")
		return 1
	}
	if first.source[0] != 'P' {
		print("FAIL first overwritten\n")
		return 1
	}
	if second.source[0] != 'Q' {
		print("FAIL second value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
