package target

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// CEndian is supplied by the target descriptor. Portable C cannot reliably
// infer target byte order during translation.
type CEndian string

const (
	CEndianLittle CEndian = "little"
	CEndianBig    CEndian = "big"
)

// CMachineProfile fixes the target-visible assumptions used by C emission.
// Automatic profiles deliberately inherit the implementation's int and
// pointer widths. Explicit profiles reject incompatible C implementations.
type CMachineProfile struct {
	Name        string
	Automatic   bool
	Hosted      bool
	CharBits    int
	IntBits     int
	PointerBits int
	Endian      CEndian
	ABI         string
	RuntimeOps  []string
}

func C89AutomaticProfile(name string, hosted bool, endian CEndian, abi string, runtimeOps ...string) CMachineProfile {
	return CMachineProfile{
		Name:       name,
		Automatic:  true,
		Hosted:     hosted,
		CharBits:   8,
		Endian:     endian,
		ABI:        abi,
		RuntimeOps: append([]string(nil), runtimeOps...),
	}
}

func C89ExplicitProfile(name string, hosted bool, intBits int, pointerBits int, endian CEndian, abi string, runtimeOps ...string) CMachineProfile {
	return CMachineProfile{
		Name:        name,
		Hosted:      hosted,
		CharBits:    8,
		IntBits:     intBits,
		PointerBits: pointerBits,
		Endian:      endian,
		ABI:         abi,
		RuntimeOps:  append([]string(nil), runtimeOps...),
	}
}

func (p CMachineProfile) Validate() error {
	if p.Name == "" || p.ABI == "" {
		return fmt.Errorf("C machine profile name and ABI are required")
	}
	if !validCProfileText(p.Name) {
		return fmt.Errorf("C machine profile name %q must contain printable ASCII only", p.Name)
	}
	if !validCProfileText(p.ABI) {
		return fmt.Errorf("C machine profile %q: ABI %q must contain printable ASCII only", p.Name, p.ABI)
	}
	if p.CharBits != 8 {
		return fmt.Errorf("C machine profile %q: only CHAR_BIT=8 is implemented", p.Name)
	}
	if p.Endian != CEndianLittle && p.Endian != CEndianBig {
		return fmt.Errorf("C machine profile %q: invalid endianness %q", p.Name, p.Endian)
	}
	if p.Automatic {
		if p.IntBits != 0 || p.PointerBits != 0 {
			return fmt.Errorf("C machine profile %q: automatic widths must be zero", p.Name)
		}
	} else {
		if !supportedCWidth(p.IntBits) {
			return fmt.Errorf("C machine profile %q: language int width must be 16, 32, or 64", p.Name)
		}
		if !supportedCWidth(p.PointerBits) {
			return fmt.Errorf("C machine profile %q: pointer width must be 16, 32, or 64", p.Name)
		}
	}
	for _, op := range p.RuntimeOps {
		if !validCProfileName(op) {
			return fmt.Errorf("C machine profile %q: invalid runtime operation %q", p.Name, op)
		}
	}
	return nil
}

func validCProfileText(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func supportedCWidth(bits int) bool {
	return bits == 16 || bits == 32 || bits == 64
}

func validCProfileName(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

// RenderC89Preamble emits the deterministic machine-contract prefix used by a
// generated translation unit. It uses only C89 declarations, preprocessor
// conditionals, and negative-array compile-time assertions.
func (p CMachineProfile) RenderC89Preamble() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("#ifndef RENVO_C_MACHINE_PROFILE\n")
	out.WriteString("#define RENVO_C_MACHINE_PROFILE 1\n")
	out.WriteString("#include <limits.h>\n")
	out.WriteString("#define RENVO_C89_ASSERT(name, expression) typedef char renvo_assumption_##name[(expression) ? 1 : -1]\n")
	out.WriteString("RENVO_C89_ASSERT(char_bit, CHAR_BIT == 8);\n")
	out.WriteString("#if UCHAR_MAX == 255U\n")
	out.WriteString("typedef unsigned char renvo_u8;\n")
	out.WriteString("#else\n#error RENVO_C_PROFILE_MISSING_U8\n#endif\n")
	out.WriteString("#if USHRT_MAX == 65535UL\n")
	out.WriteString("typedef unsigned short renvo_u16;\n")
	out.WriteString("#elif UINT_MAX == 65535UL\n")
	out.WriteString("typedef unsigned int renvo_u16;\n")
	out.WriteString("#else\n#error RENVO_C_PROFILE_MISSING_U16\n#endif\n")
	out.WriteString("#if UINT_MAX == 4294967295UL\n")
	out.WriteString("typedef unsigned int renvo_u32;\n")
	out.WriteString("#elif ULONG_MAX == 4294967295UL\n")
	out.WriteString("typedef unsigned long renvo_u32;\n")
	out.WriteString("#else\n#error RENVO_C_PROFILE_MISSING_U32\n#endif\n")
	if p.Automatic {
		out.WriteString("#if ULONG_MAX > 4294967295UL\n")
		out.WriteString("typedef unsigned long renvo_u64;\n")
		out.WriteString("RENVO_C89_ASSERT(unsigned_64_carrier, sizeof(renvo_u64) * CHAR_BIT == 64);\n")
		out.WriteString("#define RENVO_C_HAS_U64 1\n")
		out.WriteString("#else\n#define RENVO_C_HAS_U64 0\n#endif\n")
	} else if p.IntBits == 64 || p.PointerBits == 64 {
		out.WriteString("#if ULONG_MAX > 4294967295UL\n")
		out.WriteString("typedef unsigned long renvo_u64;\n")
		out.WriteString("RENVO_C89_ASSERT(unsigned_64_carrier, sizeof(renvo_u64) * CHAR_BIT == 64);\n")
		out.WriteString("#else\n#error RENVO_C_PROFILE_MISSING_U64\n#endif\n")
	}
	if p.Automatic {
		out.WriteString("typedef unsigned int renvo_uint;\n")
		out.WriteString("#define RENVO_C_LANGUAGE_INT_BITS ((int)(sizeof(renvo_uint) * CHAR_BIT))\n")
		out.WriteString("#define RENVO_C_POINTER_BITS ((int)(sizeof(void *) * CHAR_BIT))\n")
	} else {
		fmt.Fprintf(&out, "typedef renvo_u%d renvo_uint;\n", p.IntBits)
		fmt.Fprintf(&out, "RENVO_C89_ASSERT(language_int_width, sizeof(renvo_uint) * CHAR_BIT == %d);\n", p.IntBits)
		fmt.Fprintf(&out, "RENVO_C89_ASSERT(pointer_width, sizeof(void *) * CHAR_BIT == %d);\n", p.PointerBits)
		fmt.Fprintf(&out, "#define RENVO_C_LANGUAGE_INT_BITS %d\n", p.IntBits)
		fmt.Fprintf(&out, "#define RENVO_C_POINTER_BITS %d\n", p.PointerBits)
	}
	if p.Hosted {
		out.WriteString("#define RENVO_C_HOSTED 1\n")
	} else {
		out.WriteString("#define RENVO_C_HOSTED 0\n")
	}
	if p.Endian == CEndianLittle {
		out.WriteString("#define RENVO_C_ENDIAN_LITTLE 1\n#define RENVO_C_ENDIAN_BIG 0\n")
	} else {
		out.WriteString("#define RENVO_C_ENDIAN_LITTLE 0\n#define RENVO_C_ENDIAN_BIG 1\n")
	}
	fmt.Fprintf(&out, "#define RENVO_C_PROFILE_NAME %q\n", p.Name)
	fmt.Fprintf(&out, "#define RENVO_C_TARGET_ABI %q\n", p.ABI)
	ops := append([]string(nil), p.RuntimeOps...)
	sort.Strings(ops)
	previous := ""
	for _, op := range ops {
		if op == previous {
			continue
		}
		previous = op
		fmt.Fprintf(&out, "#define RENVO_C_RUNTIME_%s 1\n", strings.ToUpper(op))
	}
	out.WriteString("#endif\n")
	return out.Bytes(), nil
}
