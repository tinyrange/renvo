package unit

type Unit struct {
	ImportPath string
	Package    string
	Entry      bool
	Imports    []string
	Exports    []Symbol
	References []Symbol
	Decls      []Decl
}

type SourceFile struct {
	Path   string
	Source []byte
}

type Symbol struct {
	ImportPath string
	Name       string
	UnitName   string
}

type Decl struct {
	Path     string
	Kind     string
	Name     string
	UnitName string
	Body     string
}
