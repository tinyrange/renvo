package main

// renvo:linkstatic /usr/lib/system/libcommonCrypto.dylib,CC_SHA256
func renvoDarwinCCSHA256(data []byte, length int, digest []byte) int { return 0 }

func renvoRTGSHA256(data []byte, digest []byte) bool {
	return renvoDarwinCCSHA256(data, len(data), digest) != 0
}

func renvo_runtime_ArenaPersistString(value string) string { return value }

const renvoDarwinImportRead = 4
const renvoDarwinImportWrite = 5
const renvoDarwinImportPread = 6
const renvoDarwinImportPwrite = 7

func compileDarwinArm64(input []int, output int) int {
	return compileDarwinArm64Arena(input, output, 0)
}

func compileDarwinArm64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetDarwinArm64)
	return renvoCompileAarch64(input, output, arenaSize)
}

func renvoAsmAddDarwinStaticImport(a *renvoAsm, dylib string, name string) int {
	if len(name) == 0 || name[0] != '_' {
		name = renvo_runtime_ArenaPersistString("_" + name)
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if a.darwinImports[i].dylib == dylib && a.darwinImports[i].name == name {
			a.darwinImports[i].used = true
			return i
		}
	}
	label := renvoAsmNewLabel(a)
	a.darwinImports = append(a.darwinImports, renvoDarwinStaticImport{dylib: dylib, name: name, label: label, used: true})
	return len(a.darwinImports) - 1
}

func renvoDarwinArm64EmitGetdirentries(a *renvoAsm) {
	importIndex := renvoAsmAddDarwinStaticImport(
		a, "/usr/lib/libSystem.B.dylib", "_getdirentries")
	renvoAsmCallLabel(a, a.darwinImports[importIndex].label)
}
