//go:build rtg

package unit

type Import struct {
	NameTok int
	PathTok int
}

type Symbol struct {
	Name    string
	Package int
	Token   int
}

type Call struct {
	Kind      int
	CalleeTok int
	BaseTok   int
	DotTok    int
}

type NameRef struct {
	Kind    int
	Token   int
	Index   int
	Package int
}

type Selector struct {
	BaseTok     int
	DotTok      int
	NameTok     int
	BaseKind    int
	BaseIndex   int
	BasePackage int
	Package     int
	Symbol      int
}

type TypeRef struct {
	Kind    int
	Token   int
	BaseTok int
	DotTok  int
	Package int
	Symbol  int
}

type Program struct {
	Package    string
	ImportPath string
	Text       []byte
	Tokens     []Token
	Imports    []Import
	Symbols    []Symbol
	Decls      []Decl
	Funcs      []Func
	TypeRefs   []TypeRef
	Calls      []Call
	Refs       []NameRef
	Selectors  []Selector
}

func Marshal(program Program) ([]byte, bool) {
	return MarshalCore(program)
}
