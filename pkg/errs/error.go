package errs

import "errors"

var (
	ErrorInvalid = errors.New("invalid")
	ErrorUnknown = errors.New("unknown")
	ErrorDenied  = errors.New("denied")
)
