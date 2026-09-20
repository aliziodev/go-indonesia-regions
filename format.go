package wilayah

import "strings"

// FormatOption changes how an address is rendered.
type FormatOption func(*formatConfig)

type formatConfig struct {
	abbreviate    bool
	provinceLabel bool
	noPostal      bool
	country       string
}

// Abbreviate shortens the labels the way they are usually typed on a form:
//
//	Ds. Pasteur, Kec. Sukajadi, Kota Bandung, Jawa Barat 40161
//
// A regency is relabelled rather than prefixed, so "Kabupaten Aceh Selatan"
// becomes "Kab. Aceh Selatan" instead of growing a second label.
func Abbreviate() FormatOption {
	return func(c *formatConfig) { c.abbreviate = true }
}

// WithProvinceLabel puts a label in front of the province, which plenty of
// people write and plenty of others leave off:
//
//	... Kota Bandung, Provinsi Jawa Barat 40161
//	... Kota Bandung, Prov. Jawa Barat 40161     with Abbreviate
func WithProvinceLabel() FormatOption {
	return func(c *formatConfig) { c.provinceLabel = true }
}

// WithoutPostalCode leaves the postal code out.
func WithoutPostalCode() FormatOption {
	return func(c *formatConfig) { c.noPostal = true }
}

// WithCountry appends a country name after the province, for addresses that
// travel outside Indonesia.
func WithCountry(name string) FormatOption {
	return func(c *formatConfig) { c.country = name }
}

// String renders the address in the default style, which spells the labels
// out in full and leaves names exactly as the official data holds them:
//
//	Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat 40161
//
// Only villages and districts are labelled, because regency names already
// carry their own label in the data. Names are never re-cased: over a
// thousand of them contain Roman numerals such as "IV Jurai" or "XIII Koto
// Kampar", which a title-caser would quietly corrupt.
func (a Address) String() string {
	return a.Format()
}

// Format renders the address, adjusted by opts.
func (a Address) Format(opts ...FormatOption) string {
	var cfg formatConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	parts := make([]string, 0, 5)

	if v := a.Village; v != nil {
		parts = append(parts, villageLabel(v.Kind, cfg.abbreviate)+" "+v.Name)
	}
	if d := a.District; d != nil {
		label := "Kecamatan"
		if cfg.abbreviate {
			label = "Kec."
		}
		parts = append(parts, label+" "+d.Name)
	}
	if r := a.Regency; r != nil {
		parts = append(parts, regencyName(*r, cfg.abbreviate))
	}
	if p := a.Province; p != nil {
		province := p.Name
		if cfg.provinceLabel {
			if cfg.abbreviate {
				province = "Prov. " + province
			} else {
				province = "Provinsi " + province
			}
		}
		// The postal code belongs to the province line, the way it is written
		// on an envelope, so a country name goes after it rather than between.
		if postal := a.PostalCode(); postal != "" && !cfg.noPostal {
			province += " " + postal
		}
		parts = append(parts, province)
	}

	if cfg.country != "" {
		parts = append(parts, cfg.country)
	}

	return strings.Join(parts, ", ")
}

// Short renders the address without the village, the labels or the postal
// code, for places where a compact form reads better:
//
//	Sukajadi, Kota Bandung, Jawa Barat
func (a Address) Short() string {
	parts := make([]string, 0, 3)

	if d := a.District; d != nil {
		parts = append(parts, d.Name)
	}
	if r := a.Regency; r != nil {
		parts = append(parts, r.Name)
	}
	if p := a.Province; p != nil {
		parts = append(parts, p.Name)
	}

	return strings.Join(parts, ", ")
}

func villageLabel(kind VillageKind, abbreviate bool) string {
	if kind == Kelurahan {
		if abbreviate {
			return "Kel."
		}
		return "Kelurahan"
	}
	if abbreviate {
		return "Ds."
	}
	return "Desa"
}

// regencyName returns the name as the data holds it, or relabelled when
// abbreviating. "Kota" is left alone because it is already as short as it
// gets.
func regencyName(r Regency, abbreviate bool) string {
	if !abbreviate || r.Kind == Kota {
		return r.Name
	}
	return "Kab. " + r.ShortName()
}
