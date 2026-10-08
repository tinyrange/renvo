//go:build !renvo && linux

package runtime

import (
	"fmt"
	"io"
	"os"
	"renvo.dev/internal/runimage"
	"sync"
)

// One append-only map covers all arenas in this host process. Keep it after
// exit/Close so perf report can resolve historical samples. Exclusive creation
// rejects pre-existing files (including symlinks); profiling failure is nonfatal.
var perfSymbols struct {
	sync.Mutex
	file         *os.File
	err          error
	attempted    bool
	jit          *runimage.JITDump
	jitErr       error
	jitAttempted bool
}

func (n *Native) EnablePerfMap() error {
	perfSymbols.Lock()
	defer perfSymbols.Unlock()
	n.profileMu.Lock()
	defer n.profileMu.Unlock()
	if !perfSymbols.attempted {
		perfSymbols.attempted = true
		perfSymbols.file, perfSymbols.err = os.OpenFile(fmt.Sprintf("/tmp/perf-%d.map", os.Getpid()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	}
	if perfSymbols.err != nil {
		n.SymbolErrors++
		return perfSymbols.err
	}
	n.arena.SetSymbolWriter(perfSymbols.file)
	if os.Getenv("RENVO_RFE_JITDUMP") == "1" {
		if !perfSymbols.jitAttempted {
			perfSymbols.jitAttempted = true
			perfSymbols.jit, perfSymbols.jitErr = runimage.NewJITDump(fmt.Sprintf("jit-%d.dump", os.Getpid()))
		}
		if perfSymbols.jitErr != nil {
			n.SymbolErrors++
			return perfSymbols.jitErr
		}
		n.arena.SetJITWriter(perfSymbols.jit)
		// Stream and discovery marker are process-owned like the perf map.
		// Keep them alive across individual arena Close calls; OS exit closes
		// them, and perf inject accepts complete load records ending at EOF.
	}
	if os.Getenv("RENVO_RFE_PROFILE_CODE") == "1" && !n.profileCaptureAttempted {
		n.profileCaptureAttempted = true
		// The approved profile runs in its fixed artifact directory. Exclusive
		// creation rejects stale files and symlinks; neither file is executable.
		code, err := os.OpenFile("native-code.bin", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			n.SymbolErrors++
			return err
		}
		metadata, err := os.OpenFile("native-code.meta", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			code.Close()
			n.SymbolErrors++
			return err
		}
		n.arena.SetSymbolWriter(io.MultiWriter(perfSymbols.file, metadata))
		n.profileCapture = func() error {
			err := n.arena.WriteCodeSnapshot(code, metadata)
			codeErr := code.Close()
			metadataErr := metadata.Close()
			if err != nil {
				return err
			}
			if codeErr != nil {
				return codeErr
			}
			return metadataErr
		}
	}
	return nil
}
func (n *Native) NameEntry(entry int, name string) error {
	err := n.arena.NameEntry(entry, name)
	if err != nil {
		n.SymbolErrors++
	}
	return err
}
