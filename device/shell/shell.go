// Package shell is a small synchronous filesystem shell, not a Bash interpreter.
// It executes built-ins only; no external programs, pipes or variable expansion.
package shell

import (
	"renvo.dev/device/fat32"
	"renvo.dev/std/fmt"
	"renvo.dev/std/io"
	"renvo.dev/std/strings"
)

type Shell struct {
	Volume *fat32.Volume
	Output io.Writer
	Cwd    string
}

func New(v *fat32.Volume, out io.Writer) *Shell { return &Shell{Volume: v, Output: out, Cwd: "/"} }
func (s *Shell) Prompt()                        { fmt.Fprintf(s.Output, "\x1b[32mtab5\x1b[0m:%s$ ", s.Cwd) }

// Split accepts single/double quoted arguments and backslash escapes. It rejects
// unfinished quotes rather than accidentally executing a different pathname.
func Split(line string) ([]string, error) {
	args := []string{}
	word := ""
	quote := byte(0)
	started := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '\\' && quote != '\'' {
			i++
			if i == len(line) {
				return nil, fat32.Error("shell: trailing escape")
			}
			word += line[i : i+1]
			started = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			} else {
				word += line[i : i+1]
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			started = true
			continue
		}
		if c == ' ' || c == '\t' {
			if started {
				args = append(args, word)
				word = ""
				started = false
			}
			continue
		}
		word += line[i : i+1]
		started = true
	}
	if quote != 0 {
		return nil, fat32.Error("shell: unfinished quote")
	}
	if started {
		args = append(args, word)
	}
	return args, nil
}

func (s *Shell) path(p string) string    { return fat32.Clean(s.Cwd, p) }
func (s *Shell) usage(text string) error { return fat32.Error("usage: " + text) }

// Execute prints errors and leaves Cwd unchanged on a failed cd. Mutations occur
// only in explicit mkdir/touch/write/rm/rmdir commands; there is no formatting or
// raw-sector write command. File text is escaped to prevent embedded VT controls.
func (s *Shell) Execute(line string) {
	args, err := Split(line)
	if err == nil && len(args) > 0 {
		err = s.run(args)
	}
	if err != nil {
		fmt.Fprintf(s.Output, "%s\r\n", err)
	}
}
func (s *Shell) run(a []string) error {
	cmd := a[0]
	switch cmd {
	case "help":
		fmt.Fprint(s.Output, "pwd  ls [-a] [path]  cd [path]  cat file  head file  hex file  stat path\r\nmkdir dir  touch file  write file text...  rm file  rmdir dir  sync  clear\r\nls hides dot/FAT-hidden entries; -a reveals them. Use -- before -names.\r\nQuotes and backslash escapes supported. New names: ASCII 8.3.\r\nNo pipes, globbing, programs or scripts. Ctrl+O rotates; Ctrl+L clears.\r\n")
	case "pwd":
		fmt.Fprintf(s.Output, "%s\r\n", s.Cwd)
	case "clear":
		fmt.Fprint(s.Output, "\x1b[2J\x1b[H")
	case "sync":
		return s.Volume.Sync()
	case "ls":
		path := s.Cwd
		all, options, havePath := false, true, false
		for _, arg := range a[1:] {
			if options && arg == "--" {
				options = false
				continue
			}
			if options && strings.HasPrefix(arg, "-") && arg != "-" {
				for i := 1; i < len(arg); i++ {
					if arg[i] == 'a' || arg[i] == 'A' {
						all = true
					} else if arg[i] != 'l' {
						return s.usage("ls [-aAl] [--] [path]")
					}
				}
				continue
			}
			if havePath {
				return s.usage("ls [-aAl] [--] [path]")
			}
			path = s.path(arg)
			havePath = true
		}
		entry, err := s.Volume.Stat(path)
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			s.listEntry(entry)
			return nil
		}
		entries, err := s.Volume.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !all && e.IsHidden() {
				continue
			}
			s.listEntry(e)
		}
	case "cd":
		if len(a) > 2 {
			return s.usage("cd [path]")
		}
		path := "/"
		if len(a) == 2 {
			path = s.path(a[1])
		}
		e, err := s.Volume.Stat(path)
		if err != nil {
			return err
		}
		if !e.IsDir() {
			return fat32.ErrNotDir
		}
		s.Cwd = path
	case "stat":
		if len(a) != 2 {
			return s.usage("stat path")
		}
		e, err := s.Volume.Stat(s.path(a[1]))
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Output, "%s: bytes=%d directory=%t attributes=%d\r\n", safe(e.Name), e.Size, e.IsDir(), e.Attributes)
	case "cat", "head", "hex":
		if len(a) != 2 {
			return s.usage(cmd + " file")
		}
		return s.show(s.path(a[1]), cmd)
	case "mkdir":
		if len(a) != 2 {
			return s.usage("mkdir dir")
		}
		return s.Volume.Mkdir(s.path(a[1]))
	case "touch":
		if len(a) != 2 {
			return s.usage("touch file")
		}
		_, err := s.Volume.Stat(s.path(a[1]))
		if err == nil {
			return nil
		}
		if !fat32.Is(err, fat32.ErrNotFound) {
			return err
		}
		return s.Volume.WriteFile(s.path(a[1]), nil)
	case "write":
		if len(a) < 3 {
			return s.usage("write file text...")
		}
		return s.Volume.WriteFile(s.path(a[1]), []byte(strings.Join(a[2:], " ")+"\n"))
	case "rm", "rmdir":
		if len(a) != 2 {
			return s.usage(cmd + " path")
		}
		path := s.path(a[1])
		e, err := s.Volume.Stat(path)
		if err != nil {
			return err
		}
		if cmd == "rm" && e.IsDir() {
			return fat32.ErrIsDir
		}
		if cmd == "rmdir" && !e.IsDir() {
			return fat32.ErrNotDir
		}
		if path == s.Cwd || strings.HasPrefix(s.Cwd, path+"/") {
			return fat32.ErrReadOnly
		}
		return s.Volume.Remove(path)
	default:
		return fat32.Error("shell: unknown command: " + cmd)
	}
	return nil
}

func (s *Shell) listEntry(e fat32.Entry) {
	name := safe(e.Name)
	if e.IsDir() {
		name += "/"
	}
	// Quote whitespace and shell metacharacters so the displayed spelling can
	// be pasted back into Split, including names containing apostrophes.
	quote := false
	for _, c := range name {
		if c == ' ' || c == '\t' || c == '\'' || c == '"' || c == '\\' {
			quote = true
		}
	}
	if quote {
		name = "'" + strings.ReplaceAll(name, "'", "'\\''") + "'"
	}
	fmt.Fprintf(s.Output, "%s  %d\r\n", name, e.Size)
}

func safe(text string) string {
	b := []byte(text)
	for i, c := range b {
		if c < 32 || c == 127 {
			b[i] = '?'
		}
	}
	return string(b)
}
func (s *Shell) show(path, mode string) error {
	f, err := s.Volume.Open(path)
	if err != nil {
		return err
	}
	var buffer [4096]byte
	lines := 0
	offset := 0
	previousCR := false
	ended := true
	// Bound terminal output to 64 KiB so a large/binary file cannot monopolize
	// the single-threaded UI. head is further limited to 20 lines.
	for offset < 65536 && uint32(offset) < f.Size() {
		n, readErr := f.Read(buffer[:])
		out := make([]byte, 0, n*2)
		for i := 0; i < n; i++ {
			c := buffer[i]
			if mode == "hex" {
				const digits = "0123456789abcdef"
				out = append(out, digits[c>>4], digits[c&15], ' ')
				if (offset+i+1)%16 == 0 {
					out = append(out, '\r', '\n')
				}
				continue
			}
			if c == '\r' {
				out = append(out, '\r', '\n')
				lines++
				previousCR = true
				ended = true
			} else if c == '\n' {
				if !previousCR {
					out = append(out, '\r', '\n')
					lines++
				}
				previousCR = false
				ended = true
			} else {
				previousCR = false
				ended = false
				if c < 32 && c != '\t' || c == 127 {
					c = '.'
				}
				out = append(out, c)
			}
			if mode == "head" && lines >= 20 {
				s.Output.Write(out)
				return nil
			}
		}
		if _, err = s.Output.Write(out); err != nil {
			return err
		}
		offset += n
		if readErr != nil {
			return readErr
		}
		if n == 0 {
			break
		}
	}
	if !ended || mode == "hex" && offset%16 != 0 {
		fmt.Fprint(s.Output, "\r\n")
	}
	if offset >= 65536 {
		fmt.Fprint(s.Output, "[output limited to 64 KiB]\r\n")
	}
	return nil
}
