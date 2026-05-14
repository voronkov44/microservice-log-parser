package core

import "errors"

var ErrBadArguments = errors.New("arguments are not acceptable")
var ErrNotFound = errors.New("resource is not found")
var ErrUnsupportedFormat = errors.New("unsupported log format")
var ErrParse = errors.New("parse error")
