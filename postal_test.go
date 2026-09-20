package wilayah_test

import (
	"strings"
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestVillagesByPostalCode(t *testing.T) {
	// Pick a code from the data rather than hard-coding one, so a sync that
	// renumbers an area cannot turn this into a false failure.
	sample, ok := wilayah.VillageByCode("32.73.07.1001")
	if !ok {
		t.Fatal("fixture village not found")
	}

	got := wilayah.VillagesByPostalCode(sample.PostalCode)
	if len(got) == 0 {
		t.Fatalf("VillagesByPostalCode(%q) returned nothing", sample.PostalCode)
	}

	found := false
	var prev string
	for _, v := range got {
		if v.PostalCode != sample.PostalCode {
			t.Errorf("%s has postal code %q, want %q", v.Code, v.PostalCode, sample.PostalCode)
		}
		if prev != "" && v.Code <= prev {
			t.Errorf("results are not in code order: %q after %q", v.Code, prev)
		}
		prev = v.Code

		if v.Code == sample.Code {
			found = true
		}
	}

	if !found {
		t.Errorf("VillagesByPostalCode(%q) did not include %s", sample.PostalCode, sample.Code)
	}
}

func TestVillagesByPostalCodeRejectsMalformedInput(t *testing.T) {
	for _, code := range []string{"", "401", "401612", "abcde"} {
		t.Run(code, func(t *testing.T) {
			if got := wilayah.VillagesByPostalCode(code); len(got) != 0 {
				t.Errorf("VillagesByPostalCode(%q) returned %d rows, want 0", code, len(got))
			}
		})
	}
}

func TestVillagesByPostalCodePrefix(t *testing.T) {
	sample, ok := wilayah.VillageByCode("32.73.07.1001")
	if !ok {
		t.Fatal("fixture village not found")
	}

	prefix := sample.PostalCode[:3]
	got := wilayah.VillagesByPostalCodePrefix(prefix)

	if len(got) < len(wilayah.VillagesByPostalCode(sample.PostalCode)) {
		t.Errorf("prefix %q returned fewer rows than the exact code", prefix)
	}

	var prev string
	for _, v := range got {
		if !strings.HasPrefix(v.PostalCode, prefix) {
			t.Errorf("%s has postal code %q, which does not start with %q", v.Code, v.PostalCode, prefix)
		}
		// Ordered by postal code, then by region code.
		key := v.PostalCode + " " + v.Code
		if prev != "" && key <= prev {
			t.Errorf("results are out of order: %q after %q", key, prev)
		}
		prev = key
	}

	if got := wilayah.VillagesByPostalCodePrefix(""); len(got) != 0 {
		t.Errorf("an empty prefix returned %d rows, want 0", len(got))
	}
}

// Every village must be reachable through its own postal code, which is what
// proves the index covers the table and loses nothing.
func TestPostalIndexCoversEveryVillage(t *testing.T) {
	counts := map[string]int{}
	for v := range wilayah.AllVillages() {
		counts[v.PostalCode]++
	}

	total := 0
	for code, want := range counts {
		got := len(wilayah.VillagesByPostalCode(code))
		if got != want {
			t.Fatalf("VillagesByPostalCode(%q) returned %d rows, want %d", code, got, want)
		}
		total += got
	}

	if want := wilayah.Dataset().Counts.Villages; total != want {
		t.Errorf("the index reaches %d villages, want %d", total, want)
	}
}

func BenchmarkVillagesByPostalCode(b *testing.B) {
	wilayah.Preload()
	wilayah.VillagesByPostalCode("40161") // build the index outside the measurement

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if len(wilayah.VillagesByPostalCode("40161")) == 0 {
			b.Fatal("not found")
		}
	}
}
