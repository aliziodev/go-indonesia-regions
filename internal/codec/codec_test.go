package codec

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func encode(t *testing.T, level Level, recs []Record) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := Encode(&buf, level, recs); err != nil {
		t.Fatalf("Encode(%s): %v", level, err)
	}

	return buf.Bytes()
}

func decode(t *testing.T, blob []byte) *Table {
	t.Helper()

	table, err := Decode(blob)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	return table
}

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		level Level
		recs  []Record
	}{
		{
			level: LevelProvince,
			recs: []Record{
				{Code: "11", Name: "Aceh"},
				{Code: "32", Name: "Jawa Barat"},
			},
		},
		{
			level: LevelRegency,
			recs: []Record{
				{Code: "11.01", Name: "Kabupaten Aceh Selatan", Type: 0},
				{Code: "32.73", Name: "Kota Bandung", Type: 1},
			},
		},
		{
			level: LevelDistrict,
			recs: []Record{
				{Code: "32.73.07", Name: "Cicendo"},
			},
		},
		{
			level: LevelVillage,
			recs: []Record{
				{Code: "32.73.07.1001", Name: "Arjuna", Type: 1, PostalCode: "40172"},
				// Comma, typographic apostrophe and an empty postal code all
				// occur in the real dataset, so they belong in the round trip.
				{Code: "12.07.28.1007", Name: "Lubuk Pakam I,II", Type: 0, PostalCode: "20511"},
				{Code: "53.07.01.2021", Name: "Regapu’u", Type: 0, PostalCode: ""},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.level.String(), func(t *testing.T) {
			table := decode(t, encode(t, tc.level, tc.recs))

			if table.Level() != tc.level {
				t.Errorf("Level() = %s, want %s", table.Level(), tc.level)
			}
			if table.Len() != len(tc.recs) {
				t.Fatalf("Len() = %d, want %d", table.Len(), len(tc.recs))
			}

			for i, want := range tc.recs {
				if got := table.At(i); got != want {
					t.Errorf("At(%d) = %+v, want %+v", i, got, want)
				}
			}
		})
	}
}

func TestEmptyTable(t *testing.T) {
	table := decode(t, encode(t, LevelProvince, nil))

	if table.Len() != 0 {
		t.Errorf("Len() = %d, want 0", table.Len())
	}
}

// Postal codes repeat across villages; interning is what keeps them from
// dominating the arena, so an encoding that lost it should fail here.
func TestArenaInternsRepeatedStrings(t *testing.T) {
	const postal = "40172"

	recs := make([]Record, 0, 500)
	for i := range 500 {
		recs = append(recs, Record{
			Code:       "32.73.07." + string(rune('a'+i%26)) + strings.Repeat("x", i%7),
			Name:       "Desa " + strings.Repeat("y", i%11),
			PostalCode: postal,
		})
	}

	blob := encode(t, LevelVillage, recs)
	table := decode(t, blob)

	if got := bytes.Count(table.arena, []byte(postal)); got != 1 {
		t.Errorf("postal code stored %d times in arena, want 1", got)
	}
	for i := range table.Len() {
		if got := table.At(i).PostalCode; got != postal {
			t.Errorf("At(%d).PostalCode = %q, want %q", i, got, postal)
		}
	}
}

func TestEncodeRejectsBadInput(t *testing.T) {
	t.Run("unknown level", func(t *testing.T) {
		if err := Encode(&bytes.Buffer{}, Level(9), nil); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("missing code", func(t *testing.T) {
		err := Encode(&bytes.Buffer{}, LevelProvince, []Record{{Name: "Tanpa Kode"}})
		if !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("oversized string", func(t *testing.T) {
		err := Encode(&bytes.Buffer{}, LevelProvince, []Record{
			{Code: "11", Name: strings.Repeat("a", 256)},
		})
		if !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})
}

func TestDecodeRejectsCorruptBlob(t *testing.T) {
	valid := encode(t, LevelVillage, []Record{
		{Code: "32.73.07.1001", Name: "Arjuna", Type: 1, PostalCode: "40172"},
	})

	t.Run("not gzip", func(t *testing.T) {
		if _, err := Decode([]byte("bukan gzip")); err == nil {
			t.Error("Decode accepted a non-gzip blob")
		}
	})

	t.Run("truncated gzip", func(t *testing.T) {
		if _, err := Decode(valid[:len(valid)/2]); err == nil {
			t.Error("Decode accepted a truncated blob")
		}
	})

	// Corrupting the payload means rebuilding and re-gzipping it, since gzip
	// carries its own CRC and would otherwise reject the blob before the
	// format checks under test ever run.
	mutate := func(t *testing.T, fn func(buf []byte)) []byte {
		t.Helper()

		table := decode(t, valid)

		buf := make([]byte, headerSize, headerSize+len(table.arena)+len(table.recs))
		copy(buf, magic)
		buf[4] = formatVersion
		buf[5] = byte(table.level)
		buf[6] = table.flags
		buf[7] = byte(table.width)
		binary.LittleEndian.PutUint32(buf[8:], uint32(table.count))
		binary.LittleEndian.PutUint32(buf[12:], uint32(len(table.arena)))
		buf = append(buf, table.arena...)
		buf = append(buf, table.recs...)

		fn(buf)

		var out bytes.Buffer
		zw := gzip.NewWriter(&out)
		if _, err := zw.Write(buf); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}

		return out.Bytes()
	}

	t.Run("bad magic", func(t *testing.T) {
		blob := mutate(t, func(buf []byte) { buf[0] = 'X' })
		if _, err := Decode(blob); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		blob := mutate(t, func(buf []byte) { buf[4] = formatVersion + 1 })
		if _, err := Decode(blob); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("flags do not match level", func(t *testing.T) {
		blob := mutate(t, func(buf []byte) { buf[6] = 0 })
		if _, err := Decode(blob); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("arena length past end", func(t *testing.T) {
		blob := mutate(t, func(buf []byte) { binary.LittleEndian.PutUint32(buf[12:], 1<<20) })
		if _, err := Decode(blob); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("record count past end", func(t *testing.T) {
		blob := mutate(t, func(buf []byte) { binary.LittleEndian.PutUint32(buf[8:], 1<<20) })
		if _, err := Decode(blob); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})

	t.Run("offset past arena", func(t *testing.T) {
		blob := mutate(t, func(buf []byte) {
			recStart := headerSize + int(binary.LittleEndian.Uint32(buf[12:]))
			binary.LittleEndian.PutUint32(buf[recStart:], 1<<20)
		})
		if _, err := Decode(blob); !errors.Is(err, ErrFormat) {
			t.Errorf("err = %v, want ErrFormat", err)
		}
	})
}

func TestAtPanicsOutOfRange(t *testing.T) {
	table := decode(t, encode(t, LevelProvince, []Record{{Code: "11", Name: "Aceh"}}))

	defer func() {
		if recover() == nil {
			t.Error("At did not panic on an out of range index")
		}
	}()

	table.At(1)
}

func BenchmarkDecode(b *testing.B) {
	recs := make([]Record, 0, 10000)
	for i := range 10000 {
		recs = append(recs, Record{
			Code:       "32.73.07." + strings.Repeat("0", i%4) + string(rune('0'+i%10)),
			Name:       "Desa Contoh " + string(rune('A'+i%26)),
			Type:       uint8(i % 2),
			PostalCode: "40172",
		})
	}

	var buf bytes.Buffer
	if err := Encode(&buf, LevelVillage, recs); err != nil {
		b.Fatal(err)
	}
	blob := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := Decode(blob); err != nil {
			b.Fatal(err)
		}
	}
}
