package build

import (
	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/check"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/lower"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/unit"
)

func Units(graph *load.Graph) ([]unit.Unit, error) {
	if err := parseGraphFiles(graph); err != nil {
		return nil, err
	}
	checkMark := arena.Mark()
	if err := check.Graph(graph); err != nil {
		arena.Reset(checkMark)
		return nil, err
	}
	arena.Reset(checkMark)
	return UnitsUnchecked(graph)
}

func parseGraphFiles(graph *load.Graph) error {
	for i := 0; i < len(graph.Packages); i++ {
		files := graph.Packages[i].Files
		for j := 0; j < len(files); j++ {
			if files[j].Parsed.Path != "" {
				continue
			}
			parsed, err := parse.FileSource(files[j].Path, files[j].Source)
			if err != nil {
				return err
			}
			files[j].Parsed = parsed
		}
		graph.Packages[i].Files = files
	}
	return nil
}

func UnitsUnchecked(graph *load.Graph) ([]unit.Unit, error) {
	sources := make([]unit.TextSourceFile, 0, len(graph.Packages))
	for i := 0; i < len(graph.Packages); i++ {
		mark := arena.Mark()
		pkg := graph.Packages[i]
		u, err := lower.PackageWithGraph(pkg, graph)
		if err != nil {
			arena.Reset(mark)
			return nil, err
		}
		emitted := emit.Source(u)
		source := arena.PersistString(string(emitted))
		path := arena.PersistString(emit.FileName(u.ImportPath))
		arena.Reset(mark)
		var sourceFile unit.TextSourceFile
		sourceFile.Path = path
		sourceFile.Source = source
		sources = append(sources, sourceFile)
	}
	units, err := unit.ParseTextSources(sources)
	if err != nil {
		return nil, err
	}
	sortUnitsByImportPath(units)
	return units, nil
}

func sortUnitsByImportPath(units []unit.Unit) {
	for i := 1; i < len(units); i++ {
		value := units[i]
		j := i - 1
		for j >= 0 && units[j].ImportPath > value.ImportPath {
			units[j+1] = units[j]
			j = j - 1
		}
		units[j+1] = value
	}
}
