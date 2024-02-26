package errs

import "errors"

var (
	ErrorInvalid = errors.New("invalid")
	ErrorDirty   = errors.New("dirty")
)
