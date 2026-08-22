package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendCObjectLinux386Xv6ABISystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectLinux386Xv6ABISystemLink(t, frontendCompiler(t, root))
}

func TestFrontendStage3CObjectLinux386Xv6ABISystemLink(t *testing.T) {
	root := repoRoot(t)
	runCObjectLinux386Xv6ABISystemLink(t, selfHostedFrontendCompiler(t, root))
}

func runCObjectLinux386Xv6ABISystemLink(t *testing.T, frontend frontendConfig) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" && runtime.GOARCH != "386" {
		t.Skipf("Linux/i386 C object execution requires an x86 Linux host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("system GCC is unavailable")
	}
	linker, err := exec.LookPath("ld")
	if err != nil {
		t.Skip("system linker is unavailable")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "xv6-abi.c")
	harness := filepath.Join(dir, "harness.c")
	object := filepath.Join(dir, "xv6-abi.o")
	harnessObject := filepath.Join(dir, "harness.o")
	executable := filepath.Join(dir, "xv6-abi-test")
	if err := os.WriteFile(source, []byte(`
typedef unsigned int uint;
typedef __builtin_va_list va_list;

struct inode;
struct proc { struct inode *cwd; };
struct cpu { struct proc *proc; };
struct spinlock {
	uint locked;
	char *name;
	struct cpu *cpu;
	uint pcs[10];
};
struct sleeplock {
	uint locked;
	struct spinlock lk;
	char *name;
	int pid;
};
struct inode {
	uint dev;
	uint inum;
	int ref;
	struct sleeplock lock;
	int valid;
};

void set_inode_valid(struct inode *inode, int valid) { inode->valid = valid; }

int byte_distance(unsigned char *left, unsigned char *right) {
	return left - right;
}

uint exchange(volatile uint *address, uint replacement) {
	uint result;
	asm volatile("lock; xchgl %0, %1" :
		"+m" (*address), "=a" (result) :
		"1" (replacement) :
		"cc");
	return result;
}

uint variadic_words(uint fixed, ...) {
	va_list arguments;
	uint word;
	unsigned long long wide;
	__builtin_va_start(arguments, fixed);
	word = __builtin_va_arg(arguments, uint);
	wide = __builtin_va_arg(arguments, unsigned long long);
	__builtin_va_end(arguments);
	return fixed + word + (uint)wide + (uint)(wide >> 32);
}

int main(int argc, char **argv) {
	return argc == 2 && argv[0][0] == 'x' && argv[1][0] == 'v' ? 0 : 1;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(harness, []byte(`
typedef unsigned int uint;

struct inode;
struct proc { struct inode *cwd; };
struct cpu { struct proc *proc; };
struct spinlock {
	uint locked;
	char *name;
	struct cpu *cpu;
	uint pcs[10];
};
struct sleeplock {
	uint locked;
	struct spinlock lk;
	char *name;
	int pid;
};
struct inode {
	uint dev;
	uint inum;
	int ref;
	struct sleeplock lock;
	int valid;
};

extern void set_inode_valid(struct inode *, int);
extern int byte_distance(unsigned char *, unsigned char *);
extern uint exchange(volatile uint *, uint);
extern uint variadic_words(uint, ...);
extern int main(int, char **);

static struct inode inode;
static unsigned char bytes[16];
static uint atomic_word = 7;
static char argument0[] = "xv6-abi-test";
static char argument1[] = "vm";
static char *arguments[] = { argument0, argument1, (void *)0 };

static void exit_process(int status) {
	asm volatile("int $0x80" : : "a" (1), "b" (status) : "memory");
	__builtin_unreachable();
}

void _start(void) {
	int failed = 0;
	uint variadic_result;
	set_inode_valid(&inode, 99);
	if ((char *)&inode.valid - (char *)&inode != 76 || inode.valid != 99)
		failed |= 1;
	if (byte_distance(&bytes[11], &bytes[4]) != 7)
		failed |= 2;
	if (exchange(&atomic_word, 41) != 7 || atomic_word != 41)
		failed |= 4;
	variadic_result = variadic_words(1, 2, ((unsigned long long)17 << 32) | 5);
	if (variadic_result != 25)
		failed |= 8;
	if (main(2, arguments) != 0)
		failed |= 16;
	exit_process(failed);
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	command := frontendCommand(frontend, "cc", "-m32", "-c", filepath.Base(source), "-o", object)
	command.Dir = dir
	command.Env = frontendCommandEnv(frontend.env, dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile Linux/i386 xv6 ABI object with Renvo: %v\n%s", err, output)
	}
	hostCompile := exec.Command(gcc, "-m32", "-ffreestanding", "-fno-builtin", "-fno-pic", "-fno-pie",
		"-fno-stack-protector", "-nostdlib", "-c", harness, "-o", harnessObject)
	if output, err := hostCompile.CombinedOutput(); err != nil {
		t.Fatalf("compile Linux/i386 ABI harness: %v\n%s", err, output)
	}
	link := exec.Command(linker, "-m", "elf_i386", "-e", "_start", harnessObject, object, "-o", executable)
	if output, err := link.CombinedOutput(); err != nil {
		t.Fatalf("system-link Linux/i386 xv6 ABI object: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run Linux/i386 xv6 ABI object: %v, output %q", err, output)
	}
}
