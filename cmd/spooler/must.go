package main

// must returns v, or panics on err: for package-level values whose
// construction can only fail by a programming error.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
