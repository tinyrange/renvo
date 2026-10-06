//go:build !renvo

package driver

import "renvo.dev/targetfrontend"

// TargetVocabulary is a selected backend's physical and runtime-value contract.
// These aliases deliberately expose data, not the backend's emitter or helpers.
type TargetVocabulary = targetfrontend.TargetVocabulary
type FrontendTargetDescriptor = targetfrontend.FrontendTargetDescriptor
type TargetOperation = targetfrontend.TargetOperation
type TargetIntrinsic = targetfrontend.TargetIntrinsic
type IntrinsicBlock = targetfrontend.IntrinsicBlock
type ManagedBlock = targetfrontend.ManagedBlock
type TargetManaged = targetfrontend.TargetManaged
type TargetWidth = targetfrontend.TargetWidth
type TargetRegisterAlias = targetfrontend.TargetRegisterAlias
type TargetConstraint = targetfrontend.TargetConstraint
type InlineOperand = targetfrontend.InlineOperand
type InlineBinding = targetfrontend.InlineBinding
type InlineConstraintPlan = targetfrontend.InlineConstraintPlan
type InlineLabel = targetfrontend.InlineLabel
type InlineWord = targetfrontend.InlineWord
type InlineAssembly = targetfrontend.InlineAssembly
type TargetSyntax = targetfrontend.TargetSyntax
type TargetSyntaxForm = targetfrontend.TargetSyntaxForm
type TargetParameter = targetfrontend.TargetParameter
type TargetEffects = targetfrontend.TargetEffects
type TargetImmediate = targetfrontend.TargetImmediate
type TargetBlock = targetfrontend.TargetBlock
type TargetInstruction = targetfrontend.TargetInstruction
type TargetOperand = targetfrontend.TargetOperand
type TargetFrontendDiagnostic = targetfrontend.TargetFrontendDiagnostic
type TargetImportLoader = targetfrontend.TargetImportLoader
type TargetImportSource = targetfrontend.TargetImportSource

// ReadTargetVocabulary resolves a definition using only the caller's import
// capability. A nil loader is appropriate for a closed, standalone definition.
func ReadTargetVocabulary(source []byte, filename, target string, loader TargetImportLoader) TargetVocabulary {
	return targetfrontend.ReadTargetVocabulary(source, filename, target, loader)
}

// EncodeTargetAssembly checks frontend-produced typed operations and serializes
// a .rtgasm version 2 source. Supply it through SourceFS beside bodyless function
// declarations; ordinary package linking preserves it for CompilerJIT lowering.
// This API is for whole-function physical blocks, not C inline-asm constraints
// or compiler-managed semantic intrinsics.
func EncodeTargetAssembly(v TargetVocabulary, blocks []TargetBlock, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return targetfrontend.EncodeTargetAssembly(v, blocks, filename)
}

// ParseTargetAssembler converts conventional text into the same ordered IR.
// A blank syntax chooses the first advertised dialect; file-level syntax
// directives can choose another advertised style before function definitions.
func ParseTargetAssembler(v TargetVocabulary, syntax string, source []byte, filename string) ([]TargetBlock, []TargetFrontendDiagnostic) {
	return targetfrontend.ParseTargetAssembler(v, syntax, source, filename)
}
func ParseTargetInstructionBlock(v TargetVocabulary, syntax string, source []byte, filename, name string) (TargetBlock, []TargetFrontendDiagnostic) {
	return targetfrontend.ParseTargetInstructionBlock(v, syntax, source, filename, name)
}

// EncodeManagedAssembly serializes ordered physical bodies whose expressions,
// spill boundary, and return are compiler-owned. Direct uses emit inline.
func EncodeManagedAssembly(v TargetVocabulary, blocks []ManagedBlock, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return targetfrontend.EncodeManagedAssembly(v, blocks, filename)
}

// EncodeIntrinsicAssembly binds runtime-value intrinsic declarations without
// exposing encoders, physical register choices, or a user-owned function ABI.
func EncodeIntrinsicAssembly(v TargetVocabulary, blocks []IntrinsicBlock, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return targetfrontend.EncodeIntrinsicAssembly(v, blocks, filename)
}
func BindInlineOperands(v TargetVocabulary, outputs, inputs []InlineOperand, clobbers []string, filename string) InlineConstraintPlan {
	return targetfrontend.BindInlineOperands(v, outputs, inputs, clobbers, filename)
}
func ExpandInlineTemplate(v TargetVocabulary, plan InlineConstraintPlan, syntax string, template []byte, labels []InlineLabel, unique int, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return targetfrontend.ExpandInlineTemplate(v, plan, syntax, template, labels, unique, filename)
}
func BuildInlineAssembly(v TargetVocabulary, syntax string, template []byte, outputs, inputs []InlineOperand, clobbers []string, labels []InlineLabel, name, filename string, unique int) InlineAssembly {
	return targetfrontend.BuildInlineAssembly(v, syntax, template, outputs, inputs, clobbers, labels, name, filename, unique)
}
