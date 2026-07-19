package main

type entry struct {
	ok bool
}

var entries []entry

func appMain(args []string, env []string) int {
	entries = append(entries, entry{})
	entries[len(entries)-1].ok = true
	if !entries[0].ok {
		print("FAIL selector field assign\n")
		return 1
	}
	print("PASS\n")
	return 0
}
