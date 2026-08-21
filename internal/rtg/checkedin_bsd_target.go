//go:build !renvo

package rtg

func generateCheckedInBSDAmd64Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	operations, missing := checkedInLinuxAmd64RuntimeOperations(target.Runtime)
	if missing != "" {
		return checkedInTargetProjectionFailure(resolved.Document, target.Runtime,
			target.Descriptor.Name+" runtime operation "+missing+" requires a syscall number")
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	prefix := "renvoFreeBSDAmd64"
	body := checkedInFreeBSDAmd64Source
	if target.Descriptor.Name == "openbsd/amd64" {
		prefix = "renvoOpenBSDAmd64"
		body = checkedInOpenBSDAmd64Source
	}
	if target.Descriptor.Name == "netbsd/amd64" {
		prefix = "renvoNetBSDAmd64"
		body = checkedInNetBSDAmd64Source
	}
	for i := 0; i < len(operations); i++ {
		source = append(source, "\nconst "...)
		source = append(source, prefix...)
		suffix := operations[i].symbol[len("renvoLinuxAmd64"):]
		source = append(source, suffix...)
		source = append(source, " = "...)
		source = appendDecimalFrame(source, operations[i].value)
	}
	exit, ok := runtimeOperationInteger(target.Runtime, "exit", "number")
	if !ok || exit <= 0 {
		return checkedInTargetProjectionFailure(resolved.Document, target.Runtime,
			target.Descriptor.Name+" runtime exit requires a syscall number")
	}
	source = append(source, "\nconst "...)
	source = append(source, prefix...)
	source = append(source, "SysExit = "...)
	source = appendDecimalFrame(source, exit)
	source = append(source, '\n')
	source = append(source, body...)
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

const checkedInFreeBSDAmd64Source = `

func compileFreeBSDAmd64(input []int, output int) int {
	return compileFreeBSDAmd64Arena(input, output, 0)
}

func compileFreeBSDAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetFreeBSDAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}
`

const checkedInNetBSDAmd64Source = `

const renvoNetBSDAmd64ELFCodeOffset = 256
const renvoNetBSDAmd64ImageBase = 0x400000

func compileNetBSDAmd64(input []int, output int) int {
	return compileNetBSDAmd64Arena(input, output, 0)
}

func compileNetBSDAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetNetBSDAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}

func renvoAppendNetBSDElfHeaderAmd64(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	start := len(out)
	header := "\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00\x01\x00\x00\x00"
	for i := 0; i < len(header); i++ {
		out = append(out, header[i])
	}
	out = renvoAppendUntil(out, start+64)
	renvoPut32At(out, start+24, renvoNetBSDAmd64ImageBase+entryOff)
	renvoPut32At(out, start+32, 64)
	renvoPut32At(out, start+40, shoff)
	out[start+52] = 64
	out[start+54] = 56
	out[start+56] = 3
	if shoff != 0 {
		out[start+58] = 64
		out[start+60] = 7
		out[start+62] = 6
	}
	out = renvoAppendElf64Program(out, 1, 5, 0, renvoNetBSDAmd64ImageBase, fileSize, fileSize, 0x1000)
	out = renvoAppendElf64Program(out, 1, 6, bssOffset, renvoNetBSDAmd64ImageBase+bssOffset, 0, bssSize, 0x1000)
	out = renvoAppendElf64Program(out, 4, 4, 232, renvoNetBSDAmd64ImageBase+232, 24, 24, 4)
	out = renvoAppend32(out, 7)
	out = renvoAppend32(out, 4)
	out = renvoAppend32(out, 1)
	out = append(out, 'N', 'e', 't', 'B', 'S', 'D', 0, 0)
	return renvoAppend32(out, 1100000000)
}
`

const checkedInOpenBSDAmd64Source = `

const renvoOpenBSDAmd64ELFCodeOffset = 432
const renvoOpenBSDPTSyscalls = 0x65a3dbe9

func compileOpenBSDAmd64(input []int, output int) int {
	return compileOpenBSDAmd64Arena(input, output, 0)
}

func compileOpenBSDAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetOpenBSDAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}

func renvoAppendOpenBSDElfHeaderAmd64(out []byte, entryOff int, dataOffset int, fileSize int, bssOffset int, bssSize int, shoff int, syscallTableOff int, syscallTableSize int) []byte {
	start := len(out)
	header := "\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x3e\x00\x01\x00\x00\x00"
	for i := 0; i < len(header); i++ {
		out = append(out, header[i])
	}
	out = renvoAppendUntil(out, start+64)
	renvoPut32At(out, start+24, entryOff)
	renvoPut32At(out, start+32, 64)
	renvoPut32At(out, start+40, shoff)
	out[start+52] = 64
	out[start+54] = 56
	out[start+56] = 6
	if shoff != 0 {
		out[start+58] = 64
		out[start+60] = 7
		out[start+62] = 6
	}
	out = renvoAppendElf64Program(out, 6, 4, 64, 64, 336, 336, 8)
	dataFileSize := fileSize - dataOffset
	out = renvoAppendElf64LoadProgram(out, 5, 0, 0, dataOffset, dataOffset)
	out = renvoAppendElf64LoadProgram(out, 4, dataOffset, dataOffset, dataFileSize, dataFileSize)
	out = renvoAppendElf64LoadProgram(out, 6, bssOffset, bssOffset, 0, bssSize)
	out = renvoAppendElf64Program(out, 4, 4, 400, 400, 24, 24, 4)
	out = renvoAppendElf64Program(out, renvoOpenBSDPTSyscalls, 4, syscallTableOff, 0, syscallTableSize, syscallTableSize, 4)
	out = renvoAppendOpenBSDIdentNote(out)
	return renvoAppendUntil(out, start+renvoOpenBSDAmd64ELFCodeOffset)
}

func renvoAppendOpenBSDIdentNote(out []byte) []byte {
	out = renvoAppend32(out, 8)
	out = renvoAppend32(out, 4)
	out = renvoAppend32(out, 1)
	out = append(out, 'O', 'p', 'e', 'n', 'B', 'S', 'D', 0)
	return renvoAppend32(out, 0)
}

func renvoAppendOpenBSDSyscallTable(out []byte, a *renvoAsm) []byte {
	for i := 0; i+1 < len(a.openbsdSyscalls); i += 2 {
		label := a.openbsdSyscalls[i]
		if label < 0 || label >= len(a.labelPos) || a.labelPos[label] < 0 {
			a.patchFailed = true
			return out
		}
		out = renvoAppend32(out, a.codeOffset+int(a.labelPos[label]))
		out = renvoAppend32(out, a.openbsdSyscalls[i+1])
	}
	return out
}
`
