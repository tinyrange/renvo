//go:build renvo && darwin && arm64

package process

// renvo:linkstatic /System/Library/Frameworks/Foundation.framework/Foundation,NSHomeDirectory
func processFoundationHomeDirectory() int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_getClass
func processObjcGetClass(name string) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,sel_registerName
func processSelector(name string) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func processObjcMsg0(object, selector int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func processObjcMsg1(object, selector, value int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func processObjcMsgBytes(object, selector int, value []byte) int { return 0 }

var launchedTasks []int

func processString(value string) int {
	bytes := append([]byte(value), 0)
	return processObjcMsgBytes(processObjcGetClass("NSString"), processSelector("stringWithUTF8String:"), bytes)
}

func Start(path, directory string) bool {
	// Loading one Foundation entry point makes Objective-C runtime classes such
	// as NSTask and NSString available even when the caller is not a GUI app.
	processFoundationHomeDirectory()
	task := processObjcMsg0(processObjcGetClass("NSTask"), processSelector("alloc"))
	task = processObjcMsg0(task, processSelector("init"))
	if task == 0 {
		return false
	}
	processObjcMsg1(task, processSelector("setLaunchPath:"), processString(path))
	if directory != "" {
		processObjcMsg1(task, processSelector("setCurrentDirectoryPath:"), processString(directory))
	}
	processObjcMsg0(task, processSelector("launch"))
	if processObjcMsg0(task, processSelector("isRunning")) == 0 {
		return false
	}
	launchedTasks = append(launchedTasks, task)
	return true
}
