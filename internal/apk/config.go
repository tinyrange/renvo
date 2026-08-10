// Package apk builds installable Android NativeActivity packages without the
// Android SDK or NDK. Its implementation stays within Renvo's standard-library
// surface so cmd/renvoapk can itself be compiled by Renvo.
package apk

import "errors"

type Config struct {
	Package     string
	Name        string
	VersionCode int
	VersionName string
	MinSDK      int
	TargetSDK   int
	Orientation string
}

func DefaultConfig() Config {
	return Config{
		Name:        "Renvo App",
		VersionCode: 1,
		VersionName: "1.0",
		MinSDK:      24,
		TargetSDK:   35,
	}
}

func ParseConfig(source []byte) (Config, error) {
	config := DefaultConfig()
	lineStart := 0
	for lineStart <= len(source) {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		line := source[lineStart:lineEnd]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		line = trimASCII(line)
		if len(line) != 0 && line[0] != '#' {
			equal := byteIndex(line, '=')
			if equal <= 0 {
				return Config{}, errors.New("config line must be key=value")
			}
			key := string(trimASCII(line[:equal]))
			valueBytes := trimASCII(line[equal+1:])
			value := string(valueBytes)
			switch key {
			case "package":
				config.Package = value
			case "name":
				config.Name = value
			case "version_code":
				parsed, ok := parsePositiveDecimal(valueBytes)
				if !ok {
					return Config{}, errors.New("version_code must be a positive integer")
				}
				config.VersionCode = parsed
			case "version_name":
				config.VersionName = value
			case "min_sdk":
				parsed, ok := parsePositiveDecimal(valueBytes)
				if !ok {
					return Config{}, errors.New("min_sdk must be a positive integer")
				}
				config.MinSDK = parsed
			case "target_sdk":
				parsed, ok := parsePositiveDecimal(valueBytes)
				if !ok {
					return Config{}, errors.New("target_sdk must be a positive integer")
				}
				config.TargetSDK = parsed
			case "orientation":
				config.Orientation = value
			default:
				return Config{}, errors.New("unknown config key")
			}
		}
		if lineEnd == len(source) {
			break
		}
		lineStart = lineEnd + 1
	}
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func validateConfig(config Config) error {
	if !validPackageName(config.Package) {
		return errors.New("package must contain at least two dot-separated identifiers")
	}
	if !validPlainText(config.Name, 1, 80) {
		return errors.New("name must be 1-80 printable ASCII characters")
	}
	if !validPlainText(config.VersionName, 1, 80) {
		return errors.New("version_name must be 1-80 printable ASCII characters")
	}
	if config.VersionCode <= 0 {
		return errors.New("version_code must be positive")
	}
	if config.MinSDK < 24 {
		return errors.New("min_sdk must be at least 24 for APK Signature Scheme v2")
	}
	if config.TargetSDK < config.MinSDK || config.TargetSDK > 10000 {
		return errors.New("target_sdk must be between min_sdk and 10000")
	}
	if config.Orientation != "" && config.Orientation != "portrait" && config.Orientation != "landscape" {
		return errors.New("orientation must be portrait or landscape")
	}
	return nil
}

func validPackageName(value string) bool {
	if len(value) < 3 || value[0] == '.' || value[len(value)-1] == '.' {
		return false
	}
	segments := 1
	segmentStart := true
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '.' {
			if segmentStart {
				return false
			}
			segments++
			segmentStart = true
			continue
		}
		if segmentStart {
			if ch != '_' && (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
				return false
			}
			segmentStart = false
		} else if ch != '_' && (ch < 'A' || ch > 'Z') &&
			(ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return segments >= 2 && !segmentStart
}

func validPlainText(value string, minimum int, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func trimASCII(value []byte) []byte {
	start := 0
	for start < len(value) && (value[start] == ' ' || value[start] == '\t') {
		start++
	}
	end := len(value)
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t') {
		end--
	}
	return value[start:end]
}

func byteIndex(value []byte, target byte) int {
	for i := 0; i < len(value); i++ {
		if value[i] == target {
			return i
		}
	}
	return -1
}

func parsePositiveDecimal(value []byte) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		result = result*10 + int(value[i]-'0')
		if result > 0x7fffffff {
			return 0, false
		}
	}
	return result, result > 0
}
