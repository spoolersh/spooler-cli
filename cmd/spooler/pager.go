package main

import (
	"iter"

	"github.com/spoolersh/spooler-go"
)

// list walks a paged listing as one sequence: fn fetches the page after a
// cursor, and the walk ends at the last page or once limit items were
// yielded, fetching no page it does not need. A limit of 0 lists
// everything. The page size is the server's: the docs bound it and say
// nothing of a request beyond the bound, so the pager never asks for one.
// An error ends the walk after the items before it.
func list[T any](limit int, fn func(after string) (spooler.Page[T], error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var (
			after string
			n     int
		)
		for {
			page, err := fn(after)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
				n++
				if limit > 0 && n == limit {
					return
				}
			}
			if page.Next == "" {
				return
			}
			after = page.Next
		}
	}
}
