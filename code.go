package wilayah

import "strings"

// Region codes follow the BPS/Kemendagri scheme: dot-separated numeric
// segments, one per level, each of a fixed width.
//
//	32              province
//	32.73           regency
//	32.73.07        district
//	32.73.07.1001   village
var segmentWidths = [...]int{2, 2, 2, 4}

// LevelOf reports the administrative level a code denotes. It returns false if
// the code is not well formed, so it doubles as a validator.
func LevelOf(code string) (Level, bool) {
	rest := code

	for i, width := range segmentWidths {
		if len(rest) < width || !digits(rest[:width]) {
			return 0, false
		}
		rest = rest[width:]

		if rest == "" {
			return Level(i + 1), true
		}
		if rest[0] != '.' {
			return 0, false
		}
		rest = rest[1:]
	}

	// More segments than the hierarchy has.
	return 0, false
}

// ValidCode reports whether code is a well-formed region code of any level.
func ValidCode(code string) bool {
	_, ok := LevelOf(code)
	return ok
}

// ParentCode returns the code of the region one level up, or an empty string
// for a province code. It does not check that either region exists.
func ParentCode(code string) string {
	return parentCode(code)
}

func parentCode(code string) string {
	if i := strings.LastIndexByte(code, '.'); i > 0 {
		return code[:i]
	}
	return ""
}

func digits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
