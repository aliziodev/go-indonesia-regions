package wilayah_test

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

// Examples that assert an Output are pinned to facts that upstream does not
// revise, so a routine data sync cannot break the build. The rest demonstrate
// the API without asserting on data that is allowed to change.

func ExampleProvinceByCode() {
	p, ok := wilayah.ProvinceByCode("32")
	if !ok {
		return
	}

	fmt.Println(p.Code, p.Name)
	// Output: 32 Jawa Barat
}

func ExampleFind() {
	r, ok := wilayah.Find("32.73")
	if !ok {
		return
	}

	fmt.Printf("%s: %s\n", r.Level, r.Name)
	// Output: regency: Kota Bandung
}

func ExampleRegencyByCode() {
	r, ok := wilayah.RegencyByCode("32.73")
	if !ok {
		return
	}

	fmt.Printf("%s is a %s in province %s\n", r.Name, r.Kind, r.ProvinceCode())
	// Output: Kota Bandung is a kota in province 32
}

// Villages accepts any ancestor, so one call can cover a district, a regency
// or a whole province.
func ExampleVillages() {
	inDistrict := wilayah.Villages("32.73.07")
	inRegency := wilayah.Villages("32.73")

	fmt.Println(len(inDistrict) <= len(inRegency))
	// Output: true
}

// A cascading address form needs one call per level.
func ExampleDistricts() {
	for _, d := range wilayah.Districts("32.73") {
		_ = d.Name // populate the district dropdown
	}
}

// AllVillages streams the table without copying it, which is the way to scan
// all 80k+ rows.
func ExampleAllVillages() {
	postal := map[string]int{}
	for v := range wilayah.AllVillages() {
		postal[v.PostalCode]++
	}

	fmt.Println(len(postal) > 1000)
	// Output: true
}

// Regions marshal straight to JSON, with the enums rendered as text.
func ExampleVillage() {
	v, ok := wilayah.VillageByCode("32.73.07.1001")
	if !ok {
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// Resolve fills in every level above a code, which is what a formatted
// address is built from.
func ExampleResolve() {
	addr, ok := wilayah.Resolve("32.73.07.1001")
	if !ok {
		return
	}

	fmt.Println(addr.String())
	fmt.Println(addr.Short())
	fmt.Println(addr.PostalCode())
	// Output:
	// Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat 40161
	// Sukajadi, Kota Bandung, Jawa Barat
	// 40161
}

// Format switches to the labels people usually type on a form. A kabupaten is
// relabelled rather than prefixed, so the name never carries two labels.
func ExampleAddress_Format() {
	addr, ok := wilayah.Resolve("11.01.01.2001")
	if !ok {
		return
	}

	fmt.Println(addr.Format(wilayah.Abbreviate()))
	fmt.Println(addr.Format(wilayah.Abbreviate(), wilayah.WithProvinceLabel()))
	fmt.Println(addr.Format(wilayah.Abbreviate(), wilayah.WithoutPostalCode()))
	fmt.Println(addr.Format(wilayah.WithCountry("Indonesia")))
	// Output:
	// Ds. Keude Bakongan, Kec. Bakongan, Kab. Aceh Selatan, Aceh 23773
	// Ds. Keude Bakongan, Kec. Bakongan, Kab. Aceh Selatan, Prov. Aceh 23773
	// Ds. Keude Bakongan, Kec. Bakongan, Kab. Aceh Selatan, Aceh
	// Desa Keude Bakongan, Kecamatan Bakongan, Kabupaten Aceh Selatan, Aceh 23773, Indonesia
}

// A query may arrive shouted and labelled, the way region names are usually
// stored elsewhere. Search strips the label, uses it to rank the right level
// first, and hands back the official name.
func ExampleSearch_labels() {
	for _, q := range []string{"KAB. ACEH SELATAN", "Kota Bandung"} {
		if r := wilayah.Search(q, wilayah.Limit(1)); len(r) > 0 {
			fmt.Printf("%s -> %s %s\n", q, r[0].Code, r[0].Name)
		}
	}
	// Output:
	// KAB. ACEH SELATAN -> 11.01 Kabupaten Aceh Selatan
	// Kota Bandung -> 32.73 Kota Bandung
}

// A code at any level resolves; the levels below it stay nil.
func ExampleAddress() {
	addr, ok := wilayah.Resolve("32.73")
	if !ok {
		return
	}

	fmt.Println(addr.String())
	fmt.Println(addr.Village == nil, addr.District == nil)
	// Output:
	// Kota Bandung, Jawa Barat
	// true true
}

// One postal code usually covers several villages.
func ExampleVillagesByPostalCode() {
	for _, v := range wilayah.VillagesByPostalCode("40161") {
		_ = v.Name
	}

	// A prefix widens the net: 401 covers 40100 through 40199.
	fmt.Println(len(wilayah.VillagesByPostalCodePrefix("401")) > 0)
	// Output: true
}

// Search matches on names and ranks exact hits first, which is what an
// autocomplete needs. A regency is matched without its Kabupaten or Kota
// label, so this query finds Kota Bandung instead of burying it under the
// villages that are also called Bandung.
func ExampleSearch() {
	for _, r := range wilayah.Search("bandung", wilayah.AtLevel(wilayah.LevelRegency), wilayah.Limit(2)) {
		fmt.Printf("%s %s\n", r.Code, r.Name)
	}
	// Output:
	// 32.04 Kabupaten Bandung
	// 32.73 Kota Bandung
}

// A province or regency carries more than a name: where it governs from,
// where it sits, and which clock it keeps.
func ExampleProfileByCode() {
	p, ok := wilayah.ProfileByCode("32.73")
	if !ok {
		return
	}

	fmt.Println(p.Capital, p.Timezone)
	fmt.Printf("%.2f km2, %d people\n", p.AreaKm2, p.Population.Total)
	// Output:
	// Bandung WIB
	// 166.59 km2, 2591763 people
}

// The zone is a real time.Location, so it works with the rest of the standard
// library.
func ExampleTimezone_Location() {
	noon := time.Date(2026, 9, 21, 12, 0, 0, 0, wilayah.WIB.Location())

	fmt.Println(noon.In(wilayah.WIT.Location()).Format("15:04 MST"))
	// Output: 14:00 WIT
}

// Dataset reports which upstream snapshot the build carries, which is useful
// to surface in a health endpoint.
func ExampleDataset() {
	info := wilayah.Dataset()

	fmt.Printf("regions as of %s: %d villages\n", info.DataDate, info.Counts.Villages)
}

// Preload moves the one-off decoding cost to start-up instead of the first
// request that happens to need it.
func ExamplePreload() {
	wilayah.Preload()
}
