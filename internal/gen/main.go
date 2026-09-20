// Command gen rebuilds the embedded dataset from the neutral CSV export
// published by aliziodev/laravel-wilayah.
//
// That repository owns normalization: it parses the upstream SQL from
// cahyadsn/wilayah and cahyadsn/wilayah_kodepos and publishes the result as
// release assets. This command only changes the format, so both packages are
// guaranteed to carry the same regions.
//
// Usage:
//
//	go run ./internal/gen                      # latest published release
//	go run ./internal/gen -source ../dist      # local export, for development
//	go run ./internal/gen -source <url-base>   # a specific release
package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliziodev/go-indonesia-regions/internal/codec"
)

const (
	defaultSource   = "https://github.com/aliziodev/laravel-wilayah/releases/latest/download"
	supportedFormat = 1
	maxDownload     = 64 << 20 // the dataset is ~1 MB; this is a sanity bound
	metadataFile    = "metadata.csv"
)

// metadataColumns is the layout the generator expects, and the order the
// package reads back.
var metadataColumns = []string{
	"code", "capital", "lat", "lng", "elevation", "timezone",
	"area", "population_male", "population_female", "population_total",
}

// levels lists the CSV files in dependency order: every level is validated
// against the one before it.
var levels = []struct {
	name   string
	level  codec.Level
	fields int
}{
	{"provinces", codec.LevelProvince, 2},
	{"regencies", codec.LevelRegency, 3},
	{"districts", codec.LevelDistrict, 2},
	{"villages", codec.LevelVillage, 4},
}

type manifest struct {
	Format     int            `json:"format"`
	Version    string         `json:"version"`
	DataDate   string         `json:"data_date"`
	SourceHash string         `json:"source_hash"`
	DataHash   string         `json:"data_hash"`
	Counts     map[string]int `json:"counts"`
	Files      map[string]struct {
		Rows   int    `json:"rows"`
		Bytes  int    `json:"bytes"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("gen: ")

	source := flag.String("source", defaultSource, "release URL base, or a local directory holding the CSV export")
	out := flag.String("out", ".", "module root to write into")
	flag.Parse()

	if err := run(*source, *out); err != nil {
		log.Fatal(err)
	}
}

func run(source, out string) error {
	manifestBytes, files, err := load(source)
	if err != nil {
		return err
	}

	var m manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return fmt.Errorf("parsing version.json: %w", err)
	}
	if m.Format != supportedFormat {
		return fmt.Errorf("export format %d is not supported (this generator speaks %d)", m.Format, supportedFormat)
	}
	if m.DataHash == "" {
		return errors.New("version.json carries no data_hash")
	}

	fmt.Printf("dataset %s (%s)\n", m.Version, m.DataDate)
	fmt.Printf("data_hash %s\n\n", m.DataHash)

	parsed := make(map[string][]codec.Record, len(levels))
	parents := map[string]map[string]struct{}{}

	for _, lv := range levels {
		name := lv.name + ".csv"

		raw, ok := files[name]
		if !ok {
			return fmt.Errorf("%s missing from the export", name)
		}
		if err := verify(m, name, raw); err != nil {
			return err
		}

		recs, err := parseCSV(raw, lv.level, lv.fields, parents[parentOf(lv.name)])
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if want := m.Counts[lv.name]; want != 0 && want != len(recs) {
			return fmt.Errorf("%s: parsed %d rows, manifest says %d", name, len(recs), want)
		}

		parsed[lv.name] = recs
		parents[lv.name] = codeSet(recs)

		fmt.Printf("  %-10s %6d rows\n", lv.name, len(recs))
	}

	datasetDir := filepath.Join(out, "internal", "dataset")
	if err := os.MkdirAll(datasetDir, 0o755); err != nil {
		return err
	}

	fmt.Println()
	for _, lv := range levels {
		path := filepath.Join(datasetDir, lv.name+".bin")

		var buf bytes.Buffer
		if err := codec.Encode(&buf, lv.level, parsed[lv.name]); err != nil {
			return fmt.Errorf("encoding %s: %w", lv.name, err)
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			return err
		}

		fmt.Printf("  wrote %-34s %6.1f KB\n", filepath.ToSlash(path), float64(buf.Len())/1024)
	}

	// Metadata is 552 rows against 83k villages, so it stays gzipped CSV and is
	// parsed at run time. The packed format exists to make the village table
	// cheap to decode; wheeling it out for a table this small would buy
	// microseconds and cost a format that has to carry numbers.
	metaRaw, ok := files[metadataFile]
	if !ok {
		return fmt.Errorf("%s missing from the export", metadataFile)
	}
	if err := verify(m, metadataFile, metaRaw); err != nil {
		return err
	}

	metaRows, err := checkMetadata(metaRaw, parents["provinces"], parents["regencies"])
	if err != nil {
		return fmt.Errorf("%s: %w", metadataFile, err)
	}
	if want := m.Counts["metadata"]; want != 0 && want != metaRows {
		return fmt.Errorf("%s: parsed %d rows, manifest says %d", metadataFile, metaRows, want)
	}

	var metaBuf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&metaBuf, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := zw.Write(metaRaw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	metaPath := filepath.Join(datasetDir, "metadata.csv.gz")
	if err := os.WriteFile(metaPath, metaBuf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("  wrote %-34s %6.1f KB  (%d rows)\n", filepath.ToSlash(metaPath), float64(metaBuf.Len())/1024, metaRows)

	// The manifest is kept verbatim: release automation compares its data_hash
	// against upstream without needing to build the package.
	manifestPath := filepath.Join(datasetDir, "manifest.json")
	if err := os.WriteFile(manifestPath, append(bytes.TrimRight(manifestBytes, "\n"), '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("  wrote %s\n", filepath.ToSlash(manifestPath))

	versionPath := filepath.Join(out, "version.go")
	if err := os.WriteFile(versionPath, versionSource(m), 0o644); err != nil {
		return err
	}
	fmt.Printf("  wrote %s\n", filepath.ToSlash(versionPath))

	return nil
}

// parentOf names the level a given level must resolve its parent codes in.
func parentOf(level string) string {
	switch level {
	case "regencies":
		return "provinces"
	case "districts":
		return "regencies"
	case "villages":
		return "districts"
	default:
		return ""
	}
}

func codeSet(recs []codec.Record) map[string]struct{} {
	set := make(map[string]struct{}, len(recs))
	for _, rec := range recs {
		set[rec.Code] = struct{}{}
	}
	return set
}

// load returns the manifest bytes and the CSV files, from either a local
// directory or a release URL base.
func load(source string) ([]byte, map[string][]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return loadFromRelease(source)
	}
	return loadFromDir(source)
}

func loadFromDir(dir string) ([]byte, map[string][]byte, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(dir, "version.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("reading the export: %w", err)
	}

	names := make([]string, 0, len(levels)+1)
	for _, lv := range levels {
		names = append(names, lv.name+".csv")
	}
	names = append(names, metadataFile)

	files := make(map[string][]byte, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, fmt.Errorf("reading the export: %w", err)
		}
		files[name] = raw
	}

	fmt.Printf("source %s (local)\n", filepath.ToSlash(dir))

	return manifestBytes, files, nil
}

func loadFromRelease(base string) ([]byte, map[string][]byte, error) {
	base = strings.TrimSuffix(base, "/")
	fmt.Printf("source %s\n", base)

	manifestBytes, err := download(base + "/version.json")
	if err != nil {
		return nil, nil, err
	}
	archive, err := download(base + "/dataset.zip")
	if err != nil {
		return nil, nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, nil, fmt.Errorf("reading dataset.zip: %w", err)
	}

	files := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		// Flat archive by construction (zip -j), so a nested path means the
		// asset is not what this generator expects.
		if strings.ContainsAny(f.Name, "/\\") {
			return nil, nil, fmt.Errorf("unexpected path in dataset.zip: %q", f.Name)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", f.Name, err)
		}

		raw, err := io.ReadAll(io.LimitReader(rc, maxDownload))
		rc.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", f.Name, err)
		}

		files[f.Name] = raw
	}

	return manifestBytes, files, nil
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 2 * time.Minute}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-indonesia-regions-gen/1")

	// Release assets go through a CDN that will happily hand back the copy of
	// a file that was just replaced. The checksums in the manifest catch that,
	// but only as a failure; asking for a fresh copy avoids the trip.
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload))
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}

	return body, nil
}

// verify checks a downloaded file against the checksums in the manifest, so a
// truncated or tampered asset can never reach the embedded dataset.
func verify(m manifest, name string, raw []byte) error {
	meta, ok := m.Files[name]
	if !ok {
		return fmt.Errorf("%s: no checksum in version.json", name)
	}
	if meta.Bytes != 0 && meta.Bytes != len(raw) {
		return fmt.Errorf("%s: got %d bytes, manifest says %d", name, len(raw), meta.Bytes)
	}

	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != meta.SHA256 {
		return fmt.Errorf("%s: sha256 mismatch\n  got  %s\n  want %s", name, got, meta.SHA256)
	}

	return nil
}

// parseCSV reads one level and enforces every invariant the package relies on:
// well-formed codes of the right level, strictly ascending order (lookups use
// binary search), no duplicates, and a parent that actually exists.
func parseCSV(raw []byte, level codec.Level, fields int, parents map[string]struct{}) ([]codec.Record, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = fields
	r.ReuseRecord = true

	var (
		recs []codec.Record
		prev string
		line int
	)

	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		line++

		code := row[0]
		if got, ok := levelOf(code); !ok || got != level {
			return nil, fmt.Errorf("line %d: %q is not a %s code", line, code, level)
		}
		if prev != "" && code <= prev {
			return nil, fmt.Errorf("line %d: %q does not come after %q; the export must be sorted by code", line, code, prev)
		}
		if parents != nil {
			parent := code[:strings.LastIndexByte(code, '.')]
			if _, ok := parents[parent]; !ok {
				return nil, fmt.Errorf("line %d: %q has no parent %q in the export", line, code, parent)
			}
		}

		name := strings.TrimSpace(row[1])
		if name == "" {
			return nil, fmt.Errorf("line %d: %q has no name", line, code)
		}

		rec := codec.Record{Code: code, Name: name}

		if fields >= 3 {
			switch row[2] {
			case "0":
				rec.Type = 0
			case "1":
				rec.Type = 1
			default:
				return nil, fmt.Errorf("line %d: %q has type %q, want 0 or 1", line, code, row[2])
			}
		}
		if fields >= 4 {
			postal := strings.TrimSpace(row[3])
			if postal != "" && !isPostalCode(postal) {
				return nil, fmt.Errorf("line %d: %q has postal code %q, want five digits", line, code, postal)
			}
			rec.PostalCode = postal
		}

		recs = append(recs, rec)
		prev = code
	}

	return recs, nil
}

// checkMetadata validates the metadata table before it is embedded: every code
// must name a province or regency that exists, the rows must be sorted and
// unique, and the numbers must parse. Area is allowed to be missing, because
// the decree states none for a handful of regions.
func checkMetadata(raw []byte, provinces, regencies map[string]struct{}) (int, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = len(metadataColumns)
	r.ReuseRecord = true

	var (
		rows int
		prev string
	)

	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, err
		}
		rows++

		code := row[0]
		if _, ok := provinces[code]; !ok {
			if _, ok := regencies[code]; !ok {
				return 0, fmt.Errorf("line %d: %q is not a province or regency in the export", rows, code)
			}
		}
		if prev != "" && code <= prev {
			return 0, fmt.Errorf("line %d: %q does not come after %q; the export must be sorted by code", rows, code, prev)
		}
		prev = code

		// Columns 2..5 are always present; area (6) may be blank.
		for _, i := range []int{2, 3, 4, 5, 7, 8, 9} {
			if row[i] == "" {
				return 0, fmt.Errorf("line %d: %q has no %s", rows, code, metadataColumns[i])
			}
			if _, err := strconv.ParseFloat(row[i], 64); err != nil {
				return 0, fmt.Errorf("line %d: %q has %s %q, which is not a number", rows, code, metadataColumns[i], row[i])
			}
		}
		if row[6] != "" {
			if _, err := strconv.ParseFloat(row[6], 64); err != nil {
				return 0, fmt.Errorf("line %d: %q has area %q, which is not a number", rows, code, row[6])
			}
		}

		switch row[5] {
		case "7", "8", "9": // WIB, WITA, WIT
		default:
			return 0, fmt.Errorf("line %d: %q has timezone %q, want 7, 8 or 9", rows, code, row[5])
		}
	}

	return rows, nil
}

// levelOf mirrors wilayah.LevelOf. It is duplicated here on purpose: the
// generator must not import the package it generates code for.
func levelOf(code string) (codec.Level, bool) {
	widths := [...]int{2, 2, 2, 4}
	rest := code

	for i, width := range widths {
		if len(rest) < width || !digits(rest[:width]) {
			return 0, false
		}
		rest = rest[width:]

		if rest == "" {
			return codec.Level(i + 1), true
		}
		if rest[0] != '.' {
			return 0, false
		}
		rest = rest[1:]
	}

	return 0, false
}

func isPostalCode(s string) bool {
	return len(s) == 5 && digits(s)
}

func digits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func versionSource(m manifest) []byte {
	var b strings.Builder

	b.WriteString("// Code generated by \"go run ./internal/gen\". DO NOT EDIT.\n\n")
	b.WriteString("package wilayah\n\n")
	b.WriteString("// datasetInfo describes the snapshot embedded in this build.\n")
	b.WriteString("var datasetInfo = DatasetInfo{\n")
	fmt.Fprintf(&b, "\tVersion:    %q,\n", m.Version)
	fmt.Fprintf(&b, "\tDataDate:   %q,\n", m.DataDate)
	fmt.Fprintf(&b, "\tDataHash:   %q,\n", m.DataHash)
	fmt.Fprintf(&b, "\tSourceHash: %q,\n", m.SourceHash)
	b.WriteString("\tCounts: Counts{\n")
	fmt.Fprintf(&b, "\t\tProvinces: %d,\n", m.Counts["provinces"])
	fmt.Fprintf(&b, "\t\tRegencies: %d,\n", m.Counts["regencies"])
	fmt.Fprintf(&b, "\t\tDistricts: %d,\n", m.Counts["districts"])
	fmt.Fprintf(&b, "\t\tVillages:  %d,\n", m.Counts["villages"])
	b.WriteString("\t},\n}\n")

	return []byte(b.String())
}
