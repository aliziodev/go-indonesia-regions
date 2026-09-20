package wilayah_test

import (
	"strings"
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

// The dataset is refreshed from upstream, so these tests assert invariants and
// a few anchors that do not move (province codes and names), rather than
// pinning names and counts that a sync is allowed to change.

func TestCountsMatchDatasetInfo(t *testing.T) {
	counts := wilayah.Dataset().Counts

	cases := []struct {
		level string
		got   int
		want  int
	}{
		{"provinces", len(wilayah.Provinces()), counts.Provinces},
		{"regencies", len(wilayah.Regencies("")), counts.Regencies},
		{"districts", len(wilayah.Districts("")), counts.Districts},
		{"villages", len(wilayah.Villages("")), counts.Villages},
	}

	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: loaded %d, dataset says %d", tc.level, tc.got, tc.want)
		}
		if tc.want == 0 {
			t.Errorf("%s: dataset reports no rows", tc.level)
		}
	}
}

func TestDatasetInfoIsPopulated(t *testing.T) {
	info := wilayah.Dataset()

	if info.Version == "" || info.DataDate == "" {
		t.Errorf("Dataset() = %+v, want version and date", info)
	}
	if len(info.DataHash) != 64 {
		t.Errorf("DataHash = %q, want a 64 character sha256", info.DataHash)
	}
}

func TestCodesAreSortedAndUnique(t *testing.T) {
	t.Run("provinces", func(t *testing.T) {
		assertSorted(t, codesOf(wilayah.Provinces(), func(p wilayah.Province) string { return p.Code }))
	})
	t.Run("regencies", func(t *testing.T) {
		assertSorted(t, codesOf(wilayah.Regencies(""), func(r wilayah.Regency) string { return r.Code }))
	})
	t.Run("districts", func(t *testing.T) {
		assertSorted(t, codesOf(wilayah.Districts(""), func(d wilayah.District) string { return d.Code }))
	})
	t.Run("villages", func(t *testing.T) {
		assertSorted(t, codesOf(wilayah.Villages(""), func(v wilayah.Village) string { return v.Code }))
	})
}

// Every region below the top level must resolve to an existing parent. This is
// the invariant that caught 20 districts and 417 villages going missing when
// the upstream normalizer choked on apostrophes in names.
func TestEveryRegionHasAParent(t *testing.T) {
	for _, r := range wilayah.Regencies("") {
		if _, ok := wilayah.ProvinceByCode(r.ProvinceCode()); !ok {
			t.Fatalf("regency %s (%s) has no province %s", r.Code, r.Name, r.ProvinceCode())
		}
	}
	for _, d := range wilayah.Districts("") {
		if _, ok := wilayah.RegencyByCode(d.RegencyCode()); !ok {
			t.Fatalf("district %s (%s) has no regency %s", d.Code, d.Name, d.RegencyCode())
		}
	}
	for v := range wilayah.AllVillages() {
		if _, ok := wilayah.DistrictByCode(v.DistrictCode()); !ok {
			t.Fatalf("village %s (%s) has no district %s", v.Code, v.Name, v.DistrictCode())
		}
	}
}

// Names carrying an apostrophe (Ma'u, Sangalla', Hu'u) are escaped as a doubled apostrophe in the
// upstream SQL dump and used to be dropped silently. If they vanish again,
// these counts fall to zero.
func TestNamesWithApostrophesSurvive(t *testing.T) {
	districts := 0
	for d := range wilayah.AllDistricts() {
		if strings.Contains(d.Name, "'") {
			districts++
		}
	}
	villages := 0
	for v := range wilayah.AllVillages() {
		if strings.Contains(v.Name, "'") {
			villages++
		}
	}

	if districts == 0 || villages == 0 {
		t.Errorf("districts with apostrophes = %d, villages = %d; want both above zero", districts, villages)
	}
}

func TestEveryVillageHasAPostalCode(t *testing.T) {
	missing := 0
	for v := range wilayah.AllVillages() {
		if len(v.PostalCode) != 5 {
			missing++
		}
	}

	// Upstream currently covers every village. A regression in the join would
	// show up as a large number here, so report it rather than assert zero.
	if missing > 0 {
		t.Errorf("%d villages have no usable postal code", missing)
	}
}

func TestLookupByCode(t *testing.T) {
	p, ok := wilayah.ProvinceByCode("32")
	if !ok || p.Name != "Jawa Barat" {
		t.Errorf("ProvinceByCode(32) = %+v, %v; want Jawa Barat", p, ok)
	}

	r, ok := wilayah.RegencyByCode("32.73")
	if !ok || r.Name != "Kota Bandung" {
		t.Errorf("RegencyByCode(32.73) = %+v, %v; want Kota Bandung", r, ok)
	}
	if r.Kind != wilayah.Kota {
		t.Errorf("Kota Bandung Kind = %s, want kota", r.Kind)
	}

	kab, ok := wilayah.RegencyByCode("12.04")
	if !ok || kab.Kind != wilayah.Kabupaten {
		t.Errorf("RegencyByCode(12.04) = %+v, %v; want a kabupaten", kab, ok)
	}

	d, ok := wilayah.DistrictByCode("32.73.07")
	if !ok || d.RegencyCode() != "32.73" {
		t.Errorf("DistrictByCode(32.73.07) = %+v, %v", d, ok)
	}
}

func TestLookupRejectsUnknownAndMalformedCodes(t *testing.T) {
	cases := []string{"", "99", "32.99", "32-73", "32.", ".32", "3a", "32.73.07.1001.5", "32.073"}

	for _, code := range cases {
		t.Run(code, func(t *testing.T) {
			if _, ok := wilayah.Find(code); ok {
				t.Errorf("Find(%q) succeeded, want no match", code)
			}
		})
	}
}

func TestChildren(t *testing.T) {
	regencies := wilayah.Regencies("32")
	if len(regencies) == 0 {
		t.Fatal("Regencies(32) is empty")
	}
	for _, r := range regencies {
		if r.ProvinceCode() != "32" {
			t.Fatalf("Regencies(32) returned %s", r.Code)
		}
	}

	districts := wilayah.Districts("32.73")
	if len(districts) == 0 {
		t.Fatal("Districts(32.73) is empty")
	}
	for _, d := range districts {
		if d.RegencyCode() != "32.73" {
			t.Fatalf("Districts(32.73) returned %s", d.Code)
		}
	}

	villages := wilayah.Villages("32.73.07")
	if len(villages) == 0 {
		t.Fatal("Villages(32.73.07) is empty")
	}
	for _, v := range villages {
		if v.DistrictCode() != "32.73.07" {
			t.Fatalf("Villages(32.73.07) returned %s", v.Code)
		}
	}

	if got := wilayah.Districts("99.99"); len(got) != 0 {
		t.Errorf("Districts(99.99) returned %d rows, want 0", len(got))
	}
}

// A query may name any ancestor, not just the level directly above, and the
// result must equal the union of the levels in between.
func TestChildrenAcceptAnyAncestor(t *testing.T) {
	viaDistricts := 0
	for _, d := range wilayah.Districts("32.73") {
		viaDistricts += len(wilayah.Villages(d.Code))
	}

	if got := len(wilayah.Villages("32.73")); got != viaDistricts {
		t.Errorf("Villages(32.73) = %d rows, want %d from its districts", got, viaDistricts)
	}

	viaRegencies := 0
	for _, r := range wilayah.Regencies("32") {
		viaRegencies += len(wilayah.Villages(r.Code))
	}

	if got := len(wilayah.Villages("32")); got != viaRegencies {
		t.Errorf("Villages(32) = %d rows, want %d from its regencies", got, viaRegencies)
	}
}

// A truncated code is not an ancestor of anything: "32.7" must not behave like
// province 32, which is what the "." separator in the range bounds guarantees.
func TestPartialCodesMatchNothing(t *testing.T) {
	for _, ancestor := range []string{"3", "32.7", "32.73.0", "32.73.07.100"} {
		t.Run(ancestor, func(t *testing.T) {
			if got := len(wilayah.Villages(ancestor)); got != 0 {
				t.Errorf("Villages(%q) returned %d rows, want 0", ancestor, got)
			}
		})
	}
}

// Children of every regency must add up to the full district table, which is
// what proves the prefix range never overlaps or skips.
func TestChildrenPartitionTheLevel(t *testing.T) {
	total := 0
	for _, r := range wilayah.Regencies("") {
		total += len(wilayah.Districts(r.Code))
	}

	if want := len(wilayah.Districts("")); total != want {
		t.Errorf("districts reached through regencies = %d, want %d", total, want)
	}
}

func TestFind(t *testing.T) {
	cases := []struct {
		code  string
		level wilayah.Level
	}{
		{"32", wilayah.LevelProvince},
		{"32.73", wilayah.LevelRegency},
		{"32.73.07", wilayah.LevelDistrict},
		{"32.73.07.1001", wilayah.LevelVillage},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got, ok := wilayah.Find(tc.code)
			if !ok {
				t.Fatalf("Find(%q) found nothing", tc.code)
			}
			if got.Level != tc.level {
				t.Errorf("Level = %s, want %s", got.Level, tc.level)
			}
			if got.Code != tc.code || got.Name == "" {
				t.Errorf("Find(%q) = %+v", tc.code, got)
			}
		})
	}
}

// Callers get their own slice; editing it must not leak into the shared table.
func TestReturnedSlicesAreCopies(t *testing.T) {
	first := wilayah.Provinces()
	original := first[0].Name
	first[0].Name = "Diubah"

	if again := wilayah.Provinces(); again[0].Name != original {
		t.Errorf("shared data was modified: %q, want %q", again[0].Name, original)
	}
}

func TestIteratorsMatchSlices(t *testing.T) {
	want := wilayah.Villages("")

	i := 0
	for v := range wilayah.AllVillages() {
		if i >= len(want) {
			t.Fatalf("AllVillages yielded more than %d rows", len(want))
		}
		if v != want[i] {
			t.Fatalf("AllVillages row %d = %+v, want %+v", i, v, want[i])
		}
		i++
	}

	if i != len(want) {
		t.Errorf("AllVillages yielded %d rows, want %d", i, len(want))
	}
}

func TestIteratorStopsEarly(t *testing.T) {
	seen := 0
	for range wilayah.AllVillages() {
		seen++
		if seen == 3 {
			break
		}
	}

	if seen != 3 {
		t.Errorf("stopped after %d rows, want 3", seen)
	}
}

func TestPreloadIsIdempotent(t *testing.T) {
	wilayah.Preload()
	wilayah.Preload()

	if len(wilayah.Provinces()) != wilayah.Dataset().Counts.Provinces {
		t.Error("data changed after Preload")
	}
}

func codesOf[T any](items []T, code func(T) string) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = code(item)
	}
	return out
}

func assertSorted(t *testing.T, codes []string) {
	t.Helper()

	for i := 1; i < len(codes); i++ {
		if codes[i] <= codes[i-1] {
			t.Fatalf("code %q does not come after %q at index %d", codes[i], codes[i-1], i)
		}
	}
}

func BenchmarkVillageByCode(b *testing.B) {
	wilayah.Preload()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := wilayah.VillageByCode("32.73.07.1001"); !ok {
			b.Fatal("not found")
		}
	}
}

func BenchmarkVillagesOfDistrict(b *testing.B) {
	wilayah.Preload()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if len(wilayah.Villages("32.73.07")) == 0 {
			b.Fatal("empty")
		}
	}
}

func BenchmarkAllVillages(b *testing.B) {
	wilayah.Preload()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		n := 0
		for range wilayah.AllVillages() {
			n++
		}
	}
}
