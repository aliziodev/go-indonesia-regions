package wilayah

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/aliziodev/go-indonesia-regions/internal/dataset"
)

// Timezone is the zone a region observes, as a UTC offset in whole hours.
type Timezone int8

// The three Indonesian time zones.
const (
	WIB  Timezone = 7 // Waktu Indonesia Barat
	WITA Timezone = 8 // Waktu Indonesia Tengah
	WIT  Timezone = 9 // Waktu Indonesia Timur
)

// String implements fmt.Stringer.
func (t Timezone) String() string {
	switch t {
	case WIB:
		return "WIB"
	case WITA:
		return "WITA"
	case WIT:
		return "WIT"
	default:
		return "UTC+" + strconv.Itoa(int(t))
	}
}

// Indonesia does not observe daylight saving, so a fixed offset is the whole
// story and these three cover the country. They are built once because
// time.FixedZone allocates.
var locations = map[Timezone]*time.Location{
	WIB:  time.FixedZone("WIB", int(WIB)*3600),
	WITA: time.FixedZone("WITA", int(WITA)*3600),
	WIT:  time.FixedZone("WIT", int(WIT)*3600),
}

// Location returns the zone as a *time.Location, ready for time.Time.In.
func (t Timezone) Location() *time.Location {
	if loc, ok := locations[t]; ok {
		return loc
	}
	return time.FixedZone(t.String(), int(t)*3600)
}

// Population counts the people living in a region.
//
// It is a snapshot from the ministerial decree the dataset is built on, not a
// live figure; Dataset reports which snapshot a build carries.
type Population struct {
	Male   int `json:"male"`
	Female int `json:"female"`
	Total  int `json:"total"`
}

// Profile holds what the decree records about a province or regency beyond its
// name: where it is, what it governs from, and how big it is.
//
// Districts and villages have no profile of their own.
type Profile struct {
	Code string `json:"code"`

	// Capital is the seat of government. It is empty for the one region the
	// decree leaves blank.
	Capital string `json:"capital"`

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// Elevation is in metres, and is legitimately negative in a few places.
	Elevation float64 `json:"elevation"`

	Timezone Timezone `json:"timezone"`

	// AreaKm2 is zero for the handful of regions the decree gives no figure
	// for, since no region has an area of zero.
	AreaKm2 float64 `json:"area_km2"`

	Population Population `json:"population"`
}

// HasArea reports whether the dataset carries an area for this region.
func (p Profile) HasArea() bool { return p.AreaKm2 != 0 }

// Coordinates returns the latitude and longitude together, which is the order
// every mapping library expects.
func (p Profile) Coordinates() (lat, lng float64) {
	return p.Latitude, p.Longitude
}

var profiles struct {
	once  sync.Once
	items []Profile
}

// ProfileByCode returns the profile of a province ("32") or regency ("32.73").
// It reports false for any other code, including districts and villages, which
// the decree does not describe.
func ProfileByCode(code string) (Profile, bool) {
	items := allProfiles()

	i := sort.Search(len(items), func(i int) bool { return items[i].Code >= code })
	if i < len(items) && items[i].Code == code {
		return items[i], true
	}

	return Profile{}, false
}

func allProfiles() []Profile {
	profiles.once.Do(func() {
		items, err := decodeProfiles(dataset.Metadata)
		if err != nil {
			// Unreachable in a correctly built binary: the table ships inside
			// the executable and the generator validates it before embedding.
			panic("wilayah: embedded metadata table is unusable: " + err.Error())
		}
		profiles.items = items
	})

	return profiles.items
}

func decodeProfiles(blob []byte) ([]Profile, error) {
	zr, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	r := csv.NewReader(zr)
	r.FieldsPerRecord = 10
	r.ReuseRecord = true

	items := make([]Profile, 0, 552)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		number := func(s string) float64 {
			v, _ := strconv.ParseFloat(s, 64)
			return v
		}
		count := func(s string) int {
			v, _ := strconv.Atoi(s)
			return v
		}

		items = append(items, Profile{
			Code:      row[0],
			Capital:   row[1],
			Latitude:  number(row[2]),
			Longitude: number(row[3]),
			Elevation: number(row[4]),
			Timezone:  Timezone(count(row[5])),
			AreaKm2:   number(row[6]),
			Population: Population{
				Male:   count(row[7]),
				Female: count(row[8]),
				Total:  count(row[9]),
			},
		})
	}

	return items, nil
}
