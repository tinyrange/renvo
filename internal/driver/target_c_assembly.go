//go:build !renvo

package driver

import (
	"renvo.dev/internal/c11"
	"renvo.dev/internal/rtg"
	"strconv"
)

type targetCAssemblyCompiler struct{ vocabulary rtg.TargetVocabulary }

func (compiler targetCAssemblyCompiler) WordBits() int { return compiler.vocabulary.Target.WordBits }
func (compiler targetCAssemblyCompiler) CompileInline(request c11.AssemblyRequest) c11.AssemblyCompilation {
	result := c11.AssemblyCompilation{}
	operands := func(values []c11.AssemblyOperand) []rtg.InlineOperand {
		converted := []rtg.InlineOperand{}
		for _, value := range values {
			converted = append(converted, rtg.InlineOperand{Name: value.Name, Constraint: value.Constraint, Constant: value.Constant, Bits: value.Bits, Signed: value.Signed, Value: rtg.TargetOperand{Kind: "int64", Value: value.Value}})
		}
		return converted
	}
	v := compiler.vocabulary
	if !v.Ok || !v.Managed.Ok || len(v.Syntaxes) == 0 {
		result.Message = "selected backend has no C inline assembly boundary"
		return result
	}
	labels := []rtg.InlineLabel{}
	for i, label := range request.Labels {
		labels = append(labels, rtg.InlineLabel{Name: label, Symbol: ".LrenvoGoto_" + strconv.Itoa(request.Unique) + "_" + strconv.Itoa(i)})
	}
	lowered := rtg.BuildInlineAssembly(v, v.Syntaxes[0].Name, request.Template, operands(request.Outputs), operands(request.Inputs), request.Clobbers, labels, request.Name, "inline.rtgasm", request.Unique)
	if !lowered.Ok {
		if len(lowered.Diagnostics) != 0 {
			result.Message = lowered.Diagnostics[0].Message
		}
		return result
	}
	source, diagnostics := rtg.EncodeManagedAssembly(v, []rtg.ManagedBlock{lowered.Block}, "inline.rtgasm")
	if len(diagnostics) != 0 {
		result.Message = diagnostics[0].Message
		return result
	}
	result.Source = source
	for _, word := range lowered.Words {
		result.Words = append(result.Words, c11.AssemblyWord{Operand: word.Operand, Address: word.Address})
	}
	result.Ok = true
	return result
}
