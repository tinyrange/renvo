package target

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestArtifactFromLinkedELF(t *testing.T) {
	image := compileLinkedELFFixture(t)
	artifact, err := ArtifactFromELF(image, ELFArtifactOptions{VectorSymbol: "rtg_vectors", HeapSize: 128, StackSize: 512})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Entry == 0 || artifact.VectorAddress == 0 || artifact.HeapSize != 128 || artifact.StackSize != 512 {
		t.Fatalf("artifact header = %+v", artifact)
	}
	var executable bool
	var bssWithoutLoad bool
	for _, section := range artifact.Sections {
		if section.Flags&SectionExec != 0 && section.LoadSize != 0 {
			executable = true
		}
		if section.Name == ".bss" && section.LoadSize == 0 {
			bssWithoutLoad = true
		}
		if section.LoadSize != 0 && section.LoadAddress == 0 {
			t.Fatalf("file-backed section lacks LMA: %+v", section)
		}
	}
	if !executable || !bssWithoutLoad {
		t.Fatalf("ELF sections were not classified correctly: %+v", artifact.Sections)
	}
	if len(artifact.Imports) != 0 {
		t.Fatalf("freestanding fixture imports = %v", artifact.Imports)
	}
}

func TestArtifactFromELFRejectsObjectAndMissingVector(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF fixture linker test is Linux-specific")
	}
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "object.c")
	object := filepath.Join(dir, "object.o")
	if err := os.WriteFile(source, []byte("int value(void) { return 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(compiler, "-std=c89", "-ffreestanding", "-c", source, "-o", object)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile object: %v\n%s", err, output)
	}
	if _, err := ArtifactFromELF(object, ELFArtifactOptions{VectorSymbol: "value"}); err == nil || !strings.Contains(err.Error(), "linked executable") {
		t.Fatalf("relocatable object error = %v", err)
	}
	image := compileLinkedELFFixture(t)
	if _, err := ArtifactFromELF(image, ELFArtifactOptions{VectorSymbol: "missing"}); err == nil || !strings.Contains(err.Error(), "not defined") {
		t.Fatalf("missing vector error = %v", err)
	}
}

func compileLinkedELFFixture(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("ELF fixture linker test is Linux-specific")
	}
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "firmware.c")
	image := filepath.Join(dir, "firmware.elf")
	program := "int rtg_bss;\nint rtg_vectors(void) { return rtg_bss; }\nint rtg_entry(void) { return rtg_vectors(); }\n"
	if err := os.WriteFile(source, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(compiler, "-std=c89", "-pedantic-errors", "-ffreestanding", "-fno-pic", "-nostdlib", "-no-pie", "-Wl,-e,rtg_entry", source, "-o", image)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("link ELF fixture: %v\n%s", err, output)
	}
	file, err := elf.Open(image)
	if err != nil {
		t.Fatal(err)
	}
	if file.Type != elf.ET_EXEC {
		file.Close()
		t.Fatalf("fixture type = %s", file.Type)
	}
	file.Close()
	return image
}
