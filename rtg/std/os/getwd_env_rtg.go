//go:build rtg && !windows

package os

func Getwd() (string, *osError) {
	for i := 0; i < len(processEnv); i++ {
		item := processEnv[i]
		if len(item) >= 4 && item[0] == 'P' && item[1] == 'W' && item[2] == 'D' && item[3] == '=' {
			return item[4:], nil
		}
	}
	return "", errIO()
}
