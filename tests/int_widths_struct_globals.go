package main

type RtgIntWidthsRecord struct {
	a int16
	b int32
	c int64
}

type RtgIntWidthsTail struct {
	a int64
	b int16
}

var rtgIntWidthsGlobal16 int16 = int16(-1)
var rtgIntWidthsGlobal32 int32 = int32(-2147483648)
var rtgIntWidthsGlobalRecord RtgIntWidthsRecord = RtgIntWidthsRecord{a: int16(-32767), b: int32(-1), c: int64(42)}

func rtgIntWidthsBump(p *int16) {
	*p += int16(1)
}

func appMain(args []string) int {
	if int(rtgIntWidthsGlobal16) != -1 || int(rtgIntWidthsGlobal32) != -2147483648 {
		print("int width globals failed\n")
		return 1
	}
	if int(rtgIntWidthsGlobalRecord.a) != -32767 || int(rtgIntWidthsGlobalRecord.b) != -1 || rtgIntWidthsGlobalRecord.c != 42 {
		print("int width global struct failed\n")
		return 1
	}
	var local RtgIntWidthsRecord = RtgIntWidthsRecord{a: int16(10), b: int32(20), c: int64(30)}
	rtgIntWidthsBump(&local.a)
	if int(local.a) != 11 || int(local.b) != 20 || local.c != 30 {
		print("int width local struct failed\n")
		return 1
	}
	neg16 := 0xffff
	neg32 := 0xffffffff
	items := []RtgIntWidthsRecord{RtgIntWidthsRecord{a: int16(neg16), b: int32(neg32), c: int64(7)}}
	if int(items[0].a) != -1 || int(items[0].b) != -1 || items[0].c != 7 {
		print("int width struct slice failed\n")
		return 1
	}
	guard := int64(99)
	var tail RtgIntWidthsTail = RtgIntWidthsTail{a: int64(5), b: int16(-2)}
	if tail.a != 5 || int(tail.b) != -2 || guard != 99 {
		print("int width tail struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
