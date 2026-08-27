package uefi_test

import (
	"fmt"

	"renvo.dev/device/uefi"
)

// UEFI reports errors by setting the high bit of a machine-word status value.
func ExampleStatus() {
	fmt.Println(uefi.Success.Failed(), uefi.NotFound.Failed(), uefi.NotFound.Error())
	// Output: false true not found
}

func ExampleWriteString() {
	if status := uefi.WriteString("Hello from Renvo!\r\n"); status.Failed() {
		return
	}
}

func ExampleFirmwareVendor() {
	vendor := uefi.FirmwareVendor()
	_ = vendor
}

func ExampleReadKey() {
	key, status := uefi.ReadKey()
	if status == uefi.Success && key.UnicodeChar != 0 {
		_ = uefi.WriteUTF16([]uint16{key.UnicodeChar, '\r', '\n', 0})
	}
}

func ExampleStall() {
	_ = uefi.WriteString("Waiting...\r\n")
	uefi.Stall(500_000) // Half a second.
	_ = uefi.WriteString("Done\r\n")
}

func ExampleOpenVolume() {
	volume, status := uefi.OpenVolume()
	if status.Failed() {
		return
	}
	defer volume.Close()
}

func ExampleFile_Open() {
	volume, status := uefi.OpenVolume()
	if status.Failed() {
		return
	}
	defer volume.Close()

	file, status := volume.Open("EFI\\BOOT\\BOOTX64.EFI", uefi.FileModeRead, 0)
	if status.Failed() {
		return
	}
	defer file.Close()
}

func ExampleFile_Read() {
	volume, status := uefi.OpenVolume()
	if status.Failed() {
		return
	}
	defer volume.Close()

	file, status := volume.Open("CONFIG.TXT", uefi.FileModeRead, 0)
	if status.Failed() {
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	count, status := file.Read(buffer)
	if !status.Failed() {
		_ = buffer[:count]
	}
}

func ExampleFile_Write() {
	volume, status := uefi.OpenVolume()
	if status.Failed() {
		return
	}
	defer volume.Close()

	file, status := volume.Open("RENVO.LOG", uefi.FileModeWrite|uefi.FileModeCreate, 0)
	if status.Failed() {
		return
	}
	defer file.Close()
	_, _ = file.Write([]byte("boot started\r\n"))
}

func ExampleFile_SetPosition() {
	volume, status := uefi.OpenVolume()
	if status.Failed() {
		return
	}
	defer volume.Close()

	file, status := volume.Open("KERNEL.BIN", uefi.FileModeRead, 0)
	if status.Failed() {
		return
	}
	defer file.Close()
	_ = file.SetPosition(4096)
}

func ExampleGraphicsOutput() {
	graphics, status := uefi.GraphicsOutput()
	if status.Failed() {
		return
	}
	_ = graphics.SetDisplayMode(0)
}

func ExampleGraphicsOutputProtocol_Framebuffer() {
	graphics, status := uefi.GraphicsOutput()
	if status.Failed() {
		return
	}
	framebuffer, status := graphics.Framebuffer()
	if !status.Failed() {
		_, _ = framebuffer.Width, framebuffer.Height
	}
}

func ExampleFramebuffer_Fill() {
	graphics, status := uefi.GraphicsOutput()
	if status.Failed() {
		return
	}
	framebuffer, status := graphics.Framebuffer()
	if !status.Failed() {
		framebuffer.Fill(24, 32, 48)
	}
}

func ExampleFramebuffer_Set() {
	graphics, status := uefi.GraphicsOutput()
	if status.Failed() {
		return
	}
	framebuffer, status := graphics.Framebuffer()
	if !status.Failed() {
		framebuffer.Set(20, 20, 255, 128, 32)
	}
}

func ExampleCurrentSystemTable() {
	table := uefi.CurrentSystemTable()
	if !table.Valid() {
		return
	}
	_ = table.ConsoleOutput()
}

func ExampleSystemTable_ConfigurationTable() {
	table := uefi.CurrentSystemTable()
	address := table.ConfigurationTable(uefi.GraphicsOutputProtocolGUID)
	_ = address
}

func ExampleAllocatePool() {
	var buffer uintptr
	status := uefi.AllocatePool(uefi.LoaderData, 4096, &buffer)
	if status.Failed() {
		return
	}
	defer uefi.FreePool(buffer)
}
