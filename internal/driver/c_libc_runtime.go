package driver

// cLibcRuntime is target code, not a host syscall adapter. File access uses the
// same portable builtins as Renvo's Go runtime. The Linux/BSD flag difference
// is selected from the compilation target, never from the compiler host.
func cLibcRuntime(targetOS, targetArch string) []byte {
	flags := "const cCreate = 64; const cTruncate = 512; const cAppend = 1024; const cExclusive = 128\n"
	if targetOS == "darwin" || targetOS == "freebsd" || targetOS == "openbsd" || targetOS == "netbsd" {
		flags = "const cCreate = 512; const cTruncate = 1024; const cAppend = 8; const cExclusive = 2048\n"
	}
	processNames := ""
	processNameGlobals := ""
	openRuntime := "func __renvo_c_open_file(data []byte, flags int) int { return open(string(data), flags) }\n"
	// The portable two-argument open builtin creates executable-mode files.
	// fopen must instead supply 0666 atomically, letting the target kernel apply
	// umask only on creation. Never chmod afterwards (which changes existing files).
	if targetOS == "linux" && targetArch == "amd64" {
		processNames = ` name := ""
 if len(args)>0 {name=args[0]}
 __renvo_c_program=[]byte(name+"\x00")
 program_invocation_name=(*int8)(unsafe.Pointer(&__renvo_c_program[0]))
 last:=0
 for i:=0;i<len(name);i++ {if name[i]=='/' {last=i+1}}
 program_invocation_short_name=(*int8)(unsafe.Pointer(&__renvo_c_program[last]))
`
		processNameGlobals = "var __renvo_c_program []byte\nvar program_invocation_name *int8\nvar program_invocation_short_name *int8\n"
		openRuntime = "func __renvo_c_getrlimit(resource int32, limit *byte) int32 { return int32(syscall(97, uintptr(resource), uintptr(unsafe.Pointer(limit)), 0)) }\nfunc syscall(number int, a uintptr, b uintptr, c uintptr) int { return 0 }\nfunc __renvo_c_open_file(data []byte, flags int) int { return syscall(2, uintptr(unsafe.Pointer(&data[0])), uintptr(flags), 0666) }\nfunc __renvo_c_dup2(oldfd int32, newfd int32) int32 { return int32(syscall(33, uintptr(oldfd), uintptr(newfd), 0)) }\nfunc __renvo_c_fcntl(fd int32, action int32, argument int32) int32 { return int32(syscall(72, uintptr(fd), uintptr(action), uintptr(argument))) }\n"
	}
	return []byte("package main\nimport \"unsafe\"\n" + flags + openRuntime + `
func __renvo_c_open(name *byte, flags int32) int32 {
 data := make([]byte, 0, 128)
 for i := uintptr(0); i < 4096; i++ {
  ch := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(name)) + i))
  data = append(data, ch)
  if ch == 0 {
   native := 0
   if flags & 3 == 3 { native = 2 } else if flags & 2 != 0 { native = 1 }
   if flags & 4 != 0 { native |= cCreate }
   if flags & 8 != 0 { native |= cTruncate }
   if flags & 16 != 0 { native |= cAppend }
   if flags & 32 != 0 { native |= cExclusive }
   return int32(__renvo_c_open_file(data, native))
  }
 }
 return -36
}
func __renvo_c_close(fd int32) int32 { return int32(close(int(fd))) }
func __renvo_c_write_byte(fd int32, ch int32) int32 {
 data := []byte{byte(ch)}
 n := write(int(fd), data, -1)
 if n < 0 { return int32(n) }
 if n != 1 { return -5 }
 return int32(byte(ch))
}
func __renvo_c_read_byte(fd int32, out *byte) int32 {
 data := []byte{0}
 n := read(int(fd), data, -1)
 if n == 1 { *out = data[0] }
 return int32(n)
}
var __renvo_c_env [][]byte
` + processNameGlobals + `
func __renvo_c_set_process(args []string, env []string) {
` + processNames + ` __renvo_c_env = make([][]byte, len(env))
 for i := 0; i < len(env); i++ { __renvo_c_env[i] = []byte(env[i] + "\x00") }
}
func __renvo_c_getenv(name *byte) *byte {
 for i := 0; i < len(__renvo_c_env); i++ {
  data := __renvo_c_env[i]
  for j := 0; j < len(data); j++ {
   ch := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(name)) + uintptr(j)))
   if ch == 0 { if j != 0 && data[j] == '=' { return &data[j+1] }; break }
   if ch == '=' || ch != data[j] { break }
  }
 }
 return nil
}
func renvo_runtime_Exit(status int32) {}
func __renvo_c_abort(status int32) { renvo_runtime_Exit(status) }
`)
}
