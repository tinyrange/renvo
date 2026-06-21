package main

func appMain(args []string) int {
	var input []int
	var output int = -1

	if len(args) < 2 {
		print("usage: rtgx6 [options] <input files>\n")
		print("options:\n")
		print("  -o <file>  specify output file\n")
		return 1
	}

	for i := 0; i < len(args); i++ {
		if args[i][0] == '-' {
			if args[i][1] == 'o' {
				if i+1 >= len(args) {
					return 1
				}
				output = open(args[i+1], O_RDWR|O_CREATE|O_TRUNC)
				if output < 0 {
					return 1
				}
				i++
			} else {
				print("unknown option\n")
				return 1
			}

			continue
		}

		fd := open(args[i], O_RDONLY)
		if fd < 0 {
			return 1
		}
		input = append(input, fd)
	}

	if output < 0 {
		print("output file not specified\n")
		return 1
	}

	err := compileLinuxAmd64(input, output)
	if err != 0 {
		print("compilation failed\n")
		return 1
	}

	return 0
}
