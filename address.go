package wilayah

// Address is a region resolved together with everything above it.
//
// A level the code does not reach is nil: resolving a district code fills
// District, Regency and Province and leaves Village nil.
type Address struct {
	Province *Province `json:"province,omitempty"`
	Regency  *Regency  `json:"regency,omitempty"`
	District *District `json:"district,omitempty"`
	Village  *Village  `json:"village,omitempty"`
}

// Resolve looks a code up at its own level and walks the hierarchy up from
// there. It reports false when the code is malformed or names no region.
//
// Because a code contains the codes of its ancestors, this is one binary
// search per level and needs no extra index.
func Resolve(code string) (Address, bool) {
	level, ok := LevelOf(code)
	if !ok {
		return Address{}, false
	}

	var addr Address

	switch level {
	case LevelVillage:
		v, ok := VillageByCode(code)
		if !ok {
			return Address{}, false
		}
		addr.Village = &v
		code = v.DistrictCode()
		fallthrough

	case LevelDistrict:
		d, ok := DistrictByCode(code)
		if !ok {
			return Address{}, false
		}
		addr.District = &d
		code = d.RegencyCode()
		fallthrough

	case LevelRegency:
		r, ok := RegencyByCode(code)
		if !ok {
			return Address{}, false
		}
		addr.Regency = &r
		code = r.ProvinceCode()
		fallthrough

	case LevelProvince:
		p, ok := ProvinceByCode(code)
		if !ok {
			return Address{}, false
		}
		addr.Province = &p
	}

	return addr, true
}

// Level reports the deepest level the address reaches.
func (a Address) Level() Level {
	switch {
	case a.Village != nil:
		return LevelVillage
	case a.District != nil:
		return LevelDistrict
	case a.Regency != nil:
		return LevelRegency
	default:
		return LevelProvince
	}
}

// Code returns the code of the deepest region in the address.
func (a Address) Code() string {
	switch {
	case a.Village != nil:
		return a.Village.Code
	case a.District != nil:
		return a.District.Code
	case a.Regency != nil:
		return a.Regency.Code
	case a.Province != nil:
		return a.Province.Code
	default:
		return ""
	}
}

// PostalCode returns the postal code of the village, or an empty string when
// the address does not reach that far.
func (a Address) PostalCode() string {
	if a.Village == nil {
		return ""
	}
	return a.Village.PostalCode
}
