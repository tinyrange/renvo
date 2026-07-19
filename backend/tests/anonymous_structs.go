package main

type anonymousStructHolder struct {
	Value struct{ A int }
}

type anonymousStructAlias = struct{ A int }
type anonymousStructRows []struct{ A int }

func anonymousStructRoundTrip(value struct{ A int }) struct{ A int } {
	return value
}

func appMain() int {
	var zero struct{ A int }
	value := struct{ A int }{A: 2}
	var explicit struct{ A int } = struct{ A int }{A: 3}
	alias := anonymousStructAlias{A: 4}
	rows := anonymousStructRows{{A: 5}}
	holder := anonymousStructHolder{Value: struct{ A int }{A: 6}}
	if zero.A == 0 && anonymousStructRoundTrip(value).A == 2 && explicit.A == 3 && alias.A == 4 && rows[0].A == 5 && holder.Value.A == 6 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
