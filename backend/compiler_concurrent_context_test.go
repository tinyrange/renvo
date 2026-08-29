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

func TestConcurrentKernelCompilesOwnMetadata(t *testing.T) {
	const workers = 4
	source := []byte("package main\nvar count int\nvar stamp uint64\n// renvo:linkstatic kernel,ktime_get_ns\nfunc kernelKtimeGetNS() uint64 { return 0 }\n// renvo:linkstatic kernel,for_each_kernel_tracepoint\nfunc kernelForEach(callback func(uintptr, uintptr), data uintptr) {}\nfunc callback(tp uintptr, data uintptr) {}\nfunc bump() { count++ }\nfunc appMain() { kernelForEach(callback, 0); stamp = kernelKtimeGetNS(); for i := 0; i < 3; i++ { bump() }; if count == 3 && stamp >= 0 { print(\"100% PASS\\n\") } }\nfunc moduleExit() { kernelForEach(callback, 0); print(\"EXIT\\n\") }\n")
	baselines := make([][]byte, workers)
	for i := 0; i < workers; i++ {
		var ok bool
		baselines[i], ok = compileConcurrentKernel(source, fmt.Sprintf("module%d.ko", i), "GPL")
		if !ok {
			t.Fatalf("serial kernel compilation %d failed", i)
		}
	}
	start := make(chan struct{})
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			output, ok := compileConcurrentKernel(source, fmt.Sprintf("module%d.ko", index), "GPL")
			if !ok || !bytes.Equal(output, baselines[index]) {
				t.Errorf("parallel kernel compilation %d differs from its serial baseline", index)
			}
		}(i)
	}
	for i := 0; i < workers; i++ {
		start <- struct{}{}
	}
	wait.Wait()
}

func compileConcurrentKernel(source []byte, outputPath string, license string) ([]byte, bool) {
	context := renvoNewCompileContext(renvoTargetLinuxKernelAmd64, true, false, false)
	renvoConfigureCompileContext(context, "linux/kernel-amd64", outputPath, license)
	context.kernel.kernelRelease = "6.18.0-test"
	context.kernel.kernelVersion = "Linux version 6.18.0-test SMP PREEMPT"
	context.kernel.kernelBTF = testKernelBTF()
	context.kernel.kernelSymvers = []byte("0xb1976aeb\tmodule_layout\tvmlinux\tEXPORT_SYMBOL\n0x92997ed8\t_printk\tvmlinux\tEXPORT_SYMBOL\n0x11223344\tktime_get_ns\tvmlinux\tEXPORT_SYMBOL_GPL\n0x55667788\tfor_each_kernel_tracepoint\tvmlinux\tEXPORT_SYMBOL_GPL\n")
	context.kernel.kernelModuleSize = 128
	context.kernel.kernelNameOff = 16
	context.kernel.kernelInitOff = 32
	context.kernel.kernelExitOff = 64
	program := renvoParseProgramWithContext(source, context)
	result := renvoCompileParsedProgramArena(&program, renvoTargetLinuxKernelAmd64, 4096)
	return result.data, result.ok
}
