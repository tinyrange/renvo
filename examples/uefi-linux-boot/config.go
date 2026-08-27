package main

type bootConfig struct {
	Kernel    string
	Initramfs string
	Command   string
}

func parseConfig(data []byte) (bootConfig, string) {
	var config bootConfig
	line := 1
	for start := 0; start <= len(data); line++ {
		end := start
		for end < len(data) && data[end] != '\n' {
			end++
		}
		lineData := trimSpace(data[start:end])
		if len(lineData) != 0 && lineData[0] != '#' {
			equals := -1
			for i := 0; i < len(lineData); i++ {
				if lineData[i] == '=' {
					equals = i
					break
				}
			}
			if equals < 1 {
				return config, "config.txt line " + decimal(line) + ": expected name=value"
			}
			key := string(trimSpace(lineData[:equals]))
			value := string(trimSpace(lineData[equals+1:]))
			switch key {
			case "kernel":
				config.Kernel = value
			case "initramfs":
				config.Initramfs = value
			case "cmdline":
				config.Command = value
			default:
				return config, "config.txt line " + decimal(line) + ": unknown setting " + key
			}
		}
		if end == len(data) {
			break
		}
		start = end + 1
	}
	if config.Kernel == "" {
		return config, "config.txt: kernel is required"
	}
	if config.Initramfs == "" {
		return config, "config.txt: initramfs is required"
	}
	return config, ""
}

func trimSpace(value []byte) []byte {
	start := 0
	for start < len(value) && (value[start] == ' ' || value[start] == '\t' || value[start] == '\r') {
		start++
	}
	end := len(value)
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	at := len(digits)
	for value != 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}
