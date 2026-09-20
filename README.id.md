# go-indonesia-regions

Data wilayah administratif Indonesia — provinsi, kabupaten/kota, kecamatan, dan desa/kelurahan lengkap dengan kode pos — tertanam langsung di dalam binary Go Anda.

Tanpa database. Tanpa file data yang harus ikut dikirim. Tanpa akses jaringan saat berjalan. Tanpa dependensi.

> [Read in English](README.md)

```go
import wilayah "github.com/aliziodev/go-indonesia-regions"

p, _ := wilayah.ProvinceByCode("32")        // Jawa Barat
for _, r := range wilayah.Regencies("32") { // 27 kabupaten/kota di dalamnya
    fmt.Println(r.Code, r.Name)
}
```

## Instalasi

```bash
go get github.com/aliziodev/go-indonesia-regions
```

Butuh Go 1.23 atau lebih baru.

## Isi paket

| | |
|---|---|
| Provinsi | 38 |
| Kabupaten/kota | 514 |
| Kecamatan | 7.285 |
| Desa/kelurahan | 83.762, semuanya berkode pos |
| Profil provinsi & kab/kota | ibukota, koordinat, elevasi, zona waktu, luas, penduduk |
| Tambahan ukuran binary | ~1,2 MB |
| Dependensi | tidak ada |

## Penggunaan

### Cari berdasarkan kode

```go
p, ok := wilayah.ProvinceByCode("32")            // Jawa Barat
r, ok := wilayah.RegencyByCode("32.73")          // Kota Bandung
d, ok := wilayah.DistrictByCode("32.73.07")      // Sukajadi
v, ok := wilayah.VillageByCode("32.73.07.1001")  // Pasteur, 40161
```

Atau biarkan bentuk kodenya yang menentukan level:

```go
region, ok := wilayah.Find("32.73")
region.Level // regency
region.Name  // Kota Bandung
```

### Menelusuri hierarki

Setiap fungsi menerima kode **leluhur mana pun**, tidak harus induk langsung:

```go
wilayah.Regencies("32")         // kabupaten/kota di sebuah provinsi
wilayah.Districts("32.73")      // kecamatan di sebuah kab/kota
wilayah.Districts("32")         // semua kecamatan di provinsi itu
wilayah.Villages("32.73.07")    // desa di sebuah kecamatan
wilayah.Villages("32.73")       // semua desa di kab/kota itu
wilayah.Provinces()             // seluruh 38 provinsi
```

Hasilnya terurut berdasarkan kode, berupa slice baru yang bebas Anda ubah atau urutkan ulang.

### Menelusuri ke atas

```go
v, _ := wilayah.VillageByCode("32.73.07.1001")

v.DistrictCode()  // 32.73.07
v.RegencyCode()   // 32.73
v.ProvinceCode()  // 32
v.PostalCode      // 40161
v.Kind            // desa atau kelurahan
```

### Merangkai alamat lengkap

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

`Resolve` menerima kode level mana pun dan mengisi semua level di atasnya. Level di bawahnya dibiarkan nil, jadi kode kab/kota menghasilkan `Kota Bandung, Jawa Barat`.

Tiap opsi mengerjakan satu hal, jadi bisa dikombinasikan bebas. `Abbreviate` hanya memendekkan label yang memang sudah ada; provinsi tidak berlabel kecuali diminta lewat `WithProvinceLabel`, karena banyak juga yang menulis provinsi polos tanpa awalan.

Kab/kota **diganti labelnya**, bukan ditambahi, karena namanya di data sudah membawa label sendiri: `Kabupaten Aceh Selatan` disingkat jadi `Kab. Aceh Selatan`, bukan `Kab. Kabupaten Aceh Selatan`.

### Cari lewat kode pos

```go
wilayah.VillagesByPostalCode("40161")      // desa-desa yang berbagi kode pos itu
wilayah.VillagesByPostalCodePrefix("401")  // seluruh 40100 sampai 40199
```

### Cari berdasarkan nama

```go
wilayah.Search("bandung", wilayah.Limit(10))
wilayah.Search("sukajadi", wilayah.AtLevel(wilayah.LevelDistrict))
wilayah.Search("bandung", wilayah.AtLevel(wilayah.LevelRegency, wilayah.LevelProvince))
```

Hasil diperingkat sebelum limit diterapkan — nama persis dulu, lalu yang berawalan sama, baru sisanya — sehingga beberapa teratas memang yang pantas ditampilkan di autocomplete.

Kata kunci datang dalam bentuk apa pun sesuai cara nama itu disimpan di tempat lain, jadi labelnya dipahami dan dipakai untuk mengunggulkan level yang tepat:

```go
wilayah.Search("KAB. ACEH SELATAN")  // 11.01 Kabupaten Aceh Selatan
wilayah.Search("Kota Bandung")       // 32.73 Kota Bandung
wilayah.Search("Kab. Bandung")       // 32.04 Kabupaten Bandung
wilayah.Search("Kotamobagu")         // 71.74 Kota Kotamobagu, tidak dibaca sebagai label
```

### Catatan soal kapitalisasi

Nama disajikan persis seperti data resminya dan tidak pernah diubah kapitalisasinya. Ini disengaja: 1.324 nama mengandung angka Romawi seperti `IV Jurai` atau `XIII Koto Kampar`, dan beberapa memang akronim seperti `RD. PJKA` — semuanya akan dirusak oleh title-case menjadi `Iv Jurai`, `Xiii Koto Kampar`, dan `Rd. Pjka`.

Kalau sebuah nama sampai ke Anda dalam huruf besar semua atau berlabel dari sumber lain, cari namanya lalu pakai nama resmi yang dikembalikan. Itu jauh lebih andal daripada menebak kapitalisasi.

### Melihat profil wilayah

Provinsi dan kab/kota membawa lebih dari sekadar nama:

```go
p, ok := wilayah.ProfileByCode("32.73")

p.Capital           // Bandung
p.Latitude          // -6.91
p.Longitude         // 107.61
p.Elevation         // 726 meter
p.Timezone          // WIB
p.AreaKm2           // 166.59
p.Population.Total  // 2591763
```

Zona waktunya berupa `time.Location` sungguhan, jadi menyatu dengan pustaka standar:

```go
noon := time.Date(2026, 9, 21, 12, 0, 0, 0, wilayah.WIB.Location())
noon.In(wilayah.WIT.Location())  // 14:00 WIT
```

Kecamatan dan desa tidak punya profil karena keputusan itu memang tidak memerincinya. Luas wilayah kosong untuk lima daerah di hulu, jadi `HasArea` memberi tahu ada atau tidaknya angka, ketimbang membiarkan Anda menebak arti nilai nol.

Jumlah penduduk adalah **potret** dari keputusan yang mendasari dataset ini, bukan angka hidup. `Dataset().DataDate` memberi tahu potret kapan yang Anda pegang.

### Menyapu seluruh data

Iterator `All` mengalirkan tabel tanpa menyalinnya:

```go
byPostal := map[string][]wilayah.Village{}
for v := range wilayah.AllVillages() {
    byPostal[v.PostalCode] = append(byPostal[v.PostalCode], v)
}
```

### Menyajikan lewat HTTP

Semua tipe langsung bisa di-marshal ke JSON, dan enum-nya tampil sebagai teks yang terbaca:

```go
http.HandleFunc("/regencies", func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(wilayah.Regencies(r.URL.Query().Get("province")))
})
```

```json
{"code":"32.73","name":"Kota Bandung","kind":"kota"}
```

### Validasi kode

```go
wilayah.ValidCode("32.73.07")   // true
wilayah.LevelOf("32.73.07")     // district, true
wilayah.ParentCode("32.73.07")  // 32.73
```

`ValidCode` hanya memeriksa bentuk kode. Untuk memastikan wilayahnya benar-benar ada, lakukan lookup.

## Performa

Tabel di-decode saat pertama dipakai, per level, jadi program yang hanya membaca provinsi tidak pernah membayar biaya tabel desa. Semua fungsi aman dipakai lintas goroutine.

Diukur pada AMD Ryzen 9 5900HX:

| | |
|---|---|
| Lookup berdasarkan kode | 112 ns, 0 alokasi |
| Desa dalam satu kecamatan | 348 ns, 1 alokasi |
| Merangkai alamat lengkap | 386 ns |
| Lookup lewat kode pos | 112 ns |
| Cari di satu level | 14 µs |
| Cari di semua level | 1,7 ms |
| Sapu penuh 83.762 desa | 259 µs |
| Decode semua level (`Preload`) | 32 ms, heap 7,9 MB |

Lookup memakai binary search pada slice terurut, dan permintaan anak wilayah adalah irisan kontinu — tidak ada hash map yang dibangun dan tidak ada alokasi per baris. Semua string berbagi satu arena per level. Pencarian membandingkan huruf besar-kecil di tempat alih-alih menyalin 91.599 nama jadi huruf kecil tiap kueri, dan indeks kode pos baru dibangun saat pertama dipakai.

Di server, panggil `Preload` saat startup jika Anda lebih suka membayar biaya decode di sana ketimbang di request pertama:

```go
func main() {
    wilayah.Preload()
    // ...
}
```

## Asal data

```
cahyadsn/wilayah + cahyadsn/wilayah_kodepos   sumber resmi di hulu
        ↓
aliziodev/laravel-wilayah                     normalisasi, diterbitkan sebagai export CSV
        ↓
go-indonesia-regions                          di-encode ulang jadi tabel tertanam
```

Normalisasi hanya hidup di satu tempat, sehingga package ini dan package Laravel-nya mustahil memuat wilayah yang berbeda. `Dataset` melaporkan snapshot yang dibawa sebuah build:

```go
info := wilayah.Dataset()
info.DataDate         // 2026-09-20
info.Counts.Villages  // 83762
info.DataHash         // sidik jari isi dataset
```

Dua build dengan `DataHash` sama memuat data wilayah yang identik byte per byte.

Sebuah workflow terjadwal memeriksa export tersebut tiap hari, membangun ulang tabel saat sidik jarinya berubah, dan merilis versi patch baru hanya setelah seluruh test lulus pada data baru. Untuk tetap mutakhir, cukup `go get -u`.

## Versioning

Mengikuti semantic versioning. Pembaruan data adalah rilis patch; penambahan API adalah rilis minor. API v1 tidak akan dirusak.

## Kontribusi

Koreksi data wilayah sebaiknya diajukan ke hulu, di [cahyadsn/wilayah](https://github.com/cahyadsn/wilayah) dan [cahyadsn/wilayah_kodepos](https://github.com/cahyadsn/wilayah_kodepos) — package ini sengaja tidak menyimpan koreksi sendiri, supaya semua konsumen melihat wilayah yang sama.

Untuk package-nya sendiri, issue dan pull request dipersilakan. `internal/dataset` dan `version.go` adalah file generated; jalankan `go run ./internal/gen`, jangan diedit manual.

## Kredit

- [cahyadsn/wilayah](https://github.com/cahyadsn/wilayah) dan [cahyadsn/wilayah_kodepos](https://github.com/cahyadsn/wilayah_kodepos) untuk datanya
- [aliziodev/laravel-wilayah](https://github.com/aliziodev/laravel-wilayah) untuk normalisasi, sekaligus wilayah yang sama untuk Laravel

## Lisensi

MIT
