//go:build !renvo

package backendjit

import (
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"runtime"
	"testing"
)

func TestCompilerJITCExtendedAssemblyRuntimeContracts(t *testing.T) {
	if hostTarget() == "" {
		t.Skip("no host backend")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	definition := filepath.Join(root, "backend/definitions/linux_amd64.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
	path := filepath.Join(project, "main.c")
	source := []byte(`long count;
long value=20;
long arg(long x){count++;return x;}
long *next(void){count++;return &value;}
long with_output(long *value,long target){
 __asm__ goto("addq %1,%0;cmpq %2,%0;je %l[done]" : "+r"(*value) : "r"(22L),"r"(target) : "cc" : done);
 return 0;
 done:return 1;
}
long branch(long x,long y){
 __asm__ goto("cmpq %1,%0; je %l[match]" : : "r"(x), "r"(y) : "cc" : match);
 return 0;
 match: return 1;
}
int main(void){
 __asm__ volatile("addq %[input],%[result]" : [result] "+r"(*next()) : [input] "r"(arg(22)) : "cc");
 if(value!=42 || count!=2)return 1;
 long tied=0;
 __asm__("" : "=r"(tied) : "0"(arg(42)));
 if(tied!=42 || count!=3)return 2;
 long early=0;
 __asm__("movq %1,%0; addq %2,%0" : "=&r"(early) : "r"(arg(20)),"r"(arg(22)) : "cc");
 if(early!=42 || count!=5)return 3;
 long a=0,b=0;
 __asm__("movq %2,%0;movq %3,%1" : "=&r"(a),"=&r"(b) : "r"(arg(17)),"r"(arg(25)));
 if(a+b!=42 || count!=7)return 4;
 long memory=0,loaded=0;
 __asm__("movq %1,%0" : "=m"(memory) : "r"(arg(42)) : "memory");
 __asm__("movq %1,%0" : "=r"(loaded) : "m"(memory));
 if(memory!=42 || loaded!=42 || count!=8)return 5;
 long immediate=0;
 __asm__("movq %1,%0" : "=r"(immediate) : "i"(42));
 if(immediate!=42)return 6;
 if(branch(21,21)!=1 || branch(20,22)!=0)return 7;
 long out=20;
 if(with_output(&out,42)!=1 || out!=42)return 8;
 out=21;
 if(with_output(&out,42)!=0 || out!=43)return 9;
 struct { int value; int guard; } pair={20,1234567};
 __asm__("addl %1,%0" : "+r"(pair.value) : "r"((int)22) : "cc");
 if(pair.value!=42 || pair.guard!=1234567)return 10;
 unsigned char byte=20,byteGuard=83;
 __asm__("addb %b1,%b0" : "+r"(byte) : "D"((unsigned char)22) : "cc");
 if(byte!=42 || byteGuard!=83)return 11;
 short half=-20,halfGuard=1234;
 __asm__("addw %w1,%w0" : "+r"(half) : "r"((short)62) : "cc");
 if(half!=42 || halfGuard!=1234)return 12;
 int memory32=0,loaded32=0;
 __asm__("movl %k1,%0" : "=m"(memory32) : "r"((int)42) : "memory");
 __asm__("movl %1,%k0" : "=r"(loaded32) : "m"(memory32));
 if(memory32!=42 || loaded32!=42)return 13;
 signed char negative=-20;
 __asm__("addq %1,%q0" : "+r"(negative) : "r"((long)62) : "cc");
 if(negative!=42)return 14;
 unsigned char small=0;
 __asm__("movb %1,%0" : "=D"(small) : "i"(42));
 if(small!=42)return 15;
 __asm__("movl $1,%%ecx" ::: "ecx");
 __asm__ volatile("" ::: "memory");
 return 0;
}
`)
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	args := driver.NormalizeCCompilerCommand([]string{"renvo", "cc", "-backend", definition, "-t", "linux/amd64", "-s", "-o", "image", path})
	result := driver.CompileFromFS(args[1:], root, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("C assembly compilation: %+v build=%+v", result.Diagnostic, result.Build.Diagnostic)
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		image := filepath.Join(project, "image")
		if err := os.WriteFile(image, result.Binary, 0700); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command(image).CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("C runtime constraints/captures/goto: %s %v", output, err)
		}
	}
	for _, bad := range []string{
		`int main(void){__asm__("":::"ecx","rcx");return 0;}`,
		`int main(void){__asm__("":::"ebx");return 0;}`,
		`int x; int main(void){__asm__("movq %1,%0":"=r"(x):"r"((int)42));return 0;}`,
		`unsigned char x; int main(void){__asm__("movb $256,%0":"=r"(x));return 0;}`,
		`int x; long y; int main(void){__asm__("":"=r"(x):"0"(y));return 0;}`,
		`long x; int main(void){__asm__("movq %%rbx,%0":"=r"(x));return 0;}`,
		`long x; int main(void){__asm__("addq %1,%0":"+r"(x):"r"(x));return 0;}`,
		`long x; int main(void){__asm__("ret"::);return 0;}`,
		`long x; int main(void){__asm__("movq %1,%0":"=r"(x):"i"(arg(42)));return 0;} long arg(long x){return x;}`,
	} {
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		rejected := driver.CompileFromFS(args[1:], root, filepath.Join(root, "std"), driver.OSFS{}, backend)
		if rejected.Ok {
			t.Fatal("accepted", bad)
		}
	}
}
