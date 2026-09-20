package wilayah_test

import (
	"strings"
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestFormatStyles(t *testing.T) {
	addr, ok := wilayah.Resolve("32.73.07.1001") // Pasteur, Kota Bandung
	if !ok {
		t.Fatal("fixture village not found")
	}

	cases := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "default",
			got:  addr.String(),
			want: "Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat 40161",
		},
		{
			name: "abbreviated",
			got:  addr.Format(wilayah.Abbreviate()),
			want: "Ds. Pasteur, Kec. Sukajadi, Kota Bandung, Jawa Barat 40161",
		},
		{
			name: "abbreviated with a province label",
			got:  addr.Format(wilayah.Abbreviate(), wilayah.WithProvinceLabel()),
			want: "Ds. Pasteur, Kec. Sukajadi, Kota Bandung, Prov. Jawa Barat 40161",
		},
		{
			name: "province label spelled out",
			got:  addr.Format(wilayah.WithProvinceLabel()),
			want: "Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Provinsi Jawa Barat 40161",
		},
		{
			name: "without postal code",
			got:  addr.Format(wilayah.WithoutPostalCode()),
			want: "Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat",
		},
		{
			name: "with country",
			got:  addr.Format(wilayah.WithCountry("Indonesia")),
			want: "Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat 40161, Indonesia",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got  %q\nwant %q", tc.got, tc.want)
			}
		})
	}
}

// Abbreviating a kabupaten relabels it rather than prefixing it, so the name
// never ends up carrying two labels.
func TestFormatAbbreviatesKabupaten(t *testing.T) {
	addr, ok := wilayah.Resolve("11.01.01.2001") // Kabupaten Aceh Selatan
	if !ok {
		t.Fatal("fixture village not found")
	}

	got := addr.Format(wilayah.Abbreviate())

	if !strings.Contains(got, "Kab. Aceh Selatan") {
		t.Errorf("got %q, want it to contain %q", got, "Kab. Aceh Selatan")
	}
	if strings.Contains(got, "Kabupaten") {
		t.Errorf("got %q, want the label abbreviated", got)
	}
}

// Every regency must survive both styles with exactly one label.
func TestFormatLabelsEveryRegencyExactlyOnce(t *testing.T) {
	for _, r := range wilayah.Regencies("") {
		addr, ok := wilayah.Resolve(r.Code)
		if !ok {
			t.Fatalf("Resolve(%q) found nothing", r.Code)
		}

		want := "Kab. " + r.ShortName()
		if r.Kind == wilayah.Kota {
			want = r.Name
		}

		// Compare the regency segment as a whole rather than searching for it.
		// A substring check would accuse Kab. Kotawaringin Barat of carrying a
		// second "Kota" label.
		segments := strings.Split(addr.Format(wilayah.Abbreviate()), ", ")
		if got := segments[0]; got != want {
			t.Fatalf("abbreviated form of %q is %q, want %q", r.Name, got, want)
		}
	}
}

func TestFormatOfPartialAddress(t *testing.T) {
	addr, ok := wilayah.Resolve("32.73")
	if !ok {
		t.Fatal("fixture regency not found")
	}

	if got, want := addr.Format(wilayah.Abbreviate()), "Kota Bandung, Jawa Barat"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Names are handed back exactly as the official data holds them. Over a
// thousand contain Roman numerals, which any title-caser would turn into
// "Iv Jurai" or "Xiii Koto Kampar".
func TestFormatLeavesNamesUntouched(t *testing.T) {
	romans := 0

	for d := range wilayah.AllDistricts() {
		for _, numeral := range []string{"IV ", "IX ", "XIII ", "XVI"} {
			if strings.Contains(d.Name, numeral) {
				romans++

				addr, ok := wilayah.Resolve(d.Code)
				if !ok {
					t.Fatalf("Resolve(%q) found nothing", d.Code)
				}
				if !strings.Contains(addr.String(), d.Name) {
					t.Errorf("%q was altered in %q", d.Name, addr.String())
				}
				break
			}
		}
	}

	if romans == 0 {
		t.Error("found no Roman numeral names to check")
	}
}
