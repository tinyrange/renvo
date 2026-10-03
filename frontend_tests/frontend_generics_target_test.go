package frontend_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"renvo.dev/internal/perfgate"
	"renvo.dev/internal/targetinfo"
	"renvo.dev/std/vm"
)

func TestFrontendGenericsTargetConstantWidths(t *testing.T) {
	root := repoRoot(t)
	for _, frontend := range []struct {
		name   string
		config frontendConfig
	}{
		{"host", frontendCompiler(t, root)},
		{"stage3", selfHostedFrontendCompiler(t, root)},
	} {
		t.Run(frontend.name, func(t *testing.T) {
			for _, descriptor := range targetinfo.All() {
				if !descriptor.Advertised || descriptor.Virtual {
					continue
				}
				target := descriptor.Name
				t.Run(target, func(t *testing.T) {
					for _, test := range []struct {
						name, source string
						fits         bool
					}{
						{"int", `func Value[T ~int]()T{return 1<<31};func main(){}`, descriptor.WordBits > 32},
						{"uint", `func Value[T ~uint]()T{return 1<<32};func main(){}`, descriptor.WordBits > 32},
						{"uintptr", `func Value[T ~uintptr]()T{return 1<<32};func main(){}`, descriptor.PointerBits > 32},
						{"int64", `func Value[T ~int64]()T{return 1<<31};func main(){}`, true},
						{"array_length", `type Value[T any][1<<32]T;func main(){}`, descriptor.WordBits > 32},
						{"typed_index", `func Value[S ~[]byte](s S)byte{return s[int64(1<<32)]};func main(){}`, descriptor.WordBits > 32},
						{"slice_bound", `func Value[S ~[]byte](s S)S{return s[:uint64(1<<32)]};func main(){}`, descriptor.WordBits > 32},
						{"make_length", `func Value[S ~[]byte]()S{return make(S,1<<32)};func main(){}`, descriptor.WordBits > 32},
					} {
						t.Run(test.name, func(t *testing.T) {
							dir := t.TempDir()
							if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/width\ngo 1.25\n"), 0644); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main;"+test.source), 0644); err != nil {
								t.Fatal(err)
							}
							command := frontendCommand(frontend.config, "-t", target, "-emit-unit", "-o", filepath.Join(dir, "program.unit"), ".")
							command.Dir = dir
							command.Env = frontendCommandEnv(frontend.config.env, dir)
							output, err := command.CombinedOutput()
							if test.fits {
								if err != nil {
									t.Fatalf("compile: %v\n%s", err, output)
								}
							} else if err == nil || !strings.Contains(string(output), "RENVO-CHECK-042") {
								t.Fatalf("expected target-width diagnostic: %v\n%s", err, output)
							}
						})
					}
				})
			}
		})
	}
}

func TestFrontendGenericsWideArrayInstantiation(t *testing.T) {
	root := repoRoot(t)
	for _, frontend := range []struct {
		name   string
		config frontendConfig
	}{
		{"host", frontendCompiler(t, root)},
		{"stage3", selfHostedFrontendCompiler(t, root)},
	} {
		t.Run(frontend.name, func(t *testing.T) {
			for _, target := range []string{"linux/amd64", "linux/aarch64"} {
				t.Run(target, func(t *testing.T) {
					dir := t.TempDir()
					if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/widearray\ngo 1.25\n"), 0644); err != nil {
						t.Fatal(err)
					}
					source := `package main
import "unsafe"
type Large[T any] [1<<32]T
func Count[T any](p *Large[T])int{return len(p)}
type Wrapped[T any] struct{Prefix byte;Array Large[T];Suffix int}
func WrappedCount[T any](p *Wrapped[T])int{return len(p.Array)}
func Materialize[T any]()int{var array Large[struct{}];array[1<<32-1]=struct{}{};if unsafe.Sizeof(array)!=0{panic(3)};if _,ok:=any(array).(Large[struct{}]);!ok{panic(4)};values:=make([]struct{},1<<32);values=append(([]struct{})(nil),values...);if len(values)!=1<<32||cap(values)<len(values){panic(5)};return len(array)}
func main(){if Count[struct{}](nil)!=1<<32||Count[byte](nil)!=1<<32||Count[int](nil)!=1<<32||Count[string](nil)!=1<<32{panic(1)};if WrappedCount[struct{}](nil)!=1<<32||WrappedCount[byte](nil)!=1<<32||WrappedCount[int](nil)!=1<<32||WrappedCount[string](nil)!=1<<32{panic(2)};if Materialize[int]()!=1<<32{panic(6)};println("PASS")}
`
					if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0644); err != nil {
						t.Fatal(err)
					}
					outputPath := filepath.Join(dir, "program")
					command := frontendCommand(frontend.config, "-t", target, "-s", "-o", outputPath, ".")
					command.Dir = dir
					command.Env = frontendCommandEnv(frontend.config.env, dir)
					if output, err := command.CombinedOutput(); err != nil {
						t.Fatalf("compile instantiated wide array: %v\n%s", err, output)
					}
					if target == frontend.config.target {
						output, err := exec.Command(outputPath).CombinedOutput()
						if err != nil || string(output) != "PASS\n" {
							t.Fatalf("run instantiated wide array: %v output=%q", err, output)
						}
					}
				})
			}
		})
	}
}

func TestFrontendGenericsVMTarget(t *testing.T) {
	root := repoRoot(t)
	for _, frontend := range []struct {
		name   string
		config frontendConfig
	}{
		{"host", frontendCompiler(t, root)},
		{"stage3", selfHostedFrontendCompiler(t, root)},
	} {
		t.Run(frontend.name, func(t *testing.T) {
			for _, name := range []string{"generic_functions", "generic_alias_scope", "generic_builtin_identity", "generic_constants", "generic_complex_types", "generic_constraint_values", "generic_embedded_pointer_paths", "generic_embedded_identity", "generic_function_conversions", "generic_function_interfaces", "generic_function_initializers", "generic_callable_layout", "generic_variable_groups", "generic_nested_callbacks", "generic_namespace_aliases", "generic_intrinsic_aliases", "generic_tuple_callbacks", "generic_runtime_os", "generic_map_types", "generic_range_functions", "generic_defer_callbacks", "generic_range_defers", "generic_import_capture", "generic_dot_import_identity", "generic_aggregate_callbacks", "generic_aggregate_members", "generic_aggregate_aliases", "generic_ordinary_bodies", "generic_predeclared_shadow", "generic_predeclared_embedding", "generic_promoted_methods", "generic_unsafe", "generic_unsafe_layout", "generic_zero_layout", "generic_reflection", "generic_type_identity", "generic_upstream_cmp", "generic_upstream_slices"} {
				t.Run(name, func(t *testing.T) {
					dir := filepath.Join(root, "frontend_tests", "regressions", name)
					output := filepath.Join(t.TempDir(), "program.rnvb")
					command := frontendCommand(frontend.config, "-t", "vm/vm32", "-s", "-o", output, "./cmd/app")
					command.Dir = dir
					command.Env = frontendCommandEnv(frontend.config.env, dir)
					if result, err := command.CombinedOutput(); err != nil {
						t.Fatalf("compile: %v\n%s", err, result)
					}
					program, err := os.ReadFile(output)
					if err != nil {
						t.Fatal(err)
					}
					want, err := os.ReadFile(filepath.Join(dir, "expected.txt"))
					if err != nil {
						t.Fatal(err)
					}
					policy := perfgate.Load()
					result := vm.RunConfig(program, vm.Config{Limits: vm.Limits{Steps: int(policy.VMStepLimit), Memory: int(policy.PeakMemoryBytes)}, Args: []string{"program"}})
					if result.Trap != vm.TrapNone || result.ExitCode != 0 || !bytes.Equal(result.Output, want) {
						t.Fatalf("VM: trap=%d exit=%d output=%q stderr=%q", result.Trap, result.ExitCode, result.Output, result.Stderr)
					}
				})
			}
		})
	}
}

func TestFrontendGenericsLanguageVersions(t *testing.T) {
	root := repoRoot(t)
	for _, frontend := range []struct {
		name   string
		config frontendConfig
	}{
		{"host", frontendCompiler(t, root)},
		{"stage3", selfHostedFrontendCompiler(t, root)},
	} {
		t.Run(frontend.name, func(t *testing.T) {
			for _, test := range []struct{ name, source, before, since string }{
				{"interface_term", `type C interface{int};func main(){}`, "1.17", "1.18"},
				{"interface_approximation", `type C interface{~int};func main(){}`, "1.17", "1.18"},
				{"interface_union", `type A interface{};type C interface{A|A};func main(){}`, "1.17", "1.18"},
				{"function", `func Id[T any](v T)T{return v};func main(){_=Id(1)}`, "1.17", "1.18"},
				{"type", `type Box[T any] struct{Value T};func main(){var b Box[int];_=b}`, "1.17", "1.18"},
				{"alias", `type Box[T any] = []T;func main(){var b Box[int];_=b}`, "1.23", "1.24"},
				{"comparable", `func F[T comparable](){};func main(){F[any]()}`, "1.19", "1.20"},
				{"function_value", `func Id[T any](v T)T{return v};func main(){var f func(int)int=Id;_=f}`, "1.20", "1.21"},
				{"argument_value", `func Id[T any](v T)T{return v};func Apply[T any](f func(T)T,v T)T{return f(v)};func main(){_=Apply(Id,1)}`, "1.20", "1.21"},
				{"interface_inference", `type V int;func(v V)Value()int{return int(v)};func F[T any](v interface{Value()T})T{return v.Value()};func main(){_=F(V(1))}`, "1.20", "1.21"},
				{"min", `func F[T ~int](a,b T)T{return min(a,b)};func main(){_=F(1,2)}`, "1.20", "1.21"},
				{"clear", `func F[T any](s []T){clear(s)};func main(){F([]int{})}`, "1.20", "1.21"},
			} {
				for _, version := range []string{test.before, test.since} {
					t.Run(test.name+"/"+version, func(t *testing.T) {
						dir := t.TempDir()
						if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/version\ngo "+version+"\n"), 0644); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main;"+test.source), 0644); err != nil {
							t.Fatal(err)
						}
						command := frontendCommand(frontend.config, "-emit-unit", "-o", filepath.Join(dir, "program.unit"), ".")
						command.Dir = dir
						command.Env = frontendCommandEnv(frontend.config.env, dir)
						output, err := command.CombinedOutput()
						if version == test.since {
							if err != nil {
								t.Fatalf("compile: %v\n%s", err, output)
							}
						} else if err == nil || !strings.Contains(string(output), "RENVO-CHECK-042") {
							t.Fatalf("expected language-version diagnostic: %v\n%s", err, output)
						}
					})
				}
			}
		})
	}
}
