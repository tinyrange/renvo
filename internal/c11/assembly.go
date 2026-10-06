package c11

// AssemblyCompiler is a language-independent compiler capability. Keeping
// metadata resolution behind it avoids coupling the C parser to RTG's checker.
// A non-nil compiler is authoritative: failures cannot fall back to pattern
// recognition or pretend that an unimplemented instruction has C semantics.
type AssemblyCompiler interface {
	WordBits() int
	CompileInline(AssemblyRequest) AssemblyCompilation
}
type AssemblyOperand struct {
	Name, Constraint, Value string
	Constant                bool
	Bits                    int
	Signed                  bool
}
type AssemblyRequest struct {
	Name             string
	Template         []byte
	Outputs, Inputs  []AssemblyOperand
	Clobbers, Labels []string
	Unique           int
}
type AssemblyWord struct {
	Operand int
	Address bool
}
type AssemblyCompilation struct {
	Source  []byte
	Words   []AssemblyWord
	Message string
	Ok      bool
}
