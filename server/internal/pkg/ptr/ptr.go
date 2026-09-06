// Package ptr provides generic utilities for creating pointers to values.
package ptr

// To returns a pointer to v.
// This is a type-safe replacement for ad-hoc &v expressions
// when working with struct literals that require pointer fields.
func To[T any](v T) *T {
	return &v
}
