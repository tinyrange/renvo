package bringup

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"
)

type ELFContract struct {
	Class              elf.Class
	Data               elf.Data
	Machine            elf.Machine
	RequiredExports    []string
	AllowedUndefined   []string
	AllowedRelocations []uint32
}

type ObjectViolation struct {
	Code    string
	Subject string
	Detail  string
}

func (v ObjectViolation) Error() string {
	if v.Subject == "" {
		return v.Code + ": " + v.Detail
	}
	return v.Code + " " + v.Subject + ": " + v.Detail
}

type ObjectValidation struct {
	Violations []ObjectViolation
}

func (v ObjectValidation) OK() bool {
	return len(v.Violations) == 0
}

func ValidateELFObject(path string, contract ELFContract) ObjectValidation {
	var result ObjectValidation
	file, err := elf.Open(path)
	if err != nil {
		result.add("malformed-object", path, err.Error())
		return result
	}
	defer file.Close()
	if file.Type != elf.ET_REL {
		result.add("object-type", path, fmt.Sprintf("ELF type is %s; want ET_REL", file.Type))
	}
	if contract.Class != elf.ELFCLASSNONE && file.Class != contract.Class {
		result.add("object-class", path, fmt.Sprintf("ELF class is %s; want %s", file.Class, contract.Class))
	}
	if contract.Data != elf.ELFDATANONE && file.Data != contract.Data {
		result.add("object-endian", path, fmt.Sprintf("ELF data encoding is %s; want %s", file.Data, contract.Data))
	}
	if contract.Machine != elf.EM_NONE && file.Machine != contract.Machine {
		result.add("object-machine", path, fmt.Sprintf("ELF machine is %s; want %s", file.Machine, contract.Machine))
	}
	for _, section := range file.Sections {
		if section.Addralign != 0 && !powerOfTwo(section.Addralign) {
			result.add("section-alignment", section.Name, fmt.Sprintf("alignment %d is not a power of two", section.Addralign))
		}
		if section.Type == elf.SHT_REL || section.Type == elf.SHT_RELA {
			validateRelocationSection(&result, file, section, contract.AllowedRelocations)
		}
	}
	symbols, symbolErr := file.Symbols()
	if symbolErr != nil {
		result.add("symbol-table", path, symbolErr.Error())
		return result
	}
	defined := make(map[string]bool)
	for _, symbol := range symbols {
		if symbol.Name == "" {
			continue
		}
		if symbol.Section == elf.SHN_UNDEF {
			if elf.ST_BIND(symbol.Info) != elf.STB_LOCAL && !containsString(contract.AllowedUndefined, symbol.Name) {
				result.add("unresolved-symbol", symbol.Name, "not supplied by the trusted target shell")
			}
			continue
		}
		if elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL || elf.ST_BIND(symbol.Info) == elf.STB_WEAK {
			defined[symbol.Name] = true
		}
	}
	for _, name := range contract.RequiredExports {
		if !defined[name] {
			result.add("missing-export", name, "required milestone symbol is not globally defined")
		}
	}
	return result
}

func validateRelocationSection(result *ObjectValidation, file *elf.File, section *elf.Section, allowed []uint32) {
	if int(section.Link) >= len(file.Sections) || file.Sections[section.Link].Type != elf.SHT_SYMTAB {
		result.add("relocation-link", section.Name, "does not reference a static symbol table")
	}
	if int(section.Info) >= len(file.Sections) {
		result.add("relocation-target", section.Name, "target section index is out of range")
	}
	data, err := section.Data()
	if err != nil {
		result.add("relocation-data", section.Name, err.Error())
		return
	}
	entrySize := int(section.Entsize)
	if entrySize == 0 {
		entrySize = relocationEntrySize(file.Class, section.Type)
	}
	if entrySize <= 0 || len(data)%entrySize != 0 {
		result.add("relocation-size", section.Name, fmt.Sprintf("%d bytes is not a whole number of relocation entries", len(data)))
		return
	}
	if len(allowed) == 0 {
		return
	}
	for offset := 0; offset < len(data); offset += entrySize {
		kind, ok := relocationType(file.Class, file.ByteOrder, data[offset:offset+entrySize])
		if !ok {
			result.add("relocation-size", section.Name, "entry is too short for r_info")
			return
		}
		if !containsUint32(allowed, kind) {
			result.add("unsupported-relocation", section.Name, fmt.Sprintf("relocation type %d is not in the milestone contract", kind))
		}
	}
}

func relocationEntrySize(class elf.Class, typ elf.SectionType) int {
	if class == elf.ELFCLASS32 {
		if typ == elf.SHT_RELA {
			return 12
		}
		return 8
	}
	if class == elf.ELFCLASS64 {
		if typ == elf.SHT_RELA {
			return 24
		}
		return 16
	}
	return 0
}

func relocationType(class elf.Class, order binary.ByteOrder, entry []byte) (uint32, bool) {
	if class == elf.ELFCLASS32 {
		if len(entry) < 8 {
			return 0, false
		}
		return order.Uint32(entry[4:8]) & 0xff, true
	}
	if class == elf.ELFCLASS64 {
		if len(entry) < 16 {
			return 0, false
		}
		return uint32(order.Uint64(entry[8:16])), true
	}
	return 0, false
}

func (v *ObjectValidation) add(code string, subject string, detail string) {
	v.Violations = append(v.Violations, ObjectViolation{Code: code, Subject: subject, Detail: detail})
}

func powerOfTwo(value uint64) bool {
	return value != 0 && value&(value-1) == 0
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsUint32(values []uint32, value uint32) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func WriteValidationReport(path string, validation ObjectValidation) error {
	var data []byte
	if validation.OK() {
		data = []byte("valid relocatable object\n")
	} else {
		for _, violation := range validation.Violations {
			data = append(data, violation.Error()...)
			data = append(data, '\n')
		}
	}
	return os.WriteFile(path, data, 0o644)
}
