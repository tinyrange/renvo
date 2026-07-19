package bringup

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestELFObjectMilestoneContract(t *testing.T) {
	object := compileObject(t, "int renvo_omnibus_stage0(void) { return 7; }\n")
	contract := nativeContract(t, object)
	contract.RequiredExports = []string{"renvo_omnibus_stage0"}
	if got := ValidateELFObject(object, contract); !got.OK() {
		t.Fatalf("valid object rejected: %v", got.Violations)
	}
	contract.RequiredExports = []string{"renvo_omnibus_stage1"}
	assertViolation(t, ValidateELFObject(object, contract), "missing-export")
}

func TestELFObjectRejectsForbiddenImportsAndRelocations(t *testing.T) {
	object := compileObject(t, "extern int hosted_call(void); int renvo_omnibus_stage0(void) { return hosted_call(); }\n")
	contract := nativeContract(t, object)
	contract.RequiredExports = []string{"renvo_omnibus_stage0"}
	contract.AllowedRelocations = []uint32{^uint32(0)}
	got := ValidateELFObject(object, contract)
	assertViolation(t, got, "unresolved-symbol")
	assertViolation(t, got, "unsupported-relocation")
	contract.AllowedUndefined = []string{"hosted_call"}
	contract.AllowedRelocations = nil
	if got := ValidateELFObject(object, contract); !got.OK() {
		t.Fatalf("explicit shell import rejected: %v", got.Violations)
	}
}

func TestELFObjectRejectsMalformedAndLinkedImages(t *testing.T) {
	dir := t.TempDir()
	malformed := filepath.Join(dir, "bad.o")
	if err := os.WriteFile(malformed, []byte("not an ELF object"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertViolation(t, ValidateELFObject(malformed, ELFContract{}), "malformed-object")

	object := compileObject(t, "int renvo_omnibus_stage0(void) { return 7; }\n")
	data, err := os.ReadFile(object)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 18 {
		t.Fatal("compiled object has a short ELF header")
	}
	if data[elf.EI_DATA] == byte(elf.ELFDATA2MSB) {
		data[16], data[17] = 0, byte(elf.ET_EXEC)
	} else {
		data[16], data[17] = byte(elf.ET_EXEC), 0
	}
	linked := filepath.Join(dir, "linked.elf")
	if err := os.WriteFile(linked, data, 0o644); err != nil {
		t.Fatal(err)
	}
	assertViolation(t, ValidateELFObject(linked, ELFContract{}), "object-type")
}

func compileObject(t *testing.T, source string) string {
	t.Helper()
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "fixture.c")
	objectPath := filepath.Join(dir, "fixture.o")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(compiler, "-std=c89", "-pedantic-errors", "-ffreestanding", "-fno-pic", "-c", sourcePath, "-o", objectPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile fixture: %v\n%s", err, output)
	}
	return objectPath
}

func nativeContract(t *testing.T, path string) ELFContract {
	t.Helper()
	file, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	return ELFContract{Class: file.Class, Data: file.Data, Machine: file.Machine}
}

func assertViolation(t *testing.T, validation ObjectValidation, code string) {
	t.Helper()
	for _, violation := range validation.Violations {
		if violation.Code == code {
			return
		}
	}
	var messages []string
	for _, violation := range validation.Violations {
		messages = append(messages, violation.Error())
	}
	t.Fatalf("violations = %s; want %s", strings.Join(messages, "; "), code)
}
