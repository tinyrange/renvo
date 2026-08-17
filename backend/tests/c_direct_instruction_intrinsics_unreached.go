package main

func renvo_runtime_CSerialize()                                           {}
func renvo_runtime_CDirectMove64(destination *byte, source *byte)         {}
func renvo_runtime_CEnqueueCommand(destination *byte, source *byte) uint8 { return 0 }
func renvo_runtime_CTileRelease()                                         {}

func exerciseDirectInstructions(destination *byte, source *byte) uint8 {
	renvo_runtime_CSerialize()
	renvo_runtime_CDirectMove64(destination, source)
	renvo_runtime_CTileRelease()
	return renvo_runtime_CEnqueueCommand(destination, source)
}

func appMain(args []string) int {
	if len(args) < 0 {
		var storage [128]byte
		_ = exerciseDirectInstructions(&storage[0], &storage[64])
	}
	print("PASS\n")
	return 0
}
