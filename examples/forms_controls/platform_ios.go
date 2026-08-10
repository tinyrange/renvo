//go:build ios

package main

import "renvo.dev/forms"

const controlsPlatformSubtitle = "Native iOS • 360 pt canvas • retained Forms"

// renvo:linkstatic /usr/lib/libSystem.B.dylib,_mach_absolute_time
func machAbsoluteTime() int { return 0 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,_mach_timebase_info
func machTimebaseInfo(info *byte) int { return -1 }

var renderTimeNumer int
var renderTimeDenom int

func renderUint32(data []byte, offset int) int {
	return int(data[offset]) |
		int(data[offset+1])<<8 |
		int(data[offset+2])<<16 |
		int(data[offset+3])<<24
}

func renderTimestampMicroseconds() int {
	if renderTimeDenom == 0 {
		info := make([]byte, 8)
		if machTimebaseInfo(&info[0]) != 0 {
			return 0
		}
		renderTimeNumer = renderUint32(info, 0)
		renderTimeDenom = renderUint32(info, 4)
		if renderTimeNumer <= 0 || renderTimeDenom <= 0 {
			return 0
		}
	}
	ticks := machAbsoluteTime()
	quotient := ticks / renderTimeDenom
	remainder := ticks % renderTimeDenom
	nanoseconds := quotient*renderTimeNumer + remainder*renderTimeNumer/renderTimeDenom
	return nanoseconds / 1000
}

func configureRenderTiming(app *forms.App) {
	app.BeforeRender = demo.beforeRender
	app.AfterRender = demo.afterRender
}
