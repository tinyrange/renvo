package target

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestC89SupportUsesDefinedArithmetic(t *testing.T) {
	profile := C89AutomaticProfile("defined-arithmetic", true, CEndianLittle, "host-c")
	support, err := profile.RenderC89Support()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"//", "_Static_assert", "stdint.h", "stdbool.h", " inline ", "long long"} {
		if strings.Contains(string(support), forbidden) {
			t.Fatalf("generated support contains non-C89 surface %q", forbidden)
		}
	}
	for _, alias := range []string{"#define renvo_sex8 rgx08", "#define renvo_sdiv rgsdv", "#define renvo_sshr rgshr"} {
		if !strings.Contains(string(support), alias) {
			t.Fatalf("generated support lacks short external alias %q", alias)
		}
	}
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "arithmetic.c")
	executablePath := filepath.Join(dir, "arithmetic")
	program := `
int main(void)
{
    renvo_uint all;
    renvo_uint minimum;
    all = ~(renvo_uint)0U;
    minimum = (renvo_uint)1U << (RENVO_C_LANGUAGE_INT_BITS - 1);
    if (!renvo_slt(all, 0U) || renvo_slt(0U, all)) return 1;
    if (!renvo_sle(all, all) || renvo_sle(0U, all)) return 2;
    if (renvo_sdiv(all, 2U) != 0U) return 3;
    if (renvo_sdiv(minimum, all) != minimum) return 4;
    if (renvo_srem(all, 2U) != all) return 5;
    if (renvo_shl(1U, RENVO_C_LANGUAGE_INT_BITS) != 0U) return 6;
    if (renvo_ushr(all, RENVO_C_LANGUAGE_INT_BITS) != 0U) return 7;
    if (renvo_sshr(all, 1U) != all) return 8;
    if (renvo_sshr(minimum, RENVO_C_LANGUAGE_INT_BITS) != all) return 9;
    if (renvo_sex8((renvo_u8)128U) != (all ^ (renvo_uint)127U)) return 10;
    if (renvo_sex16((renvo_u16)32768UL) != (all ^ (renvo_uint)32767UL)) return 11;
    return 0;
}
`
	source := append(append([]byte(nil), support...), []byte(program)...)
	if err := os.WriteFile(sourcePath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	compilers := map[string]string{"cc": compiler}
	for _, name := range []string{"gcc", "clang"} {
		if path, lookupErr := exec.LookPath(name); lookupErr == nil {
			compilers[name] = path
		}
	}
	for name, path := range compilers {
		name := name
		path := path
		t.Run(name, func(t *testing.T) {
			executablePath := executablePath + "-" + name
			command := exec.Command(path, "-std=c89", "-pedantic-errors", "-Wall", "-Werror", sourcePath, "-o", executablePath)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("strict C89 compilation failed: %v\n%s", err, output)
			}
			if output, err := exec.Command(executablePath).CombinedOutput(); err != nil {
				t.Fatalf("defined-arithmetic fixture failed: %v\n%s", err, output)
			}
		})
	}
}

func TestC89SupportIsDeterministic(t *testing.T) {
	profile := C89ExplicitProfile("deterministic", false, 32, 32, CEndianBig, "ilp32")
	first, err := profile.RenderC89Support()
	if err != nil {
		t.Fatal(err)
	}
	second, err := profile.RenderC89Support()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("C89 support output is not deterministic")
	}
}
