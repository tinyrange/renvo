package main

func renvo_runtime_CCacheLineFlush(address *byte) {}
func renvo_runtime_CPrefetch(address *byte)       {}

func appMain(args []string) int {
	value := byte(1)
	renvo_runtime_CCacheLineFlush(&value)
	renvo_runtime_CPrefetch(&value)
	print("PASS\n")
	return 0
}
