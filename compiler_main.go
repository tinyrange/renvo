package main

func rtgEnv() []string {
	var empty []string
	return empty
}

func rtgOpenInputFallback(path string) int {
	if len(path) == 0 || path[0] == '/' {
		return -1
	}
	env := rtgEnv()
	for e := 0; e < len(env); e++ {
		pwd := env[e]
		if len(pwd) > 4 {
			if pwd[0] == 'P' && pwd[1] == 'W' && pwd[2] == 'D' && pwd[3] == '=' {
				var full []byte
				for i := 4; i < len(pwd); i++ {
					full = append(full, pwd[i])
				}
				full = append(full, '/')
				for i := 0; i < len(path); i++ {
					full = append(full, path[i])
				}
				full = append(full, 0)
				return open(string(full), O_RDONLY)
			}
		}
	}
	return -1
}

func appMain(args []string) int {
	var input []int
	if len(args) < 4 {
		return 1
	}
	var output int = open(args[2], O_RDWR|O_CREATE|O_TRUNC)
	if output < 0 {
		return 1
	}
	for i := 3; i < len(args); i++ {
		fd := open(args[i], O_RDONLY)
		if fd < 0 {
			fd = rtgOpenInputFallback(args[i])
		}
		if fd < 0 {
			return 1
		}
		input = append(input, fd)
	}
	if compileLinuxAmd64(input, output) != 0 {
		return 1
	}

	return 0
}
