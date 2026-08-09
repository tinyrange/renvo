package main

import (
	"fmt"
	"os"

	"renvo.dev/internal/apk"
)

func run(args []string) int {
	sharedObjectPath := ""
	configPath := ""
	outputPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-so":
			i++
			if i < len(args) {
				sharedObjectPath = args[i]
			}
		case "-config":
			i++
			if i < len(args) {
				configPath = args[i]
			}
		case "-o":
			i++
			if i < len(args) {
				outputPath = args[i]
			}
		case "-h", "--help":
			usage()
			return 0
		default:
			fmt.Println("renvoapk: unexpected argument: " + args[i])
			usage()
			return 2
		}
	}
	if sharedObjectPath == "" || configPath == "" || outputPath == "" {
		usage()
		return 2
	}
	sharedObject, err := os.ReadFile(sharedObjectPath)
	if err != nil {
		fmt.Println("renvoapk: could not read shared object: " + err.Error())
		return 1
	}
	configSource, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println("renvoapk: could not read config: " + err.Error())
		return 1
	}
	config, err := apk.ParseConfig(configSource)
	if err != nil {
		fmt.Println("renvoapk: invalid config: " + err.Error())
		return 1
	}
	result, err := apk.Build(sharedObject, config)
	if err != nil {
		fmt.Println("renvoapk: packaging failed: " + err.Error())
		return 1
	}
	if err := os.WriteFile(outputPath, result, 0644); err != nil {
		fmt.Println("renvoapk: could not write APK: " + err.Error())
		return 1
	}
	fmt.Println("renvoapk: wrote " + outputPath)
	return 0
}

func usage() {
	fmt.Println("usage: renvoapk -so librenvo.so -config app.conf -o app.apk")
}
