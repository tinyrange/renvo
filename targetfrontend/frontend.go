// Package targetfrontend provides backend-owned assembler and runtime-intrinsic
// contracts without a host filesystem, process API, or compiler driver. It is
// suitable for self-hosted and custom language frontends.
package targetfrontend

import (
	"renvo.dev/internal/asmunit"
	"renvo.dev/internal/rtg"
)

// TargetVocabulary is a selected backend's physical and runtime-value contract.
// These aliases deliberately expose data, not the backend's emitter or helpers.
type TargetVocabulary = rtg.TargetVocabulary
type FrontendTargetDescriptor = rtg.TargetDescriptor
type TargetOperation = rtg.TargetOperation
type TargetIntrinsic = rtg.TargetIntrinsic
type IntrinsicBlock = rtg.IntrinsicBlock
type ManagedBlock = rtg.ManagedBlock
type TargetManaged = rtg.TargetManaged
type TargetWidth = rtg.TargetWidth
type TargetRegisterAlias = rtg.TargetRegisterAlias
type TargetConstraint = rtg.TargetConstraint
type InlineOperand = rtg.InlineOperand
type InlineBinding = rtg.InlineBinding
type InlineConstraintPlan = rtg.InlineConstraintPlan
type InlineLabel = rtg.InlineLabel
type InlineWord = rtg.InlineWord
type InlineAssembly = rtg.InlineAssembly
type TargetSyntax = rtg.TargetSyntax
type TargetSyntaxForm = rtg.TargetSyntaxForm
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

// ParseTargetAssembler converts conventional text into the same ordered IR.
// A blank syntax chooses the first advertised dialect; file-level syntax
// directives can choose another advertised style before function definitions.
func ParseTargetAssembler(v TargetVocabulary, syntax string, source []byte, filename string) ([]TargetBlock, []TargetFrontendDiagnostic) {
	return rtg.ParseTargetAssembler(v, syntax, source, filename)
}
func ParseTargetInstructionBlock(v TargetVocabulary, syntax string, source []byte, filename, name string) (TargetBlock, []TargetFrontendDiagnostic) {
	return rtg.ParseTargetInstructionBlock(v, syntax, source, filename, name)
}

// EncodeManagedAssembly serializes ordered physical bodies whose expressions,
// spill boundary, and return are compiler-owned. Direct uses emit inline.
func EncodeManagedAssembly(v TargetVocabulary, blocks []ManagedBlock, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return rtg.EncodeManagedAssembly(v, blocks, filename)
}

// EncodeIntrinsicAssembly binds runtime-value intrinsic declarations without
// exposing encoders, physical register choices, or a user-owned function ABI.
func EncodeIntrinsicAssembly(v TargetVocabulary, blocks []IntrinsicBlock, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return rtg.EncodeIntrinsicAssembly(v, blocks, filename)
}
func BindInlineOperands(v TargetVocabulary, outputs, inputs []InlineOperand, clobbers []string, filename string) InlineConstraintPlan {
	return rtg.BindInlineOperands(v, outputs, inputs, clobbers, filename)
}
func ExpandInlineTemplate(v TargetVocabulary, plan InlineConstraintPlan, syntax string, template []byte, labels []InlineLabel, unique int, filename string) ([]byte, []TargetFrontendDiagnostic) {
	return rtg.ExpandInlineTemplate(v, plan, syntax, template, labels, unique, filename)
}
func BuildInlineAssembly(v TargetVocabulary, syntax string, template []byte, outputs, inputs []InlineOperand, clobbers []string, labels []InlineLabel, name, filename string, unique int) InlineAssembly {
	return rtg.BuildInlineAssembly(v, syntax, template, outputs, inputs, clobbers, labels, name, filename, unique)
}

// AssemblyEvaluatorCompiler compiles the generated, closed encoder program to
// raw vm/vm32 bytecode. The capability can use a separate arena or compiler
// process; this package always executes it in its own bounded VM.
type AssemblyEvaluatorCompiler = asmunit.Compiler
type AssemblyMaterialization = asmunit.Result

// MaterializeAssembly resolves a selected definition and re-evaluates every
// binding from its preserved assembly source. Embedded code is never trusted.
// memory=0 uses the 96 MiB ceiling; a nonzero value imposes a stricter bound.
// Execution is always limited to 500 million VM steps. A caller with an already
// resolved definition can use the internal service, avoiding repeated imports.
func MaterializeAssembly(data, definition []byte, filename, target string, loader TargetImportLoader, compiler AssemblyEvaluatorCompiler, memory int) AssemblyMaterialization {
	resolved := rtg.Resolve(rtg.ParseImports(definition, filename, loader))
	if !resolved.Ok {
		return AssemblyMaterialization{Diagnostics: resolved.Diagnostics}
	}
	if memory == 0 {
		return asmunit.Materialize(data, resolved, target, compiler)
	}
	return asmunit.MaterializeBounded(data, resolved, target, compiler, memory)
}
