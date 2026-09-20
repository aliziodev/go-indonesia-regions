package codec

import "unsafe"

// unsafeString reinterprets b as a string without copying.
//
// This is sound here, and only here, because of one invariant the package
// maintains: the arena is written once by Decode and never mutated afterwards.
// The strings handed to callers alias that buffer, which is what keeps lookups
// allocation-free — a full decode of the village table would otherwise cost a
// quarter of a million small copies.
//
// The string keeps the whole arena alive, which is intended: the table owns it
// for its entire lifetime anyway.
func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}
