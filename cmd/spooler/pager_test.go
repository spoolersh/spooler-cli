package main

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

// pages serves items two per page, failing once the cursor reaches failAt
// when that is positive, and records the cursors asked for.
func pages(items []int, failAt int, log *[]string) func(after string) (spooler.Page[int], error) {
	return func(after string) (spooler.Page[int], error) {
		*log = append(*log, after)
		start := 0
		if after != "" {
			start = int(after[0]-'0') * 2
		}
		if failAt > 0 && start >= failAt {
			return spooler.Page[int]{}, errors.New("boom")
		}
		end := min(start+2, len(items))
		page := spooler.Page[int]{
			Items: items[start:end],
		}
		if end < len(items) {
			page.Next = string(rune('0' + (end / 2)))
		}
		return page, nil
	}
}

func TestList(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	for _, test := range []struct {
		name      string
		limit     int
		failAt    int
		exp       []int
		expErr    bool
		expAfters []string
	}{
		{
			name:      "every page",
			exp:       []int{1, 2, 3, 4, 5},
			expAfters: []string{"", "1", "2"},
		},
		{
			name:      "limit inside a page",
			limit:     3,
			exp:       []int{1, 2, 3},
			expAfters: []string{"", "1"},
		},
		{
			name:      "limit at a page boundary fetches no more",
			limit:     2,
			exp:       []int{1, 2},
			expAfters: []string{""},
		},
		{
			name:      "limit beyond the end",
			limit:     9,
			exp:       []int{1, 2, 3, 4, 5},
			expAfters: []string{"", "1", "2"},
		},
		{
			name:      "error after the first page",
			failAt:    2,
			exp:       []int{1, 2},
			expErr:    true,
			expAfters: []string{"", "1"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				log    []string
				act    []int
				actErr error
			)
			for item, err := range list(test.limit, pages(items, test.failAt, &log)) {
				if err != nil {
					actErr = err
					break
				}
				act = append(act, item)
			}
			if diff := cmp.Diff(test.exp, act); diff != "" {
				t.Errorf("items:\n%s", diff)
			}
			if (actErr != nil) != test.expErr {
				t.Errorf("error: %v; want %t", actErr, test.expErr)
			}
			if diff := cmp.Diff(test.expAfters, log); diff != "" {
				t.Errorf("cursors:\n%s", diff)
			}
		})
	}
}
