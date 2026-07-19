//go:build renvo && darwin && arm64

package native

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_getClass
func objcGetClass(name string) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,sel_registerName
func selRegisterName(name string) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_lookUpClass
func objcLookUpClass(name string) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_getMetaClass
func objcGetMetaClass(name string) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,class_getName
func classGetName(class int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,object_getClass
func objectGetClass(object int) int { return 0 }

func Lookup() bool {
	class := objcGetClass("NSObject")
	return class != 0 &&
		selRegisterName("description") != 0 &&
		objcLookUpClass("NSObject") != 0 &&
		objcGetMetaClass("NSObject") != 0 &&
		classGetName(class) != 0 &&
		objectGetClass(class) != 0
}
