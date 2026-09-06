//go:build !renvo_bundle

package renvo

import "embed"

// BundledExtrasEnabled reports whether the optional module and libc sources are embedded.
// The standard library is embedded in every build.
const BundledExtrasEnabled = false

//go:embed std
var bundledStdFiles embed.FS
