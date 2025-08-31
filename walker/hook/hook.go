package hook

import "strings"

func match[T comparable](b T) func(a T) bool {
	return func(a T) bool {
		return a == b
	}
}

func containsStr(b string) func(a string) bool {
	return func(a string) bool {
		return strings.Contains(a, b)
	}
}
