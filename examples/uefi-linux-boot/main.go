package main

import (
	"fmt"

	"renvo.dev/device/uefi"
)

func main() {
	var configPath [260]uint16
	encodePath(configPath[:], "CONFIG.TXT")
	volume, status := uefi.OpenVolume()
	if status.Failed() {
		fmt.Println("Cannot open boot volume: " + status.Error())
		return
	}
	configFile, status := volume.OpenUTF16(configPath[:], uefi.FileModeRead, 0)
	if status.Failed() {
		volume.Close()
		fmt.Println("Cannot open \\config.txt: " + status.Error())
		return
	}
	configData := make([]byte, 0, 1024)
	buffer := make([]byte, 512)
	for {
		count, readStatus := configFile.Read(buffer)
		if readStatus.Failed() {
			configFile.Close()
			volume.Close()
			fmt.Println("Cannot read \\config.txt: " + readStatus.Error())
			return
		}
		if count == 0 {
			break
		}
		configData = append(configData, buffer[:count]...)
		if len(configData) > 16384 {
			configFile.Close()
			volume.Close()
			fmt.Println("config.txt is larger than 16 KiB")
			return
		}
	}
	configFile.Close()
	volume.Close()
	config, problem := parseConfig(configData)
	if problem != "" {
		volume.Close()
		fmt.Println(problem)
		return
	}
	volume, status = uefi.OpenVolume()
	if status.Failed() {
		fmt.Println("Cannot reopen boot volume: " + status.Error())
		return
	}
	if problem := BootLinux64(volume, config.Kernel, config.Initramfs, config.Command); problem != nil {
		fmt.Println("Boot failed: " + problem.Error())
	}
}
