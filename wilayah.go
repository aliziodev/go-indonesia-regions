// Package wilayah provides the official Indonesian administrative regions —
// provinces, regencies, districts and villages — with postal codes, embedded
// in your binary.
//
// There is nothing to install or configure: no database, no data files to
// ship, no network access at run time. Every lookup is served from memory.
//
//	p, ok := wilayah.ProvinceByCode("32")      // Jawa Barat
//	for _, r := range wilayah.Regencies("32") {
//		fmt.Println(r.Code, r.Name)
//	}
//
// Beyond lookups, Resolve turns a code into a full address, VillagesByPostalCode goes
// the other way, and Search backs an autocomplete:
//
//	addr, _ := wilayah.Resolve("32.73.07.1001")
//	addr.String()                     // Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat 40161
//	addr.Format(wilayah.Abbreviate()) // Ds. Pasteur, Kec. Sukajadi, Kota Bandung, Jawa Barat 40161
//
//	wilayah.VillagesByPostalCode("40161")
//	wilayah.Search("KAB. ACEH SELATAN", wilayah.Limit(5))
//
// Provinces and regencies carry a profile as well: capital, coordinates,
// elevation, time zone, area and population, all from the same ministerial
// decree as the codes themselves.
//
//	p, _ := wilayah.ProfileByCode("32.73")
//	p.Capital           // Bandung
//	p.Timezone          // WIB
//	p.Population.Total  // 2591763
//
// # Names
//
// Names are served exactly as the official data holds them, and are never
// re-cased. Over a thousand contain Roman numerals, such as "IV Jurai" or
// "XIII Koto Kampar", and a few are genuine acronyms like "RD. PJKA"; a
// title-caser corrupts all of those. To clean up a name that arrives shouted
// or labelled from somewhere else, search for it and keep the official name
// that comes back.
//
// # Codes
//
// Regions are identified by the BPS/Kemendagri code, whose shape tells you the
// level: "32" is a province, "32.73" a regency, "32.73.07" a district and
// "32.73.07.1001" a village. A code always contains the code of its parent,
// which is how the hierarchy is navigated.
//
// # Cost
//
// Tables decode on first use, one level at a time, so a program that only
// reads provinces never pays for the village table. Decoding is guarded by
// sync.Once and every function here is safe for concurrent use.
//
// Functions that return a slice return a fresh copy, so callers are free to
// sort or edit it. For a full scan without the copy, range over the All
// iterators instead.
//
// # Data
//
// The dataset comes from cahyadsn/wilayah and cahyadsn/wilayah_kodepos by way
// of aliziodev/laravel-wilayah, which owns normalization. Dataset reports which
// snapshot a build carries.
package wilayah

import (
	"iter"
	"slices"
)

// Provinces returns all provinces, ordered by code.
func Provinces() []Province {
	return slices.Clone(provinces.all())
}

// ProvinceByCode returns the province with the given code, such as "32".
func ProvinceByCode(code string) (Province, bool) {
	return provinces.find(code)
}

// Regencies returns the regencies of a province, ordered by code. An empty
// provinceCode returns every regency in the country.
func Regencies(provinceCode string) []Regency {
	return slices.Clone(regencies.descendants(provinceCode))
}

// RegencyByCode returns the regency with the given code, such as "32.73".
func RegencyByCode(code string) (Regency, bool) {
	return regencies.find(code)
}

// Districts returns the districts under a region, ordered by code. The
// ancestor may be a regency ("32.73") or a province ("32"); an empty string
// returns every district in the country.
func Districts(ancestor string) []District {
	return slices.Clone(districts.descendants(ancestor))
}

// DistrictByCode returns the district with the given code, such as "32.73.07".
func DistrictByCode(code string) (District, bool) {
	return districts.find(code)
}

// Villages returns the villages under a region, ordered by code. The ancestor
// may be a district ("32.73.07"), a regency ("32.73") or a province ("32").
//
// An empty ancestor returns every village in the country, which copies the
// whole table; range over AllVillages to avoid that.
func Villages(ancestor string) []Village {
	return slices.Clone(villages.descendants(ancestor))
}

// VillageByCode returns the village with the given code, such as "32.73.07.1001".
func VillageByCode(code string) (Village, bool) {
	return villages.find(code)
}

// Find looks a code up at whichever level it belongs to, and reports the match
// in a level-agnostic form.
func Find(code string) (Region, bool) {
	level, ok := LevelOf(code)
	if !ok {
		return Region{}, false
	}

	switch level {
	case LevelProvince:
		if p, ok := ProvinceByCode(code); ok {
			return Region{Level: level, Code: p.Code, Name: p.Name}, true
		}
	case LevelRegency:
		if r, ok := RegencyByCode(code); ok {
			return Region{Level: level, Code: r.Code, Name: r.Name}, true
		}
	case LevelDistrict:
		if d, ok := DistrictByCode(code); ok {
			return Region{Level: level, Code: d.Code, Name: d.Name}, true
		}
	case LevelVillage:
		if v, ok := VillageByCode(code); ok {
			return Region{Level: level, Code: v.Code, Name: v.Name}, true
		}
	}

	return Region{}, false
}

// AllProvinces iterates over every province in code order, without copying.
func AllProvinces() iter.Seq[Province] {
	return slices.Values(provinces.all())
}

// AllRegencies iterates over every regency in code order, without copying.
func AllRegencies() iter.Seq[Regency] {
	return slices.Values(regencies.all())
}

// AllDistricts iterates over every district in code order, without copying.
func AllDistricts() iter.Seq[District] {
	return slices.Values(districts.all())
}

// AllVillages iterates over every village in code order, without copying.
func AllVillages() iter.Seq[Village] {
	return slices.Values(villages.all())
}

// Dataset reports which snapshot of the region data this build carries.
func Dataset() DatasetInfo {
	return datasetInfo
}

// Preload decodes every table up front.
//
// Call it at start-up in a server if you would rather pay the one-off decoding
// cost there than on whichever request happens to be first.
func Preload() {
	provinces.all()
	regencies.all()
	districts.all()
	villages.all()
}
