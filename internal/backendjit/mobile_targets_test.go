package backendjit

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/apk"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func compileMobileTarget(
	t *testing.T, definitionName string, target string, output string,
) []byte {
	return compileMobileProgram(t, definitionName, target, output,
		filepath.Join("internal", "backendjit", "testdata", "mobile_entry.go"))
}

func compileMobileProgram(
	t *testing.T, definitionName string, target string, output string,
	program string,
) []byte {
	t.Helper()
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s",
			runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "examples", definitionName)
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", target,
		"-s",
		"-o", output,
		filepath.Join(root, program),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("%s custom backend compile failed: %#v", target, result.Diagnostic)
	}
	return result.Binary
}

func TestCompilerJITAndroidARM64NativeActivityImage(t *testing.T) {
	image := compileMobileTarget(t,
		filepath.Join("android", "android_arm64.rtg"),
		"android/arm64", "librenvo.so")
	if len(image) < 64 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("Android output is not ELF64: % x", image[:minInt(4, len(image))])
	}
	if image[4] != 2 || image[5] != 1 {
		t.Fatalf("Android ELF class/data = %d/%d, want ELF64 little-endian",
			image[4], image[5])
	}
	if kind := binary.LittleEndian.Uint16(image[16:18]); kind != 3 {
		t.Fatalf("Android ELF type = %d, want ET_DYN (3)", kind)
	}
	if machine := binary.LittleEndian.Uint16(image[18:20]); machine != 183 {
		t.Fatalf("Android ELF machine = %d, want AArch64 (183)", machine)
	}
	if entry := binary.LittleEndian.Uint64(image[24:32]); entry != 0 {
		t.Fatalf("Android ELF entry = %#x, want no process entry", entry)
	}

	phoff := int(binary.LittleEndian.Uint64(image[32:40]))
	phentsize := int(binary.LittleEndian.Uint16(image[54:56]))
	phnum := int(binary.LittleEndian.Uint16(image[56:58]))
	if phentsize != 56 || phnum < 4 || phoff+phentsize*phnum > len(image) {
		t.Fatalf("invalid Android program-header table: off=%d size=%d count=%d",
			phoff, phentsize, phnum)
	}
	dynamicOffset := -1
	dynamicSize := 0
	loadCount := 0
	hasExecutableStack := false
	for i := 0; i < phnum; i++ {
		at := phoff + i*phentsize
		kind := binary.LittleEndian.Uint32(image[at : at+4])
		flags := binary.LittleEndian.Uint32(image[at+4 : at+8])
		offset := binary.LittleEndian.Uint64(image[at+8 : at+16])
		address := binary.LittleEndian.Uint64(image[at+16 : at+24])
		fileSize := binary.LittleEndian.Uint64(image[at+32 : at+40])
		alignment := binary.LittleEndian.Uint64(image[at+48 : at+56])
		if offset+fileSize > uint64(len(image)) {
			t.Fatalf("Android segment %d extends past the file", i)
		}
		if kind == 1 {
			loadCount++
			if alignment != 0x4000 || offset%alignment != address%alignment {
				t.Fatalf("Android PT_LOAD %d is not 16 KiB congruent", i)
			}
			if flags&3 == 3 {
				t.Fatalf("Android PT_LOAD %d is writable and executable", i)
			}
		}
		if kind == 2 {
			dynamicOffset = int(offset)
			dynamicSize = int(fileSize)
		}
		if kind == 0x6474e551 && flags&1 != 0 {
			hasExecutableStack = true
		}
	}
	if loadCount != 2 || dynamicOffset < 0 || dynamicSize == 0 {
		t.Fatalf("Android ELF loads=%d dynamic=[%d,%d)",
			loadCount, dynamicOffset, dynamicOffset+dynamicSize)
	}
	if hasExecutableStack {
		t.Fatal("Android ELF requests an executable stack")
	}

	dynamic := make(map[uint64]uint64)
	for at := dynamicOffset; at+16 <= dynamicOffset+dynamicSize; at += 16 {
		tag := binary.LittleEndian.Uint64(image[at : at+8])
		value := binary.LittleEndian.Uint64(image[at+8 : at+16])
		dynamic[tag] = value
		if tag == 0 {
			break
		}
	}
	strtab := int(dynamic[5])
	strsize := int(dynamic[10])
	symtab := int(dynamic[6])
	if strtab <= 0 || strsize <= 0 || strtab+strsize > len(image) ||
		symtab <= 0 || symtab+48 > len(image) || dynamic[11] != 24 {
		t.Fatalf("invalid Android dynamic tables: %#v", dynamic)
	}
	strings := image[strtab : strtab+strsize]
	nameOffset := int(binary.LittleEndian.Uint32(image[symtab+24 : symtab+28]))
	if got := mobileCString(strings, nameOffset); got != "ANativeActivity_onCreate" {
		t.Fatalf("Android exported symbol = %q", got)
	}
	if image[symtab+28] != 0x12 ||
		binary.LittleEndian.Uint16(image[symtab+30:symtab+32]) == 0 {
		t.Fatal("Android NativeActivity entry is not a defined global function")
	}
	if value := binary.LittleEndian.Uint64(image[symtab+32 : symtab+40]); value != 0x4000 {
		t.Fatalf("Android NativeActivity address = %#x, want 0x4000", value)
	}
	if got := mobileCString(strings, int(dynamic[14])); got != "librenvo.so" {
		t.Fatalf("Android SONAME = %q", got)
	}
	if binary.LittleEndian.Uint32(image[0x4000:0x4004]) != 0xa9bf7bf3 {
		t.Fatal("Android entry does not preserve x19 and the NativeActivity link register")
	}
	if binary.LittleEndian.Uint32(image[0x4004:0x4008]) != 0xaa0003f3 {
		t.Fatal("Android entry does not retain the NativeActivity pointer in x19")
	}
	if !bytes.Contains(image[0x4000:], []byte{0xe9, 0x0f, 0x1f, 0xf8}) {
		t.Fatal("Android AAPCS64 stack path omitted its aligned push hook")
	}
	if bytes.Contains(image[0x4000:], []byte{0xbf, 0xff, 0x00, 0x00}) ||
		bytes.Contains(image[0x4000:], []byte{0xc0, 0xdf, 0x00, 0x00}) {
		t.Fatal("Android code contains a truncated packed AArch64 instruction")
	}
	returnSequence := []byte{0xf3, 0x7b, 0xc1, 0xa8, 0xc0, 0x03, 0x5f, 0xd6}
	if !bytes.Contains(image[0x4000:], returnSequence) {
		t.Fatal("Android entry does not return to the NativeActivity caller")
	}
}

func TestCompilerJITAndroidARM64FormsImage(t *testing.T) {
	image := compileMobileProgram(t,
		filepath.Join("android", "android_arm64.rtg"),
		"android/arm64", "librenvo.so",
		filepath.Join("examples", "forms_hello"))

	expectedImports := []string{
		"AInputEvent_getType",
		"AKeyEvent_getAction",
		"AKeyEvent_getKeyCode",
		"AKeyEvent_getMetaState",
		"AKeyEvent_getRepeatCount",
		"AInputQueue_attachLooper",
		"AInputQueue_detachLooper",
		"AInputQueue_finishEvent",
		"AInputQueue_getEvent",
		"AInputQueue_preDispatchEvent",
		"ALooper_forThread",
		"AMotionEvent_getAction",
		"AMotionEvent_getPointerCount",
		"AMotionEvent_getPointerId",
		"AMotionEvent_getX",
		"AMotionEvent_getY",
		"ANativeWindow_getHeight",
		"ANativeWindow_getWidth",
		"ANativeWindow_setBuffersGeometry",
		"ANativeWindow_lock",
		"ANativeWindow_unlockAndPost",
		"memcpy",
	}
	for _, name := range expectedImports {
		if !bytes.Contains(image, append([]byte(name), 0)) {
			t.Errorf("Android Forms image omits dynamic import %q", name)
		}
	}
	if !bytes.Contains(image, []byte("Renvo for Android")) ||
		bytes.Contains(image, []byte("Renvo for iPhone")) {
		t.Fatal("shared Forms hello selected the wrong Android platform title")
	}
	if bytes.Contains(image, []byte("__android_log_write")) {
		t.Fatal("Android Forms image contains a temporary device logging probe")
	}
	if !bytes.Contains(image[0x4000:], []byte{
		0x00, 0x00, 0x38, 0x9e, // fcvtzs x0, s0
	}) {
		t.Fatal("Android Forms image does not bridge C float motion coordinates to integer pixels")
	}
	// Callback-table layout belongs to the Go client. The target materializes
	// function pointers but must not encode NativeActivity callback slots.
	for _, store := range [][]byte{
		{0x2a, 0x81, 0x03, 0xf8}, // stur x10, [x9, #56]
		{0x2a, 0x01, 0x05, 0xf8}, // stur x10, [x9, #80]
	} {
		if bytes.Contains(image[0x4000:], store) {
			t.Fatalf("Android target still owns callback-table store % x", store)
		}
	}

	phoff := int(binary.LittleEndian.Uint64(image[32:40]))
	phentsize := int(binary.LittleEndian.Uint16(image[54:56]))
	phnum := int(binary.LittleEndian.Uint16(image[56:58]))
	dynamicOffset, dynamicSize := -1, 0
	for i := 0; i < phnum; i++ {
		at := phoff + i*phentsize
		if binary.LittleEndian.Uint32(image[at:at+4]) == 2 {
			dynamicOffset = int(binary.LittleEndian.Uint64(image[at+8 : at+16]))
			dynamicSize = int(binary.LittleEndian.Uint64(image[at+32 : at+40]))
		}
	}
	if dynamicOffset < 0 {
		t.Fatal("Android Forms image has no PT_DYNAMIC")
	}
	dynamic := make(map[uint64]uint64)
	for at := dynamicOffset; at+16 <= dynamicOffset+dynamicSize; at += 16 {
		tag := binary.LittleEndian.Uint64(image[at : at+8])
		dynamic[tag] = binary.LittleEndian.Uint64(image[at+8 : at+16])
		if tag == 0 {
			break
		}
	}
	if dynamic[7] == 0 || dynamic[8] != uint64(len(expectedImports)*24) || dynamic[9] != 24 {
		t.Fatalf("Android Forms RELA contract is invalid: %#v", dynamic)
	}
	for at := int(dynamic[7]); at < int(dynamic[7]+dynamic[8]); at += 24 {
		if kind := binary.LittleEndian.Uint32(image[at+8 : at+12]); kind != 1025 {
			t.Fatalf("Android Forms relocation at %#x = %d, want R_AARCH64_GLOB_DAT", at, kind)
		}
	}
}

func TestCompilerJITAndroidARM64ControlsImage(t *testing.T) {
	image := compileMobileProgram(t,
		filepath.Join("android", "android_arm64.rtg"),
		"android/arm64", "librenvo-controls.so",
		filepath.Join("examples", "forms_controls"))

	if !bytes.Contains(image, []byte("Native Android")) {
		t.Fatal("shared controls gallery omitted its Android platform specialization")
	}
	if bytes.Contains(image, []byte("Native iOS")) {
		t.Fatal("shared controls gallery retained its iOS platform subtitle")
	}
	for _, name := range []string{"ANativeWindow_lock", "AMotionEvent_getX"} {
		if !bytes.Contains(image, append([]byte(name), 0)) {
			t.Errorf("Android controls image omits dynamic import %q", name)
		}
	}
}

func TestCompilerJITIOSARM64MachOImage(t *testing.T) {
	image := compileMobileTarget(t,
		filepath.Join("ios", "ios_arm64.rtg"),
		"ios/arm64", "renvo-ios")
	if len(image) < 32 || binary.LittleEndian.Uint32(image[:4]) != 0xfeedfacf {
		t.Fatalf("iOS output is not Mach-O 64: % x", image[:minInt(4, len(image))])
	}
	if cpu := binary.LittleEndian.Uint32(image[4:8]); cpu != 0x0100000c {
		t.Fatalf("iOS Mach-O CPU = %#x, want ARM64", cpu)
	}
	if kind := binary.LittleEndian.Uint32(image[12:16]); kind != 2 {
		t.Fatalf("iOS Mach-O type = %d, want MH_EXECUTE", kind)
	}
	if flags := binary.LittleEndian.Uint32(image[24:28]); flags&0x200000 == 0 {
		t.Fatalf("iOS Mach-O flags = %#x, want MH_PIE", flags)
	}

	commands := int(binary.LittleEndian.Uint32(image[16:20]))
	commandBytes := int(binary.LittleEndian.Uint32(image[20:24]))
	if commands <= 0 || 32+commandBytes > len(image) {
		t.Fatalf("invalid iOS load-command table: count=%d size=%d",
			commands, commandBytes)
	}
	at := 32
	hasIOSVersion := false
	hasMain := false
	hasLibSystem := false
	signatureOffset := 0
	signatureSize := 0
	for i := 0; i < commands; i++ {
		if at+8 > 32+commandBytes {
			t.Fatalf("iOS load command %d has no header", i)
		}
		command := binary.LittleEndian.Uint32(image[at : at+4])
		size := int(binary.LittleEndian.Uint32(image[at+4 : at+8]))
		if size < 8 || at+size > 32+commandBytes {
			t.Fatalf("iOS load command %d has invalid size %d", i, size)
		}
		if command == 0x19 && size >= 72 {
			fileOffset := binary.LittleEndian.Uint64(image[at+40 : at+48])
			fileSize := binary.LittleEndian.Uint64(image[at+48 : at+56])
			maxProtection := binary.LittleEndian.Uint32(image[at+56 : at+60])
			if fileOffset+fileSize > uint64(len(image)) {
				t.Fatalf("iOS segment %d extends past the file", i)
			}
			if maxProtection&6 == 6 {
				t.Fatalf("iOS segment %d is writable and executable", i)
			}
		}
		if command == 0x32 && size >= 24 {
			platform := binary.LittleEndian.Uint32(image[at+8 : at+12])
			minimum := binary.LittleEndian.Uint32(image[at+12 : at+16])
			sdk := binary.LittleEndian.Uint32(image[at+16 : at+20])
			hasIOSVersion = platform == 2 && minimum == 0x000d0000 && sdk == 0x000d0000
		}
		if command == 0x80000028 && size >= 24 {
			hasMain = binary.LittleEndian.Uint64(image[at+8:at+16]) == 0x1000
		}
		if command == 0x0c && size >= 24 {
			nameOffset := int(binary.LittleEndian.Uint32(image[at+8 : at+12]))
			if nameOffset >= 0 && nameOffset < size &&
				mobileCString(image[at:at+size], nameOffset) ==
					"/usr/lib/libSystem.B.dylib" {
				hasLibSystem = true
			}
		}
		if command == 0x1d && size >= 16 {
			signatureOffset = int(binary.LittleEndian.Uint32(image[at+8 : at+12]))
			signatureSize = int(binary.LittleEndian.Uint32(image[at+12 : at+16]))
		}
		at += size
	}
	if at != 32+commandBytes {
		t.Fatalf("iOS load commands end at %d, want %d", at, 32+commandBytes)
	}
	if !hasIOSVersion || !hasMain || !hasLibSystem {
		t.Fatalf("iOS contracts: build-version=%v main=%v libSystem=%v",
			hasIOSVersion, hasMain, hasLibSystem)
	}
	validateIOSAdHocSignature(t, image, signatureOffset, signatureSize)
}

func TestCompilerJITIOSARM64FormsImage(t *testing.T) {
	image := compileMobileProgram(t,
		filepath.Join("ios", "ios_arm64.rtg"),
		"ios/arm64", "RenvoForms",
		filepath.Join("examples", "forms_hello"))

	for _, name := range []string{
		"_UIApplicationMain",
		"_objc_allocateClassPair",
		"_objc_getClass",
		"_objc_msgSend",
		"_class_addMethod",
		"_sel_registerName",
		"_CGColorSpaceCreateDeviceRGB",
		"_CFDataCreate",
		"_CFRelease",
		"_CGDataProviderCreateWithCFData",
		"_CGImageCreate",
	} {
		if !bytes.Contains(image, append([]byte(name), 0)) {
			t.Errorf("iOS Forms image omits dynamic import %q", name)
		}
	}
	if !bytes.Contains(image, []byte("Renvo for iPhone")) ||
		bytes.Contains(image, []byte("Renvo for Android")) {
		t.Fatal("shared Forms hello selected the wrong iOS platform title")
	}
	for _, pseudo := range []string{
		"renvoIOSObjcMsgRect",
		"renvoIOSObjcMsgFloat",
		"renvoIOSDelegateDidFinishCallback",
		"renvoIOSTouchesBeganCallback",
	} {
		if bytes.Contains(image, []byte(pseudo)) {
			t.Errorf("iOS Forms image leaks compiler-only pseudo import %q", pseudo)
		}
	}
	for _, dylib := range []string{
		"/System/Library/Frameworks/UIKit.framework/UIKit",
		"/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
		"/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
		"/usr/lib/libobjc.A.dylib",
	} {
		if !bytes.Contains(image, append([]byte(dylib), 0)) {
			t.Errorf("iOS Forms image omits dylib %q", dylib)
		}
	}
	if bytes.Contains(image, []byte("_CGDataProviderCreateWithData")) {
		t.Fatal("iOS Forms image aliases its mutable framebuffer through CoreGraphics")
	}
	// UIKit returns CGPoint/CGRect components in floating-point registers. The
	// target bridge converts those values to integral Forms coordinates.
	if !bytes.Contains(image[0x1000:], []byte{
		0x20, 0x00, 0x78, 0x9e, // fcvtzs x0, d1
	}) || !bytes.Contains(image[0x1000:], []byte{
		0x40, 0x00, 0x78, 0x9e, // fcvtzs x0, d2
	}) {
		t.Fatal("iOS Forms image omits UIKit geometry result bridges")
	}

	commands := int(binary.LittleEndian.Uint32(image[16:20]))
	at := 32
	signatureOffset, signatureSize := 0, 0
	for i := 0; i < commands; i++ {
		size := int(binary.LittleEndian.Uint32(image[at+4 : at+8]))
		if binary.LittleEndian.Uint32(image[at:at+4]) == 0x1d && size >= 16 {
			signatureOffset = int(binary.LittleEndian.Uint32(image[at+8 : at+12]))
			signatureSize = int(binary.LittleEndian.Uint32(image[at+12 : at+16]))
		}
		at += size
	}
	validateIOSAdHocSignature(t, image, signatureOffset, signatureSize)
}

func TestCompilerJITIOSARM64ControlsImage(t *testing.T) {
	image := compileMobileProgram(t,
		filepath.Join("ios", "ios_arm64.rtg"),
		"ios/arm64", "RenvoControls",
		filepath.Join("examples", "forms_controls"))

	if !bytes.Contains(image, []byte("Native iOS")) {
		t.Fatal("shared controls gallery omitted its iOS platform specialization")
	}
	if bytes.Contains(image, []byte("Native Android")) {
		t.Fatal("shared controls gallery retained its Android platform subtitle")
	}
	for _, name := range []string{
		"_UIApplicationMain", "_CGImageCreate",
		"_mach_absolute_time", "_mach_timebase_info",
	} {
		if !bytes.Contains(image, append([]byte(name), 0)) {
			t.Errorf("iOS controls image omits dynamic import %q", name)
		}
	}
}

func TestRenvoBuiltAPKPackagerMatchesHostBuilder(t *testing.T) {
	sharedObject := compileMobileTarget(t,
		filepath.Join("android", "android_arm64.rtg"),
		"android/arm64", "librenvo.so")
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	packager := driver.CompileFromFS([]string{
		"-t", hostTarget(),
		"-s",
		"-o", "renvoapk",
		filepath.Join(root, "cmd", "renvoapk"),
	}, root, filepath.Join(root, "std"), driver.OSFS{}, backendcompiled.Backend{})
	if !packager.Ok {
		t.Fatalf("Renvo packager compile failed: %#v", packager.Diagnostic)
	}
	directory := t.TempDir()
	executableName := "renvoapk"
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}
	executable := filepath.Join(directory, executableName)
	if err := os.WriteFile(executable, packager.Binary, 0755); err != nil {
		t.Fatal(err)
	}
	sharedObjectPath := filepath.Join(directory, "librenvo.so")
	if err := os.WriteFile(sharedObjectPath, sharedObject, 0644); err != nil {
		t.Fatal(err)
	}
	configSource := []byte("package=dev.renvo.compilerjit\n" +
		"name=CompilerJIT APK\n" +
		"version_code=1\n" +
		"version_name=1.0\n" +
		"min_sdk=24\n" +
		"target_sdk=35\n")
	configPath := filepath.Join(directory, "app.conf")
	if err := os.WriteFile(configPath, configSource, 0644); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(directory, "app.apk")
	command := exec.Command(executable,
		"-so", sharedObjectPath, "-config", configPath, "-o", outputPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Renvo-built packager failed: %v\n%s", err, output)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	config, err := apk.ParseConfig(configSource)
	if err != nil {
		t.Fatal(err)
	}
	want, err := apk.Build(sharedObject, config)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Renvo-built APK differs from host builder: got %d bytes, want %d",
			len(got), len(want))
	}
}

func validateIOSAdHocSignature(t *testing.T, image []byte, offset int, size int) {
	t.Helper()
	if offset <= 0 || size < 20 || offset+size > len(image) {
		t.Fatalf("invalid iOS code-signature range [%d,%d)", offset, offset+size)
	}
	signature := image[offset : offset+size]
	be := binary.BigEndian
	if be.Uint32(signature[:4]) != 0xfade0cc0 ||
		int(be.Uint32(signature[4:8])) != size || be.Uint32(signature[8:12]) != 1 {
		t.Fatal("iOS code signature is not a one-entry embedded signature")
	}
	codeDirectoryOffset := int(be.Uint32(signature[16:20]))
	if codeDirectoryOffset < 20 || codeDirectoryOffset+40 > len(signature) {
		t.Fatal("iOS code-directory offset is invalid")
	}
	directory := signature[codeDirectoryOffset:]
	if be.Uint32(directory[:4]) != 0xfade0c02 {
		t.Fatal("iOS signature does not contain a code directory")
	}
	directorySize := int(be.Uint32(directory[4:8]))
	hashOffset := int(be.Uint32(directory[16:20]))
	codeSlots := int(be.Uint32(directory[28:32]))
	codeLimit := int(be.Uint32(directory[32:36]))
	if directorySize > len(directory) || hashOffset < 40 || codeSlots <= 0 ||
		codeLimit != offset || directory[36] != 32 || directory[37] != 2 ||
		directory[39] != 14 || hashOffset+codeSlots*32 > directorySize {
		t.Fatal("iOS code-directory layout is invalid")
	}
	for slot := 0; slot < codeSlots; slot++ {
		start := slot * 16384
		end := start + 16384
		if end > codeLimit {
			end = codeLimit
		}
		hash := sha256.Sum256(image[start:end])
		stored := directory[hashOffset+slot*32 : hashOffset+(slot+1)*32]
		if !bytes.Equal(stored, hash[:]) {
			t.Fatalf("iOS code-signature hash %d does not cover the final image", slot)
		}
	}
}

func mobileCString(data []byte, offset int) string {
	if offset < 0 || offset >= len(data) {
		return ""
	}
	end := offset
	for end < len(data) && data[end] != 0 {
		end++
	}
	return string(data[offset:end])
}
