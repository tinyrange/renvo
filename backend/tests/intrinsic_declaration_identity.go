package main

var calls int

func renvo_runtime_FmtPrintln_other() { calls++ }

//renvo:intrinsic renvo_runtime_FmtPrintln
func renvo_runtime_FmtPrintln_(value string) { print("FAIL\n") }

func appMain() int {
	renvo_runtime_FmtPrintln_other()
	if calls != 1 {
		print("FAIL\n")
		return 1
	}
	renvo_runtime_FmtPrintln_("PASS")
	return 0
}
