// Command census summarizes a prepared Linux build tree without
// modifying it. The JSON output is a blocker dashboard, not a compatibility
// gate: time and memory are telemetry while semantic progress is monotonic.
package main

import (
	"debug/elf"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type dashboard struct {
	Schema              int            `json:"schema"`
	Kernel              string         `json:"kernel"`
	Config              string         `json:"config"`
	CommandFiles        int            `json:"command_files"`
	CompilerCommands    int            `json:"compiler_commands"`
	TargetCCommands     int            `json:"target_c_commands"`
	RenvoTargetCommands int            `json:"renvo_target_commands"`
	SourceFiles         map[string]int `json:"source_files"`
	Flags               map[string]int `json:"flags"`
	Syntax              map[string]int `json:"syntax"`
	Builtins            map[string]int `json:"builtins"`
	Objects             objectCensus   `json:"objects"`
	Telemetry           telemetry      `json:"telemetry"`
}

type objectCensus struct {
	Count        int            `json:"count"`
	TotalBytes   int64          `json:"total_bytes"`
	MaximumBytes int64          `json:"maximum_bytes"`
	Sections     map[string]int `json:"sections"`
	Relocations  map[string]int `json:"relocations"`
}

type telemetry struct {
	ElapsedSeconds string `json:"elapsed_seconds,omitempty"`
	UserSeconds    string `json:"user_seconds,omitempty"`
	SystemSeconds  string `json:"system_seconds,omitempty"`
	PeakRSSKiB     string `json:"peak_rss_kib,omitempty"`
}

func main() {
	kernel := flag.String("kernel", "", "prepared Linux source tree")
	output := flag.String("out", "-", "dashboard output path, or - for stdout")
	metrics := flag.String("metrics", "", "optional key=value timing file")
	flag.Parse()
	if *kernel == "" {
		fmt.Fprintln(os.Stderr, "usage: renvokbuildcensus -kernel <tree> [-metrics file] [-out file]")
		os.Exit(2)
	}
	result, err := collect(*kernel, *metrics)
	if err != nil {
		fmt.Fprintln(os.Stderr, "renvokbuildcensus:", err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "renvokbuildcensus:", err)
		os.Exit(1)
	}
	encoded = append(encoded, '\n')
	if *output == "-" {
		_, err = os.Stdout.Write(encoded)
	} else {
		err = os.WriteFile(*output, encoded, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "renvokbuildcensus:", err)
		os.Exit(1)
	}
}

func collect(root string, metricsPath string) (dashboard, error) {
	result := dashboard{
		Schema: 1, Kernel: "linux-6.12.99", Config: "x86_64 tinyconfig",
		SourceFiles: map[string]int{}, Flags: map[string]int{}, Syntax: map[string]int{}, Builtins: map[string]int{},
		Objects: objectCensus{Sections: map[string]int{}, Relocations: map[string]int{}},
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".cmd") {
			return censusCommandFile(root, path, &result)
		}
		if strings.HasSuffix(path, ".o") {
			censusObject(path, &result.Objects)
		}
		return nil
	})
	if err != nil {
		return dashboard{}, err
	}
	if metricsPath != "" {
		data, err := os.ReadFile(metricsPath)
		if err != nil {
			return dashboard{}, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			switch key {
			case "elapsed_seconds":
				result.Telemetry.ElapsedSeconds = value
			case "user_seconds":
				result.Telemetry.UserSeconds = value
			case "system_seconds":
				result.Telemetry.SystemSeconds = value
			case "peak_rss_kib":
				result.Telemetry.PeakRSSKiB = value
			}
		}
	}
	return result, nil
}

func censusCommandFile(root string, path string, result *dashboard) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result.CommandFiles++
	for _, line := range strings.Split(string(data), "\n") {
		at := strings.Index(line, ":=")
		if at < 0 || !strings.Contains(line[:at], "cmd_") {
			continue
		}
		command := strings.TrimSpace(line[at+2:])
		words := shellWords(command)
		if len(words) == 0 {
			continue
		}
		compiler := false
		target := false
		for _, word := range words {
			if word == "-c" || word == "-S" || word == "-E" {
				compiler = true
			}
			if word == "-nostdinc" || word == "-D__KERNEL__" {
				target = true
			}
			if strings.HasPrefix(word, "-") {
				result.Flags[classifyFlag(word)]++
			}
			extension := filepath.Ext(word)
			if extension == ".c" || extension == ".S" || extension == ".s" {
				result.SourceFiles[extension]++
				if extension == ".c" {
					censusSource(filepath.Join(root, word), result)
				}
			}
		}
		if compiler {
			result.CompilerCommands++
			if target {
				result.TargetCCommands++
				if strings.Contains(command, "renvo") {
					result.RenvoTargetCommands++
				}
			}
		}
	}
	return nil
}

func shellWords(command string) []string {
	var out []string
	var word strings.Builder
	quote := rune(0)
	escaped := false
	for _, ch := range command + " " {
		if escaped {
			word.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			} else {
				word.WriteRune(ch)
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			if word.Len() > 0 {
				out = append(out, word.String())
				word.Reset()
			}
			continue
		}
		word.WriteRune(ch)
	}
	return out
}

func classifyFlag(flag string) string {
	for _, prefix := range []string{"-D", "-I", "-Wp,", "-Wa,", "-W", "-f", "-m", "-O", "-std="} {
		if strings.HasPrefix(flag, prefix) {
			return prefix
		}
	}
	return flag
}

func censusSource(path string, result *dashboard) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(data)
	for name, spelling := range map[string]string{
		"asm": "asm", "attribute": "__attribute__", "typeof": "typeof", "auto_type": "__auto_type", "statement_expression": "({",
	} {
		result.Syntax[name] += strings.Count(text, spelling)
	}
	for at := 0; ; {
		index := strings.Index(text[at:], "__builtin_")
		if index < 0 {
			break
		}
		start := at + index
		end := start
		for end < len(text) && (text[end] == '_' || text[end] >= 'a' && text[end] <= 'z' || text[end] >= 'A' && text[end] <= 'Z' || text[end] >= '0' && text[end] <= '9') {
			end++
		}
		result.Builtins[text[start:end]]++
		at = end
	}
}

func censusObject(path string, result *objectCensus) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	file, err := elf.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	result.Count++
	result.TotalBytes += info.Size()
	if info.Size() > result.MaximumBytes {
		result.MaximumBytes = info.Size()
	}
	for _, section := range file.Sections {
		result.Sections[section.Name]++
		if section.Type != elf.SHT_RELA {
			continue
		}
		data, err := section.Data()
		if err != nil {
			continue
		}
		for at := 0; at+24 <= len(data); at += 24 {
			kind := elf.R_X86_64(uint32(binary.LittleEndian.Uint64(data[at+8 : at+16])))
			result.Relocations[kind.String()]++
		}
	}
}

// Keep deterministic iteration available to tests and future text formats.
func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
