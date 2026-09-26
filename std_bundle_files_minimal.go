//go:build !renvo_bundle

package renvo

import "embed"

// BundledExtrasEnabled reports whether optional application modules and libc
// sources are embedded. The standard library and core runtime are always embedded.
const BundledExtrasEnabled = false

//go:embed std x/runtime
var bundledStdFiles embed.FS
