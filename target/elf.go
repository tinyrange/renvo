package target

import (
	"debug/elf"
	"fmt"
)

type ELFArtifactOptions struct {
	VectorSymbol string
	HeapSize     uint64
	StackSize    uint64
}

// ArtifactFromELF derives the board-budget input from a linked ELF image. Load
// addresses come from PT_LOAD physical addresses rather than assuming VMA=LMA.
func ArtifactFromELF(path string, options ELFArtifactOptions) (Artifact, error) {
	file, err := elf.Open(path)
	if err != nil {
		return Artifact{}, fmt.Errorf("open linked ELF artifact: %w", err)
	}
	defer file.Close()
	if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN {
		return Artifact{}, fmt.Errorf("ELF artifact type is %s; want a linked executable image", file.Type)
	}
	if options.VectorSymbol == "" {
		return Artifact{}, fmt.Errorf("ELF vector symbol is required")
	}
	vectorAddress, err := linkedELFSymbolAddress(file, options.VectorSymbol)
	if err != nil {
		return Artifact{}, err
	}
	artifact := Artifact{
		Entry:         file.Entry,
		VectorAddress: vectorAddress,
		HeapSize:      options.HeapSize,
		StackSize:     options.StackSize,
	}
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_ALLOC == 0 || section.Size == 0 {
			continue
		}
		converted := Section{Name: section.Name, Address: section.Addr, Size: section.Size, Flags: SectionAlloc}
		if section.Flags&elf.SHF_WRITE != 0 {
			converted.Flags |= SectionWrite
		}
		if section.Flags&elf.SHF_EXECINSTR != 0 {
			converted.Flags |= SectionExec
		}
		if section.Type != elf.SHT_NOBITS {
			loadAddress, ok := elfSectionLoadAddress(file, section)
			if !ok {
				return Artifact{}, fmt.Errorf("allocated ELF section %q is not covered by a file-backed PT_LOAD segment", section.Name)
			}
			converted.LoadAddress = loadAddress
			converted.LoadSize = section.Size
		}
		artifact.Sections = append(artifact.Sections, converted)
	}
	artifact.Imports = linkedELFImports(file)
	return artifact, nil
}

func elfSectionLoadAddress(file *elf.File, section *elf.Section) (uint64, bool) {
	sectionEnd, ok := checkedELFEnd(section.Offset, section.Size)
	if !ok {
		return 0, false
	}
	for _, program := range file.Progs {
		if program.Type != elf.PT_LOAD || section.Offset < program.Off {
			continue
		}
		programEnd, valid := checkedELFEnd(program.Off, program.Filesz)
		if !valid || sectionEnd > programEnd {
			continue
		}
		delta := section.Offset - program.Off
		if program.Paddr > ^uint64(0)-delta {
			return 0, false
		}
		return program.Paddr + delta, true
	}
	return 0, false
}

func linkedELFSymbolAddress(file *elf.File, name string) (uint64, error) {
	groups := make([][]elf.Symbol, 0, 2)
	if symbols, err := file.Symbols(); err == nil {
		groups = append(groups, symbols)
	}
	if symbols, err := file.DynamicSymbols(); err == nil {
		groups = append(groups, symbols)
	}
	for _, symbols := range groups {
		for _, symbol := range symbols {
			if symbol.Name == name && symbol.Section != elf.SHN_UNDEF {
				return symbol.Value, nil
			}
		}
	}
	return 0, fmt.Errorf("ELF vector symbol %q is not defined", name)
}

func linkedELFImports(file *elf.File) []string {
	var imports []string
	groups := make([][]elf.Symbol, 0, 2)
	if symbols, err := file.Symbols(); err == nil {
		groups = append(groups, symbols)
	}
	if symbols, err := file.DynamicSymbols(); err == nil {
		groups = append(groups, symbols)
	}
	for _, symbols := range groups {
		for _, symbol := range symbols {
			if symbol.Name == "" || symbol.Section != elf.SHN_UNDEF || elf.ST_BIND(symbol.Info) == elf.STB_LOCAL || stringInList(imports, symbol.Name) {
				continue
			}
			imports = append(imports, symbol.Name)
		}
	}
	return imports
}

func checkedELFEnd(start uint64, size uint64) (uint64, bool) {
	if start > ^uint64(0)-size {
		return 0, false
	}
	return start + size, true
}
