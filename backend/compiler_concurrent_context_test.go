package main

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentTargetCompilesAreIndependent(t *testing.T) {
	initialTarget := renvoTarget
	initialOS := renvoTargetOS
	initialArch := renvoTargetArch
	initialIntSize := renvoNativeIntSize
	initialStrip := renvoCompilerStripSymbols
	source := []byte(`package main

func value(x int) int {
	return x*7 + 3
}

func appMain() int {
	if value(5) != 38 {
		return 1
	}
	print("PASS\n")
	return 0
}
`)
	cases := []struct {
		target  string
		options RenvoCompileOptions
	}{
		{target: "linux/amd64"},
		{target: "linux/386", options: RenvoCompileOptions{StripSymbols: true}},
		{target: "linux/aarch64"},
		{target: "linux/arm", options: RenvoCompileOptions{StripSymbols: true}},
		{target: "windows/amd64", options: RenvoCompileOptions{WindowsGUI: true}},
		{target: "windows/386", options: RenvoCompileOptions{StripSymbols: true}},
		{target: "windows/arm64", options: RenvoCompileOptions{WindowsGUI: true}},
		{target: "darwin/arm64"},
		{target: "wasi/wasm32", options: RenvoCompileOptions{StripSymbols: true}},
		{target: "vm/vm32"},
	}
	baselines := make([][]byte, len(cases))
	for i := range cases {
		output, ok := RenvoCompileSourceToBytesWithOptions(source, cases[i].target, cases[i].options)
		if !ok {
			t.Fatalf("serial %s compilation failed", cases[i].target)
		}
		baselines[i] = output
	}

	type compileJob struct {
		caseIndex int
		iteration int
	}
	const iterations = 2
	const parallelCompiles = 3
	start := make(chan struct{})
	jobs := make(chan compileJob, len(cases)*iterations)
	var errors []string
	var errorsMu sync.Mutex
	var workers sync.WaitGroup
	for worker := 0; worker < parallelCompiles; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for {
				job := <-jobs
				if job.caseIndex < 0 {
					return
				}
				output, ok := RenvoCompileSourceToBytesWithOptions(source, cases[job.caseIndex].target, cases[job.caseIndex].options)
				if !ok {
					errorsMu.Lock()
					errors = append(errors, fmt.Sprintf("%s iteration %d compilation failed", cases[job.caseIndex].target, job.iteration))
					errorsMu.Unlock()
					continue
				}
				if !bytes.Equal(output, baselines[job.caseIndex]) {
					errorsMu.Lock()
					errors = append(errors, fmt.Sprintf("%s iteration %d output differs from serial baseline", cases[job.caseIndex].target, job.iteration))
					errorsMu.Unlock()
				}
			}
		}()
	}
	for caseIndex := range cases {
		for iteration := 0; iteration < iterations; iteration++ {
			jobs <- compileJob{caseIndex: caseIndex, iteration: iteration}
		}
	}
	for worker := 0; worker < parallelCompiles; worker++ {
		start <- struct{}{}
	}
	for worker := 0; worker < parallelCompiles; worker++ {
		jobs <- compileJob{caseIndex: -1}
	}
	workers.Wait()
	for _, message := range errors {
		t.Error(message)
	}
	if renvoTarget != initialTarget || renvoTargetOS != initialOS || renvoTargetArch != initialArch ||
		renvoNativeIntSize != initialIntSize || renvoCompilerStripSymbols != initialStrip {
		t.Fatal("host compilation mutated legacy target or strip globals")
	}
}
