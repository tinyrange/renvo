//go:build renvo_bundle

package renvo

import "embed"

// BundledExtrasEnabled reports whether the optional module and libc sources are embedded.
// The standard library is embedded in every build.
const BundledExtrasEnabled = true

//go:embed std forms device x libc
var bundledStdFiles embed.FS
