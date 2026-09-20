package wilayah

import (
	"slices"
	"sort"
	"strings"
	"sync"
)

// One postal code covers many villages, and villages sorted by code are not
// sorted by postal code, so this needs an index of its own. It holds village
// positions rather than copies: 83k int32 costs a third of a megabyte where
// copying the rows would cost several.
//
// It is built on first use, so a program that never asks about postal codes
// never pays for it.
var postal struct {
	once  sync.Once
	order []int32
}

func postalOrder() []int32 {
	postal.once.Do(func() {
		items := villages.all()

		order := make([]int32, len(items))
		for i := range order {
			order[i] = int32(i)
		}

		// Ties keep village code order, so results are stable and read
		// naturally: same postal code, ascending region code.
		slices.SortFunc(order, func(a, b int32) int {
			if c := strings.Compare(items[a].PostalCode, items[b].PostalCode); c != 0 {
				return c
			}
			return strings.Compare(items[a].Code, items[b].Code)
		})

		postal.order = order
	})

	return postal.order
}

// VillagesByPostalCode returns the villages sharing a postal code, ordered by region
// code. One postal code usually covers several villages.
func VillagesByPostalCode(code string) []Village {
	if len(code) != 5 {
		return nil
	}
	return postalMatches(code, true)
}

// VillagesByPostalCodePrefix returns the villages whose postal code starts with
// prefix, ordered by postal code and then by region code. It is the way to
// widen a search: "401" covers everything from 40100 to 40199.
//
// An empty prefix returns nothing; use AllVillages to walk the whole table.
func VillagesByPostalCodePrefix(prefix string) []Village {
	if prefix == "" {
		return nil
	}
	return postalMatches(prefix, false)
}

func postalMatches(prefix string, exact bool) []Village {
	items := villages.all()
	order := postalOrder()

	lo := sort.Search(len(order), func(i int) bool {
		return items[order[i]].PostalCode >= prefix
	})

	// The matches are contiguous, so walking forward from the lower bound
	// costs one step per result rather than a second search.
	var out []Village
	for i := lo; i < len(order); i++ {
		got := items[order[i]].PostalCode
		if exact {
			if got != prefix {
				break
			}
		} else if !strings.HasPrefix(got, prefix) {
			break
		}

		out = append(out, items[order[i]])
	}

	return out
}
