package main

type renvoHostLayoutMarker struct {
	_ struct{} `r:"h"`
}

type renvoHostLayoutAlias renvoHostLayoutMarker

type renvoHostHeader struct {
	_     renvoHostLayoutAlias
	Kind  byte
	Value uint32
}

func appMain(args []string) int {
	var single renvoHostHeader
	single.Kind = 0x12
	single.Value = 0x3456789a
	if single.Kind != 0x12 || single.Value != 0x3456789a {
		print("single header\n")
		return 1
	}
	headers := make([]renvoHostHeader, 2)
	headers[0] = single
	headers[1].Kind = 0xab
	headers[1].Value = 0x4def0123
	if headers[0].Kind != 0x12 || headers[0].Value != 0x3456789a {
		print("first header\n")
		return 1
	}
	if headers[1].Kind != 0xab {
		print("second kind\n")
		return 1
	}
	if headers[1].Value != 0x4def0123 {
		print("second value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
