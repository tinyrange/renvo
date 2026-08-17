package main

func renvo_runtime_CMonitor(address uintptr, extensions uint64, hints uint64) {}
func renvo_runtime_CMonitorExtended(address uintptr, extensions uint64, hints uint64) {
}
func renvo_runtime_CWait(extensions uint64, hints uint64) {}
func renvo_runtime_CWaitExtended(extensions uint64, counter uint64, hints uint64) {
}
func renvo_runtime_CEnableInterruptsAndWait(extensions uint64, hints uint64) {}
func renvo_runtime_CTimedPause(control uint64, high uint64, low uint64)      {}
func renvo_runtime_CInt3Selftest(value *uint32)                              {}

func exerciseMonitorWaitIntrinsics(address uintptr) {
	renvo_runtime_CMonitor(address, 1, 2)
	renvo_runtime_CMonitorExtended(address, 3, 4)
	renvo_runtime_CWait(5, 6)
	renvo_runtime_CWaitExtended(7, 8, 9)
	renvo_runtime_CEnableInterruptsAndWait(10, 11)
	renvo_runtime_CTimedPause(12, 13, 14)
	var value uint32
	renvo_runtime_CInt3Selftest(&value)
}

func appMain(args []string) int {
	if len(args) < 0 {
		exerciseMonitorWaitIntrinsics(0)
	}
	print("PASS\n")
	return 0
}
