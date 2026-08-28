package c11

import (
	"bytes"
	"testing"
)

type headerTestReader struct {
	name string
	src  []byte
}

func (r headerTestReader) ReadInclude(_ string, name string, angled bool) ([]byte, string, bool) {
	if !angled || name != r.name {
		return nil, "", false
	}
	return r.src, "/usr/include/" + name, true
}

func (r headerTestReader) ReadIncludeNext(from string, name string, angled bool) ([]byte, string, bool) {
	return r.ReadInclude(from, name, angled)
}

func TestBuildObjectPreludeRetainsOnlyReferencedHeaderDeclaration(t *testing.T) {
	source := []byte("#include <stdio.h>\nint main(void) { puts(\"PASS\"); return 0; }\n")
	header := []byte("extern int remove (const char *__name);\nextern int puts (const char *__s);\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{name: "stdio.h", src: header})
	if !result.Ok || !bytes.Equal(result.Prelude, []byte("extern int puts ( const char * __s );\n")) {
		t.Fatalf("BuildObjectPrelude = %#v, prelude %q", result, result.Prelude)
	}
}

func TestBuildObjectPreludeReportsMissingHeaderAtDirective(t *testing.T) {
	result := BuildObjectPrelude("/tmp/main.c", []byte("  #include <missing.h>\nint main(void) { return 0; }\n"), headerTestReader{})
	if result.Ok || result.ErrorPath != "missing.h" || result.ErrorAt != 2 {
		t.Fatalf("BuildObjectPrelude missing header = %#v", result)
	}
}

func TestBuildObjectPreludeIgnoresInactiveIncludeWithDependencies(t *testing.T) {
	source := []byte("#if 0\n#include <missing.h>\n#endif\n#include <api.h>\nint main(void) { return answer(); }\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{}, HeaderSource{Path: "/include/api.h", Source: []byte("int answer(void);\n")})
	if !result.Ok || string(result.Prelude) != "int answer ( void );\n" {
		t.Fatalf("prelude = %#v, %q", result, result.Prelude)
	}
}

func TestBuildObjectPreludeAcceptsDefaultExternalLibraryPrototypes(t *testing.T) {
	source := []byte("#include <crypto.h>\nint main(void) { return EVP_DigestUpdate(0, 0, 0); }\n")
	header := []byte("static int private_helper(void);\n#define UNUSED_HELPER() private_helper()\n#endif\n__owur int EVP_DigestUpdate(EVP_MD_CTX *ctx, const void *data, size_t count);\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{name: "crypto.h", src: header})
	want := []byte("int EVP_DigestUpdate ( EVP_MD_CTX * ctx , const void * data , size_t count );\n")
	if !result.Ok || !bytes.Equal(result.Prelude, want) {
		t.Fatalf("BuildObjectPrelude library prototype = %#v, prelude %q", result, result.Prelude)
	}
}

func TestBuildObjectPreludeDropsPostfixHeaderDecorations(t *testing.T) {
	source := []byte("#include <stdio.h>\nint main(void) { return snprintf(0, 0, \"ok\"); }\n")
	header := []byte("extern int snprintf(char *s, size_t n, const char *format, ...) __THROWNL __attribute__((__format__(__printf__, 3, 4)));\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{name: "stdio.h", src: header})
	want := []byte("extern int snprintf ( char * s , size_t n , const char * format , ... );\n")
	if !result.Ok || !bytes.Equal(result.Prelude, want) {
		t.Fatalf("BuildObjectPrelude decorated prototype = %#v, prelude %q", result, result.Prelude)
	}
}

func TestBuildObjectPreludeKeepsDeclarationsInsideCXXLinkageGuard(t *testing.T) {
	source := []byte("#include <api.h>\nint main(void) { return answer(); }\n")
	header := []byte("#ifdef __cplusplus\nextern \"C\" {\n#endif\nint answer(void);\n#ifdef __cplusplus\n}\n#endif\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{name: "api.h", src: header})
	if !result.Ok || string(result.Prelude) != "int answer ( void );\n" {
		t.Fatalf("prelude = %#v, %q", result, result.Prelude)
	}
}

func TestBuildObjectPreludeDoesNotRetainMultilineMacroBody(t *testing.T) {
	source := []byte("#include <rand.h>\nint main(void) { return RAND_bytes(0, 0); }\n")
	header := []byte("#define RAND_cleanup() \\\n+    while (0) \\\n+    continue\n#endif\nint RAND_bytes(unsigned char *buf, int num);\n")
	result := BuildObjectPrelude("/tmp/main.c", source, headerTestReader{name: "rand.h", src: header})
	want := []byte("int RAND_bytes ( unsigned char * buf , int num );\n")
	if !result.Ok || !bytes.Equal(result.Prelude, want) {
		t.Fatalf("BuildObjectPrelude multiline macro = %#v, prelude %q", result, result.Prelude)
	}
}
