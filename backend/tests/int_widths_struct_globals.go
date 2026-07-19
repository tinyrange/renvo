package main

type RenvoIntWidthsRecord struct {
	a int16
	b int32
	c int64
}

type RenvoIntWidthsTail struct {
	a int64
	b int16
}

var renvoIntWidthsGlobal16 int16 = int16(-1)
var renvoIntWidthsGlobal32 int32 = int32(-2147483648)
var renvoIntWidthsGlobalRecord RenvoIntWidthsRecord = RenvoIntWidthsRecord{a: int16(-32767), b: int32(-1), c: int64(42)}

func renvoIntWidthsBump(p *int16) {
	*p += int16(1)
}

func appMain(args []string) int {
	if int(renvoIntWidthsGlobal16) != -1 || int(renvoIntWidthsGlobal32) != -2147483648 {
		print("int width globals failed\n")
		return 1
	}
	if int(renvoIntWidthsGlobalRecord.a) != -32767 || int(renvoIntWidthsGlobalRecord.b) != -1 || renvoIntWidthsGlobalRecord.c != 42 {
		print("int width global struct failed\n")
		return 1
	}
	var local RenvoIntWidthsRecord = RenvoIntWidthsRecord{a: int16(10), b: int32(20), c: int64(30)}
	renvoIntWidthsBump(&local.a)
	if int(local.a) != 11 || int(local.b) != 20 || local.c != 30 {
		print("int width local struct failed\n")
		return 1
	}
	neg16 := 0xffff
	neg32 := 0xffffffff
	items := []RenvoIntWidthsRecord{RenvoIntWidthsRecord{a: int16(neg16), b: int32(neg32), c: int64(7)}}
	if int(items[0].a) != -1 || int(items[0].b) != -1 || items[0].c != 7 {
		print("int width struct slice failed\n")
		return 1
	}
	guard := int64(99)
	var tail RenvoIntWidthsTail = RenvoIntWidthsTail{a: int64(5), b: int16(-2)}
	if tail.a != 5 || int(tail.b) != -2 || guard != 99 {
		print("int width tail struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
