package targetinfo

// descriptorReader reads trusted records emitted by gentargets, never external
// input. Length-prefixed strings allow arbitrary bytes, including NUL. Numbers
// fit a nonnegative int on every supported compiler host, including 32-bit hosts.
type descriptorReader struct {
	data string
	pos  int
}

func (r *descriptorReader) number() int {
	value := 0
	for shift := 0; shift < 32; shift += 8 {
		value |= int(r.data[r.pos]) << shift
		r.pos++
	}
	return value
}

func (r *descriptorReader) text() string {
	n := r.number()
	value := r.data[r.pos : r.pos+n]
	r.pos += n
	return value
}

func (r *descriptorReader) list() []string {
	n := r.number()
	if n == 0 {
		return nil
	}
	values := make([]string, n-1)
	for i := 0; i < len(values); i++ {
		values[i] = r.text()
	}
	return values
}

// Return fresh slices so callers cannot mutate the registry through a result.
func decodeDescriptor(data string) Descriptor {
	r := descriptorReader{data: data}
	var d Descriptor
	d.Name = r.text()
	d.Backend = r.text()
	d.Aliases = r.list()
	d.Family = r.text()
	d.OS = r.text()
	d.ISA = r.text()
	d.WordBits = r.number()
	d.PointerBits = r.number()
	d.CodePointerBits = r.number()
	d.FunctionPointerBits = r.number()
	d.MaxAlign = r.number()
	d.Endian = r.text()
	d.ABI = r.text()
	d.Image = r.text()
	d.Runtime = r.list()
	d.Tags = r.list()
	d.Capabilities = r.list()
	for i := 0; i < len(d.Definition); i++ {
		d.Definition[i] = data[r.pos]
		r.pos++
	}
	d.DescriptorVersion = r.number()
	flags := r.number()
	d.Advertised = flags&1 != 0
	d.Virtual = flags&2 != 0
	d.IDE = flags&4 != 0
	d.DefaultArena = r.number()
	d.ReleaseArtifact = r.text()
	return d
}

// Membership checks do not allocate a descriptor or temporary slice.
func descriptorSetContains(data string, value string) bool {
	r := descriptorReader{data: data}
	for r.pos < len(data) {
		if r.text() == value {
			return true
		}
	}
	return false
}
