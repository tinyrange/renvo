//go:build renvo_bundle

package renvo

import "embed"

// BundledExtrasEnabled reports whether optional application modules and libc
// sources are embedded. The standard library and core runtime are always embedded.
const BundledExtrasEnabled = true

//go:embed std forms device x libc
var bundledStdFiles embed.FS
