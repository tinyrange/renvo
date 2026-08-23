package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	backend := flag.String("backend", "", "backend source directory")
	output := flag.String("o", "", "generated output")
	sourcesOutput := flag.String("sources", "", "generated embedded source bundle")
	packageName := flag.String("package", "backendcompiled", "generated Go package name")
	fixedTarget := flag.String("fixed-target", "", "fixed target identifier for the generated compiler")
	stubSources := flag.String("stub-sources", "", "comma-separated backend sources whose functions are replaced by unreachable stubs")
	stubFunctions := flag.String("stub-functions", "", "comma-separated functions replaced by unreachable stubs")
	prepareSource := flag.String("prepare-source", "", "specialize one compiler source for a prepared backend")
	flag.Parse()
	if *prepareSource != "" {
		if *output == "" {
			fmt.Fprintln(os.Stderr, "usage: gen -prepare-source input.go -o output.go")
			os.Exit(2)
		}
		source, err := os.ReadFile(*prepareSource)
		if err != nil {
			fail(err)
		}
		prepared, err := specializePreparationSource(filepath.Base(*prepareSource), source)
		if err != nil {
			fail(err)
		}
		if err = os.WriteFile(*output, prepared, 0o644); err != nil {
			fail(err)
		}
		return
	}
	if *backend == "" || *output == "" || *packageName == "" {
		fmt.Fprintln(os.Stderr, "usage: gen -backend directory -o output.go [-sources sources.go] [-package name] [-fixed-target identifier] [-stub-sources names] [-stub-functions names]")
		os.Exit(2)
	}
	stubbed := make(map[string]bool)
	for _, name := range strings.Split(*stubSources, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			stubbed[name] = true
		}
	}
	stubbedFunctions := make(map[string]bool)
	for _, name := range strings.Split(*stubFunctions, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			stubbedFunctions[name] = true
		}
	}
	manifest, err := os.ReadFile(filepath.Join(*backend, "compiler_sources.txt"))
	if err != nil {
		fail(err)
	}
	var names []string
	for _, line := range strings.Split(string(manifest), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			names = append(names, name)
		}
	}
	var out bytes.Buffer
	var sourceBundle bytes.Buffer
	var digestSource bytes.Buffer
	out.WriteString("// Code generated from checked-in RTG backend outputs; DO NOT EDIT.\n")
	out.WriteString("//go:build !renvo\n\n")
	out.WriteString("package ")
	out.WriteString(*packageName)
	out.WriteString("\n\n")
	sourceBundle.WriteString("// Code generated from checked-in RTG backend outputs; DO NOT EDIT.\n")
	sourceBundle.WriteString("//go:build !renvo\n\n")
	sourceBundle.WriteString("package ")
	sourceBundle.WriteString(*packageName)
	sourceBundle.WriteString("\n\n")
	sourceBundle.WriteString("const CompilerSourceCount = ")
	sourceBundle.WriteString(strconv.Itoa(len(names)))
	sourceBundle.WriteString("\n\n")
	var sourceNames bytes.Buffer
	var sourceSizes bytes.Buffer
	var sourceChunkCounts bytes.Buffer
	var sourceChunks bytes.Buffer
	var sourceConstants bytes.Buffer
	foundStub := make(map[string]bool)
	foundStubFunction := make(map[string]bool)
	for sourceIndex, name := range names {
		source, err := os.ReadFile(filepath.Join(*backend, name))
		if err != nil {
			fail(err)
		}
		packageAt := bytes.Index(source, []byte("package main\n"))
		if packageAt < 0 {
			fail(fmt.Errorf("%s has no package main declaration", name))
		}
		preparedSource, err := specializePreparationSource(name, source)
		if err != nil {
			fail(err)
		}
		digestSource.WriteString(name)
		digestSource.WriteByte(0)
		digestSource.Write(preparedSource)
		if !bytes.HasPrefix(source, []byte("//go:build renvo")) {
			out.WriteString("// source: backend/")
			out.WriteString(name)
			out.WriteByte('\n')
			compiledSource := source[packageAt+len("package main\n"):]
			if *fixedTarget != "" || stubbed[name] || len(stubbedFunctions) != 0 {
				compiledSource, err = specializeCompiledSource(
					name, source, *fixedTarget, stubbed[name], stubbedFunctions, foundStubFunction)
				if err != nil {
					fail(err)
				}
			}
			if stubbed[name] {
				foundStub[name] = true
			}
			out.Write(compactCompiledSource(compiledSource))
			out.WriteByte('\n')
		}
		indexText := strconv.Itoa(sourceIndex)
		sourceNames.WriteString("\tif index == ")
		sourceNames.WriteString(indexText)
		sourceNames.WriteString(" { return ")
		sourceNames.WriteString(strconv.Quote(name))
		sourceNames.WriteString(" }\n")
		sourceSizes.WriteString("\tif index == ")
		sourceSizes.WriteString(indexText)
		sourceSizes.WriteString(" { return ")
		sourceSizes.WriteString(strconv.Itoa(len(preparedSource)))
		sourceSizes.WriteString(" }\n")
		chunks := compressedChunks(compress(preparedSource))
		sourceChunkCounts.WriteString("\tif index == ")
		sourceChunkCounts.WriteString(indexText)
		sourceChunkCounts.WriteString(" { return ")
		sourceChunkCounts.WriteString(strconv.Itoa(len(chunks)))
		sourceChunkCounts.WriteString(" }\n")
		sourceChunks.WriteString("\tif index == ")
		sourceChunks.WriteString(indexText)
		sourceChunks.WriteString(" {\n")
		for chunkIndex, chunk := range chunks {
			constantName := "compilerSourceData" + indexText + "Chunk" + strconv.Itoa(chunkIndex)
			sourceChunks.WriteString("\t\tif chunk == ")
			sourceChunks.WriteString(strconv.Itoa(chunkIndex))
			sourceChunks.WriteString(" { return ")
			sourceChunks.WriteString(constantName)
			sourceChunks.WriteString(" }\n")
			sourceConstants.WriteString("const ")
			sourceConstants.WriteString(constantName)
			sourceConstants.WriteString(" = ")
			writeQuotedChunks(&sourceConstants, chunk)
			sourceConstants.WriteByte('\n')
		}
		sourceChunks.WriteString("\t}\n")
	}
	for name := range stubbed {
		if !foundStub[name] {
			fail(fmt.Errorf("stub source %s is not an ordinary compiler source", name))
		}
	}
	for name := range stubbedFunctions {
		if !foundStubFunction[name] {
			fail(fmt.Errorf("stub function %s is not in an ordinary compiler source", name))
		}
	}
	sourceBundle.WriteString("func compilerSourceName(index int) string {\n")
	sourceBundle.Write(sourceNames.Bytes())
	sourceBundle.WriteString(`	return ""
}
func compilerSourceSize(index int) int {
`)
	sourceBundle.Write(sourceSizes.Bytes())
	sourceBundle.WriteString(`	return 0
}
func compilerSourceChunkCount(index int) int {
`)
	sourceBundle.Write(sourceChunkCounts.Bytes())
	sourceBundle.WriteString(`	return 0
}
func compilerSourceChunk(index int, chunk int) string {
`)
	sourceBundle.Write(sourceChunks.Bytes())
	sourceBundle.WriteString(`	return ""
}
`)
	sourceBundle.Write(sourceConstants.Bytes())
	digest := fmt.Sprintf("%x", sha256.Sum256(digestSource.Bytes()))
	compilerSource := out.Bytes()
	packageDeclaration := []byte("package " + *packageName + "\n")
	packageEnd := bytes.Index(compilerSource, packageDeclaration)
	packageEnd += len(packageDeclaration)
	var withDigest bytes.Buffer
	withDigest.Write(compilerSource[:packageEnd])
	withDigest.WriteString("\nconst CompilerSourceDigest = ")
	withDigest.WriteString(strconv.Quote(digest))
	withDigest.WriteString("\n")
	withDigest.Write(compilerSource[packageEnd:])
	compiled := bytes.TrimRight(withDigest.Bytes(), "\n")
	compiled = append(compiled, '\n')
	if err := os.WriteFile(*output, compiled, 0o644); err != nil {
		fail(err)
	}
	if *sourcesOutput != "" {
		if err := os.WriteFile(*sourcesOutput, sourceBundle.Bytes(), 0o644); err != nil {
			fail(err)
		}
	}
}

func specializeCompiledSource(
	name string, source []byte, fixedTarget string, stubAllFunctions bool,
	stubFunctions map[string]bool, foundStubFunction map[string]bool,
) ([]byte, error) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, name, source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s for fixed-backend generation: %w", name, err)
	}
	fixedTargetFound := false
	for _, declaration := range file.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		stubFunction := isFunction && (stubAllFunctions || stubFunctions[function.Name.Name])
		if stubFunction {
			if stubFunctions[function.Name.Name] {
				foundStubFunction[function.Name.Name] = true
			}
			if function.Name.Name == "init" {
				function.Body = &ast.BlockStmt{}
			} else {
				function.Body = &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{
					Fun:  ast.NewIdent("panic"),
					Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("unavailable specialized backend function: " + function.Name.Name)}},
				}}}}
			}
			continue
		}
		generic, ok := declaration.(*ast.GenDecl)
		if !ok || fixedTarget == "" || name != "compiler_main.go" {
			continue
		}
		for _, item := range generic.Specs {
			specification, ok := item.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, declared := range specification.Names {
				if declared.Name != "renvoFixedTarget" {
					continue
				}
				if len(specification.Names) != 1 || len(specification.Values) != 1 || i != 0 {
					return nil, fmt.Errorf("%s must declare renvoFixedTarget as one variable with one value", name)
				}
				specification.Values[0] = ast.NewIdent(fixedTarget)
				fixedTargetFound = true
			}
		}
	}
	if fixedTarget != "" && name == "compiler_main.go" && !fixedTargetFound {
		return nil, fmt.Errorf("%s does not declare renvoFixedTarget", name)
	}
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		return nil, fmt.Errorf("format specialized %s: %w", name, err)
	}
	formatted := output.Bytes()
	packageAt := bytes.Index(formatted, []byte("package main\n"))
	if packageAt < 0 {
		return nil, fmt.Errorf("specialized %s has no package main declaration", name)
	}
	return formatted[packageAt+len("package main\n"):], nil
}

func specializePreparationSource(name string, source []byte) ([]byte, error) {
	prepared := source
	if name == "compiler_target_policy_impl.go" {
		const ordinaryTag = "//go:build !renvo_prepared\n"
		if !bytes.HasPrefix(source, []byte(ordinaryTag)) {
			return nil, fmt.Errorf("%s does not declare the ordinary preparation build tag", name)
		}
		const identifier = "renvoPreparedBackendActive"
		files := token.NewFileSet()
		file, err := parser.ParseFile(files, name, source, 0)
		if err != nil {
			return nil, fmt.Errorf("parse preparation setting in %s: %w", name, err)
		}
		start := -1
		end := -1
		for _, declaration := range file.Decls {
			generic, ok := declaration.(*ast.GenDecl)
			if !ok || generic.Tok != token.CONST {
				continue
			}
			for _, item := range generic.Specs {
				specification, ok := item.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, declared := range specification.Names {
					if declared.Name != identifier {
						continue
					}
					if start >= 0 {
						return nil, fmt.Errorf("%s declares %s more than once", name, identifier)
					}
					if len(specification.Names) != 1 || len(specification.Values) != 1 || i != 0 {
						return nil, fmt.Errorf("%s must declare %s as one const with one value", name, identifier)
					}
					start = files.Position(specification.Values[0].Pos()).Offset
					end = files.Position(specification.Values[0].End()).Offset
				}
			}
		}
		if start < 0 || end <= start {
			return nil, fmt.Errorf("%s does not declare the preparation const %s", name, identifier)
		}
		preparedTag := "//go:build renvo_prepared\n\n// Code generated from compiler_target_policy_impl.go; DO NOT EDIT.\n"
		prepared = make([]byte, 0, len(source)+len(preparedTag)-len(ordinaryTag)-end+start+1)
		prepared = append(prepared, preparedTag...)
		prepared = append(prepared, source[len(ordinaryTag):start]...)
		prepared = append(prepared, '1')
		prepared = append(prepared, source[end:]...)
		const ordinaryStructuredMode = "const renvoRTGStructuredFunctions = 0\n"
		if bytes.Count(prepared, []byte(ordinaryStructuredMode)) != 1 {
			return nil, fmt.Errorf("%s does not declare one ordinary structured-function const", name)
		}
		prepared = bytes.Replace(prepared, []byte(ordinaryStructuredMode), nil, 1)
	}
	return foldPreparationBranches(name, prepared)
}

func foldPreparationBranches(name string, source []byte) ([]byte, error) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, name, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse prepared branches in %s: %w", name, err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		if preparationBodyHasGoto(function.Body) {
			continue
		}
		function.Body.List = foldPreparationStatements(function.Body.List)
		markUnusedPreparationLocals(function.Body)
	}
	var output bytes.Buffer
	if err = format.Node(&output, files, file); err != nil {
		return nil, fmt.Errorf("format prepared branches in %s: %w", name, err)
	}
	return output.Bytes(), nil
}

func preparationBodyHasGoto(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if branch, ok := node.(*ast.BranchStmt); ok && branch.Tok == token.GOTO {
			found = true
			return false
		}
		return !found
	})
	return found
}

func markUnusedPreparationLocals(body *ast.BlockStmt) {
	uses := make(map[*ast.Object]int)
	ast.Inspect(body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && identifier.Obj != nil && identifier.Obj.Kind == ast.Var && identifier.Obj.Pos() != identifier.Pos() {
			uses[identifier.Obj]++
		}
		return true
	})
	body.List = markUnusedPreparationStatements(body.List, uses)
}

func markUnusedPreparationStatements(statements []ast.Stmt, uses map[*ast.Object]int) []ast.Stmt {
	var output []ast.Stmt
	for _, statement := range statements {
		switch item := statement.(type) {
		case *ast.BlockStmt:
			item.List = markUnusedPreparationStatements(item.List, uses)
		case *ast.IfStmt:
			item.Body.List = markUnusedPreparationStatements(item.Body.List, uses)
			if block, ok := item.Else.(*ast.BlockStmt); ok {
				block.List = markUnusedPreparationStatements(block.List, uses)
			}
		case *ast.ForStmt:
			item.Body.List = markUnusedPreparationStatements(item.Body.List, uses)
		case *ast.RangeStmt:
			item.Body.List = markUnusedPreparationStatements(item.Body.List, uses)
		case *ast.SwitchStmt:
			for _, child := range item.Body.List {
				clause := child.(*ast.CaseClause)
				clause.Body = markUnusedPreparationStatements(clause.Body, uses)
			}
		case *ast.TypeSwitchStmt:
			for _, child := range item.Body.List {
				clause := child.(*ast.CaseClause)
				clause.Body = markUnusedPreparationStatements(clause.Body, uses)
			}
		case *ast.SelectStmt:
			for _, child := range item.Body.List {
				clause := child.(*ast.CommClause)
				clause.Body = markUnusedPreparationStatements(clause.Body, uses)
			}
		}
		output = append(output, statement)
		for _, identifier := range preparationStatementDeclarations(statement) {
			if identifier.Name == "_" || identifier.Obj == nil || uses[identifier.Obj] != 0 {
				continue
			}
			output = append(output, &ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN,
				Rhs: []ast.Expr{ast.NewIdent(identifier.Name)},
			})
		}
	}
	return output
}

func preparationStatementDeclarations(statement ast.Stmt) []*ast.Ident {
	switch item := statement.(type) {
	case *ast.AssignStmt:
		if item.Tok != token.DEFINE {
			return nil
		}
		var output []*ast.Ident
		for _, expression := range item.Lhs {
			if identifier, ok := expression.(*ast.Ident); ok {
				output = append(output, identifier)
			}
		}
		return output
	case *ast.DeclStmt:
		declaration, ok := item.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR {
			return nil
		}
		var output []*ast.Ident
		for _, specification := range declaration.Specs {
			output = append(output, specification.(*ast.ValueSpec).Names...)
		}
		return output
	}
	return nil
}

func foldPreparationStatements(statements []ast.Stmt) []ast.Stmt {
	output := make([]ast.Stmt, 0, len(statements))
	for _, statement := range statements {
		folded := foldPreparationStatement(statement)
		if folded == nil {
			continue
		}
		output = append(output, folded)
		if preparationStatementTerminates(folded) {
			break
		}
	}
	return output
}

func foldPreparationStatement(statement ast.Stmt) ast.Stmt {
	switch item := statement.(type) {
	case *ast.BlockStmt:
		item.List = foldPreparationStatements(item.List)
	case *ast.IfStmt:
		item.Body.List = foldPreparationStatements(item.Body.List)
		if item.Else != nil {
			item.Else = foldPreparationStatement(item.Else)
		}
		if value, known := preparationBool(item.Cond); known {
			if value {
				return item.Body
			}
			return item.Else
		}
	case *ast.ForStmt:
		item.Body.List = foldPreparationStatements(item.Body.List)
	case *ast.RangeStmt:
		item.Body.List = foldPreparationStatements(item.Body.List)
	case *ast.SwitchStmt:
		for _, statement := range item.Body.List {
			clause := statement.(*ast.CaseClause)
			clause.Body = foldPreparationStatements(clause.Body)
		}
	case *ast.TypeSwitchStmt:
		for _, statement := range item.Body.List {
			clause := statement.(*ast.CaseClause)
			clause.Body = foldPreparationStatements(clause.Body)
		}
	case *ast.SelectStmt:
		for _, statement := range item.Body.List {
			clause := statement.(*ast.CommClause)
			clause.Body = foldPreparationStatements(clause.Body)
		}
	case *ast.LabeledStmt:
		item.Stmt = foldPreparationStatement(item.Stmt)
	}
	return statement
}

func preparationBool(expression ast.Expr) (bool, bool) {
	switch item := expression.(type) {
	case *ast.ParenExpr:
		return preparationBool(item.X)
	case *ast.UnaryExpr:
		if item.Op == token.NOT {
			value, known := preparationBool(item.X)
			return !value, known
		}
	case *ast.BinaryExpr:
		if item.Op == token.LAND || item.Op == token.LOR {
			left, leftKnown := preparationBool(item.X)
			right, rightKnown := preparationBool(item.Y)
			if item.Op == token.LAND {
				// A known right-hand false value does not make the complete
				// expression removable: Go still evaluates an unknown left side.
				if leftKnown && !left {
					return false, true
				}
				if leftKnown && rightKnown {
					return left && right, true
				}
			} else {
				// Likewise, x || true must preserve evaluation of x.
				if leftKnown && left {
					return true, true
				}
				if leftKnown && rightKnown {
					return left || right, true
				}
			}
		}
		if item.Op == token.EQL || item.Op == token.NEQ {
			left, leftKnown := preparationInteger(item.X)
			right, rightKnown := preparationInteger(item.Y)
			if leftKnown && rightKnown {
				equal := left == right
				return equal == (item.Op == token.EQL), true
			}
		}
	}
	return false, false
}

func preparationInteger(expression ast.Expr) (int64, bool) {
	switch item := expression.(type) {
	case *ast.ParenExpr:
		return preparationInteger(item.X)
	case *ast.Ident:
		if item.Name == "renvoPreparedBackendActive" && item.Obj != nil && item.Obj.Kind == ast.Con {
			return 1, true
		}
	case *ast.BasicLit:
		if item.Kind == token.INT {
			value, err := strconv.ParseInt(item.Value, 0, 64)
			return value, err == nil
		}
	}
	return 0, false
}

func preparationStatementTerminates(statement ast.Stmt) bool {
	switch item := statement.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BlockStmt:
		return len(item.List) != 0 && preparationStatementTerminates(item.List[len(item.List)-1])
	case *ast.IfStmt:
		return item.Else != nil && preparationStatementTerminates(item.Body) && preparationStatementTerminates(item.Else)
	}
	return false
}

func compactCompiledSource(source []byte) []byte {
	files := token.NewFileSet()
	file := files.AddFile("backend.go", files.Base(), len(source))
	var scan scanner.Scanner
	scan.Init(file, source, nil, scanner.ScanComments)
	out := make([]byte, 0, len(source))
	last := 0
	for {
		position, kind, literal := scan.Scan()
		if kind == token.EOF {
			break
		}
		if kind != token.COMMENT {
			continue
		}
		start := file.Offset(position)
		out = append(out, source[last:start]...)
		newlines := 0
		for i := 0; i < len(literal); i++ {
			if literal[i] == '\n' {
				out = append(out, '\n')
				newlines++
			}
		}
		if newlines == 0 {
			out = append(out, ' ')
		}
		last = start + len(literal)
	}
	out = append(out, source[last:]...)
	return removeCompiledIndentation(out)
}

func removeCompiledIndentation(source []byte) []byte {
	out := make([]byte, 0, len(source))
	state := byte(0)
	quote := byte(0)
	escaped := false
	lineStart := true
	for i := 0; i < len(source); i++ {
		ch := source[i]
		if lineStart && state == 0 && (ch == ' ' || ch == '\t' || ch == '\r') {
			continue
		}
		if ch == '\n' && state == 0 {
			for len(out) != 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == '\t' ||
				out[len(out)-1] == '\r') {
				out = out[:len(out)-1]
			}
		}
		out = append(out, ch)
		if state == 0 {
			if ch == '`' {
				state = 2
			} else if ch == '"' || ch == '\'' {
				state = 1
				quote = ch
				escaped = false
			}
		} else if state == 1 {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == quote {
				state = 0
			}
		} else if ch == '`' {
			state = 0
		}
		if ch == '\n' {
			lineStart = state == 0
		} else if lineStart {
			lineStart = false
		}
	}
	return out
}

func compressedChunks(source []byte) [][]byte {
	const chunkLimit = 8192
	var chunks [][]byte
	var chunk []byte
	for at := 0; at < len(source); {
		size := 3
		if source[at] < 128 {
			size = int(source[at]) + 2
		}
		if at+size > len(source) {
			panic("invalid compressed source")
		}
		if len(chunk) > 0 && len(chunk)+size > chunkLimit {
			chunks = append(chunks, chunk)
			chunk = nil
		}
		chunk = append(chunk, source[at:at+size]...)
		at += size
	}
	if len(chunk) > 0 {
		chunks = append(chunks, chunk)
	}
	return chunks
}

func compress(source []byte) []byte {
	const (
		maxDistance = 65535
		maxLength   = 130
		maxChain    = 96
	)
	last := make([]int, 65536)
	for i := range last {
		last[i] = -1
	}
	previous := make([]int, len(source))
	for i := range previous {
		previous[i] = -1
	}
	var out []byte
	literalStart := 0
	flushLiterals := func(end int) {
		for literalStart < end {
			count := end - literalStart
			if count > 128 {
				count = 128
			}
			out = append(out, byte(count-1))
			out = append(out, source[literalStart:literalStart+count]...)
			literalStart += count
		}
	}
	at := 0
	for at < len(source) {
		bestAt := -1
		bestLength := 0
		hash := -1
		if at+2 < len(source) {
			hash = (int(source[at])*251 + int(source[at+1])*31 + int(source[at+2])) & 65535
			candidate := last[hash]
			for searched := 0; candidate >= 0 && at-candidate <= maxDistance && searched < maxChain; searched++ {
				length := 0
				for length < maxLength && at+length < len(source) &&
					source[candidate+length] == source[at+length] {
					length++
				}
				if length > bestLength {
					bestAt = candidate
					bestLength = length
					if length == maxLength {
						break
					}
				}
				candidate = previous[candidate]
			}
		}
		if bestLength >= 3 {
			flushLiterals(at)
			distance := at - bestAt
			out = append(out, 0x80|byte(bestLength-3), byte(distance), byte(distance>>8))
			for i := 0; i < bestLength; i++ {
				position := at + i
				if position+2 >= len(source) {
					continue
				}
				key := (int(source[position])*251 + int(source[position+1])*31 + int(source[position+2])) & 65535
				previous[position] = last[key]
				last[key] = position
			}
			at += bestLength
			literalStart = at
			continue
		}
		if hash >= 0 {
			previous[at] = last[hash]
			last[hash] = at
		}
		at++
		if at-literalStart == 128 {
			flushLiterals(at)
			literalStart = at
		}
	}
	flushLiterals(len(source))
	return out
}

func writeQuotedChunks(out *bytes.Buffer, source []byte) {
	const chunkSize = 8192
	if len(source) == 0 {
		out.WriteString(`""`)
		return
	}
	encoded := base64.RawStdEncoding.EncodeToString(source)
	for start := 0; start < len(encoded); start += chunkSize {
		if start != 0 {
			out.WriteByte('+')
		}
		end := start + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		out.WriteString(strconv.Quote(encoded[start:end]))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "backend bundle:", err)
	os.Exit(1)
}
