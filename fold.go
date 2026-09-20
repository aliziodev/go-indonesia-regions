package wilayah

// Case folding for region names, shared by the name helpers and by search.

// trimRegencyPrefix removes the "Kabupaten" or "Kota" label that regency names
// carry in the source data.
func trimRegencyPrefix(name string) string {
	for _, prefix := range [...]string{"Kabupaten ", "Kota "} {
		if len(name) > len(prefix) && equalFold(name[:len(prefix)], prefix) {
			return name[len(prefix):]
		}
	}
	return name
}

// Region names are Latin script, so folding ASCII case is enough. Comparing in
// place like this avoids allocating a lowercased copy of all 91k names on
// every search.
func containsFold(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}

	first := lowerASCII(substr[0])
	for i := 0; i+len(substr) <= len(s); i++ {
		if lowerASCII(s[i]) != first {
			continue
		}
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}

	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		if lowerASCII(a[i]) != lowerASCII(b[i]) {
			return false
		}
	}
	return true
}

func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
