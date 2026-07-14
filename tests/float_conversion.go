package main

type conversionScalar = float64

type conversionPoint struct {
	x conversionScalar
	y conversionScalar
}

type conversionMatrix struct {
	a conversionScalar
	b conversionScalar
	c conversionScalar
	d conversionScalar
}

func conversionMakePoint(x, y int) conversionPoint {
	return conversionPoint{x: conversionScalar(x) + 0.5, y: conversionScalar(y) + 0.5}
}

func conversionIdentity() conversionMatrix {
	return conversionMatrix{a: 1, d: 1}
}

func conversionScalarIdentity(value conversionScalar) conversionScalar {
	return value
}

func conversionReturnIntegerLiteral() conversionScalar {
	return 7
}

func conversionGlyphCoordinate(position conversionPoint, scale conversionScalar, row int) conversionScalar {
	return position.y + conversionScalar(row)*scale
}

func appMain(args []string) int {
	p := conversionMakePoint(1, -2)
	direct := conversionPoint{x: 100, y: 290}
	matrix := conversionIdentity()
	mixed := conversionScalar(2) * 3
	one := conversionScalar(1)
	if p.x != 1.5 {
		print("point x failed\n")
		return 1
	}
	if p.y != -1.5 {
		print("point y failed\n")
		return 1
	}
	if int(p.x) != 1 {
		print("point int x failed\n")
		return 1
	}
	if int(p.y) != -1 {
		print("point int y failed\n")
		return 1
	}
	if int(direct.x) != 100 || int(direct.y) != 290 {
		print("literal field conversion failed\n")
		return 1
	}
	if int(matrix.a) != 1 || int(matrix.d) != 1 {
		print("matrix conversion failed\n")
		return 1
	}
	if int(conversionScalarIdentity(40)) != 40 {
		print("parameter conversion failed\n")
		return 1
	}
	if int(conversionReturnIntegerLiteral()) != 7 {
		print("return conversion failed\n")
		return 1
	}
	if int(mixed) != 6 {
		print("multiplication conversion failed\n")
		return 1
	}
	if one != 1 {
		print("float comparison with integer literal failed\n")
		return 1
	}
	for row := 0; row < 7; row++ {
		if int(conversionScalar(row)) != row {
			print("local integer to named float conversion failed\n")
			return 1
		}
		got := int(conversionGlyphCoordinate(conversionPoint{x: 2, y: 2}, 1, row))
		if got != row+2 {
			if got == row*4+2 {
				print("row conversion scaled four times\n")
				return 1
			}
			if got == row*2+2 {
				print("row conversion scaled twice\n")
				return 1
			}
			if got == row+8 {
				print("struct position scaled four times\n")
				return 1
			}
			if row == 0 {
				if got == 0 {
					print("struct argument got zero\n")
					return 1
				}
				if got == 1 {
					print("struct argument got one\n")
					return 1
				}
				if got == 4 {
					print("struct argument got four\n")
					return 1
				}
				if got == 8 {
					print("struct argument got eight\n")
					return 1
				}
			}
			print("struct argument arithmetic failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
