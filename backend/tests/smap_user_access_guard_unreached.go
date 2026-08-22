package main

func renvo_runtime_CEnableUserAccess()  {}
func renvo_runtime_CDisableUserAccess() {}

func guardedUserAccessInstructions() {
	renvo_runtime_CEnableUserAccess()
	renvo_runtime_CDisableUserAccess()
}

func appMain(args []string) int {
	if len(args) < 0 {
		guardedUserAccessInstructions()
	}
	print("PASS\n")
	return 0
}
