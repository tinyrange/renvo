package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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
	result, err := testmeasure.Run(cmd)
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
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 0 {
			os.Exit(cmd.ProcessState.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
