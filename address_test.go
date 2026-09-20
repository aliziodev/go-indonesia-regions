package wilayah_test

import (
	"strings"
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestResolveFillsEveryLevelAboveTheCode(t *testing.T) {
	cases := []struct {
		code    string
		level   wilayah.Level
		village bool
	}{
		{"32.73.07.1001", wilayah.LevelVillage, true},
		{"32.73.07", wilayah.LevelDistrict, false},
		{"32.73", wilayah.LevelRegency, false},
		{"32", wilayah.LevelProvince, false},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			addr, ok := wilayah.Resolve(tc.code)
			if !ok {
				t.Fatalf("Resolve(%q) found nothing", tc.code)
			}

			if addr.Level() != tc.level {
				t.Errorf("Level() = %s, want %s", addr.Level(), tc.level)
			}
			if addr.Code() != tc.code {
				t.Errorf("Code() = %q, want %q", addr.Code(), tc.code)
			}
			if addr.Province == nil {
				t.Error("Province is nil")
			}
			if (addr.Village != nil) != tc.village {
				t.Errorf("Village = %v, want present: %v", addr.Village, tc.village)
			}

			// Levels below the one the code names must stay empty.
			if tc.level < wilayah.LevelDistrict && addr.District != nil {
				t.Errorf("District = %v, want nil", addr.District)
			}
			if tc.level < wilayah.LevelRegency && addr.Regency != nil {
				t.Errorf("Regency = %v, want nil", addr.Regency)
			}
		})
	}
}

func TestResolveRejectsUnknownAndMalformedCodes(t *testing.T) {
	for _, code := range []string{"", "99", "32.99", "32.73.99", "32.7", "abc"} {
		t.Run(code, func(t *testing.T) {
			if _, ok := wilayah.Resolve(code); ok {
				t.Errorf("Resolve(%q) succeeded, want no match", code)
			}
		})
	}
}

func TestAddressString(t *testing.T) {
	addr, ok := wilayah.Resolve("32.73.07.1001")
	if !ok {
		t.Fatal("fixture village not found")
	}

	got := addr.String()

	for _, want := range []string{addr.Village.Name, addr.District.Name, addr.Regency.Name, addr.Province.Name, addr.PostalCode()} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
	if !strings.HasPrefix(got, "Desa ") && !strings.HasPrefix(got, "Kelurahan ") {
		t.Errorf("String() = %q, want it to start with the village label", got)
	}
	if !strings.HasSuffix(got, addr.PostalCode()) {
		t.Errorf("String() = %q, want the postal code last", got)
	}
}

// Regency names already carry their own "Kabupaten" or "Kota" prefix, so a
// formatter that adds one produces "Kota Kota Bandung". The check is that the
// name appears exactly as the data holds it, never with a label bolted on —
// a plain search for "Kota Kota" would libel Kota Kotamobagu, which is a real
// city.
func TestAddressNeverDoublesTheRegencyPrefix(t *testing.T) {
	labels := []string{"Kabupaten ", "Kota ", "Kab. ", "Kota. "}

	for _, r := range wilayah.Regencies("") {
		addr, ok := wilayah.Resolve(r.Code)
		if !ok {
			t.Fatalf("Resolve(%q) found nothing", r.Code)
		}

		for _, form := range []string{addr.String(), addr.Short()} {
			if !strings.Contains(form, r.Name) {
				t.Fatalf("%q does not contain %q", form, r.Name)
			}
			for _, label := range labels {
				if strings.Contains(form, label+r.Name) {
					t.Fatalf("%q carries a second label before %q", form, r.Name)
				}
			}
		}
	}
}

func TestAddressShort(t *testing.T) {
	addr, ok := wilayah.Resolve("32.73.07.1001")
	if !ok {
		t.Fatal("fixture village not found")
	}

	got := addr.Short()

	if strings.Contains(got, addr.Village.Name) {
		t.Errorf("Short() = %q, want no village", got)
	}
	if strings.Contains(got, addr.PostalCode()) {
		t.Errorf("Short() = %q, want no postal code", got)
	}
	if !strings.Contains(got, addr.Regency.Name) || !strings.Contains(got, addr.Province.Name) {
		t.Errorf("Short() = %q, want the regency and province", got)
	}
}

func TestAddressOfProvinceHasNoPostalCode(t *testing.T) {
	addr, ok := wilayah.Resolve("32")
	if !ok {
		t.Fatal("fixture province not found")
	}

	if got := addr.PostalCode(); got != "" {
		t.Errorf("PostalCode() = %q, want empty", got)
	}
	if got := addr.String(); got != addr.Province.Name {
		t.Errorf("String() = %q, want %q", got, addr.Province.Name)
	}
}

func TestZeroAddress(t *testing.T) {
	var addr wilayah.Address

	if got := addr.String(); got != "" {
		t.Errorf("String() = %q, want empty", got)
	}
	if got := addr.Code(); got != "" {
		t.Errorf("Code() = %q, want empty", got)
	}
}

func BenchmarkResolve(b *testing.B) {
	wilayah.Preload()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := wilayah.Resolve("32.73.07.1001"); !ok {
			b.Fatal("not found")
		}
	}
}

func BenchmarkAddressString(b *testing.B) {
	addr, ok := wilayah.Resolve("32.73.07.1001")
	if !ok {
		b.Fatal("not found")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = addr.String()
	}
}
