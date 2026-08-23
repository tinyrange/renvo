package testing

func printFailure(t *T, name string) {
	for i := 0; i < len(t.logs); i++ {
		print("    ")
		print(t.logs[i])
		print("\n")
	}
	print("--- FAIL: ")
	print(name)
	print("\n")
}
