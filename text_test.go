package wilayah_test

import (
	"encoding/json"
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestJSONRoundTrip(t *testing.T) {
	v, ok := wilayah.VillageByCode("32.73.07.1001")
	if !ok {
		t.Fatal("fixture village not found")
	}

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var back wilayah.Village
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal(%s): %v", raw, err)
	}
	if back != v {
		t.Errorf("round trip changed the village:\n got %+v\nwant %+v", back, v)
	}
}

func TestEnumsMarshalAsText(t *testing.T) {
	cases := []struct {
		value any
		want  string
	}{
		{wilayah.LevelProvince, `"province"`},
		{wilayah.LevelVillage, `"village"`},
		{wilayah.Kabupaten, `"kabupaten"`},
		{wilayah.Kota, `"kota"`},
		{wilayah.Desa, `"desa"`},
		{wilayah.Kelurahan, `"kelurahan"`},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			raw, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(raw) != tc.want {
				t.Errorf("Marshal = %s, want %s", raw, tc.want)
			}
		})
	}
}

func TestEnumsRejectUnknownText(t *testing.T) {
	var level wilayah.Level
	if err := json.Unmarshal([]byte(`"kecamatan"`), &level); err == nil {
		t.Error("Level accepted an unknown name")
	}

	var kind wilayah.VillageKind
	if err := json.Unmarshal([]byte(`"dusun"`), &kind); err == nil {
		t.Error("VillageKind accepted an unknown name")
	}
}

func TestRegionMarshalsLevelAsText(t *testing.T) {
	r, ok := wilayah.Find("32.73")
	if !ok {
		t.Fatal("fixture regency not found")
	}

	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := `{"level":"regency","code":"32.73","name":"Kota Bandung"}`
	if string(raw) != want {
		t.Errorf("Marshal = %s, want %s", raw, want)
	}
}
