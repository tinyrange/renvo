package check

import "testing"

func TestGenericTypedFloatingConstantRounding(t *testing.T) {
	for _, tc := range []struct{ value, typ, want string }{
		{"127.000001", "float32", "127"},
		{"16777217", "float32", "16777216"},
		{"16777219", "float32", "16777220"},
		{"9007199254740993", "float64", "9007199254740992"},
		{"9007199254740995", "float64", "9007199254740996"},
		{"0x1p-150", "float32", "0"},
		{"0x1.8p-149", "float32", "0x1p-148"},
		{"0x1p-1075", "float64", "0"},
		{"1e-1000", "float64", "0"},
		{"0.1", "float32", "0x1.99999ap-4"},
	} {
		e := genericEnvironment{}
		got := e.roundConstant(genericConstantLiteral(tc.value), e.types.basic(tc.typ))
		diff := genericConstantBinary(got, genericConstantLiteral(tc.want), "-", false)
		if diff == nil || len(diff.real.numerator.words) != 0 {
			t.Errorf("%s(%s) != %s", tc.typ, tc.value, tc.want)
		}
	}
}

func TestGenericConstantRepresentability(t *testing.T) {
	for _, test := range []struct {
		literal, typ string
		fits         bool
	}{
		{"127", "int8", true}, {"128", "int8", false},
		{"255", "uint8", true}, {"256", "uint8", false},
		{"1.0", "int8", true}, {"1.5", "int8", false},
		{"0x1.fcp6", "int8", true}, {"0x1p7", "int8", false},
		{"18446744073709551615", "uint64", true},
		{"18446744073709551616", "uint64", false},
		{"9223372036854775808", "int64", false},
		{"0x1.fffffep127", "float32", true},
		{"0x1.ffffffp127", "float32", false},
		{"1e-1000", "float32", true},
		{"1e100", "float32", false}, {"1e100", "float64", true},
		{"0i", "int", true}, {"1i", "int", false}, {"1i", "complex64", true},
	} {
		t.Run(test.literal+"/"+test.typ, func(t *testing.T) {
			e := &genericEnvironment{}
			value := genericArgument{untyped: genericUntypedComplex, constant: genericConstantLiteral(test.literal)}
			if value.constant == nil || e.constantFits(value, e.types.basic(test.typ)) != test.fits {
				t.Fatalf("representability mismatch for %s in %s", test.literal, test.typ)
			}
		})
	}
}

func TestGenericExactConstantArithmetic(t *testing.T) {
	for _, test := range []struct {
		a, b, op, want string
		integer        bool
	}{
		{"0.1", "0.2", "+", "0.3", false},
		{"1.5", "0.5", "+", "2", false},
		{"5", "2", "/", "2", true},
		{"5", "2", "/", "2.5", false},
		{"1i", "1i", "/", "1", false},
		{"1", "64", "<<", "18446744073709551616", true},
		{"5", "2", "%", "1", true},
	} {
		got := genericConstantBinary(genericConstantLiteral(test.a), genericConstantLiteral(test.b), test.op, test.integer)
		want := genericConstantLiteral(test.want)
		if got == nil {
			t.Fatalf("%s %s %s failed", test.a, test.op, test.b)
		}
		diff := genericConstantBinary(got, want, "-", false)
		if diff == nil || len(diff.real.numerator.words) != 0 || len(diff.imaginary.numerator.words) != 0 {
			t.Fatalf("%s %s %s != %s", test.a, test.op, test.b, test.want)
		}
	}
	if genericConstantBinary(genericConstantLiteral("1"), genericConstantLiteral("0"), "/", false) != nil {
		t.Fatal("accepted constant division by zero")
	}
}
