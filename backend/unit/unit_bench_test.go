package unit

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var benchmarkCompilerFiles = benchmarkCompilerSourceManifest()

func benchmarkCompilerSourceManifest() []string {
	data, err := os.ReadFile("../compiler_sources.txt")
	if err != nil {
		panic(err)
	}
	var files []string
	for _, line := range strings.Split(string(data), "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			files = append(files, "../"+path)
		}
	}
	if len(files) == 0 {
		panic("compiler source manifest is empty")
	}
	return files
}

func BenchmarkUnitEncodeCompiler(b *testing.B) {
	program := benchmarkCompilerProgram(b)
	unit, err := Marshal(program)
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}
	b.SetBytes(int64(len(unit)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Marshal(program); err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

func BenchmarkUnitDecodeCompiler(b *testing.B) {
	unit := benchmarkCompilerUnit(b)
	b.SetBytes(int64(len(unit)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Unmarshal(unit); err != nil {
			b.Fatalf("Unmarshal failed: %v", err)
		}
	}
}

func BenchmarkUnitReadCompiler(b *testing.B) {
	unit := benchmarkCompilerUnit(b)
	path := filepath.Join(b.TempDir(), "compiler.unit")
	if err := os.WriteFile(path, unit, 0o644); err != nil {
		b.Fatalf("WriteFile failed: %v", err)
	}
	b.SetBytes(int64(len(unit)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("ReadFile failed: %v", err)
		}
		if _, err := Unmarshal(data); err != nil {
			b.Fatalf("Unmarshal failed: %v", err)
		}
	}
}

func BenchmarkUnitWriteCompiler(b *testing.B) {
	program := benchmarkCompilerProgram(b)
	unit, err := Marshal(program)
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}
	path := filepath.Join(b.TempDir(), "compiler.unit")
	b.SetBytes(int64(len(unit)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := Marshal(program)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			b.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func BenchmarkGoReadCompilerFiles(b *testing.B) {
	raw := benchmarkCompilerSourceSize(b)
	b.SetBytes(int64(raw))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range benchmarkCompilerFiles {
			if _, err := os.ReadFile(path); err != nil {
				b.Fatalf("ReadFile(%s) failed: %v", path, err)
			}
		}
	}
}

func BenchmarkGoParseCompilerFiles(b *testing.B) {
	raw := benchmarkCompilerSourceSize(b)
	b.SetBytes(int64(raw))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fset := token.NewFileSet()
		for _, path := range benchmarkCompilerFiles {
			if _, err := parser.ParseFile(fset, path, nil, parser.ParseComments); err != nil {
				b.Fatalf("ParseFile(%s) failed: %v", path, err)
			}
		}
	}
}

func BenchmarkGoWriteCompilerBundle(b *testing.B) {
	raw := benchmarkCompilerSourceBundle(b)
	path := filepath.Join(b.TempDir(), "compiler.go.txt")
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			b.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func benchmarkCompilerProgram(b *testing.B) Program {
	b.Helper()
	program, err := ConvertFiles(benchmarkCompilerFiles)
	if err != nil {
		b.Fatalf("ConvertFiles failed: %v", err)
	}
	return program
}

func benchmarkCompilerUnit(b *testing.B) []byte {
	b.Helper()
	unit, err := Marshal(benchmarkCompilerProgram(b))
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}
	return unit
}

func benchmarkCompilerSourceSize(b *testing.B) int {
	b.Helper()
	total := 0
	for _, path := range benchmarkCompilerFiles {
		info, err := os.Stat(path)
		if err != nil {
			b.Fatalf("Stat(%s) failed: %v", path, err)
		}
		total += int(info.Size())
	}
	return total
}

func benchmarkCompilerSourceBundle(b *testing.B) []byte {
	b.Helper()
	var out bytes.Buffer
	for _, path := range benchmarkCompilerFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("ReadFile(%s) failed: %v", path, err)
		}
		out.Write(data)
		out.WriteByte('\n')
	}
	return out.Bytes()
}
