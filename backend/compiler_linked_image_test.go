package main

import (
	"testing"

	"renvo.dev/internal/linkedimage"
)

func TestBackendLinkedImageAcrossExecutableTargets(t *testing.T) {
	savedTarget := renvoTarget
	savedOS := renvoTargetOS
	savedArch := renvoTargetArch
	savedIntSize := renvoNativeIntSize
	savedStrip := renvoCompilerStripSymbols
	savedEmitImage := renvoCompilerEmitImage
	savedSubsystem := renvoCompilerWindowsSubsystem
	defer func() {
		renvoTarget = savedTarget
		renvoTargetOS = savedOS
		renvoTargetArch = savedArch
		renvoNativeIntSize = savedIntSize
		renvoCompilerStripSymbols = savedStrip
		renvoCompilerEmitImage = savedEmitImage
		renvoCompilerWindowsSubsystem = savedSubsystem
	}()
	source := []byte("package main\nvar renvo_repl_storage_1 = 7\nfunc appMain() int { print(\"PASS\\n\"); return renvo_repl_storage_1 - 7 }\n")
	targets := []string{
		"linux/amd64",
		"linux/386",
		"linux/aarch64",
		"linux/arm",
		"windows/amd64",
		"windows/386",
		"windows/arm64",
		"darwin/arm64",
		"wasi/wasm32",
		"vm/bytecode",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			data, ok := RenvoCompileSourceToBytesWithOptions(source, target, RenvoCompileOptions{
				StripSymbols: true,
				EmitImage:    true,
			})
			if !ok {
				t.Fatal("backend compilation failed")
			}
			image, err := linkedimage.Decode(data)
			if err != nil {
				t.Fatalf("decode linked image: %v", err)
			}
			if image.Target != target || len(image.Native) == 0 || len(image.Segments) == 0 {
				t.Fatalf("linked image = %#v", image)
			}
			switch target {
			case "windows/amd64", "windows/386", "windows/arm64":
				_, memorySize, _, _, _, imports, layoutOK := linkedimage.WindowsLayout(image.Native)
				if !layoutOK || memorySize == 0 || len(imports) == 0 {
					t.Fatalf("WindowsLayout = memory %d, imports %#v, ok %v", memorySize, imports, layoutOK)
				}
				symbols, symbolsOK := linkedimage.PersistentSymbols(image.Native, memorySize)
				if !symbolsOK || len(symbols) != 1 {
					t.Fatalf("PersistentSymbols = %#v, %v", symbols, symbolsOK)
				}
				if target == "windows/386" {
					relocations, relocationsOK := linkedimage.BaseRelocations(image.Native, memorySize)
					if !relocationsOK || len(relocations) == 0 {
						t.Fatalf("BaseRelocations = %#v, %v", relocations, relocationsOK)
					}
				}
			case "darwin/arm64":
				_, memorySize, _, imports, libraries, layoutOK := linkedimage.DarwinLayout(image.Native)
				if !layoutOK || memorySize == 0 || len(imports) == 0 || len(libraries) == 0 {
					t.Fatalf("DarwinLayout = memory %d, imports %#v, libraries %#v, ok %v", memorySize, imports, libraries, layoutOK)
				}
				symbols, symbolsOK := linkedimage.PersistentSymbols(image.Native, memorySize)
				if !symbolsOK || len(symbols) != 1 {
					t.Fatalf("PersistentSymbols = %#v, %v", symbols, symbolsOK)
				}
			}
		})
	}
}
