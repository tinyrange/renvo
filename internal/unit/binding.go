package unit

const (
	TagDefinitionTarget    = 4
	TagDefinitionHash      = 5
	TagDescriptorVersion   = 6
	targetBindingFixedSize = 52
	// Leave enough capacity on a newly marshaled unit for every registered
	// target name, plus room for future names, without copying the whole unit.
	unboundTargetBindingReserve = 80
)

type TargetBinding struct {
	Target            string
	Definition        string
	DescriptorVersion int
}

// BindUnboundTarget appends a binding to a newly marshaled RTGU. MarshalCore
// reserves the small tail capacity, so the ordinary frontend path writes into
// that storage directly instead of rescanning or copying the complete unit.
// The pointer parameter avoids an escaping-slice copy in arena-backed
// self-hosted callers.
func BindUnboundTarget(data *[]byte, binding TargetBinding) bool {
	if !validUnitRoot(*data) || binding.Target == "" || len(binding.Definition) != 32 || binding.DescriptorVersion <= 0 {
		return false
	}
	required := targetBindingFixedSize + len(binding.Target)
	if cap(*data)-len(*data) < required {
		out := *data
		out = appendStringNodeCore(out, TagDefinitionTarget, binding.Target)
		out = appendStringNodeCore(out, TagDefinitionHash, binding.Definition)
		var version [2]byte
		version[0] = byte(binding.DescriptorVersion)
		version[1] = byte(binding.DescriptorVersion >> 8)
		out = appendNode(out, TagDescriptorVersion, version[:])
		patchUint32Core(out, 10, len(out)-14)
		*data = out
		return true
	}
	start := len(*data)
	out := (*data)[:start+required]
	at := start
	at = writeStringNodeCore(out, at, TagDefinitionTarget, binding.Target)
	at = writeStringNodeCore(out, at, TagDefinitionHash, binding.Definition)
	at = writeNodeHeaderCore(out, at, TagDescriptorVersion, 2)
	out[at] = byte(binding.DescriptorVersion)
	out[at+1] = byte(binding.DescriptorVersion >> 8)
	at += 2
	if at != len(out) {
		return false
	}
	patchUint32Core(out, 10, len(out)-14)
	*data = out
	return true
}

func writeStringNodeCore(out []byte, at int, tag int, payload string) int {
	at = writeNodeHeaderCore(out, at, tag, len(payload))
	for i := 0; i < len(payload); i++ {
		out[at+i] = payload[i]
	}
	return at + len(payload)
}

func writeNodeHeaderCore(out []byte, at int, tag int, size int) int {
	out[at] = byte(tag)
	out[at+1] = byte(tag >> 8)
	out[at+2] = byte(size)
	out[at+3] = byte(size >> 8)
	out[at+4] = byte(size >> 16)
	out[at+5] = byte(size >> 24)
	return at + 6
}

// BindTarget sets the target-definition identity on a canonical RTGU root,
// replacing an existing valid binding when a frontend resolves an external
// definition after its bootstrap target. Version-1 readers skip these optional
// child tags.
func BindTarget(data []byte, binding TargetBinding) ([]byte, bool) {
	if !validUnitRoot(data) || binding.Target == "" || len(binding.Definition) != 32 || binding.DescriptorVersion <= 0 {
		return nil, false
	}
	out := make([]byte, 0, len(data)+len(binding.Target)+58)
	out = append(out, data[:14]...)
	hadBinding := false
	for at := 14; at < len(data); {
		if at+6 > len(data) {
			return nil, false
		}
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		next := at + 6 + size
		if size < 0 || next < at || next > len(data) {
			return nil, false
		}
		if tag == TagDefinitionTarget || tag == TagDefinitionHash || tag == TagDescriptorVersion {
			hadBinding = true
		} else {
			out = append(out, data[at:next]...)
		}
		at = next
	}
	if hadBinding {
		if _, ok := ReadTargetBinding(data); !ok {
			return nil, false
		}
	}
	out = appendStringNodeCore(out, TagDefinitionTarget, binding.Target)
	out = appendStringNodeCore(out, TagDefinitionHash, binding.Definition)
	var version [2]byte
	version[0] = byte(binding.DescriptorVersion)
	version[1] = byte(binding.DescriptorVersion >> 8)
	out = appendNode(out, TagDescriptorVersion, version[:])
	patchUint32Core(out, 10, len(out)-14)
	return out, true
}

func ReadTargetBinding(data []byte) (TargetBinding, bool) {
	var binding TargetBinding
	if !validUnitRoot(data) {
		return binding, false
	}
	at := 14
	haveTarget, haveHash, haveVersion := false, false, false
	for at < len(data) {
		if at+6 > len(data) {
			return TargetBinding{}, false
		}
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		at += 6
		if size < 0 || at+size < at || at+size > len(data) {
			return TargetBinding{}, false
		}
		if tag == TagDefinitionTarget {
			if haveTarget || size == 0 {
				return TargetBinding{}, false
			}
			binding.Target = string(data[at : at+size])
			haveTarget = true
		} else if tag == TagDefinitionHash {
			if haveHash || size != 32 {
				return TargetBinding{}, false
			}
			binding.Definition = string(data[at : at+size])
			haveHash = true
		} else if tag == TagDescriptorVersion {
			if haveVersion || size != 2 {
				return TargetBinding{}, false
			}
			binding.DescriptorVersion = int(data[at]) | int(data[at+1])<<8
			haveVersion = true
		}
		at += size
	}
	if !haveTarget && !haveHash && !haveVersion {
		return TargetBinding{}, false
	}
	return binding, haveTarget && haveHash && haveVersion && binding.DescriptorVersion > 0
}

func validUnitRoot(data []byte) bool {
	if len(data) < 14 || string(data[:4]) != Magic {
		return false
	}
	version := int(data[4]) | int(data[5])<<8
	flags := int(data[6]) | int(data[7])<<8
	tag := int(data[8]) | int(data[9])<<8
	length := int(data[10]) | int(data[11])<<8 | int(data[12])<<16 | int(data[13])<<24
	return version == Version && flags == 0 && tag == TagUnit && length == len(data)-14
}
