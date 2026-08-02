//go:build !tiny

package forms

import _ "embed"

//go:embed iconset.rvi
var embeddedIconSet string

//go:embed assets/control-icons.rim
var embeddedControlIconMasks []byte
