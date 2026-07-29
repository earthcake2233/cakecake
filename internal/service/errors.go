package service

import "fmt"

// SvcError wraps a business error code.
type SvcError struct {
	Code int
	Msg  string
}

func (e *SvcError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

func (e *SvcError) Is(target error) bool {
	if t, ok := target.(*SvcError); ok {
		return e.Code == t.Code
	}
	return false
}

// Predefined service errors.
var (
	ErrNotFound         = &SvcError{Code: 40400, Msg: "resource not found"}
	ErrForbidden        = &SvcError{Code: 40300, Msg: "forbidden"}
	ErrParamError       = &SvcError{Code: 40001, Msg: "parameter error"}
	ErrCommentSensitive = &SvcError{Code: 40021, Msg: "content contains blocked words"}
	ErrCommentsClosed   = &SvcError{Code: 40303, Msg: "comments closed"}
	ErrInternalError    = &SvcError{Code: 50000, Msg: "internal error"}
	ErrUnauthorized     = &SvcError{Code: 40100, Msg: "unauthorized"}
)