package main

// renvo:linkstatic example.dll,DefinitionStaticImport
func definitionStaticImport(value int) {
}

func appMain(args []string) int {
	definitionStaticImport(7)
	buffer := []byte("definition runtime")
	fd := open("definition-runtime.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd >= 0 {
		write(fd, buffer, -1)
		write(fd, buffer, 0)
		read(fd, buffer, -1)
		read(fd, buffer, 0)
		chmod(fd, 493)
		close(fd)
	}
	print("PASS\n")
	return 0
}
