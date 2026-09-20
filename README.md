# go-indonesia-regions

[![Go Reference](https://pkg.go.dev/badge/github.com/aliziodev/go-indonesia-regions.svg)](https://pkg.go.dev/github.com/aliziodev/go-indonesia-regions)
[![CI](https://github.com/aliziodev/go-indonesia-regions/actions/workflows/ci.yml/badge.svg)](https://github.com/aliziodev/go-indonesia-regions/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/aliziodev/go-indonesia-regions)](https://github.com/aliziodev/go-indonesia-regions/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/aliziodev/go-indonesia-regions)](go.mod)

The official Indonesian administrative regions — provinces, regencies, districts and villages — with postal codes, embedded in your binary.

No database. No data files to ship. No network calls at run time. No dependencies.

> [Baca dalam Bahasa Indonesia](README.id.md)

```go
import wilayah "github.com/aliziodev/go-indonesia-regions"

p, _ := wilayah.ProvinceByCode("32")        // Jawa Barat
for _, r := range wilayah.Regencies("32") { // its 27 regencies and cities
    fmt.Println(r.Code, r.Name)
}
```

## Install

```bash
go get github.com/aliziodev/go-indonesia-regions
```

Requires Go 1.23 or newer.

## What you get

| | |
|---|---|
| Provinces | 38 |
| Regencies and cities | 514 |
| Districts | 7,285 |
| Villages | 83,762, each with a postal code |
| Province and regency profiles | capital, coordinates, elevation, time zone, area, population |
| Added to your binary | ~1.2 MB |
| Dependencies | none |

## Usage

### Look up by code

```go
p, ok := wilayah.ProvinceByCode("32")            // Jawa Barat
r, ok := wilayah.RegencyByCode("32.73")          // Kota Bandung
d, ok := wilayah.DistrictByCode("32.73.07")      // Sukajadi
v, ok := wilayah.VillageByCode("32.73.07.1001")  // Pasteur, 40161
```

Or let the code shape decide the level:

```go
region, ok := wilayah.Find("32.73")
region.Level // regency
region.Name  // Kota Bandung
```

### Walk the hierarchy

Each function takes the code of **any** ancestor, not just the level directly above:

```go
wilayah.Regencies("32")         // regencies of a province
wilayah.Districts("32.73")      // districts of a regency
wilayah.Districts("32")         // every district in the province
wilayah.Villages("32.73.07")    // villages of a district
wilayah.Villages("32.73")       // every village in the regency
wilayah.Provinces()             // all 38
```

Results come back sorted by code, as a fresh slice you are free to sort or edit.

### Go up

```go
v, _ := wilayah.VillageByCode("32.73.07.1001")

v.DistrictCode()  // 32.73.07
v.RegencyCode()   // 32.73
v.ProvinceCode()  // 32
v.PostalCode      // 40161
v.Kind            // desa or kelurahan
```

### Format an address

```go
addr, ok := wilayah.Resolve("32.73.07.1001")

addr.String()
// Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Jawa Barat 40161

addr.Format(wilayah.Abbreviate())
// Ds. Pasteur, Kec. Sukajadi, Kota Bandung, Jawa Barat 40161

addr.Format(wilayah.Abbreviate(), wilayah.WithProvinceLabel())
// Ds. Pasteur, Kec. Sukajadi, Kota Bandung, Prov. Jawa Barat 40161

addr.Format(wilayah.WithProvinceLabel())
// Desa Pasteur, Kecamatan Sukajadi, Kota Bandung, Provinsi Jawa Barat 40161

addr.Format(wilayah.Abbreviate(), wilayah.WithoutPostalCode())
addr.Format(wilayah.WithCountry("Indonesia"))

addr.Short()  // Sukajadi, Kota Bandung, Jawa Barat
```

`Resolve` takes a code at any level and fills in everything above it. The levels below stay nil, so a regency code gives `Kota Bandung, Jawa Barat`.

Each option does one thing, so they combine freely. `Abbreviate` only shortens the labels that are already there; the province carries none unless `WithProvinceLabel` asks for one, because plenty of people write the province plain.

A regency is relabelled rather than prefixed, because its name already carries a label in the data: `Kabupaten Aceh Selatan` abbreviates to `Kab. Aceh Selatan`, never `Kab. Kabupaten Aceh Selatan`.

### Find by postal code

```go
wilayah.VillagesByPostalCode("40161")      // the villages sharing that code
wilayah.VillagesByPostalCodePrefix("401")  // everything from 40100 to 40199
```

### Search by name

```go
wilayah.Search("bandung", wilayah.Limit(10))
wilayah.Search("sukajadi", wilayah.AtLevel(wilayah.LevelDistrict))
wilayah.Search("bandung", wilayah.AtLevel(wilayah.LevelRegency, wilayah.LevelProvince))
```

Results are ranked before the limit is applied — exact names first, then prefixes, then the rest — so the top few are what an autocomplete should show.

Queries arrive in whatever form the name was stored in elsewhere, so labels are understood and used to rank the right level first:

```go
wilayah.Search("KAB. ACEH SELATAN")  // 11.01 Kabupaten Aceh Selatan
wilayah.Search("Kota Bandung")       // 32.73 Kota Bandung
wilayah.Search("Kab. Bandung")       // 32.04 Kabupaten Bandung
wilayah.Search("Kotamobagu")         // 71.74 Kota Kotamobagu, not read as a label
```

### A note on capitalisation

Names are served exactly as the official data holds them, and are never re-cased. That is deliberate: 1,324 names contain Roman numerals such as `IV Jurai` or `XIII Koto Kampar`, and a few are genuine acronyms like `RD. PJKA` — a title-caser turns those into `Iv Jurai`, `Xiii Koto Kampar` and `Rd. Pjka`.

If a name reaches you shouted or labelled from elsewhere, search for it and keep the official name that comes back. That is more reliable than guessing at capitalisation.

### Look up a profile

A province or regency carries more than a name:

```go
p, ok := wilayah.ProfileByCode("32.73")

p.Capital           // Bandung
p.Latitude          // -6.91
p.Longitude         // 107.61
p.Elevation         // 726 metres
p.Timezone          // WIB
p.AreaKm2           // 166.59
p.Population.Total  // 2591763
```

The time zone is a real `time.Location`, so it composes with the standard library:

```go
noon := time.Date(2026, 9, 21, 12, 0, 0, 0, wilayah.WIB.Location())
noon.In(wilayah.WIT.Location())  // 14:00 WIT
```

Districts and villages have no profile; the decree does not describe them. Area is missing for five regions upstream, so `HasArea` tells you whether a figure exists rather than leaving you to guess at a zero.

Population is a snapshot from the decree the dataset is built on, not a live figure. `Dataset().DataDate` tells you which snapshot you have.

### Scan everything

The `All` iterators stream the table without copying it:

```go
byPostal := map[string][]wilayah.Village{}
for v := range wilayah.AllVillages() {
    byPostal[v.PostalCode] = append(byPostal[v.PostalCode], v)
}
```

### Serve it over HTTP

Regions marshal straight to JSON, with the enums rendered as readable text:

```go
http.HandleFunc("/regencies", func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(wilayah.Regencies(r.URL.Query().Get("province")))
})
```

```json
{"code":"32.73","name":"Kota Bandung","kind":"kota"}
```

### Validate a code

```go
wilayah.ValidCode("32.73.07")   // true
wilayah.LevelOf("32.73.07")     // district, true
wilayah.ParentCode("32.73.07")  // 32.73
```

`ValidCode` checks the shape of a code. To check that a region actually exists, look it up.

## Performance

Tables decode on first use, one level at a time, so a program that only reads provinces never pays for the village table. Everything is safe for concurrent use.

Measured on an AMD Ryzen 9 5900HX:

| | |
|---|---|
| Lookup by code | 112 ns, 0 allocations |
| Villages of a district | 348 ns, 1 allocation |
| Resolve a full address | 386 ns |
| Lookup by postal code | 112 ns |
| Search one level | 14 µs |
| Search every level | 1.7 ms |
| Full scan of 83,762 villages | 259 µs |
| Decoding every level (`Preload`) | 32 ms, 7.9 MB of heap |

Lookups are binary searches over sorted slices, and a query for children is a contiguous range, so no hash map is built and no per-row allocation happens. Strings share one arena per level. Search compares case in place rather than lowercasing 91,599 names on every query, and the postal index is built on first use.

In a server, call `Preload` at start-up if you would rather pay the decoding cost there than on the first request that needs it:

```go
func main() {
    wilayah.Preload()
    // ...
}
```

## Where the data comes from

```
cahyadsn/wilayah + cahyadsn/wilayah_kodepos   upstream source of record
        ↓
aliziodev/laravel-wilayah                     normalization, published as a CSV export
        ↓
go-indonesia-regions                          re-encoded into the embedded tables
```

Normalization lives in one place only, so this package and the Laravel ones can never hold different regions. `Dataset` reports the snapshot a build carries:

```go
info := wilayah.Dataset()
info.DataDate         // 2026-09-20
info.Counts.Villages  // 83762
info.DataHash         // fingerprint of the dataset contents
```

Two builds with the same `DataHash` hold byte-identical region data.

A scheduled workflow checks the published export daily, rebuilds the tables when the fingerprint moves, and releases a new patch version only after the test suite passes on the new data. Staying current is `go get -u`.

## Versioning

Semantic versioning. A data refresh is a patch release; new API is a minor release. The v1 API will not break.

## Known data issues

The data is served exactly as the decree states it, because this package holds no corrections of its own — that is what keeps it identical to the Laravel packages built from the same export. A few things are worth knowing:

- **Kabupaten Wakatobi (74.07) has the wrong longitude.** Upstream records 23.54 where its own boundary polygon starts at 123.58, so the leading 1 was lost during entry. It maps to the coast of Africa until upstream fixes it.
- **Five regions have no area**, DKI Jakarta among them. `HasArea` reports this rather than leaving a zero to be read as a measurement.
- **Population is a snapshot** from Kepmendagri No 300.2.2-2430 Tahun 2025, not a live figure.
- **Names are never re-cased**, so `IV Jurai` and `RD. PJKA` survive intact. See the note on capitalisation above.

Found another? Report it to [cahyadsn/wilayah](https://github.com/cahyadsn/wilayah/issues) and every package downstream of it gets the fix.

## Contributing

Corrections to the region data belong upstream, at [cahyadsn/wilayah](https://github.com/cahyadsn/wilayah) and [cahyadsn/wilayah_kodepos](https://github.com/cahyadsn/wilayah_kodepos) — this package deliberately holds no edits of its own, so that every consumer sees the same regions.

For the package itself, issues and pull requests are welcome. `internal/dataset` and `version.go` are generated; run `go run ./internal/gen` rather than editing them.

Releases are cut automatically from the commit subjects, so they follow [Conventional Commits](https://www.conventionalcommits.org):

| Prefix | Effect |
|---|---|
| `feat:` | minor release |
| `fix:` `perf:` `docs:` | patch release |
| `chore:` `ci:` `test:` `refactor:` | no release; rides along with the next one |
| `feat!:` or `BREAKING CHANGE` | refused — Go needs the module path to move to `/v2`, which is done by hand |

The changelog is written by the same workflow, so there is nothing to update by hand.

## Credits

- [cahyadsn/wilayah](https://github.com/cahyadsn/wilayah) and [cahyadsn/wilayah_kodepos](https://github.com/cahyadsn/wilayah_kodepos) for the data
- [aliziodev/laravel-wilayah](https://github.com/aliziodev/laravel-wilayah) for normalization, and the same regions for Laravel

## License

MIT
