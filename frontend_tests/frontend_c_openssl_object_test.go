package frontend_tests

import (
	"debug/elf"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCOpenSSLObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCOpenSSLObjectSystemLink(t, root, frontendCompiler(t, root), "")
}

func TestFrontendStage3COpenSSLObjectSystemLink(t *testing.T) {
	root := repoRoot(t)
	runCOpenSSLObjectSystemLink(t, root, selfHostedFrontendCompiler(t, root), "")
}

func TestFrontendCOpenSSLCustomRTGObjectSystemLink(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the hosted OpenSSL object target is Linux/amd64")
	}
	root := repoRoot(t)
	runCOpenSSLObjectSystemLink(t, root, integratedFrontendCompiler(t, root), root,
		"-backend", filepath.Join(root, "backends", "linux_amd64_object.rtg"),
		"-t", "linux-object/amd64")
}

func runCOpenSSLObjectSystemLink(
	t *testing.T, root string, frontend frontendConfig, compileDir string,
	backendArgs ...string,
) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("the hosted OpenSSL object target is Linux/amd64")
	}
	system := requireSystemOpenSSL(t)
	dir := t.TempDir()
	source := filepath.Join(root, "frontend_tests", "testdata", "openssl_object.c")
	object := filepath.Join(dir, "openssl_c_object.o")
	executable := filepath.Join(dir, "openssl_c_object_test")

	compileArgs := []string{"cc"}
	compileArgs = append(compileArgs, backendArgs...)
	compileArgs = append(compileArgs, system.includeArgs...)
	compileArgs = append(compileArgs, "-c", source, "-o", object)
	command := frontendCommand(frontend, compileArgs...)
	command.Dir = dir
	if compileDir != "" {
		command.Dir = compileDir
	}
	command.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile real-header OpenSSL C object with Renvo: %v\n%s", err, combined)
	}

	file, err := elf.Open(object)
	if err != nil {
		t.Fatalf("open Renvo C OpenSSL object: %v", err)
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Machine != elf.EM_X86_64 {
		t.Fatalf("ELF type/machine = %v/%v", file.Type, file.Machine)
	}
	wantedImports := []string{
		"OpenSSL_version_num", "OpenSSL_version", "SHA256", "CRYPTO_memcmp",
		"RAND_bytes", "EVP_MD_CTX_new", "EVP_MD_CTX_free", "EVP_sha256",
		"EVP_DigestInit_ex", "EVP_DigestUpdate", "EVP_DigestFinal_ex",
	}
	foundImports := make(map[string]bool)
	foundMain := false
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "main" && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL &&
			symbol.Section != elf.SHN_UNDEF {
			foundMain = true
		}
		if symbol.Section == elf.SHN_UNDEF && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL {
			foundImports[symbol.Name] = true
		}
	}
	if !foundMain {
		t.Fatal("C OpenSSL object does not export main")
	}
	for _, name := range wantedImports {
		if !foundImports[name] {
			t.Fatalf("C OpenSSL object does not import %s", name)
		}
	}
	if file.Section(".rela.text") == nil {
		t.Fatal("C OpenSSL object has no .rela.text section")
	}

	linkArgs := []string{object, "-o", executable}
	linkArgs = append(linkArgs, system.linkArgs...)
	link := exec.Command(system.linker, linkArgs...)
	if combined, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Renvo C OpenSSL object: %v\n%s", err, combined)
	}
	if combined, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked C OpenSSL object: %v, output %q", err, combined)
	}
}
