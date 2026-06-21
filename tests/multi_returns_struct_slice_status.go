package main

type rtg1013Packet struct {
	id int
	ok bool
}

func rtg1013Build() (rtg1013Packet, []byte, bool) {
	var data []byte
	data = append(data, 'O')
	data = append(data, 'K')
	return rtg1013Packet{id: 7, ok: true}, data, true
}

func appMain(args []string) int {
	packet, data, ok := rtg1013Build()
	if !ok || !packet.ok || packet.id != 7 || len(data) != 2 || data[1] != 'K' {
		print("RTG-1013 struct slice status failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
