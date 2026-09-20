package wilayah

import (
	"slices"
	"strings"
)

// SearchOption narrows or shapes a search.
type SearchOption func(*searchConfig)

type searchConfig struct {
	levels []Level
	limit  int
}

// AtLevel restricts a search to the given levels. Without it, every level is
// searched.
func AtLevel(levels ...Level) SearchOption {
	return func(c *searchConfig) {
		c.levels = levels
	}
}

// Limit caps how many results come back. Without it, every match is returned,
// which for a short query can be thousands of rows.
func Limit(n int) SearchOption {
	return func(c *searchConfig) {
		c.limit = n
	}
}

// Search finds regions whose name contains the query, ignoring case.
//
// Results are ranked before the limit is applied: an exact name first, then
// names that start with the query, then the rest, each group in code order.
// That ordering is what makes the top few results useful for an autocomplete.
//
// Matching is on names only. To resolve a code, use Find.
func Search(query string, opts ...SearchOption) []Region {
	query, hint := parseQuery(query)
	if query == "" {
		return nil
	}

	cfg := searchConfig{limit: -1}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.limit == 0 {
		return nil
	}

	type scored struct {
		region Region
		rank   int
		hinted int
	}

	var matches []scored

	// searchable is what the query is matched against, which is not always the
	// name that gets returned: a regency is matched on "Bandung" so that the
	// query "bandung" finds "Kota Bandung" as an exact hit rather than burying
	// it under the villages that happen to be named Bandung.
	collect := func(level Level, code, name, searchable string, hinted bool) {
		rank, ok := rankMatch(searchable, query)
		if !ok {
			return
		}

		penalty := 0
		if hint.level != 0 && !hinted {
			penalty = 1
		}

		matches = append(matches, scored{
			region: Region{Level: level, Code: code, Name: name},
			rank:   rank,
			hinted: penalty,
		})
	}

	if cfg.wants(LevelProvince) {
		for _, p := range provinces.all() {
			collect(LevelProvince, p.Code, p.Name, p.Name, hint.level == LevelProvince)
		}
	}
	if cfg.wants(LevelRegency) {
		for _, r := range regencies.all() {
			hinted := hint.level == LevelRegency && (!hint.hasRegencyKind || hint.regencyKind == r.Kind)
			collect(LevelRegency, r.Code, r.Name, r.ShortName(), hinted)
		}
	}
	if cfg.wants(LevelDistrict) {
		for _, d := range districts.all() {
			collect(LevelDistrict, d.Code, d.Name, d.Name, hint.level == LevelDistrict)
		}
	}
	if cfg.wants(LevelVillage) {
		for _, v := range villages.all() {
			hinted := hint.level == LevelVillage && (!hint.hasVillageKind || hint.villageKind == v.Kind)
			collect(LevelVillage, v.Code, v.Name, v.Name, hinted)
		}
	}

	// Ranking order: how well the name matched, then whether the region is the
	// kind the query asked for, then the level, so a province outranks a
	// village of the same name. The sort is stable, so within a group the code
	// order the rows were collected in survives.
	slices.SortStableFunc(matches, func(a, b scored) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		if a.hinted != b.hinted {
			return a.hinted - b.hinted
		}
		return int(a.region.Level) - int(b.region.Level)
	})

	if cfg.limit >= 0 && len(matches) > cfg.limit {
		matches = matches[:cfg.limit]
	}

	out := make([]Region, len(matches))
	for i, m := range matches {
		out[i] = m.region
	}

	return out
}

func (c searchConfig) wants(level Level) bool {
	if len(c.levels) == 0 {
		return true
	}
	return slices.Contains(c.levels, level)
}

// hint is what a label at the front of a query says about the region being
// looked for.
type hint struct {
	level          Level
	regencyKind    RegencyKind
	hasRegencyKind bool
	villageKind    VillageKind
	hasVillageKind bool
}

// labels maps the words people put in front of a region name, in both the
// written-out and the abbreviated form.
var labels = map[string]hint{
	"prov":      {level: LevelProvince},
	"provinsi":  {level: LevelProvince},
	"kab":       {level: LevelRegency, regencyKind: Kabupaten, hasRegencyKind: true},
	"kabupaten": {level: LevelRegency, regencyKind: Kabupaten, hasRegencyKind: true},
	"kota":      {level: LevelRegency, regencyKind: Kota, hasRegencyKind: true},
	"kec":       {level: LevelDistrict},
	"kecamatan": {level: LevelDistrict},
	"distrik":   {level: LevelDistrict},
	"ds":        {level: LevelVillage, villageKind: Desa, hasVillageKind: true},
	"desa":      {level: LevelVillage, villageKind: Desa, hasVillageKind: true},
	"kel":       {level: LevelVillage, villageKind: Kelurahan, hasVillageKind: true},
	"kelurahan": {level: LevelVillage, villageKind: Kelurahan, hasVillageKind: true},
}

// parseQuery cleans a query up and lifts a leading label off it.
//
// People paste region names in whatever form they were stored in, often
// shouted and labelled: "KAB. ACEH SELATAN". Matching is case insensitive, the
// label is stripped so it does not have to appear in the name, and the label
// is kept as a hint that ranks the region it asked for first.
//
// A label only counts as its own word, so "Kotamobagu" is searched for as
// written rather than read as "Kota" plus "mobagu".
func parseQuery(query string) (string, hint) {
	query = strings.Join(strings.Fields(query), " ")
	if query == "" {
		return "", hint{}
	}

	first, rest, found := strings.Cut(query, " ")
	if !found {
		return query, hint{}
	}

	key := strings.ToLower(strings.TrimSuffix(first, "."))
	h, ok := labels[key]
	if !ok {
		return query, hint{}
	}

	rest = strings.TrimSpace(rest)
	if rest == "" {
		return query, hint{}
	}

	return rest, h
}

// rankMatch scores how well a name matches a query: 0 for the whole name, 1
// for a prefix, 2 for a match anywhere else.
func rankMatch(name, query string) (int, bool) {
	switch {
	case len(name) == len(query) && equalFold(name, query):
		return 0, true
	case len(name) > len(query) && equalFold(name[:len(query)], query):
		return 1, true
	case containsFold(name, query):
		return 2, true
	default:
		return 0, false
	}
}
