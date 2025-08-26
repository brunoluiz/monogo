package hook

import "errors"

var ErrStopCondition = errors.New("stop walker condtion reached")

func match[T comparable](b T) func(a T) bool {
	return func(a T) bool {
		return a == b
	}
}
