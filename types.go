package wilayah

import "fmt"

// Level is an administrative level of the Indonesian region hierarchy.
type Level uint8

// The four levels, ordered from the broadest to the narrowest.
const (
	LevelProvince Level = iota + 1
	LevelRegency
	LevelDistrict
	LevelVillage
)

// String implements fmt.Stringer.
func (l Level) String() string {
	switch l {
	case LevelProvince:
		return "province"
	case LevelRegency:
		return "regency"
	case LevelDistrict:
		return "district"
	case LevelVillage:
		return "village"
	default:
		return fmt.Sprintf("Level(%d)", uint8(l))
	}
}

// RegencyKind distinguishes a regency (kabupaten) from a city (kota).
type RegencyKind uint8

// The two kinds of second-level region.
const (
	Kabupaten RegencyKind = 0
	Kota      RegencyKind = 1
)

// String implements fmt.Stringer.
func (k RegencyKind) String() string {
	if k == Kota {
		return "kota"
	}
	return "kabupaten"
}

// VillageKind distinguishes a rural village (desa) from an urban one
// (kelurahan).
type VillageKind uint8

// The two kinds of fourth-level region.
const (
	Desa      VillageKind = 0
	Kelurahan VillageKind = 1
)

// String implements fmt.Stringer.
func (k VillageKind) String() string {
	if k == Kelurahan {
		return "kelurahan"
	}
	return "desa"
}

// Province is a first-level region, coded "32".
type Province struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Regency is a second-level region, coded "32.73".
type Regency struct {
	Code string      `json:"code"`
	Name string      `json:"name"`
	Kind RegencyKind `json:"kind"`
}

// ProvinceCode returns the code of the province this regency belongs to.
func (r Regency) ProvinceCode() string { return parentCode(r.Code) }

// ShortName returns the name without its "Kabupaten" or "Kota" prefix, so
// "Kota Bandung" becomes "Bandung".
//
// That prefix is a type label rather than part of the name, which is why Kind
// carries the same information.
func (r Regency) ShortName() string { return trimRegencyPrefix(r.Name) }

// District is a third-level region, coded "32.73.07".
type District struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// RegencyCode returns the code of the regency this district belongs to.
func (d District) RegencyCode() string { return parentCode(d.Code) }

// ProvinceCode returns the code of the province this district belongs to.
func (d District) ProvinceCode() string { return parentCode(d.RegencyCode()) }

// Village is a fourth-level region, coded "32.73.07.1001".
type Village struct {
	Code       string      `json:"code"`
	Name       string      `json:"name"`
	Kind       VillageKind `json:"kind"`
	PostalCode string      `json:"postal_code"`
}

// DistrictCode returns the code of the district this village belongs to.
func (v Village) DistrictCode() string { return parentCode(v.Code) }

// RegencyCode returns the code of the regency this village belongs to.
func (v Village) RegencyCode() string { return parentCode(v.DistrictCode()) }

// ProvinceCode returns the code of the province this village belongs to.
func (v Village) ProvinceCode() string { return parentCode(v.RegencyCode()) }

// Region is a level-agnostic view of a region, for code that handles several
// levels at once, such as a lookup by code or a search across the hierarchy.
type Region struct {
	Level Level  `json:"level"`
	Code  string `json:"code"`
	Name  string `json:"name"`
}

// Counts reports how many regions the embedded dataset holds per level.
type Counts struct {
	Provinces int `json:"provinces"`
	Regencies int `json:"regencies"`
	Districts int `json:"districts"`
	Villages  int `json:"villages"`
}

// DatasetInfo describes the embedded dataset: which upstream snapshot it was
// built from, and how large it is.
//
// DataHash fingerprints the dataset contents and is the value the release
// automation compares against upstream. Two builds carrying the same DataHash
// hold byte-identical region data.
type DatasetInfo struct {
	Version    string `json:"version"`
	DataDate   string `json:"data_date"`
	DataHash   string `json:"data_hash"`
	SourceHash string `json:"source_hash"`
	Counts     Counts `json:"counts"`
}
