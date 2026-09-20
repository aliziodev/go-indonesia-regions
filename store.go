package wilayah

import (
	"sort"
	"sync"

	"github.com/aliziodev/go-indonesia-regions/internal/codec"
	"github.com/aliziodev/go-indonesia-regions/internal/dataset"
)

// store decodes one embedded table on first use and keeps the result for the
// lifetime of the process.
//
// Laziness is what keeps the cost honest: a program that only ever touches
// provinces never pays to decompress the 83k-row village table.
type store[T any] struct {
	blob  []byte
	level codec.Level
	conv  func(codec.Record) T
	code  func(T) string

	once  sync.Once
	items []T
}

// all returns every row of the table, in ascending code order. The returned
// slice is shared and must not be modified; callers that hand data out clone
// the range they need.
func (s *store[T]) all() []T {
	s.once.Do(s.decode)
	return s.items
}

func (s *store[T]) decode() {
	table, err := codec.Decode(s.blob)
	if err != nil {
		// Unreachable in a correctly built binary: the blobs ship inside the
		// executable and are verified at generation time. Failing loudly beats
		// forcing an error return onto every lookup in the package.
		panic("wilayah: embedded " + s.level.String() + " table is unusable: " + err.Error())
	}

	items := make([]T, table.Len())
	for i := range items {
		items[i] = s.conv(table.At(i))
	}

	s.items = items
}

// find resolves an exact code through binary search.
func (s *store[T]) find(code string) (T, bool) {
	items := s.all()

	i := sort.Search(len(items), func(i int) bool { return s.code(items[i]) >= code })
	if i < len(items) && s.code(items[i]) == code {
		return items[i], true
	}

	var zero T
	return zero, false
}

// descendants returns the rows of this table that fall under ancestor, which
// may sit any number of levels above them. An empty ancestor yields the whole
// table.
//
// Codes are sorted and every descendant starts with the ancestor code followed
// by ".", so they occupy one contiguous range. The upper bound uses "/"
// because it is the byte right after "." in ASCII, which makes it the first
// key past the last descendant. That separator is also what keeps a partial
// code from matching: "32.7" bounds the empty range between "32.7." and
// "32.7/", and every real code under province 32 sorts above both.
func (s *store[T]) descendants(ancestor string) []T {
	items := s.all()
	if ancestor == "" {
		return items
	}

	lo := s.lowerBound(items, ancestor+".")
	hi := s.lowerBound(items, ancestor+"/")

	return items[lo:hi]
}

func (s *store[T]) lowerBound(items []T, key string) int {
	return sort.Search(len(items), func(i int) bool { return s.code(items[i]) >= key })
}

var (
	provinces = &store[Province]{
		blob:  dataset.Provinces,
		level: codec.LevelProvince,
		conv: func(r codec.Record) Province {
			return Province{Code: r.Code, Name: r.Name}
		},
		code: func(p Province) string { return p.Code },
	}

	regencies = &store[Regency]{
		blob:  dataset.Regencies,
		level: codec.LevelRegency,
		conv: func(r codec.Record) Regency {
			return Regency{Code: r.Code, Name: r.Name, Kind: RegencyKind(r.Type)}
		},
		code: func(r Regency) string { return r.Code },
	}

	districts = &store[District]{
		blob:  dataset.Districts,
		level: codec.LevelDistrict,
		conv: func(r codec.Record) District {
			return District{Code: r.Code, Name: r.Name}
		},
		code: func(d District) string { return d.Code },
	}

	villages = &store[Village]{
		blob:  dataset.Villages,
		level: codec.LevelVillage,
		conv: func(r codec.Record) Village {
			return Village{
				Code:       r.Code,
				Name:       r.Name,
				Kind:       VillageKind(r.Type),
				PostalCode: r.PostalCode,
			}
		},
		code: func(v Village) string { return v.Code },
	}
)
