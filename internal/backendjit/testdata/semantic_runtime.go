package main

var preparedSemanticGlobal = 7

type preparedSemanticPair struct {
	left  int
	right int
}

type preparedSemanticOuter struct {
	pair *preparedSemanticPair
}

func preparedSemanticIndex(values []preparedSemanticPair, index int) preparedSemanticPair {
	return values[index]
}

func preparedSemanticWideSigned(left int64, right int64) int64 {
	return left/right + left%right
}

func appMain() int {
	local := 4
	local++
	local--
	preparedSemanticGlobal++
	preparedSemanticGlobal--
	if local != 4 || preparedSemanticGlobal != 7 {
		return 1
	}

	values := []preparedSemanticPair{
		{left: 2, right: 3},
	}
	values = append(values, preparedSemanticPair{left: 5, right: 8})
	if len(values) != 2 || cap(values) < 2 {
		return 2
	}
	if values[1].left != 5 {
		return 11
	}
	if values[1].right != 8 {
		return 12
	}

	pair := preparedSemanticIndex(values, 1)
	if pair.left != 5 {
		return 9
	}
	if pair.right != 8 {
		return 10
	}
	outer := preparedSemanticOuter{pair: &pair}
	if outer.pair != &pair {
		return 14
	}
	outer.pair.left++
	if outer.pair.left != 6 || outer.pair.right != 8 {
		return 3
	}

	left := 19
	right := 4
	if !(left > right) || left <= right || left == right {
		return 4
	}
	shift := uint(2)
	unsigned := uint(0x80)
	if unsigned>>shift != 0x20 || uint(3)<<shift != 12 {
		return 5
	}

	value := 1.5 * 2.0
	if value != 3.0 {
		return 6
	}

	wideLeft := uint64(0x100000002)
	wideRight := uint64(0x200000103)
	wide := wideLeft + wideRight
	wide = wide - wideRight
	wide = wide * 3
	wide = wide / 3
	wide = wide ^ wideRight
	wide = wide &^ uint64(0xff)
	wide = (wide | 0x40) << 5
	wide = wide >> 3
	if wide != 0xc00000500 || preparedSemanticWideSigned(-0x100000007, 3) != -0x55555559 {
		return 7
	}
	if !(wideLeft < wideRight) || wideLeft >= wideRight {
		return 8
	}

	print("PASS\n")
	return 0
}
