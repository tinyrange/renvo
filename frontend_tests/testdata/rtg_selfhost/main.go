package main

import "renvo.dev/internal/rtg"

const definition = `definition 1
unit tiny
implements direct_emitter_v1
@import "machine.rtg"
`

const machine = `
go backend {
	func addOne(value int) int { return value + 1 }
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

func main() {
	parsed := rtg.ParseImports([]byte(definition), "tiny.rtg", machineImportLoader{})
	if !parsed.Ok {
		if len(parsed.Diagnostics) != 0 {
			print(parsed.Diagnostics[0].Code + ": " + parsed.Diagnostics[0].Message + "\n")
		} else {
			print("PARSE\n")
		}
		return
	}
	resolved := rtg.ResolveDefinitions(parsed)
	if !resolved.Ok {
		print("RESOLVE\n")
		return
	}
	generated := rtg.GenerateFixedBackend(resolved, "example/tiny64")
	if !generated.Ok || len(generated.Source) == 0 {
		print("GENERATE\n")
		return
	}
	print("PASS\n")
}
