package main

func appMain(args []string) int {
	fd := open("rtg_1002_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-1002 create failed\n")
		return 1
	}
	close(fd)
	var path []byte
	path = append(path, 'r')
	path = append(path, 't')
	path = append(path, 'g')
	path = append(path, '_')
	path = append(path, '1')
	path = append(path, '0')
	path = append(path, '0')
	path = append(path, '2')
	path = append(path, '_')
	path = append(path, 'o')
	path = append(path, 'p')
	path = append(path, 'e')
	path = append(path, 'n')
	path = append(path, '.')
	path = append(path, 't')
	path = append(path, 'm')
	path = append(path, 'p')
	fd = open(string(path), O_RDONLY)
	if fd < 0 {
		print("RTG-1002 open failed\n")
		return 2
	}
	close(fd)
	print("PASS\n")
	return 0
}
