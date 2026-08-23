package main

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend,float64=12
func metadataAggregateArguments(receiver, selector, width, height int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend,result-float64=2
func metadataAggregateResult(receiver, selector int) float64 { return 0 }

// renvo:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glPixelZoom,float32=3
func metadataSingleArguments(x, y int) {}

func appMain() int {
	metadataAggregateArguments(1, 2, 3, 4)
	_ = metadataAggregateResult(1, 2)
	metadataSingleArguments(1, 2)
	print("PASS\n")
	return 0
}
