package wilayah_test

import (
	"strings"
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestSearchFindsAKnownRegion(t *testing.T) {
	results := wilayah.Search("bandung")
	if len(results) == 0 {
		t.Fatal("Search(bandung) returned nothing")
	}

	found := false
	for _, r := range results {
		if r.Code == "32.73" {
			found = true
		}
		if !strings.Contains(strings.ToLower(r.Name), "bandung") {
			t.Errorf("%s (%s) does not contain the query", r.Code, r.Name)
		}
	}

	if !found {
		t.Error("Search(bandung) did not return Kota Bandung")
	}
}

// A query for a regency should surface the regency rather than bury it under
// the districts and villages that happen to share its name. Matching ignores
// the "Kabupaten"/"Kota" label, so "bandung" is an exact hit on Kota Bandung,
// and a level tiebreak puts the broader region first.
func TestSearchRanksTheRegencyFirst(t *testing.T) {
	results := wilayah.Search("bandung")
	if len(results) == 0 {
		t.Fatal("Search(bandung) returned nothing")
	}

	position := map[string]int{}
	firstNarrow := -1

	for i, r := range results {
		position[r.Code] = i
		if firstNarrow < 0 && r.Level > wilayah.LevelRegency {
			firstNarrow = i
		}
	}

	for _, code := range []string{"32.73", "32.04"} { // Kota and Kabupaten Bandung
		at, ok := position[code]
		if !ok {
			t.Fatalf("%s is missing from the results", code)
		}
		if firstNarrow >= 0 && at > firstNarrow {
			t.Errorf("%s ranks at %d, below the first district or village at %d", code, at, firstNarrow)
		}
	}
}

func TestSearchRanksExactThenPrefixThenTheRest(t *testing.T) {
	results := wilayah.Search("jawa bar", wilayah.Limit(5))
	if len(results) == 0 {
		t.Fatal("Search(jawa bar) returned nothing")
	}

	if got := results[0]; got.Code != "32" {
		t.Errorf("first result is %s (%s), want Jawa Barat", got.Code, got.Name)
	}
}

func TestSearchIsCaseInsensitive(t *testing.T) {
	lower := wilayah.Search("bandung", wilayah.Limit(5))
	upper := wilayah.Search("BANDUNG", wilayah.Limit(5))
	mixed := wilayah.Search("BaNdUnG", wilayah.Limit(5))

	if len(lower) == 0 {
		t.Fatal("Search returned nothing")
	}
	for i := range lower {
		if upper[i] != lower[i] || mixed[i] != lower[i] {
			t.Fatalf("result %d differs between cases: %v / %v / %v", i, lower[i], upper[i], mixed[i])
		}
	}
}

func TestSearchAtLevel(t *testing.T) {
	results := wilayah.Search("bandung", wilayah.AtLevel(wilayah.LevelDistrict))
	if len(results) == 0 {
		t.Fatal("no districts matched")
	}

	for _, r := range results {
		if r.Level != wilayah.LevelDistrict {
			t.Errorf("got a %s, want districts only", r.Level)
		}
	}

	both := wilayah.Search("bandung", wilayah.AtLevel(wilayah.LevelRegency, wilayah.LevelProvince))
	for _, r := range both {
		if r.Level != wilayah.LevelRegency && r.Level != wilayah.LevelProvince {
			t.Errorf("got a %s, want regencies and provinces only", r.Level)
		}
	}
}

func TestSearchLimit(t *testing.T) {
	all := wilayah.Search("sari")
	if len(all) <= 3 {
		t.Fatalf("the query matched %d regions, too few to test the limit", len(all))
	}

	limited := wilayah.Search("sari", wilayah.Limit(3))
	if len(limited) != 3 {
		t.Fatalf("Limit(3) returned %d rows", len(limited))
	}
	for i := range limited {
		if limited[i] != all[i] {
			t.Errorf("row %d = %v, want %v; the limit must cut the ranking, not reorder it", i, limited[i], all[i])
		}
	}

	if got := wilayah.Search("sari", wilayah.Limit(0)); len(got) != 0 {
		t.Errorf("Limit(0) returned %d rows, want 0", len(got))
	}
}

// Region names get pasted in whatever form they were stored in, often shouted
// and carrying a label. Those queries have to land on the right region.
func TestSearchUnderstandsLabelsAndShouting(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{"KAB. ACEH SELATAN", "11.01"},
		{"Kabupaten Aceh Selatan", "11.01"},
		{"kab aceh selatan", "11.01"},
		{"Kota Bandung", "32.73"},
		{"KOTA BANDUNG", "32.73"},
		{"Kab. Bandung", "32.04"},
		{"Ds. Pasteur", "32.73.07.1001"},
		{"PROVINSI JAWA BARAT", "32"},
		{"prov. jawa barat", "32"},
		{"  Kab.   Aceh   Selatan  ", "11.01"},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			results := wilayah.Search(tc.query, wilayah.Limit(1))
			if len(results) == 0 {
				t.Fatalf("Search(%q) returned nothing", tc.query)
			}
			if results[0].Code != tc.want {
				t.Errorf("Search(%q) ranked %s (%s) first, want %s", tc.query, results[0].Code, results[0].Name, tc.want)
			}
		})
	}
}

// A label pins the level even when several regions share the name. There are
// two districts called Sukajadi, so the test is that districts win, not that
// one particular district does.
func TestSearchLabelPinsTheLevel(t *testing.T) {
	for _, query := range []string{"kec. sukajadi", "Kecamatan Sukajadi", "KEC. SUKAJADI"} {
		t.Run(query, func(t *testing.T) {
			results := wilayah.Search(query, wilayah.Limit(5))
			if len(results) == 0 {
				t.Fatalf("Search(%q) returned nothing", query)
			}

			if got := results[0]; got.Level != wilayah.LevelDistrict || got.Name != "Sukajadi" {
				t.Errorf("first result is %s %s (%s), want a district named Sukajadi", got.Level, got.Code, got.Name)
			}

			found := false
			for _, r := range results {
				if r.Code == "32.73.07" {
					found = true
				}
			}
			if !found {
				t.Errorf("Search(%q) did not surface 32.73.07 in the top %d", query, len(results))
			}
		})
	}
}

// A label only counts as a whole word, or Kota Kotamobagu becomes unfindable.
func TestSearchDoesNotMistakeANameForALabel(t *testing.T) {
	results := wilayah.Search("Kotamobagu", wilayah.Limit(1))
	if len(results) == 0 {
		t.Fatal("Search(Kotamobagu) returned nothing")
	}
	if results[0].Code != "71.74" {
		t.Errorf("ranked %s (%s) first, want Kota Kotamobagu", results[0].Code, results[0].Name)
	}

	// A bare label with nothing after it is a query in its own right.
	if got := wilayah.Search("kota", wilayah.AtLevel(wilayah.LevelRegency), wilayah.Limit(1)); len(got) == 0 {
		t.Error("Search(kota) returned nothing")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	for _, q := range []string{"", "   "} {
		if got := wilayah.Search(q); len(got) != 0 {
			t.Errorf("Search(%q) returned %d rows, want 0", q, len(got))
		}
	}
}

func TestSearchNoMatch(t *testing.T) {
	if got := wilayah.Search("zzzznotaplace"); len(got) != 0 {
		t.Errorf("returned %d rows, want 0", len(got))
	}
}

func TestRegencyShortName(t *testing.T) {
	cases := map[string]string{
		"32.73": "Bandung",      // Kota Bandung
		"11.01": "Aceh Selatan", // Kabupaten Aceh Selatan
	}

	for code, want := range cases {
		r, ok := wilayah.RegencyByCode(code)
		if !ok {
			t.Fatalf("regency %s not found", code)
		}
		if got := r.ShortName(); got != want {
			t.Errorf("%q.ShortName() = %q, want %q", r.Name, got, want)
		}
	}

	// No regency name should be left starting with its own label.
	for _, r := range wilayah.Regencies("") {
		short := r.ShortName()
		if strings.HasPrefix(short, "Kabupaten ") || strings.HasPrefix(short, "Kota ") {
			t.Errorf("%q.ShortName() = %q, which still carries the label", r.Name, short)
		}
		if short == "" {
			t.Errorf("%q.ShortName() is empty", r.Name)
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	wilayah.Preload()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if len(wilayah.Search("bandung", wilayah.Limit(10))) == 0 {
			b.Fatal("no results")
		}
	}
}

func BenchmarkSearchRegenciesOnly(b *testing.B) {
	wilayah.Preload()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if len(wilayah.Search("bandung", wilayah.AtLevel(wilayah.LevelRegency))) == 0 {
			b.Fatal("no results")
		}
	}
}
