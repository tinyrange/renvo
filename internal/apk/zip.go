package apk

type zipEntry struct {
	name string
	data []byte
}

type zipCentralEntry struct {
	entry  zipEntry
	crc    uint32
	offset int
}

func buildZIPSections(entries []zipEntry) ([]byte, []byte, []byte, bool) {
	local := make([]byte, 0, 65536)
	records := make([]zipCentralEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry.name) == 0 || len(entry.name) > 65535 ||
			len(entry.data) > 0x7fffffff || len(local) > 0x7fffffff {
			return nil, nil, nil, false
		}
		crc := zipCRC32(entry.data)
		records = append(records, zipCentralEntry{
			entry: entry, crc: crc, offset: len(local),
		})
		local = append32(local, 0x04034b50)
		local = append16(local, 20)
		local = append16(local, 0)
		local = append16(local, 0)
		local = append16(local, 0)
		local = append16(local, 0x21)
		local = append32(local, int(crc))
		local = append32(local, len(entry.data))
		local = append32(local, len(entry.data))
		local = append16(local, len(entry.name))
		local = append16(local, 0)
		local = append(local, []byte(entry.name)...)
		local = append(local, entry.data...)
	}
	central := make([]byte, 0, len(entries)*80)
	for i := 0; i < len(records); i++ {
		record := records[i]
		central = append32(central, 0x02014b50)
		central = append16(central, 20)
		central = append16(central, 20)
		central = append16(central, 0)
		central = append16(central, 0)
		central = append16(central, 0)
		central = append16(central, 0x21)
		central = append32(central, int(record.crc))
		central = append32(central, len(record.entry.data))
		central = append32(central, len(record.entry.data))
		central = append16(central, len(record.entry.name))
		central = append16(central, 0)
		central = append16(central, 0)
		central = append16(central, 0)
		central = append16(central, 0)
		central = append32(central, 0)
		central = append32(central, record.offset)
		central = append(central, []byte(record.entry.name)...)
	}
	if len(central) > 0x7fffffff || len(entries) > 65535 {
		return nil, nil, nil, false
	}
	end := make([]byte, 0, 22)
	end = append32(end, 0x06054b50)
	end = append16(end, 0)
	end = append16(end, 0)
	end = append16(end, len(entries))
	end = append16(end, len(entries))
	end = append32(end, len(central))
	end = append32(end, len(local))
	end = append16(end, 0)
	return local, central, end, true
}

func zipCRC32(data []byte) uint32 {
	crc := uint32(0xffffffff)
	for i := 0; i < len(data); i++ {
		crc ^= uint32(data[i])
		for bit := 0; bit < 8; bit++ {
			mask := uint32(0) - (crc & 1)
			crc = crc>>1 ^ (0xedb88320 & mask)
		}
	}
	return crc ^ 0xffffffff
}
