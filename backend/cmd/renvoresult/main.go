package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"renvo.dev/backend/omnibus/resultabi"
)

type output struct {
	Profile         uint32 `json:"profile"`
	State           string `json:"state"`
	CurrentProbe    uint32 `json:"current_probe"`
	FailureProbe    uint32 `json:"failure_probe,omitempty"`
	CompletedProbes uint32 `json:"completed_probes"`
	Expected        string `json:"expected"`
	Observed        string `json:"observed"`
	Signature       string `json:"signature"`
	Sequence        uint32 `json:"sequence"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "renvoresult:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("renvoresult", flag.ContinueOnError)
	artifact := flags.String("artifact", "", "ELF artifact containing the result symbol")
	memory := flags.String("memory", "", "raw target memory dump")
	baseText := flags.String("base", "0", "address represented by byte zero of the memory dump")
	symbol := flags.String("symbol", resultabi.SymbolName, "result symbol name")
	profileText := flags.String("expected-profile", "", "required profile identifier")
	signatureText := flags.String("expected-signature", "", "required final signature")
	hosted := flags.Bool("hosted", false, "print only PASS on validated success")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *artifact == "" || *memory == "" {
		return errors.New("-artifact and -memory are required")
	}
	base, err := strconv.ParseUint(*baseText, 0, 64)
	if err != nil {
		return fmt.Errorf("parse -base: %w", err)
	}
	snapshot, err := resultabi.DecodeMemoryDump(*artifact, *memory, base, *symbol)
	if err != nil {
		return err
	}
	if *hosted {
		if *profileText == "" || *signatureText == "" {
			return errors.New("-hosted requires -expected-profile and -expected-signature")
		}
		profile, err := strconv.ParseUint(*profileText, 0, 32)
		if err != nil {
			return fmt.Errorf("parse -expected-profile: %w", err)
		}
		signature, err := strconv.ParseUint(*signatureText, 0, 64)
		if err != nil {
			return fmt.Errorf("parse -expected-signature: %w", err)
		}
		if err := snapshot.ValidatePass(uint32(profile), signature); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, "PASS")
		return err
	}
	value := output{
		Profile:         snapshot.Profile,
		State:           snapshot.State.String(),
		CurrentProbe:    snapshot.CurrentProbe,
		FailureProbe:    snapshot.FailureProbe,
		CompletedProbes: snapshot.CompletedProbes,
		Expected:        fmt.Sprintf("%#x", snapshot.Expected),
		Observed:        fmt.Sprintf("%#x", snapshot.Observed),
		Signature:       fmt.Sprintf("%#x", snapshot.Signature),
		Sequence:        snapshot.Sequence,
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
