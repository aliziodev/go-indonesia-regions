package wilayah_test

import (
	"testing"

	wilayah "github.com/aliziodev/go-indonesia-regions"
)

func TestLevelOf(t *testing.T) {
	cases := []struct {
		code  string
		level wilayah.Level
		ok    bool
	}{
		{"32", wilayah.LevelProvince, true},
		{"32.73", wilayah.LevelRegency, true},
		{"32.73.07", wilayah.LevelDistrict, true},
		{"32.73.07.1001", wilayah.LevelVillage, true},
		{"00", wilayah.LevelProvince, true}, // shape only; existence is a lookup

		{"", 0, false},
		{"3", 0, false},
		{"323", 0, false},
		{"32.", 0, false},
		{".32", 0, false},
		{"32..73", 0, false},
		{"32.7", 0, false},
		{"32.733", 0, false},
		{"32-73", 0, false},
		{"3a", 0, false},
		{"32.73.07.100", 0, false},
		{"32.73.07.10012", 0, false},
		{"32.73.07.1001.11", 0, false},
		{" 32", 0, false},
		{"32 ", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			level, ok := wilayah.LevelOf(tc.code)
			if ok != tc.ok || (ok && level != tc.level) {
				t.Errorf("LevelOf(%q) = %v, %v; want %v, %v", tc.code, level, ok, tc.level, tc.ok)
			}
			if got := wilayah.ValidCode(tc.code); got != tc.ok {
				t.Errorf("ValidCode(%q) = %v, want %v", tc.code, got, tc.ok)
			}
		})
	}
}

func TestParentCode(t *testing.T) {
	cases := map[string]string{
		"32":            "",
		"32.73":         "32",
		"32.73.07":      "32.73",
		"32.73.07.1001": "32.73.07",
	}

	for code, want := range cases {
		if got := wilayah.ParentCode(code); got != want {
			t.Errorf("ParentCode(%q) = %q, want %q", code, got, want)
		}
	}
}

// Whatever the input, code handling must not panic, and a valid code must have
// a valid parent exactly one level up.
func FuzzLevelOf(f *testing.F) {
	for _, seed := range []string{"32", "32.73", "32.73.07", "32.73.07.1001", "", ".", "32..", "9999999999999"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, code string) {
		level, ok := wilayah.LevelOf(code)
		if !ok {
			return
		}

		if level < wilayah.LevelProvince || level > wilayah.LevelVillage {
			t.Fatalf("LevelOf(%q) = %v, which is not a level", code, level)
		}

		parent := wilayah.ParentCode(code)
		if level == wilayah.LevelProvince {
			if parent != "" {
				t.Fatalf("province %q has parent %q", code, parent)
			}
			return
		}

		parentLevel, ok := wilayah.LevelOf(parent)
		if !ok || parentLevel != level-1 {
			t.Fatalf("parent of %q is %q, whose level is %v, %v", code, parent, parentLevel, ok)
		}
	})
}
