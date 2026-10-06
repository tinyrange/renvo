package main

import (
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/unit"
	"renvo.dev/targetfrontend"
)

const definition = `definition 1
unit tiny
implements direct_emitter_v1
@import "machine.rtg"
`

const machine = `
go backend {
	func addOne(value int) int { return value + 1 }
	func emitValue(out *RTGEmitter, value int) { out.Int8(value) }
}
arch tiny64 {
	endian = little
	word_bits = 64
	pointer_bits = 64
	reject = [
		move, address,
		load.native, load.i32, load.u32, load.i16, load.u16, load.i8, load.u8,
		store.native, store.u32, store.u16, store.u8,
		add, subtract, multiply, bit_and, bit_or, bit_xor, compare, test,
		increment, decrement,
		shift_left_immediate, shift_right_unsigned_immediate, shift_right_signed_immediate,
		call, call_indirect, jump, jump_condition, set_condition, return, leave,
		host_syscall, move_immediate, variable_shift, signed_divide, copy_bytes
	]
	frontend_operations {
        emit(value:int) {
            lower = go emitValue
            reads = []
            writes = []
            clobbers = []
            memory = none
            control = none
            ordering = ordered
            requires = []
            immediates { value = unsigned 8 }
        }
    }
    frontend_syntax {
        native {
            style = native
            emit(value:int) = emit(value)
        }
    }
	exports { renvoTinyAddOne = go addOne }
}
abi tiny_abi { arch = tiny64 }
runtime tiny_runtime { operation print { builtin = true } }
format tiny_image { address_bits = 64 }
target example/tiny64 {
	family = native_v1
	os = example
	arch = tiny64
	abi = tiny_abi
	runtime = tiny_runtime
	executable = tiny_image
	build_tags = [example, tiny64]
	capabilities = [hosted, executable]
}
`

type machineImportLoader struct{}

func (machineImportLoader) LoadImport(_ string, path string) rtg.ImportSource {
	if path == "machine.rtg" {
		return rtg.ImportSource{
			Source: []byte(machine), Filename: path, Ok: true,
		}
	}
	return rtg.ImportSource{}
}

func appMain(args []string) int {
	return run(args)
}

func run(args []string) int {
	parsed := rtg.ParseImports([]byte(definition), "tiny.rtg", machineImportLoader{})
	if !parsed.Ok {
		if len(parsed.Diagnostics) != 0 {
			print(parsed.Diagnostics[0].Code + ": " + parsed.Diagnostics[0].Message + "\n")
		} else {
			print("PARSE\n")
		}
		return 1
	}
	resolved := rtg.ResolveDefinitions(parsed)
	if !resolved.Ok {
		print("RESOLVE\n")
		return 1
	}
	generated := rtg.GenerateFixedBackend(resolved, "example/tiny64")
	if !generated.Ok || len(generated.Source) == 0 {
		print("GENERATE\n")
		return 1
	}
	vocabulary := rtg.FrontendOperations(resolved, "example/tiny64")
	blocks, diagnostics := rtg.ParseTargetAssembler(vocabulary, "native", []byte(".text\n.globl answer\nanswer: emit 42; emit 7\n"), "answer.s")
	if len(diagnostics) != 0 || len(blocks) != 1 || len(blocks[0].Instructions) != 2 {
		print("ASSEMBLER\n")
		return 1
	}
	encoded, diagnostics := rtg.EncodeTargetAssembly(vocabulary, blocks, "answer.rtgasm")
	if len(diagnostics) != 0 {
		print("ENCODE\n")
		return 1
	}
	typed := rtg.LowerTargetAssembly(resolved, "example/tiny64", rtg.ParseAssembly(encoded, "answer.rtgasm"))
	ordinary := rtg.LowerAssemblerFile(resolved, "example/tiny64", []byte(".globl answer\nanswer: emit 42; emit 7\n"), "answer.s")
	if !typed.Ok || !ordinary.Ok || len(typed.Entries[0].Steps) != 2 || len(ordinary.Entries[0].Steps) != 2 {
		print("MATERIALIZE\n")
		return 1
	}
	_, diagnostics = rtg.ParseTargetInstructionBlock(vocabulary, "native", []byte("emit 256"), "invalid.s", "bad")
	if len(diagnostics) == 0 {
		print("RANGE\n")
		return 1
	}
	raw, ok := unit.MarshalCore(unit.CoreProgram{
		RTGAssembly:      []unit.RTGAssemblySource{{Path: "answer.rtgasm", Source: encoded}},
		RTGAssemblyFuncs: []unit.RTGAssemblyBinding{{Func: 0, Source: 0, Entry: 0}},
	})
	if !ok {
		print("UNIT\n")
		return 1
	}
	generated = rtg.GenerateAssemblyEvaluators(resolved, "example/tiny64", []rtg.AssemblyEvaluatorEntry{{Assembly: typed, EntryIndex: 0}})
	if !generated.Ok {
		print("EVALUATOR\n")
		return 1
	}
	if len(args) == 2 && args[1] == "generate" {
		print(string(generated.Source))
		return 0
	}
	if len(args) != 3 {
		print("ARGUMENTS\n")
		return 1
	}
	compiledSource, err := readFixtureFile(args[1])
	if !err || !sameBytes(compiledSource, generated.Source) {
		print("SOURCE\n")
		return 1
	}
	bytecode, err := readFixtureFile(args[2])
	if !err {
		print("BYTECODE\n")
		return 1
	}
	materialized := targetfrontend.MaterializeAssembly(raw, []byte(definition), "tiny.rtg", "example/tiny64", machineImportLoader{}, suppliedEvaluatorCompiler{Source: compiledSource, Bytecode: bytecode}, 2*1024*1024)
	if !materialized.Ok {
		if len(materialized.Diagnostics) > 0 {
			print(materialized.Diagnostics[0].Message + "\n")
		}
		print("EVALUATE\n")
		return 1
	}
	_, bindings, ok := unit.ReadRTGAssemblyFragments(materialized.Unit)
	if !ok || len(bindings) != 1 || len(bindings[0].Code) != 2 || bindings[0].Code[0] != 42 || bindings[0].Code[1] != 7 {
		print("EMISSION\n")
		return 1
	}
	print("PASS\n")
	return 0
}

// The test supplies an evaluator compiled from this exact generated source by
// the self-hosted compiler. Keeping the compiler capability outside this arena
// exercises the same API as a separately prepared backend, not a fake emitter.
type suppliedEvaluatorCompiler struct{ Source, Bytecode []byte }

func (c suppliedEvaluatorCompiler) CompileAssemblyEvaluator(source []byte) ([]byte, string, bool) {
	return c.Bytecode, "", sameBytes(c.Source, source)
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func readFixtureFile(name string) ([]byte, bool) {
	path := name + "\x00"
	fd := open(path, 0)
	if fd < 0 {
		return nil, false
	}
	var out []byte
	var buffer [4096]byte
	for {
		n := read(fd, buffer[:], -1)
		if n < 0 || len(out)+n > 4*1024*1024 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		out = append(out, buffer[:n]...)
	}
	close(fd)
	return out, true
}
