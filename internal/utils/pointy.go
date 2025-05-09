package utils

func Pointy[T any](v T) *T {
	return &v
}
