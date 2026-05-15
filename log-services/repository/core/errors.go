package core

import "errors"

var ErrBadArguments = errors.New("arguments are not acceptable")
var ErrNotFound = errors.New("resource is not found")
var ErrUnavailable = errors.New("dependency unavailable")
var ErrInvalidStatus = errors.New("invalid log status")
