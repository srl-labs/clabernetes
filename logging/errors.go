package logging

import "errors"

// ErrLoggingInstance is the error returned when encountering issues registering, removing,
// fetching, or updating logging instances from the logging Manager.
var ErrLoggingInstance = errors.New("errLoggingInstance")
