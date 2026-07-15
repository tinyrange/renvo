//go:build rtg && windows && amd64

package graphics

const windowsPointerSize = 8

const windowsWindowClassSize = 72
const windowsWindowClassProcOffset = 8
const windowsWindowClassInstanceOffset = 24
const windowsWindowClassCursorOffset = 40
const windowsWindowClassNameOffset = 64

const windowsMessageStructSize = 48
const windowsMessageWindowOffset = 0
const windowsMessageKindOffset = 8
const windowsMessageWParamOffset = 16
const windowsMessageLParamOffset = 24
const windowsMessageTimeOffset = 32
const windowsMessagePointXOffset = 36
const windowsMessagePointYOffset = 40
const windowsMessagePrivateOffset = 44

const windowsTrackMouseEventSize = 24
const windowsTrackMouseEventHoverTimeOffset = 16
