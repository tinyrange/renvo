package frontend_tests

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFrontendGoOpenSSLObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runGoOpenSSLObjectSystemLink(t, root, frontendCompiler(t, root), "")
}

func TestFrontendStage3GoOpenSSLObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runGoOpenSSLObjectSystemLink(t, root, selfHostedFrontendCompiler(t, root), "")
}

func TestFrontendGoOpenSSLCustomRTGObjectSystemLink(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the hosted OpenSSL object target is Linux/amd64")
	}
	root := repoRoot(t)
	runGoOpenSSLObjectSystemLink(t, root, integratedFrontendCompiler(t, root), root,
		"-backend", filepath.Join(root, "backends", "linux_amd64_object.rtg"),
		"-t", "linux-object/amd64")
}

func integratedFrontendCompiler(t *testing.T, root string) frontendConfig {
	t.Helper()
	compiler := filepath.Join(t.TempDir(), "renvo")
	build := exec.Command("go", "build", "-o", compiler, "./cmd/renvo")
	build.Dir = root
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build integrated Renvo command: %v\n%s", err, combined)
	}
	return frontendConfig{
		compiler: compiler,
		env:      []string{"RENVO_STDROOT=" + filepath.Join(root, "std")},
	}
}

func runGoOpenSSLObjectSystemLink(
	t *testing.T, root string, frontend frontendConfig, compileDir string,
	compileArgs ...string,
) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the hosted OpenSSL object target is Linux/amd64")
	}
	system := requireSystemOpenSSL(t)

	dir := t.TempDir()
	source := filepath.Join(root, "frontend_tests", "testdata", "openssl_object.go")
	object := filepath.Join(dir, "openssl_object.o")
	harness := filepath.Join(dir, "harness.c")
	executable := filepath.Join(dir, "openssl_object_test")
	compileArgs = append(compileArgs, "-mode=object", "-o", object, source)
	command := frontendCommand(frontend, compileArgs...)
	command.Dir = dir
	if compileDir != "" {
		command.Dir = compileDir
	}
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile OpenSSL Go object with Renvo: %v\n%s", err, combined)
	}

	file, err := elf.Open(object)
	if err != nil {
		t.Fatalf("open Renvo OpenSSL object: %v", err)
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF type/machine = %v/%v", file.Type, file.Machine)
	}
	wantedImports := []string{
		"OpenSSL_version_num", "OpenSSL_version", "SHA256", "CRYPTO_memcmp",
		"RAND_bytes", "EVP_MD_CTX_new", "EVP_MD_CTX_free", "EVP_sha256",
		"EVP_DigestInit_ex", "EVP_DigestUpdate", "EVP_DigestFinal_ex",
		"BN_new", "BN_free", "BN_set_word", "BN_add_word", "BN_get_word",
		"BN_bn2binpad",
	}
	foundImports := make(map[string]bool)
	foundExport := false
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "renvo_openssl_self_test" &&
			elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL &&
			symbol.Section != elf.SHN_UNDEF {
			foundExport = true
		}
		if symbol.Section == elf.SHN_UNDEF &&
			elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL {
			foundImports[symbol.Name] = true
		}
	}
	if !foundExport {
		t.Fatal("OpenSSL object does not export renvo_openssl_self_test")
	}
	for _, name := range wantedImports {
		if !foundImports[name] {
			t.Fatalf("OpenSSL object does not import %s", name)
		}
	}
	if relocations := file.Section(".rela.text"); relocations == nil {
		t.Fatal("OpenSSL object has no .rela.text section")
	}

	harnessSource := []byte(
		"#include <stdint.h>\n" +
			"#include <stdio.h>\n" +
			"extern int32_t renvo_openssl_self_test(void);\n" +
			"extern int32_t call_renvo_preserving_rbx(void);\n" +
			"__asm__(\n" +
			"  \".text\\n\"\n" +
			"  \".globl call_renvo_preserving_rbx\\n\"\n" +
			"  \".type call_renvo_preserving_rbx,@function\\n\"\n" +
			"  \"call_renvo_preserving_rbx:\\n\"\n" +
			"  \"pushq %rbx\\n\"\n" +
			"  \"movabsq $0x1122334455667788, %rbx\\n\"\n" +
			"  \"call renvo_openssl_self_test\\n\"\n" +
			"  \"movabsq $0x1122334455667788, %rcx\\n\"\n" +
			"  \"cmpq %rcx, %rbx\\n\"\n" +
			"  \"je 1f\\n\"\n" +
			"  \"movl $24, %eax\\n\"\n" +
			"  \"1: popq %rbx\\n\"\n" +
			"  \"ret\\n\"\n" +
			");\n" +
			"int main(void) {\n" +
			"  int32_t result = call_renvo_preserving_rbx();\n" +
			"  if (result != 0) fprintf(stderr, \"OpenSSL interop failure %d\\n\", result);\n" +
			"  return result;\n" +
			"}\n",
	)
	if err := os.WriteFile(harness, harnessSource, 0o644); err != nil {
		t.Fatal(err)
	}
	linkArgs := []string{harness, object, "-o", executable}
	linkArgs = append(linkArgs, system.linkArgs...)
	link := exec.Command(system.linker, linkArgs...)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo OpenSSL object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked OpenSSL object: %v, output %q", err, combined)
	}
}

type systemOpenSSLConfig struct {
	linker      string
	includeArgs []string
	linkArgs    []string
}

func requireSystemOpenSSL(t *testing.T) systemOpenSSLConfig {
	t.Helper()
	linker, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker driver is unavailable")
	}
	pkgConfig, err := exec.LookPath("pkg-config")
	if err != nil {
		t.Skip("pkg-config is unavailable")
	}
	if err := exec.Command(pkgConfig, "--exists", "openssl").Run(); err != nil {
		t.Skip("system OpenSSL development package is unavailable")
	}
	cflagsOutput, err := exec.Command(pkgConfig, "--cflags", "openssl").Output()
	if err != nil {
		t.Fatalf("query OpenSSL compiler flags: %v", err)
	}
	var includeArgs []string
	for _, flag := range strings.Fields(string(cflagsOutput)) {
		if strings.HasPrefix(flag, "-I") && len(flag) > 2 {
			includeArgs = append(includeArgs, flag)
		}
	}
	linkOutput, err := exec.Command(pkgConfig, "--libs", "openssl").Output()
	if err != nil {
		t.Fatalf("query OpenSSL linker flags: %v", err)
	}
	linkArgs := strings.Fields(string(linkOutput))
	if len(linkArgs) == 0 {
		t.Fatal("pkg-config returned no OpenSSL linker flags")
	}
	return systemOpenSSLConfig{
		linker: linker, includeArgs: includeArgs, linkArgs: linkArgs,
	}
}
