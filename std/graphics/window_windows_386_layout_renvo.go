//go:build renvo && windows && 386

package graphics

const windowsPointerSize = 4

const windowsWindowClassSize = 40
const windowsWindowClassProcOffset = 4
const windowsWindowClassInstanceOffset = 16
const windowsWindowClassCursorOffset = 24
const windowsWindowClassNameOffset = 36

const windowsMessageStructSize = 32
const windowsMessageWindowOffset = 0
const windowsMessageKindOffset = 4
const windowsMessageWParamOffset = 8
const windowsMessageLParamOffset = 12
const windowsMessageTimeOffset = 16
const windowsMessagePointXOffset = 20
const windowsMessagePointYOffset = 24
const windowsMessagePrivateOffset = 28

const windowsTrackMouseEventSize = 16
const windowsTrackMouseEventHoverTimeOffset = 12
