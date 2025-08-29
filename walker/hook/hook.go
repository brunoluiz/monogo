package hook

import "errors"

var ErrEarlyExit = errors.New("stop walker early exit condition reached")

func match[T comparable](b T) func(a T) bool {
	return func(a T) bool {
		return a == b
	}
}
