//go:build !renvo

package driver

import "renvo.dev/internal/rtg"

// TargetVocabulary is a selected backend's physical machine-block contract.
// These aliases deliberately expose data, not the backend's emitter or helpers.
type TargetVocabulary = rtg.TargetVocabulary
type FrontendTargetDescriptor = rtg.TargetDescriptor
type TargetOperation = rtg.TargetOperation
type TargetParameter = rtg.TargetParameter
type TargetEffects = rtg.TargetEffects
type TargetImmediate = rtg.TargetImmediate
type TargetBlock = rtg.TargetBlock
type TargetInstruction = rtg.TargetInstruction
type TargetOperand = rtg.TargetOperand
type TargetFrontendDiagnostic = rtg.Diagnostic
type TargetImportLoader = rtg.ImportLoader
type TargetImportSource = rtg.ImportSource

// ReadTargetVocabulary resolves a definition using only the caller's import
// capability. A nil loader is appropriate for a closed, standalone definition.
func ReadTargetVocabulary(source []byte, filename, target string, loader TargetImportLoader) TargetVocabulary {
	return rtg.FrontendOperations(rtg.Resolve(rtg.ParseImports(source, filename, loader)), target)
}

// EncodeTargetAssembly checks frontend-produced typed operations and serializes
// a .rtgasm version 2 source. Supply it through SourceFS beside bodyless function
// declarations; ordinary package linking preserves it for CompilerJIT lowering.
// This API is for whole-function physical blocks, not C inline-asm constraints
// or compiler-managed semantic intrinsics.
func EncodeTargetAssembly(v TargetVocabulary, blocks []TargetBlock, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return rtg.EncodeTargetAssembly(v, blocks, filename)
}
