package json

import "testing"

type scalarTestLabel string
type scalarTestCount uint64

//renvo:reflect
type scalarTestRecord struct {
	Label scalarTestLabel
	Count scalarTestCount
}

func TestDecodeNamedScalar(t *testing.T) {
	// Keep the type in the linked program; each field registers its named type.
	if _, ok := describeFields(scalarTestRecord{}); !ok {
		t.Fatal("missing opted-in metadata")
	}
	label, err := decodeScalar(scalarToken(t, "\"named\""), scalarTestLabel(""))
	if err != nil || label != scalarTestLabel("named") {
		t.Fatal("named string conversion")
	}
	count, err := decodeScalar(scalarToken(t, "18446744073709551615"), scalarTestCount(0))
	if err != nil || count != scalarTestCount(18446744073709551615) {
		t.Fatal("named integer conversion")
	}
}

func scalarToken(t *testing.T, source string) jsonValue {
	v, _, err := parseJSON([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestDecodeScalarRanges(t *testing.T) {
	cases := []struct {
		source          string
		prototype, want any
	}{
		{"127", int8(0), int8(127)},
		{"-128", int8(0), int8(-128)},
		{"32767", int16(0), int16(32767)},
		{"-32768", int16(0), int16(-32768)},
		{"2147483647", int32(0), int32(2147483647)},
		{"-2147483648", int32(0), int32(-2147483648)},
		{"9223372036854775807", int64(0), int64(9223372036854775807)},
		{"-9223372036854775808", int64(0), int64(-9223372036854775808)},
		{"255", uint8(0), uint8(255)},
		{"65535", uint16(0), uint16(65535)},
		{"4294967295", uint32(0), uint32(4294967295)},
		{"18446744073709551615", uint64(0), uint64(18446744073709551615)},
		{"42", int(0), int(42)}, {"42", uint(0), uint(42)}, {"42", uintptr(0), uintptr(42)},
		{"1.5", float32(0), float32(1.5)}, {"-2.5e2", float64(0), float64(-250)},
		{"true", false, true}, {"false", true, false}, {"\"text\"", "", "text"},
		{"null", int(42), int(42)},
	}
	for _, tc := range cases {
		got, err := decodeScalar(scalarToken(t, tc.source), tc.prototype)
		if err != nil || got != tc.want {
			t.Fatalf("decode %s: got %v, error %v; want %v", tc.source, got, err, tc.want)
		}
	}
}

func TestDecodeScalarRejectsInvalidConversion(t *testing.T) {
	cases := []struct {
		source    string
		prototype any
	}{
		{"128", int8(0)}, {"-129", int8(0)},
		{"32768", int16(0)}, {"2147483648", int32(0)},
		{"9223372036854775808", int64(0)}, {"-9223372036854775809", int64(0)},
		{"256", uint8(0)}, {"65536", uint16(0)}, {"4294967296", uint32(0)},
		{"18446744073709551616", uint64(0)}, {"-1", uint64(0)},
		{"1.0", int(0)}, {"1e2", int(0)},
		{"1e40", float32(0)}, {"1e400", float64(0)},
		{"true", int(0)}, {"1", ""}, {"\"true\"", false},
	}
	for _, tc := range cases {
		if _, err := decodeScalar(scalarToken(t, tc.source), tc.prototype); err == nil {
			t.Fatalf("accepted %s", tc.source)
		}
	}
}
