package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"renvo.dev/internal/testmeasure"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: renvomeasure RESULT COMMAND [ARG]...")
		os.Exit(2)
	}
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	started := time.Now()
	err := cmd.Run()
	result := testmeasure.Result{ElapsedNanoseconds: int64(time.Since(started))}
	if cmd.ProcessState != nil {
		result.CPUNanoseconds = int64(cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime())
		result.MaxRSSKB = processMaxRSSKB(cmd.ProcessState)
	}
	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, marshalErr)
		os.Exit(1)
	}
	if writeErr := os.WriteFile(os.Args[1], data, 0o644); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(1)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
