package hook

func match[T comparable](b T) func(a T) bool {
	return func(a T) bool {
		return a == b
	}
}
