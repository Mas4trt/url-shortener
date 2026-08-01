package service

import "errors"

// ErrInvalidConfig is returned by New when the supplied Config can't
// produce a working Service. Kept as a sentinel (rather than a bare
// fmt.Errorf string) so callers — and tests — can assert on it with
// errors.Is instead of matching on message text.
var ErrInvalidConfig = errors.New("service: invalid config")
