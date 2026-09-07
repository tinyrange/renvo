package big

import "testing"

func integer(t *testing.T, text string) *Int {
	x, ok := new(Int).SetString(text, 10)
	if !ok {
		t.Fatal("parse", text)
	}
	return x
}
func requireInt(t *testing.T, x *Int, want string) {
	if x.String() != want {
		t.Fatal("integer mismatch", x.String(), want)
	}
}

func TestIntegerArithmetic(t *testing.T) {
	x := integer(t, "18446744073709551615")
	y := NewInt(1)
	requireInt(t, new(Int).Add(x, y), "18446744073709551616")
	requireInt(t, new(Int).Sub(y, x), "-18446744073709551614")
	requireInt(t, new(Int).Mul(x, x), "340282366920938463426481119284349108225")
	q, r := new(Int), new(Int)
	q.QuoRem(integer(t, "340282366920938463463374607431768211455"), x, r)
	requireInt(t, q, "18446744073709551617")
	requireInt(t, r, "0")
	for _, a := range []int64{-17, 17} {
		for _, b := range []int64{-5, 5} {
			q.QuoRem(NewInt(a), NewInt(b), r)
			if q.Int64() != a/b || r.Int64() != a%b {
				t.Fatal("division signs")
			}
		}
	}
	z := new(Int).Set(x)
	z.Add(z, z)
	requireInt(t, z, "36893488147419103230")
	z.Sub(z, z)
	requireInt(t, z, "0")
	z.Set(x)
	z.Mul(z, z)
	requireInt(t, z, "340282366920938463426481119284349108225")
	z.Set(x)
	q.QuoRem(z, NewInt(10), z)
	requireInt(t, q, "1844674407370955161")
	requireInt(t, z, "5")
}

func TestIntegerBits(t *testing.T) {
	x := new(Int).Lsh(NewInt(1), 128)
	requireInt(t, x, "340282366920938463463374607431768211456")
	if x.BitLen() != 129 || x.IsInt64() || x.IsUint64() {
		t.Fatal("large width")
	}
	requireInt(t, new(Int).Rsh(x, 64), "18446744073709551616")
	requireInt(t, new(Int).Rsh(NewInt(-17), 2), "-5")
	requireInt(t, new(Int).Rsh(NewInt(-17), 1000), "-1")
	requireInt(t, new(Int).Not(x), "-340282366920938463463374607431768211457")
	requireInt(t, new(Int).And(NewInt(-17), NewInt(31)), "15")
	requireInt(t, new(Int).Or(NewInt(-17), NewInt(31)), "-1")
	requireInt(t, new(Int).Xor(NewInt(-17), NewInt(31)), "-16")
	minimum := NewInt(-9223372036854775808)
	if !minimum.IsInt64() || minimum.IsUint64() || minimum.Int64() != -9223372036854775808 {
		t.Fatal("minimum int64")
	}
	maximum := new(Int).SetUint64(18446744073709551615)
	if maximum.IsInt64() || !maximum.IsUint64() || maximum.Uint64() != 18446744073709551615 {
		t.Fatal("maximum uint64")
	}
	var zero Int
	if zero.Sign() != 0 || zero.BitLen() != 0 || !zero.IsInt64() || !zero.IsUint64() {
		t.Fatal("zero value")
	}
}

func TestIntegerText(t *testing.T) {
	for _, source := range []string{"0", "-0", "+123456789012345678901234567890", "-18446744073709551616"} {
		x := integer(t, source)
		for base := 2; base <= 62; base++ {
			y, ok := new(Int).SetString(x.Text(base), base)
			if !ok || x.Cmp(y) != 0 {
				t.Fatal("base round trip", base)
			}
		}
	}
	for _, source := range []string{"0xff", "0x_ff", "0b1111_1111", "0o377", "0377", "2_55"} {
		x, ok := new(Int).SetString(source, 0)
		if !ok || x.Int64() != 255 {
			t.Fatal("prefix", source)
		}
	}
	for _, source := range []string{"", "+", "-", "0x", "_1", "1_", "1__2", "0x__1", "09"} {
		if _, ok := new(Int).SetString(source, 0); ok {
			t.Fatal("invalid accepted", source)
		}
	}
}

func TestIntegerLargeIdentities(t *testing.T) {
	seed := uint64(123)
	for iteration := 0; iteration < 20; iteration++ {
		a, b := new(Int), new(Int)
		for i := 0; i < 4; i++ {
			seed = seed*6364136223846793005 + 1
			a.Lsh(a, 64).Add(a, new(Int).SetUint64(seed))
			seed = seed*6364136223846793005 + 1
			b.Lsh(b, 64).Add(b, new(Int).SetUint64(seed))
		}
		if iteration%2 == 0 {
			a.Neg(a)
		}
		if iteration%3 == 0 {
			b.Neg(b)
		}
		if new(Int).Sub(new(Int).Add(a, b), b).Cmp(a) != 0 {
			t.Fatal("addition identity")
		}
		product := new(Int).Mul(a, b)
		q, r := new(Int), new(Int)
		q.QuoRem(product, b, r)
		if q.Cmp(a) != 0 || r.Sign() != 0 {
			t.Fatal("product division identity")
		}
		q.QuoRem(a, b, r)
		if new(Int).Add(new(Int).Mul(q, b), r).Cmp(a) != 0 {
			t.Fatal("division reconstruction")
		}
		if new(Int).Abs(r).Cmp(new(Int).Abs(b)) >= 0 || r.Sign() != 0 && r.Sign() != a.Sign() {
			t.Fatal("remainder constraint")
		}
	}
}
