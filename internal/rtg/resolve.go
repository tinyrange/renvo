package rtg

const DescriptorVersion = 3

const (
	BackendFamilyNativeV1     = "native_v1"
	BackendFamilyStructured32 = "structured32"
)

type TargetDescriptor struct {
	Name                string
	Aliases             []string
	Family              string
	OS                  string
	ISA                 string
	WordBits            int
	PointerBits         int
	CodePointerBits     int
	FunctionPointerBits int
	MaxAlign            int
	ArenaDefault        int
	Endian              string
	ABI                 string
	Runtime             string
	RuntimeOps          []string
	Executable          string
	Object              string
	OutputKind          string
	BuildTags           []string
	Capabilities        []string
	Definition          [32]byte
	Version             int
}

type ResolvedTarget struct {
	Declaration Declaration
	Descriptor  TargetDescriptor
	Arch        Declaration
	ABI         Declaration
	Runtime     Declaration
	Executable  Declaration
	Object      Declaration
}

type ResolveResult struct {
	Document    Document
	Targets     []ResolvedTarget
	Diagnostics []Diagnostic
	Ok          bool
}

func Resolve(document Document) ResolveResult {
	return resolve(document, true)
}

// ResolveArchitectureDefinition validates a closed architecture projection
// without requiring it to export a target. Checked-in architecture generation
// uses this for roots that import only one shared ISA fragment.
func ResolveArchitectureDefinition(document Document) ResolveResult {
	return resolve(document, false)
}

func resolve(document Document, requireTarget bool) ResolveResult {
	result := ResolveResult{Document: document, Ok: document.Ok}
	if !document.Ok {
		result.Diagnostics = append(result.Diagnostics, document.Diagnostics...)
		return result
	}
	result.Diagnostics = append(result.Diagnostics, validateMachineDeclarations(document)...)
	if len(result.Diagnostics) != 0 {
		result.Ok = false
		return result
	}
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclTarget {
			continue
		}
		target, diagnostic, ok := resolveTarget(document, document.Declarations[i])
		if !ok {
			result.Diagnostics = append(result.Diagnostics, diagnostic)
			result.Ok = false
			continue
		}
		for j := 0; j < len(result.Targets); j++ {
			if targetNameCollides(target.Descriptor, result.Targets[j].Descriptor) {
				result.Diagnostics = append(result.Diagnostics, resolveDiagnostic(document, document.Declarations[i], "RTG-RESOLVE-001", "canonical target or alias collision for "+target.Descriptor.Name))
				result.Ok = false
			}
		}
		result.Targets = append(result.Targets, target)
		compositionDiagnostics := validateTargetComposition(document, target)
		if len(compositionDiagnostics) != 0 {
			result.Diagnostics = append(result.Diagnostics, compositionDiagnostics...)
			result.Ok = false
		}
	}
	if requireTarget && len(result.Targets) == 0 && hasMachineDeclaration(document) {
		result.Diagnostics = append(result.Diagnostics, documentDiagnostic(document,
			sourceSpan(document.Source, 0, 0), "RTG-RESOLVE-002",
			"machine definition exports no targets"))
		result.Ok = false
	}
	return result
}

func ResolveDefinitions(parsed Document) ResolveResult {
	return Resolve(parsed)
}

func resolveTarget(document Document, declaration Declaration) (ResolvedTarget, Diagnostic, bool) {
	target := ResolvedTarget{Declaration: declaration}
	target.Descriptor.Name = declaration.Name
	family, ok := requiredNameField(document, declaration, "family")
	if !ok {
		diagnostic := resolveDiagnostic(document, declaration, "RTG-RESOLVE-022",
			"target "+declaration.Name+" is missing family")
		return target, diagnostic, false
	}
	if family != BackendFamilyNativeV1 && family != BackendFamilyStructured32 {
		diagnostic := resolveDiagnostic(document, declaration, "RTG-RESOLVE-023",
			"target "+declaration.Name+" has unknown backend family "+family)
		return target, diagnostic, false
	}
	target.Descriptor.Family = family
	osName, ok := requiredNameField(document, declaration, "os")
	if !ok {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-015", "target "+declaration.Name+" is missing os"), false
	}
	target.Descriptor.OS = osName
	target.Descriptor.Version = DescriptorVersion

	archName, ok := requiredNameField(document, declaration, "arch")
	if !ok {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-003", "target "+declaration.Name+" is missing arch"), false
	}
	abiName, ok := requiredNameField(document, declaration, "abi")
	if !ok {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-004", "target "+declaration.Name+" is missing abi"), false
	}
	runtimeName, ok := requiredNameField(document, declaration, "runtime")
	if !ok {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-005", "target "+declaration.Name+" is missing runtime"), false
	}
	var diagnostic Diagnostic
	target.Arch, diagnostic, ok = requireDeclaration(document, DeclArch, archName, declaration)
	if !ok {
		return target, diagnostic, false
	}
	target.ABI, diagnostic, ok = requireDeclaration(document, DeclABI, abiName, declaration)
	if !ok {
		return target, diagnostic, false
	}
	target.Runtime, diagnostic, ok = requireDeclaration(document, DeclRuntime, runtimeName, declaration)
	if !ok {
		return target, diagnostic, false
	}
	target.Descriptor.ABI = abiName
	target.Descriptor.Runtime = runtimeName
	target.Descriptor.ISA = archName
	if alias, found := fieldValue(document, target.Arch, "alias"); found {
		target.Descriptor.ISA = valueName(alias)
	}
	if frontendArch, found := fieldValue(document, declaration, "frontend_arch"); found {
		target.Descriptor.ISA = valueName(frontendArch)
		if target.Descriptor.ISA == "" {
			diagnostic := resolveDiagnostic(document, declaration, "RTG-RESOLVE-021",
				"target "+declaration.Name+" has invalid frontend_arch")
			return target, diagnostic, false
		}
	}
	target.Descriptor.WordBits, ok = integerField(document, target.Arch, "word_bits")
	if !ok || target.Descriptor.WordBits != 8 && target.Descriptor.WordBits != 16 &&
		target.Descriptor.WordBits != 32 && target.Descriptor.WordBits != 64 {
		return target, resolveDiagnostic(document, target.Arch, "RTG-RESOLVE-006", "architecture "+archName+" has invalid word_bits"), false
	}
	target.Descriptor.PointerBits, ok = integerField(document, target.Arch, "pointer_bits")
	if !ok {
		target.Descriptor.PointerBits = target.Descriptor.WordBits
	}
	if target.Descriptor.PointerBits != 8 && target.Descriptor.PointerBits != 16 &&
		target.Descriptor.PointerBits != 32 && target.Descriptor.PointerBits != 64 {
		return target, resolveDiagnostic(document, target.Arch, "RTG-RESOLVE-007", "architecture "+archName+" has invalid pointer_bits"), false
	}
	target.Descriptor.CodePointerBits = target.Descriptor.PointerBits
	target.Descriptor.FunctionPointerBits = target.Descriptor.PointerBits
	target.Descriptor.MaxAlign = target.Descriptor.PointerBits / 8
	if bits, found := integerField(document, declaration, "code_pointer_bits"); found {
		target.Descriptor.CodePointerBits = bits
	}
	if bits, found := integerField(document, declaration, "function_pointer_bits"); found {
		target.Descriptor.FunctionPointerBits = bits
	}
	if align, found := integerField(document, declaration, "max_align"); found {
		target.Descriptor.MaxAlign = align
	}
	if arena, found := integerField(document, declaration, "arena_default"); found {
		target.Descriptor.ArenaDefault = arena
	}
	if !validDescriptorWidth(target.Descriptor.CodePointerBits) {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-011", "target "+declaration.Name+" has invalid code_pointer_bits"), false
	}
	if !validDescriptorWidth(target.Descriptor.FunctionPointerBits) {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-012", "target "+declaration.Name+" has invalid function_pointer_bits"), false
	}
	if target.Descriptor.MaxAlign <= 0 || target.Descriptor.MaxAlign&(target.Descriptor.MaxAlign-1) != 0 {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-013", "target "+declaration.Name+" has invalid max_align"), false
	}
	if target.Descriptor.ArenaDefault < 0 {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-014", "target "+declaration.Name+" has invalid arena_default"), false
	}
	endian, found := fieldValue(document, target.Arch, "endian")
	if !found || valueName(endian) != "little" && valueName(endian) != "big" {
		return target, resolveDiagnostic(document, target.Arch, "RTG-RESOLVE-008", "architecture "+archName+" has invalid endian"), false
	}
	target.Descriptor.Endian = valueName(endian)

	if value, found := fieldValue(document, declaration, "executable"); found {
		name := valueName(value)
		target.Executable, diagnostic, ok = requireDeclaration(document, DeclFormat, name, declaration)
		if !ok {
			return target, diagnostic, false
		}
		target.Descriptor.Executable = name
	}
	if value, found := fieldValue(document, declaration, "object"); found {
		name := valueName(value)
		target.Object, diagnostic, ok = requireDeclaration(document, DeclFormat, name, declaration)
		if !ok {
			return target, diagnostic, false
		}
		target.Descriptor.Object = name
	}
	if target.Descriptor.Executable == "" && target.Descriptor.Object == "" {
		return target, resolveDiagnostic(document, declaration, "RTG-RESOLVE-009", "target "+declaration.Name+" has no output format"), false
	}
	target.Descriptor.Aliases = listField(document, declaration, "aliases")
	target.Descriptor.BuildTags = listField(document, declaration, "build_tags")
	target.Descriptor.Capabilities = listField(document, declaration, "capabilities")
	target.Descriptor.RuntimeOps = runtimeOperations(document, target.Runtime)
	output := target.Executable
	if output.Name == "" {
		output = target.Object
	}
	target.Descriptor.OutputKind = output.Name
	if kind, found := fieldValue(document, output, "kind"); found {
		target.Descriptor.OutputKind = valueName(kind)
	}
	if target.Descriptor.Family == BackendFamilyNativeV1 &&
		(target.Descriptor.ISA == "wasm32" || target.Descriptor.ISA == "vm32" ||
			target.Descriptor.OutputKind == "wasm" || target.Descriptor.OutputKind == "html-wasm" ||
			target.Descriptor.OutputKind == "rnvm") {
		diagnostic := resolveDiagnostic(document, declaration, "RTG-RESOLVE-024",
			"native_v1 target "+declaration.Name+" uses a structured backend machine or output")
		return target, diagnostic, false
	}
	if target.Descriptor.Family == BackendFamilyStructured32 &&
		target.Descriptor.ISA != "wasm32" && target.Descriptor.ISA != "vm32" {
		diagnostic := resolveDiagnostic(document, declaration, "RTG-RESOLVE-025",
			"structured32 target "+declaration.Name+" requires wasm32 or vm32 frontend architecture")
		return target, diagnostic, false
	}
	sortStrings(target.Descriptor.Aliases)
	sortStrings(target.Descriptor.BuildTags)
	sortStrings(target.Descriptor.Capabilities)
	target.Descriptor.Definition = targetSemanticIdentity(document, target)
	return target, Diagnostic{}, true
}

func validDescriptorWidth(bits int) bool {
	return bits == 8 || bits == 16 || bits == 32 || bits == 64
}

func runtimeOperations(document Document, declaration Declaration) []string {
	operations := listField(document, declaration, "operations")
	for i := 0; i < len(declaration.Statements); i++ {
		tokens := declaration.Statements[i].Tokens
		if len(tokens) >= 2 && tokens[0] == "operation" &&
			stringIndex(operations, tokens[1]) < 0 {
			operations = append(operations, tokens[1])
		}
	}
	return operations
}

func requireDeclaration(document Document, kind string, name string, owner Declaration) (Declaration, Diagnostic, bool) {
	declaration, ok := document.Declaration(kind, name)
	if ok {
		return declaration, Diagnostic{}, true
	}
	return Declaration{}, resolveDiagnostic(document, owner, "RTG-RESOLVE-010", owner.Kind+" "+owner.Name+" references unknown "+kind+" "+name), false
}

func fieldValue(document Document, declaration Declaration, name string) (string, bool) {
	for i := 0; i < len(declaration.Fields); i++ {
		if declaration.Fields[i].Name == name {
			field := declaration.Fields[i]
			if field.Value != "" {
				return field.Value, true
			}
			return compactSource(document.Source[field.ValueStart:field.ValueEnd]), true
		}
	}
	return "", false
}

func requiredNameField(document Document, declaration Declaration, name string) (string, bool) {
	value, ok := fieldValue(document, declaration, name)
	if !ok {
		return "", false
	}
	value = valueName(value)
	return value, value != ""
}

func integerField(document Document, declaration Declaration, name string) (int, bool) {
	value, ok := fieldValue(document, declaration, name)
	if !ok {
		return 0, false
	}
	return parseInteger(value)
}

func listField(document Document, declaration Declaration, name string) []string {
	value, ok := fieldValue(document, declaration, name)
	if !ok || len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil
	}
	var result []string
	start := 1
	for i := 1; i <= len(value)-1; i++ {
		if i == len(value)-1 || value[i] == ',' {
			item := valueName(value[start:i])
			if item != "" {
				result = append(result, item)
			}
			start = i + 1
		}
	}
	return result
}

func valueName(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'' || value[0] == '`') {
		return unquoteSimple(value)
	}
	return value
}

func compactSource(source []byte) string {
	tokens, diagnostics := scan(source, "")
	if len(diagnostics) != 0 {
		return ""
	}
	return compactTokens(source, tokens)
}

func parseInteger(text string) (int, bool) {
	base := 10
	at := 0
	sign := 1
	if len(text) != 0 && text[0] == '-' {
		sign = -1
		at = 1
	}
	if len(text) > at+2 && text[at] == '0' && text[at+1] == 'x' {
		base = 16
		at += 2
	}
	if at == len(text) {
		return 0, false
	}
	value := 0
	for at < len(text) {
		digit := int(text[at] - '0')
		if text[at] >= 'a' && text[at] <= 'f' {
			digit = int(text[at]-'a') + 10
		} else if text[at] >= 'A' && text[at] <= 'F' {
			digit = int(text[at]-'A') + 10
		}
		if digit < 0 || digit >= base || value > (1073741824-digit)/base {
			return 0, false
		}
		value = value*base + digit
		at++
	}
	return sign * value, true
}

func targetNameCollides(left TargetDescriptor, right TargetDescriptor) bool {
	leftNames := append([]string{left.Name}, left.Aliases...)
	rightNames := append([]string{right.Name}, right.Aliases...)
	for i := 0; i < len(leftNames); i++ {
		for j := 0; j < len(rightNames); j++ {
			if leftNames[i] == rightNames[j] {
				return true
			}
		}
	}
	return false
}

func hasMachineDeclaration(document Document) bool {
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind != DeclSystem && document.Declarations[i].Kind != DeclGo {
			return true
		}
	}
	return false
}

func resolveDiagnostic(document Document, declaration Declaration, code string, message string) Diagnostic {
	return documentDiagnostic(document, declaration.Span, code, message)
}
