package wilayah

import "fmt"

// The enums marshal as text rather than numbers, so a region serialized to
// JSON reads as {"kind":"kelurahan"} instead of {"kind":1}. Implementing
// encoding.TextMarshaler covers encoding/json and anything else that honours
// it, and keeps the wire format stable even if the numeric values ever move.

// MarshalText implements encoding.TextMarshaler.
func (l Level) MarshalText() ([]byte, error) {
	if l < LevelProvince || l > LevelVillage {
		return nil, fmt.Errorf("wilayah: cannot marshal %s", l)
	}
	return []byte(l.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *Level) UnmarshalText(text []byte) error {
	switch string(text) {
	case "province":
		*l = LevelProvince
	case "regency":
		*l = LevelRegency
	case "district":
		*l = LevelDistrict
	case "village":
		*l = LevelVillage
	default:
		return fmt.Errorf("wilayah: %q is not a level", text)
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (k RegencyKind) MarshalText() ([]byte, error) {
	if k != Kabupaten && k != Kota {
		return nil, fmt.Errorf("wilayah: cannot marshal regency kind %d", uint8(k))
	}
	return []byte(k.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (k *RegencyKind) UnmarshalText(text []byte) error {
	switch string(text) {
	case "kabupaten":
		*k = Kabupaten
	case "kota":
		*k = Kota
	default:
		return fmt.Errorf("wilayah: %q is not a regency kind", text)
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (k VillageKind) MarshalText() ([]byte, error) {
	if k != Desa && k != Kelurahan {
		return nil, fmt.Errorf("wilayah: cannot marshal village kind %d", uint8(k))
	}
	return []byte(k.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (k *VillageKind) UnmarshalText(text []byte) error {
	switch string(text) {
	case "desa":
		*k = Desa
	case "kelurahan":
		*k = Kelurahan
	default:
		return fmt.Errorf("wilayah: %q is not a village kind", text)
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (t Timezone) MarshalText() ([]byte, error) {
	if t != WIB && t != WITA && t != WIT {
		return nil, fmt.Errorf("wilayah: cannot marshal timezone %d", int8(t))
	}
	return []byte(t.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (t *Timezone) UnmarshalText(text []byte) error {
	switch string(text) {
	case "WIB":
		*t = WIB
	case "WITA":
		*t = WITA
	case "WIT":
		*t = WIT
	default:
		return fmt.Errorf("wilayah: %q is not an Indonesian timezone", text)
	}
	return nil
}
