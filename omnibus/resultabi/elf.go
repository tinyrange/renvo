package resultabi

import (
	"debug/elf"
	"fmt"
	"os"
)

// ELFSymbolAddress locates the authoritative result symbol in an ELF artifact.
func ELFSymbolAddress(path string, name string) (uint64, error) {
	file, err := elf.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open ELF artifact: %w", err)
	}
	defer file.Close()
	groups := make([][]elf.Symbol, 0, 2)
	if symbols, symbolErr := file.Symbols(); symbolErr == nil {
		groups = append(groups, symbols)
	}
	if symbols, symbolErr := file.DynamicSymbols(); symbolErr == nil {
		groups = append(groups, symbols)
	}
	for _, symbols := range groups {
		for _, symbol := range symbols {
			if symbol.Name == name {
				if symbol.Size != 0 && symbol.Size < Size {
					return 0, fmt.Errorf("ELF symbol %q is %d bytes; need %d", name, symbol.Size, Size)
				}
				return symbol.Value, nil
			}
		}
	}
	return 0, fmt.Errorf("ELF symbol %q not found", name)
}

// DecodeMemoryDump locates SymbolName in artifact and decodes its bytes from a
// memory dump whose first byte corresponds to baseAddress.
func DecodeMemoryDump(artifact string, memoryDump string, baseAddress uint64, symbol string) (Snapshot, error) {
	address, err := ELFSymbolAddress(artifact, symbol)
	if err != nil {
		return Snapshot{}, err
	}
	if address < baseAddress {
		return Snapshot{}, errAddressBeforeBase
	}
	offset := address - baseAddress
	data, err := os.ReadFile(memoryDump)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read memory dump: %w", err)
	}
	if offset > uint64(len(data)) || uint64(len(data))-offset < Size {
		return Snapshot{}, fmt.Errorf("result symbol %#x is outside memory dump base %#x (%d bytes)", address, baseAddress, len(data))
	}
	return Decode(data[offset : offset+Size])
}
