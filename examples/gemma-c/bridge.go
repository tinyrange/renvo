package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"
)

var modelData []byte
var promptData []byte
var arguments []string

func llm_args(argc int32, argv uintptr) {
	// A C entry point receives argc/argv directly; it does not initialize os.Args.
	arguments = make([]string, int(argc))
	for i := 0; i < int(argc); i++ {
		p := *(*uintptr)(unsafe.Pointer(argv + uintptr(i)*unsafe.Sizeof(argv)))
		b := make([]byte, 0)
		for j := uintptr(0); ; j++ {
			c := *(*byte)(unsafe.Pointer(p + j))
			if c == 0 {
				break
			}
			b = append(b, c)
		}
		arguments[i] = string(b)
	}
}

func llm_load() uintptr {
	if len(arguments) < 4 {
		fmt.Fprintln(os.Stderr, "usage: gemma model.bin prompt.ids steps [logits.bin]")
		return 0
	}
	f, err := os.Open(arguments[1])
	if err != nil {
		return 0
	}
	defer f.Close()
	prefix := make([]byte, 12)
	for at := 0; at < len(prefix); {
		n, err := f.Read(prefix[at:])
		at += n
		if err != nil || n == 0 {
			return 0
		}
	}
	size := int(uint32(prefix[8]) | uint32(prefix[9])<<8 | uint32(prefix[10])<<16 | uint32(prefix[11])<<24)
	if size < 2048 || size > 700*1024*1024 {
		return 0
	}
	modelData = make([]byte, size)
	copy(modelData, prefix)
	for at := 12; at < len(modelData); {
		n, err := f.Read(modelData[at:])
		at += n
		if err != nil || n == 0 {
			return 0
		}
	}
	if !textMode {
		promptData, err = os.ReadFile(arguments[2])
		if err != nil || len(promptData)%4 != 0 {
			return 0
		}
	}
	return uintptr(unsafe.Pointer(&modelData[0]))
}
func llm_size() uint32         { return uint32(len(modelData)) }
func llm_prompt_length() int32 { return int32(len(promptData) / 4) }
func llm_prompt_token(i int32) int32 {
	at := int(i) * 4
	return int32(uint32(promptData[at]) | uint32(promptData[at+1])<<8 | uint32(promptData[at+2])<<16 | uint32(promptData[at+3])<<24)
}
func llm_steps() int32 {
	n, err := strconv.Atoi(arguments[3])
	if err != nil || n < 0 || n > 1024 {
		return -1
	}
	return int32(n)
}
func llm_emit(token int32) {
	if textMode {
		tokEmit(int(token))
	} else {
		fmt.Println(token)
	}
}
func llm_finish() {
	if textMode {
		tokFlushBytes()
		fmt.Println()
	}
}

func llm_command() int32 {
	if len(arguments) < 2 {
		return -1
	}
	mode := arguments[1]
	if mode == "--encode" || mode == "--tokenize" || mode == "--decode" {
		if len(arguments) != 4 {
			tokFail("usage: gemma --encode|--tokenize|--decode tokenizer.bin text-or-ids-file")
		}
		tokLoad(arguments[2])
		if mode == "--decode" {
			data, err := os.ReadFile(arguments[3])
			if err != nil || len(data)%4 != 0 {
				tokFail("Invalid token ID file")
			}
			for i := 0; i < len(data); i += 4 {
				id := int(uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16 | uint32(data[i+3])<<24)
				tokEmit(id)
			}
			tokFlushBytes()
		} else {
			var ids []int
			if mode == "--tokenize" {
				ids = tokChat(arguments[3])
			} else {
				ids = tokEncode(arguments[3])
			}
			for i := 0; i < len(ids); i++ {
				fmt.Println(ids[i])
			}
		}
		return 0
	}
	if mode == "--chat" {
		if len(arguments) != 6 {
			tokFail("usage: gemma --chat model.bin tokenizer.bin prompt steps")
		}
		tokLoad(arguments[3])
		ids := tokChat(arguments[4])
		if len(ids) > 1024 {
			tokFail("Prompt exceeds 1024 tokens")
		}
		budget, err := strconv.Atoi(arguments[5])
		if err != nil || budget < 0 || budget > 1024-len(ids) {
			tokFail("Prompt plus generation budget must fit 1024 tokens")
		}
		promptData = make([]byte, len(ids)*4)
		for i := 0; i < len(ids); i++ {
			if ids[i] >= 262144 {
				tokFail("Prompt contains a token outside the text model vocabulary")
			}
			id := uint32(ids[i])
			promptData[i*4] = byte(id)
			promptData[i*4+1] = byte(id >> 8)
			promptData[i*4+2] = byte(id >> 16)
			promptData[i*4+3] = byte(id >> 24)
		}
		arguments = []string{arguments[0], arguments[2], "", arguments[5]}
		textMode = true
	}
	return -1
}
func llm_logits(address uintptr, count int32) {
	if len(arguments) < 5 {
		return
	}
	data := make([]byte, int(count)*4)
	for i := 0; i < len(data); i++ {
		data[i] = *(*byte)(unsafe.Pointer(address + uintptr(i)))
	}
	if err := os.WriteFile(arguments[4], data, 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
