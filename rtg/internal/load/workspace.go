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
	graph := LoadGraph(module, stdRoot, workDir, arg, normalized)
	workspace.Graph = graph
	if !graph.Ok {
		return workspaceFail(workspace, WorkspaceErrGraph, graph.ErrorPackage)
	}
	return workspace
}

func normalizeSourceFiles(files []SourceFile) ([]SourceFile, int) {
	out := make([]SourceFile, 0, len(files))
	for i := 0; i < len(files); i++ {
		out = append(out, SourceFile{Path: CleanPath(files[i].Path), Src: files[i].Src})
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
