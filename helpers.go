package event

import (
	"context"
	"fmt"
)

func merge(parent, child context.Context) context.Context {
	if child == nil {
		return parent
	}

	return child
}

type errWrapper struct {
	err        string
	underlying error
}

func (e *errWrapper) Error() string {
	return e.err
}

func (e *errWrapper) Unwrap() error {
	return e.underlying
}

func recoverErr(err error, r interface{}) error {
	switch v := r.(type) {
	case nil:
		if err != nil {
			return err
		}

		return nil
	case string:
		return &errWrapper{
			err:        v,
			underlying: err,
		}
	case error:
		return &errWrapper{
			err:        v.Error(),
			underlying: err,
		}
	default:
		// Fallback err (per specs, error strings
		// should be lowercase w/o punctuation
		return &errWrapper{
			err:        fmt.Sprintf("panic: %v", r),
			underlying: err,
		}
	}
}
