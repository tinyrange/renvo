package main

import "math/big"

func main() {
	x, ok := new(big.Int).SetString("340282366920938463463374607431768211455", 10)
	if !ok {
		panic("large integer parse")
	}
	y, ok := new(big.Int).SetString("18446744073709551615", 10)
	if !ok {
		panic("divisor parse")
	}
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(x, y, r)
	if q.String() != "18446744073709551617" || r.Sign() != 0 {
		panic("division")
	}
	if new(big.Int).Mul(q, y).Cmp(x) != 0 {
		panic("multiplication")
	}
	if new(big.Int).Rsh(big.NewInt(-17), 2).Int64() != -5 {
		panic("signed shift")
	}
	if new(big.Int).Xor(big.NewInt(-17), big.NewInt(31)).Int64() != -16 {
		panic("bitwise")
	}
	if new(big.Rat).SetFloat64(0.1).RatString() != "3602879701896397/36028797018963968" {
		panic("rational")
	}
	integer, accuracy := new(big.Float).SetFloat64(-1.75).Int(nil)
	if integer.Int64() != -1 || accuracy != big.Above {
		panic("float truncation")
	}
	large, ok := new(big.Int).SetString("9007199254740993", 10)
	if !ok {
		panic("rounding input")
	}
	f, accuracy := new(big.Float).SetInt(large).Float64()
	if f != 9007199254740992 || accuracy != big.Below {
		panic("float rounding")
	}
	print("PASS\n")
}
