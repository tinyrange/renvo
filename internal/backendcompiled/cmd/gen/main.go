package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	backend := flag.String("backend", "", "backend source directory")
	output := flag.String("o", "", "generated output")
	sourcesOutput := flag.String("sources", "", "generated embedded source bundle")
	flag.Parse()
	if *backend == "" || *output == "" || *sourcesOutput == "" {
		fmt.Fprintln(os.Stderr, "usage: gen -backend directory -o output.go -sources sources.go")
		os.Exit(2)
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
	out.WriteString("package backendcompiled\n\n")
	sourceBundle.WriteString("// Code generated from checked-in RTG backend outputs; DO NOT EDIT.\n")
	sourceBundle.WriteString("//go:build !renvo\n\n")
	sourceBundle.WriteString("package backendcompiled\n\n")
	sourceBundle.WriteString("const CompilerSourceCount = ")
	sourceBundle.WriteString(strconv.Itoa(len(names)))
	sourceBundle.WriteString("\n\n")
	var sourceNames bytes.Buffer
	var sourceSizes bytes.Buffer
	var sourceChunkCounts bytes.Buffer
	var sourceChunks bytes.Buffer
	var sourceConstants bytes.Buffer
	for sourceIndex, name := range names {
		source, err := os.ReadFile(filepath.Join(*backend, name))
		if err != nil {
			fail(err)
		}
		packageAt := bytes.Index(source, []byte("package main\n"))
		if packageAt < 0 {
			fail(fmt.Errorf("%s has no package main declaration", name))
		}
		digestSource.WriteString(name)
		digestSource.WriteByte(0)
		digestSource.Write(source)
		out.WriteString("// source: backend/")
		out.WriteString(name)
		out.WriteByte('\n')
		out.Write(source[:packageAt])
		out.Write(source[packageAt+len("package main\n"):])
		out.WriteByte('\n')
		indexText := strconv.Itoa(sourceIndex)
		sourceNames.WriteString("\tif index == ")
		sourceNames.WriteString(indexText)
		sourceNames.WriteString(" { return ")
		sourceNames.WriteString(strconv.Quote(name))
		sourceNames.WriteString(" }\n")
		sourceSizes.WriteString("\tif index == ")
		sourceSizes.WriteString(indexText)
		sourceSizes.WriteString(" { return ")
		sourceSizes.WriteString(strconv.Itoa(len(source)))
		sourceSizes.WriteString(" }\n")
		chunks := compressedChunks(compress(source))
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
	packageEnd := bytes.Index(compilerSource, []byte("package backendcompiled\n"))
	packageEnd += len("package backendcompiled\n")
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
	if err := os.WriteFile(*sourcesOutput, sourceBundle.Bytes(), 0o644); err != nil {
		fail(err)
	}
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
	for start := 0; start < len(source); start += chunkSize {
		if start != 0 {
			out.WriteByte('+')
		}
		end := start + chunkSize
		if end > len(source) {
			end = len(source)
		}
		out.WriteString(strconv.Quote(string(source[start:end])))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "backend bundle:", err)
	os.Exit(1)
}
