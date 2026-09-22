package load

const (
	WorkspaceOK = iota
	WorkspaceErrDuplicateFile
	WorkspaceErrMissingModule
	WorkspaceErrModule
	WorkspaceErrGraph
)

type Workspace struct {
	Module    Module
	Graph     Graph
	Files     []SourceFile
	Ok        bool
	Error     int
	ErrorFile int
}

func LoadWorkspace(workDir string, stdRoot string, arg string, files []SourceFile) Workspace {
	workspace := Workspace{Ok: true, Error: WorkspaceOK, ErrorFile: -1}
	normalized, duplicate := normalizeSourceFiles(files)
	if duplicate >= 0 {
		workspace.Files = normalized
		return workspaceFail(workspace, WorkspaceErrDuplicateFile, duplicate)
	}
	workspace.Files = normalized
	workDir = CleanPath(workDir)
	moduleRoot, moduleSrc, moduleFile := findNearestModule(workDir, normalized)
	if moduleFile < 0 {
		return workspaceFail(workspace, WorkspaceErrMissingModule, -1)
	}
	module := ParseModule(moduleRoot, moduleSrc)
	workspace.Module = module
	if !module.Ok {
		return workspaceFail(workspace, WorkspaceErrModule, moduleFile)
	}
	var dependencies []ModuleDependency
	for i := 0; i < len(normalized); i++ {
		if i == moduleFile || BasePath(normalized[i].Path) != "go.mod" {
			continue
		}
		// Collector manifests retain the logical (possibly replaced) module
		// path and declared language version, not dependency resolution rules.
		root := DirPath(normalized[i].Path)
		dependency := ParseModule(root, normalized[i].Src)
		if !dependency.Ok {
			// Accept the historical path-only manifest for existing callers.
			path, end, ok := parseModulePath(normalized[i].Src, 0)
			if !ok || path == "" || end != len(normalized[i].Src) || path != string(normalized[i].Src) {
				return workspaceFail(workspace, WorkspaceErrModule, i)
			}
			dependency = Module{Root: root, Path: path, Ok: true}
		}
		if dependency.Path == "" {
			return workspaceFail(workspace, WorkspaceErrModule, i)
		}
		dependencies = append(dependencies, ModuleDependency{Path: dependency.Path, Root: root, GoVersion: dependency.GoVersion})
	}
	graph := LoadGraphWithDependencies(module, stdRoot, workDir, arg, dependencies, normalized)
	workspace.Graph = graph
	if !graph.Ok {
		return workspaceFail(workspace, WorkspaceErrGraph, graph.ErrorPackage)
	}
	return workspace
}

func normalizeSourceFiles(files []SourceFile) ([]SourceFile, int) {
	out := make([]SourceFile, 0, len(files))
	for i := 0; i < len(files); i++ {
		out = append(out, SourceFile{Path: CleanPath(files[i].Path), Src: files[i].Src, CPrelude: files[i].CPrelude,
			CObject: files[i].CObject, CCompiler: files[i].CCompiler, CDataModel: files[i].CDataModel, CFunctionSections: files[i].CFunctionSections,
			CDataSections: files[i].CDataSections, CShortWChar: files[i].CShortWChar,
			CUnsignedChar:    files[i].CUnsignedChar,
			CKernelCodeModel: files[i].CKernelCodeModel, COptimize: files[i].COptimize,
			ArenaStart: files[i].ArenaStart, ArenaEnd: files[i].ArenaEnd})
	}
	sortSourceFiles(out)
	for i := 1; i < len(out); i++ {
		if out[i-1].Path == out[i].Path {
			return out, i
		}
	}
	return out, -1
}

func findNearestModule(workDir string, files []SourceFile) (string, []byte, int) {
	dir := CleanPath(workDir)
	for {
		goMod := JoinPath(dir, "go.mod")
		index := findSourceFile(files, goMod)
		if index >= 0 {
			return dir, files[index].Src, index
		}
		next := DirPath(dir)
		if next == dir {
			break
		}
		if dir == "." || dir == "/" {
			break
		}
		dir = next
	}
	return "", nil, -1
}

func findSourceFile(files []SourceFile, path string) int {
	path = CleanPath(path)
	for i := 0; i < len(files); i++ {
		if files[i].Path == path {
			return i
		}
	}
	return -1
}

func workspaceFail(workspace Workspace, err int, file int) Workspace {
	workspace.Ok = false
	workspace.Error = err
	workspace.ErrorFile = file
	return workspace
}
