package platform

import "errors"

var ErrOpenUnsupported = errors.New("opening external resources is unsupported on this platform")
