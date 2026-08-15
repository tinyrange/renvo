package rtg

// Generated seed for the bounded Linux-module ELF constructor; placeholders are
// specialized from the selected format and typed helper bindings.
const declarativeLinuxModuleSource = `func LINUXMODULEGet32(data []byte, at int) int {
if at < 0 || at+4 > len(data) {
return 0
}
value := int(data[at])
value |= int(data[at+1]) << 8
value |= int(data[at+2]) << 16
value |= int(data[at+3]) << 24
return value
}

func LINUXMODULEBTFString(data []byte, base int, offset int, value string) bool {
position := base + offset
if offset < 0 || position < 0 || position+len(value) >= len(data) {
return false
}
for i := 0; i < len(value); i++ {
if data[position+i] != value[i] {
return false
}
}
return data[position+len(value)] == 0
}

type LINUXMODULEModuleLayout struct {
size		int
nameOffset	int
initOffset	int
exitOffset	int
ok		bool
}

func LINUXMODULEBTFModuleLayout(data []byte) LINUXMODULEModuleLayout {
var result LINUXMODULEModuleLayout
if len(data) < 24 || data[0] != 0x9f || data[1] != 0xeb {
return result
}
headerLength := LINUXMODULEGet32(data, 4)
typeStart := headerLength + LINUXMODULEGet32(data, 8)
typeEnd := typeStart + LINUXMODULEGet32(data, 12)
stringStart := headerLength + LINUXMODULEGet32(data, 16)
if headerLength < 24 || typeStart < headerLength ||
typeEnd > len(data) || stringStart < headerLength ||
stringStart >= len(data) {
return result
}
position := typeStart
for position+12 <= typeEnd {
name := LINUXMODULEGet32(data, position)
info := LINUXMODULEGet32(data, position+4)
size := LINUXMODULEGet32(data, position+8)
kind := (info >> 24) & 31
count := info & 65535
extra := 0
if kind == 1 {
extra = 4
} else if kind == 3 {
extra = 12
} else if kind == 4 || kind == 5 {
extra = count * 12
} else if kind == 6 || kind == 13 {
extra = count * 8
} else if kind == 14 || kind == 17 {
extra = 4
} else if kind == 15 || kind == 19 {
extra = count * 12
}
next := position + 12 + extra
if next > typeEnd || next <= position {
return result
}
if kind == 4 &&
LINUXMODULEBTFString(data, stringStart, name, "module") {
nameOffset := -1
initOffset := -1
exitOffset := -1
member := position + 12
for i := 0; i < count; i++ {
memberName := LINUXMODULEGet32(data, member)
bitOffset := LINUXMODULEGet32(data, member+8) & 0x00ffffff
if bitOffset%8 == 0 {
if LINUXMODULEBTFString(
data, stringStart, memberName, "name",
) {
nameOffset = bitOffset / 8
} else if LINUXMODULEBTFString(
data, stringStart, memberName, "init",
) {
initOffset = bitOffset / 8
} else if LINUXMODULEBTFString(
data, stringStart, memberName, "exit",
) {
exitOffset = bitOffset / 8
}
}
member += 12
}
result.size = size
result.nameOffset = nameOffset
result.initOffset = initOffset
result.exitOffset = exitOffset
result.ok = nameOffset >= 0 && initOffset >= 0
return result
}
position = next
}
return result
}

func LINUXMODULEHexDigit(ch byte) int {
if ch >= '0' && ch <= '9' {
return int(ch - '0')
}
if ch >= 'a' && ch <= 'f' {
return int(ch-'a') + 10
}
if ch >= 'A' && ch <= 'F' {
return int(ch-'A') + 10
}
return -1
}

func LINUXMODULESymbolCRC(data []byte, symbol string) (int, bool) {
line := 0
for line < len(data) {
end := line
for end < len(data) && data[end] != '\n' {
end++
}
firstTab := line
for firstTab < end && data[firstTab] != '\t' {
firstTab++
}
secondTab := firstTab + 1
for secondTab < end && data[secondTab] != '\t' {
secondTab++
}
if secondTab-firstTab-1 == len(symbol) {
match := true
for i := 0; i < len(symbol); i++ {
if data[firstTab+1+i] != symbol[i] {
match = false
}
}
if match {
value := 0
start := line
if start+2 <= firstTab &&
data[start] == '0' && data[start+1] == 'x' {
start += 2
}
for i := start; i < firstTab; i++ {
digit := LINUXMODULEHexDigit(data[i])
if digit < 0 {
return 0, false
}
value = value<<4 | digit
}
return value, true
}
}
line = end + 1
}
return 0, false
}

func LINUXMODULESymbolGPLOnly(data []byte, symbol string) bool {
line := 0
for line < len(data) {
end := line
for end < len(data) && data[end] != '\n' {
end++
}
firstTab := line
for firstTab < end && data[firstTab] != '\t' {
firstTab++
}
secondTab := firstTab + 1
for secondTab < end && data[secondTab] != '\t' {
secondTab++
}
if secondTab-firstTab-1 == len(symbol) {
match := true
for i := 0; i < len(symbol); i++ {
if data[firstTab+1+i] != symbol[i] {
match = false
}
}
if match {
thirdTab := secondTab + 1
for thirdTab < end && data[thirdTab] != '\t' {
thirdTab++
}
exportStart := thirdTab + 1
exportEnd := exportStart
for exportEnd < end && data[exportEnd] != '\t' {
exportEnd++
}
if exportEnd-exportStart < 4 {
return false
}
if data[exportEnd-4] != '_' || data[exportEnd-3] != 'G' {
return false
}
return data[exportEnd-2] == 'P' && data[exportEnd-1] == 'L'
}
}
line = end + 1
}
return false
}

func LINUXMODULELicenseGPLCompatible(license string) bool {
if license == "GPL" || license == "GPL v2" {
return true
}
if license == "GPL and additional rights" ||
license == "Dual BSD/GPL" {
return true
}
return license == "Dual MIT/GPL" || license == "Dual MPL/GPL"
}

func LINUXMODULEContains(text string, needle string) bool {
if len(needle) == 0 || len(needle) > len(text) {
return false
}
for i := 0; i+len(needle) <= len(text); i++ {
match := true
for j := 0; j < len(needle); j++ {
if text[i+j] != needle[j] {
match = false
}
}
if match {
return true
}
}
return false
}

func LINUXMODULEVermagic(
release string, version string, exitOffset int, symvers []byte,
) string {
out := release
if LINUXMODULEContains(version, " SMP ") ||
LINUXMODULEContains(version, " SMP") {
out += " SMP"
}
if LINUXMODULEContains(version, "PREEMPT") {
out += " preempt"
}
if exitOffset >= 0 {
out += " mod_unload"
}
crc, ok := LINUXMODULESymbolCRC(symvers, "module_layout")
if ok && crc != 0 {
out += " modversions"
}
return out + " "
}

func LINUXMODULEAppendString(out []byte, value string) []byte {
for i := 0; i < len(value); i++ {
out = append(out, value[i])
}
return append(out, 0)
}

func LINUXMODULEUntil(out []byte, size int) []byte {
for len(out) < size {
out = append(out, 0)
}
return out
}

func LINUXMODULEAppendVersion(out []byte, symbol string, crc int) []byte {
start := len(out)
out = MODULEAPPEND64(out, crc)
for i := 0; i < len(symbol) && i < 55; i++ {
out = append(out, symbol[i])
}
out = append(out, 0)
return LINUXMODULEUntil(out, start+64)
}

func LINUXMODULEModuleImage(
emitter *MODULEEMITTER, initLabel int, exitLabel int,
) []byte {
initPosition := emitter.LabelPosition(initLabel)
exitPosition := emitter.LabelPosition(exitLabel)
layout := LINUXMODULEBTFModuleLayout(emitter.KernelBTF())
if initPosition < 0 || !layout.ok || layout.size <= 0 {
return nil
}
moduleName := emitter.KernelModuleName()
license := emitter.KernelLicense()
symvers := emitter.KernelSymvers()
if moduleName == "" || license == "" || len(symvers) == 0 {
return nil
}

strings := []byte("\x00init_module\x00cleanup_module\x00__this_module\x00")

var symbols []byte
symbols = MODULESYMBOL(symbols, 0, 0, 0, 0, 0)
symbols = MODULESYMBOL(symbols, 0, 3, 1, 0, 0)
symbols = MODULESYMBOL(symbols, 0, 3, 3, 0, 0)
symbols = MODULESYMBOL(symbols, 0, 3, 4, 0, 0)
symbols = MODULESYMBOL(symbols, 0, 3, 7, 0, 0)
initSymbol := 5
symbols = MODULESYMBOL(
symbols, 1, 18, 1, initPosition, 0,
)
exitSymbol := 0
if exitPosition >= 0 {
exitSymbol = len(symbols) / 24
symbols = MODULESYMBOL(
symbols, 13, 18, 1, exitPosition, 0,
)
}
symbols = MODULESYMBOL(
symbols, 28, 17, 7, 0, layout.size,
)
firstImportSymbol := len(symbols)/24
nameOffset := len(strings)
for i := 0; i < emitter.ExternalImportCount(); i++ {
symbols = MODULESYMBOL(
symbols, nameOffset, 16, 0, 0, 0,
)
strings = LINUXMODULEAppendString(strings, emitter.ExternalImportName(i))
nameOffset = len(strings)
}

var textRelocations []byte
for i := 0; i < emitter.AbsoluteRelocationCount(); i++ {
at := emitter.AbsoluteRelocationOffset(i)
addend := emitter.AbsoluteRelocationAddend(i)
kind := emitter.AbsoluteRelocationKind(i)
if kind == RTGRelocationAbsoluteBSS {
textRelocations = MODULERELOCATION(
textRelocations, at, 3, 2, addend-4,
)
} else if kind == RTGRelocationAbsoluteBSSEnd {
alignment := addend
if alignment <= 0 {
alignment = 1
}
textRelocations = MODULERELOCATION(
textRelocations, at, 3, 2,
MODULEALIGN(emitter.BSSSize(), alignment)-4,
)
} else if kind == RTGRelocationImport {
if addend < 0 || addend >= emitter.ExternalImportCount() {
return nil
}
textRelocations = MODULERELOCATION(
textRelocations, at, firstImportSymbol+addend, 4, -4,
)
} else {
textRelocations = MODULERELOCATION(
textRelocations, at, 2, 2, addend-4,
)
}
}

thisModule := make([]byte, layout.size)
if layout.nameOffset+len(moduleName) >= len(thisModule) {
return nil
}
for i := 0; i < len(moduleName); i++ {
thisModule[layout.nameOffset+i] = moduleName[i]
}
var moduleRelocations []byte
moduleRelocations = MODULERELOCATION(
moduleRelocations, layout.initOffset, initSymbol, 1, 0,
)
if exitSymbol != 0 && layout.exitOffset >= 0 {
moduleRelocations = MODULERELOCATION(
moduleRelocations, layout.exitOffset, exitSymbol, 1, 0,
)
}

var versions []byte
moduleCRC, moduleOK := LINUXMODULESymbolCRC(
symvers, "module_layout",
)
if moduleOK && moduleCRC != 0 {
versions = LINUXMODULEAppendVersion(
versions, "module_layout", moduleCRC,
)
}
for i := 0; i < emitter.ExternalImportCount(); i++ {
name := emitter.ExternalImportName(i)
if LINUXMODULESymbolGPLOnly(symvers, name) &&
!LINUXMODULELicenseGPLCompatible(license) {
return nil
}
crc, ok := LINUXMODULESymbolCRC(symvers, name)
if !ok {
return nil
}
if crc != 0 {
versions = LINUXMODULEAppendVersion(versions, name, crc)
}
}

var moduleInfo []byte
moduleInfo = LINUXMODULEAppendString(
moduleInfo, "license="+license,
)
moduleInfo = LINUXMODULEAppendString(moduleInfo, "depends=")
moduleInfo = LINUXMODULEAppendString(
moduleInfo, "name="+moduleName,
)
moduleInfo = LINUXMODULEAppendString(
moduleInfo,
"vermagic="+LINUXMODULEVermagic(
emitter.KernelRelease(), emitter.KernelVersion(),
layout.exitOffset, symvers,
),
)

sectionNames := []byte("\x00.text\x00.rela.text\x00.rodata\x00.bss\x00.modinfo\x00__versions\x00.gnu.linkonce.this_module\x00.rela.gnu.linkonce.this_module\x00.symtab\x00.strtab\x00.shstrtab\x00")

code := emitter.Code()
data := emitter.Data()
var image []byte
image = LINUXMODULEUntil(image, 64)
textOffset := MODULEALIGN(len(image), 16)
image = LINUXMODULEUntil(image, textOffset)
image = append(image, code...)
textRelocationOffset := MODULEALIGN(len(image), 8)
image = LINUXMODULEUntil(image, textRelocationOffset)
image = append(image, textRelocations...)
dataOffset := MODULEALIGN(len(image), 8)
image = LINUXMODULEUntil(image, dataOffset)
image = append(image, data...)
bssOffset := len(image)
infoOffset := len(image)
image = append(image, moduleInfo...)
versionOffset := MODULEALIGN(len(image), 8)
image = LINUXMODULEUntil(image, versionOffset)
image = append(image, versions...)
moduleOffset := MODULEALIGN(len(image), 64)
image = LINUXMODULEUntil(image, moduleOffset)
image = append(image, thisModule...)
moduleRelocationOffset := MODULEALIGN(len(image), 8)
image = LINUXMODULEUntil(image, moduleRelocationOffset)
image = append(image, moduleRelocations...)
symbolOffset := MODULEALIGN(len(image), 8)
image = LINUXMODULEUntil(image, symbolOffset)
image = append(image, symbols...)
stringOffset := len(image)
image = append(image, strings...)
sectionStringOffset := len(image)
image = append(image, sectionNames...)
sectionOffset := MODULEALIGN(len(image), 8)
image = LINUXMODULEUntil(image, sectionOffset)
image = MODULESECTION(image, 0, 0, 0, 0, 0, 0, 0, 0, 0)
image = MODULESECTION(
image, 1, 1, 6, textOffset, len(code), 0, 0, 16, 0,
)
image = MODULESECTION(
image, 7, 4, 64, textRelocationOffset,
len(textRelocations), 9, 1, 8, 24,
)
image = MODULESECTION(
image, 18, 1, 2, dataOffset, len(data), 0, 0, 8, 0,
)
image = MODULESECTION(
image, 26, 8, 3, bssOffset, emitter.BSSSize(),
0, 0, 8, 0,
)
image = MODULESECTION(
image, 31, 1, 2, infoOffset, len(moduleInfo),
0, 0, 1, 0,
)
image = MODULESECTION(
image, 40, 1, 2, versionOffset, len(versions),
0, 0, 8, 0,
)
image = MODULESECTION(
image, 51, 1, 3, moduleOffset,
len(thisModule), 0, 0, 64, 0,
)
image = MODULESECTION(
image, 77, 4, 64, moduleRelocationOffset,
len(moduleRelocations), 9, 7, 8, 24,
)
image = MODULESECTION(
image, 108, 2, 0, symbolOffset, len(symbols),
10, 5, 8, 24,
)
image = MODULESECTION(
image, 116, 3, 0, stringOffset, len(strings),
0, 0, 1, 0,
)
image = MODULESECTION(
image, 124, 3, 0, sectionStringOffset,
len(sectionNames), 0, 0, 1, 0,
)
var header []byte
header = MODULEHEADER(header, sectionOffset, 12, 11)
for i := 0; i < len(header); i++ {
image[i] = header[i]
}
return image
}

`
