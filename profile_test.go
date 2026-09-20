package wilayah_test

import (
	"encoding/json"
	"testing"
	"time"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestProfileByCode(t *testing.T) {
	p, ok := wilayah.ProfileByCode("32")
	if !ok {
		t.Fatal("Jawa Barat has no profile")
	}

	if p.Capital != "Bandung" {
		t.Errorf("Capital = %q, want Bandung", p.Capital)
	}
	if p.Timezone != wilayah.WIB {
		t.Errorf("Timezone = %s, want WIB", p.Timezone)
	}
	if !p.HasArea() {
		t.Error("Jawa Barat has no area")
	}
	if p.Population.Total == 0 {
		t.Error("Jawa Barat has no population")
	}

	// The country spans three zones, so at least one region must sit in each.
	papua, ok := wilayah.ProfileByCode("94")
	if !ok || papua.Timezone != wilayah.WIT {
		t.Errorf("94 = %+v, %v; want a WIT region", papua, ok)
	}
}

// The decree describes provinces and regencies. Districts and villages have no
// profile, and asking for one is not an error, just a miss.
func TestProfileOnlyCoversProvincesAndRegencies(t *testing.T) {
	for _, code := range []string{"32.73.07", "32.73.07.1001", "", "99", "abc"} {
		t.Run(code, func(t *testing.T) {
			if _, ok := wilayah.ProfileByCode(code); ok {
				t.Errorf("ProfileByCode(%q) returned a profile", code)
			}
		})
	}
}

func TestEveryProvinceAndRegencyHasAProfile(t *testing.T) {
	missing := 0

	for _, p := range wilayah.Provinces() {
		if _, ok := wilayah.ProfileByCode(p.Code); !ok {
			t.Errorf("province %s (%s) has no profile", p.Code, p.Name)
			missing++
		}
	}
	for _, r := range wilayah.Regencies("") {
		if _, ok := wilayah.ProfileByCode(r.Code); !ok {
			t.Errorf("regency %s (%s) has no profile", r.Code, r.Name)
			missing++
		}
	}

	if missing > 0 {
		t.Fatalf("%d regions are missing a profile", missing)
	}
}

// Every field is checked against what is physically possible for Indonesia, so
// a column that silently shifts in the export shows up here rather than in
// somebody else's map.
func TestProfileValuesArePlausible(t *testing.T) {
	codes := []string{}
	for _, p := range wilayah.Provinces() {
		codes = append(codes, p.Code)
	}
	for _, r := range wilayah.Regencies("") {
		codes = append(codes, r.Code)
	}

	withoutArea := 0
	offMap := []string{}

	for _, code := range codes {
		p, ok := wilayah.ProfileByCode(code)
		if !ok {
			t.Fatalf("%s has no profile", code)
		}

		// Indonesia runs from roughly 6°N to 11°S and 95°E to 141°E.
		//
		// Upstream has a typo or two - Kabupaten Wakatobi is recorded at
		// longitude 23.5 when its own boundary polygon starts at 123.5 - so
		// these are collected rather than failed one by one. A handful is the
		// data being human; dozens would mean a column had shifted.
		if p.Latitude < -11.5 || p.Latitude > 6.5 || p.Longitude < 94.5 || p.Longitude > 141.5 {
			offMap = append(offMap, code)
		}
		if p.Elevation < -50 || p.Elevation > 5000 {
			t.Errorf("%s has elevation %f", code, p.Elevation)
		}
		if p.Timezone != wilayah.WIB && p.Timezone != wilayah.WITA && p.Timezone != wilayah.WIT {
			t.Errorf("%s has timezone %d", code, int8(p.Timezone))
		}
		if p.AreaKm2 < 0 {
			t.Errorf("%s has a negative area", code)
		}
		if !p.HasArea() {
			withoutArea++
		}

		if got := p.Population.Male + p.Population.Female; got != p.Population.Total {
			t.Errorf("%s: %d men plus %d women is %d, but the total says %d",
				code, p.Population.Male, p.Population.Female, got, p.Population.Total)
		}
		if p.Population.Total <= 0 {
			t.Errorf("%s has a population of %d", code, p.Population.Total)
		}
	}

	// Upstream leaves the area blank for a handful of regions. A jump here
	// means the column moved, not that Indonesia changed.
	if withoutArea > 20 {
		t.Errorf("%d regions have no area, which is more than the known gap", withoutArea)
	}
	t.Logf("regions without an area figure: %d", withoutArea)

	if len(offMap) > 5 {
		t.Errorf("%d regions sit outside Indonesia: %v", len(offMap), offMap)
	}
	if len(offMap) > 0 {
		t.Logf("coordinates upstream has wrong: %v", offMap)
	}
}

func TestTimezoneLocation(t *testing.T) {
	cases := []struct {
		zone   wilayah.Timezone
		name   string
		offset int
	}{
		{wilayah.WIB, "WIB", 7 * 3600},
		{wilayah.WITA, "WITA", 8 * 3600},
		{wilayah.WIT, "WIT", 9 * 3600},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.zone.String(); got != tc.name {
				t.Errorf("String() = %q, want %q", got, tc.name)
			}

			at := time.Date(2026, 9, 21, 12, 0, 0, 0, tc.zone.Location())
			name, offset := at.Zone()
			if name != tc.name || offset != tc.offset {
				t.Errorf("Zone() = %q, %d; want %q, %d", name, offset, tc.name, tc.offset)
			}
		})
	}

	// Noon in Jakarta is two in the afternoon in Papua.
	noon := time.Date(2026, 9, 21, 12, 0, 0, 0, wilayah.WIB.Location())
	if got := noon.In(wilayah.WIT.Location()).Hour(); got != 14 {
		t.Errorf("noon WIB is %d o'clock WIT, want 14", got)
	}
}

func TestProfileJSON(t *testing.T) {
	p, ok := wilayah.ProfileByCode("32.73")
	if !ok {
		t.Fatal("fixture regency has no profile")
	}

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var back wilayah.Profile
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal(%s): %v", raw, err)
	}
	if back != p {
		t.Errorf("round trip changed the profile:\n got %+v\nwant %+v", back, p)
	}

	// The zone is text on the wire, not a number.
	var decoded struct {
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Timezone != "WIB" {
		t.Errorf("timezone marshalled as %q, want WIB", decoded.Timezone)
	}
}

func TestTimezoneRejectsUnknownText(t *testing.T) {
	var zone wilayah.Timezone
	if err := json.Unmarshal([]byte(`"WITB"`), &zone); err == nil {
		t.Error("Timezone accepted an unknown name")
	}
}

func BenchmarkProfileByCode(b *testing.B) {
	wilayah.ProfileByCode("32.73") // decode outside the measurement

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := wilayah.ProfileByCode("32.73"); !ok {
			b.Fatal("not found")
		}
	}
}
