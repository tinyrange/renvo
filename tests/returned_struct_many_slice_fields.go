package main

type rtg0718Entry struct {
	start int
	end   int
	typ   int
	off   int
}

type rtg0718Meta struct {
	prog  int
	first []rtg0718Entry
	mid   []int
	last  []rtg0718Entry
	more  []rtg0718Entry
	next  []int
	ok    bool
}

func rtg0718Build(seed int) rtg0718Meta {
	var out rtg0718Meta
	out.first = make([]rtg0718Entry, 0, 32)
	out.mid = make([]int, 0, 32)
	out.last = make([]rtg0718Entry, 0, 32)
	out.more = make([]rtg0718Entry, 0, 512)
	out.next = make([]int, 0, 512)
	for i := 0; i < 256; i++ {
		rtg0718AppendFirst(&out, rtg0718Entry{start: seed + 10 + i, end: seed + 14 + i, typ: 1, off: 2})
		out.mid = append(out.mid, seed+99+i)
		rtg0718AppendLast(&out, rtg0718Entry{start: seed + 20 + i, end: seed + 24 + i, typ: 3, off: 4})
		out.more = append(out.more, rtg0718Entry{start: seed + 30 + i, end: seed + 34 + i, typ: 5, off: 6})
		out.next = append(out.next, i-1)
	}
	out.ok = true
	return out
}

func rtg0718AppendFirst(out *rtg0718Meta, entry rtg0718Entry) {
	out.first = append(out.first, entry)
}

func rtg0718AppendLast(out *rtg0718Meta, entry rtg0718Entry) {
	out.last = append(out.last, entry)
}

func appMain(args []string, env []string) int {
	meta := rtg0718Build(0)
	other := rtg0718Build(1000)
	if !meta.ok {
		print("FAIL ok\n")
		return 1
	}
	if len(meta.first) != 256 || len(meta.mid) != 256 || len(meta.last) != 256 || len(meta.more) != 256 || len(meta.next) != 256 {
		print("FAIL len\n")
		return 1
	}
	if meta.first[0].start != 10 || meta.first[0].end != 14 {
		print("FAIL first\n")
		return 1
	}
	if meta.mid[0] != 99 || meta.mid[255] != 354 {
		print("FAIL mid\n")
		return 1
	}
	if meta.last[0].start != 20 || meta.last[0].end != 24 || meta.last[255].start != 275 {
		print("FAIL last\n")
		return 1
	}
	if meta.more[0].start != 30 || meta.more[255].end != 289 {
		print("FAIL more\n")
		return 1
	}
	if meta.next[0] != -1 || meta.next[255] != 254 {
		print("FAIL next\n")
		return 1
	}
	if other.first[0].start != 1010 || other.last[255].start != 1275 {
		print("FAIL other\n")
		return 1
	}
	if meta.first[0].start != 10 || meta.last[255].start != 275 {
		print("FAIL overwritten\n")
		return 1
	}
	print("PASS\n")
	return 0
}
