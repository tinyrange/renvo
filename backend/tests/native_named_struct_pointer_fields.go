package main

type nativeNamedSetup struct {
	RequestType byte
	Request     byte
	Value       uint16
	Index       uint16
	Length      uint16
}

func nativeNamedRequestMatches(setup *nativeNamedSetup) bool {
	return setup.Request == 0x22
}

func appMain(args []string) int {
	setup := nativeNamedSetup{
		RequestType: 0x21,
		Request:     0x22,
		Value:       0x0003,
	}
	if !nativeNamedRequestMatches(&setup) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
