//go:build !renvo

package driver

import (
	"path/filepath"
	"renvo.dev/internal/c11"
	"testing"
)

func TestTargetCAssemblyTranslation(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	backend := resolveBackendBuildOptions([]string{"-backend", filepath.Join(root, "backend/definitions/linux_amd64.rtg"), "-t", "linux/amd64", "-o", "image", "main.c"}, root, OSFS{})
	if backend.options.CAssemblyCompiler == nil {
		t.Fatal(backend.options)
	}
	compiler := backend.options.CAssemblyCompiler
	sources := []string{
		`long tied=0; long arg(long x){return x;} int main(void){__asm__("":"=r"(tied):"0"(arg(42)));return 0;}`,
		`long a,b; long arg(long x){return x;} int main(void){__asm__("movq %2,%0;movq %3,%1":"=&r"(a),"=&r"(b):"r"(arg(17)),"r"(arg(25)));return 0;}`,
		`long memory,loaded;long arg(long x){return x;}int main(void){__asm__("movq %1,%0":"=m"(memory):"r"(arg(42)):"memory");__asm__("movq %1,%0":"=r"(loaded):"m"(memory));return 0;}`,
		`long immediate;int main(void){__asm__("movq %1,%0":"=r"(immediate):"i"(42));return 0;}`,
		`long x; int main(void){__asm__("addq %1,%0":"+r"(x):"r"(x):"cc");return 0;}`,
		`long branch(long x,long y){__asm__ goto("cmpq %1,%0;je %l[match]"::"r"(x),"r"(y):"cc":match);return 0;match:return 1;} int main(void){return branch(21,21);}`,
		`long count;long value;long *next(void){count++;return &value;}long arg(long x){count++;return x;}int main(void){__asm__("addq %1,%0":"+r"(*next()):"r"(arg(22)):"cc");return 0;}`,
	}
	for i, source := range sources {
		translated := c11.TranslateWithPreludeConfig("main", []byte(source), nil, c11.ObjectConfig{DataModel: c11.DataModelLP64, AssemblyCompiler: compiler, AssemblyNamespace: "test", IsolateGoBuiltins: true})
		if !translated.Ok {
			t.Fatalf("source %d failed: %+v", i, translated)
		}
		if len(translated.Assemblies) == 0 {
			t.Fatalf("source %d emitted no assembly: %s", i, translated.Source)
		}
	}
}
