// Package codec implements the packed binary format used to embed the region
// dataset in the compiled binary.
//
// A table is stored as a gzip-compressed blob holding a fixed-width record
// block plus a single string arena. Decoding therefore costs one allocation
// for the decompressed buffer instead of one allocation per string: the
// records reference the arena, and so do the strings handed to callers.
package codec

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// Header layout:
//
//	0..3   magic "WLYH"
//	4      format version
//	5      level
//	6      flags
//	7      record width in bytes
//	8..11  record count (uint32)
//	12..15 arena length (uint32)
//	16..   arena, then count*width record bytes
const (
	magic         = "WLYH"
	formatVersion = 1
	headerSize    = 16
)

// Level identifies the administrative level a table holds.
type Level uint8

const (
	LevelProvince Level = 1
	LevelRegency  Level = 2
	LevelDistrict Level = 3
	LevelVillage  Level = 4
)

// String implements fmt.Stringer.
func (l Level) String() string {
	switch l {
	case LevelProvince:
		return "province"
	case LevelRegency:
		return "regency"
	case LevelDistrict:
		return "district"
	case LevelVillage:
		return "village"
	default:
		return fmt.Sprintf("level(%d)", uint8(l))
	}
}

// Optional per-record fields. Which ones a table carries is fixed by its level.
const (
	flagType   uint8 = 1 << 0
	flagPostal uint8 = 1 << 1
)

// ErrFormat reports a blob that is not a valid table.
var ErrFormat = errors.New("codec: invalid table format")

// Record is one row of a table.
type Record struct {
	Code       string
	Name       string
	PostalCode string
	Type       uint8
}

// Table is a decoded, read-only table. The zero value is not usable; obtain one
// from Decode. A Table is safe for concurrent use.
type Table struct {
	level Level
	flags uint8
	width int
	count int
	arena []byte
	recs  []byte
}

func flagsFor(level Level) (uint8, error) {
	switch level {
	case LevelProvince, LevelDistrict:
		return 0, nil
	case LevelRegency:
		return flagType, nil
	case LevelVillage:
		return flagType | flagPostal, nil
	default:
		return 0, fmt.Errorf("%w: unknown level %d", ErrFormat, uint8(level))
	}
}

// widthFor returns the fixed record width implied by flags: code offset+length
// (5 bytes) and name offset+length (5 bytes), plus the optional fields.
func widthFor(flags uint8) int {
	width := 10
	if flags&flagType != 0 {
		width++
	}
	if flags&flagPostal != 0 {
		width += 5
	}
	return width
}

// arenaBuilder concatenates strings into one buffer, reusing identical entries.
// Interning matters most for postal codes: ~83k villages share ~8k distinct
// values, so deduplication removes almost all of that cost.
type arenaBuilder struct {
	buf   []byte
	index map[string]uint32
}

func newArenaBuilder(sizeHint int) *arenaBuilder {
	return &arenaBuilder{
		buf:   make([]byte, 0, sizeHint),
		index: make(map[string]uint32),
	}
}

func (a *arenaBuilder) add(s string) (uint32, uint8, error) {
	if len(s) > math.MaxUint8 {
		return 0, 0, fmt.Errorf("%w: string longer than 255 bytes: %q", ErrFormat, s)
	}
	if s == "" {
		return 0, 0, nil
	}
	if off, ok := a.index[s]; ok {
		return off, uint8(len(s)), nil
	}
	// Widened to uint64 so the comparison also compiles where int is 32 bits
	// and the constant would not fit.
	if uint64(len(a.buf))+uint64(len(s)) > math.MaxUint32 {
		return 0, 0, fmt.Errorf("%w: arena overflow", ErrFormat)
	}

	off := uint32(len(a.buf))
	a.buf = append(a.buf, s...)
	a.index[s] = off

	return off, uint8(len(s)), nil
}

// Encode writes recs as a gzip-compressed table. Record order is preserved
// verbatim; callers decide it.
func Encode(w io.Writer, level Level, recs []Record) error {
	flags, err := flagsFor(level)
	if err != nil {
		return err
	}
	if int64(len(recs)) > math.MaxUint32 {
		return fmt.Errorf("%w: too many records", ErrFormat)
	}

	width := widthFor(flags)
	arena := newArenaBuilder(len(recs) * 24)
	block := make([]byte, 0, len(recs)*width)
	field := make([]byte, 5)

	putField := func(s string) error {
		off, length, err := arena.add(s)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(field, off)
		field[4] = length
		block = append(block, field...)
		return nil
	}

	for i, rec := range recs {
		if rec.Code == "" {
			return fmt.Errorf("%w: record %d has no code", ErrFormat, i)
		}
		if err := putField(rec.Code); err != nil {
			return err
		}
		if err := putField(rec.Name); err != nil {
			return err
		}
		if flags&flagType != 0 {
			block = append(block, rec.Type)
		}
		if flags&flagPostal != 0 {
			if err := putField(rec.PostalCode); err != nil {
				return err
			}
		}
	}

	header := make([]byte, headerSize)
	copy(header, magic)
	header[4] = formatVersion
	header[5] = byte(level)
	header[6] = flags
	header[7] = byte(width)
	binary.LittleEndian.PutUint32(header[8:], uint32(len(recs)))
	binary.LittleEndian.PutUint32(header[12:], uint32(len(arena.buf)))

	zw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return err
	}
	for _, chunk := range [][]byte{header, arena.buf, block} {
		if _, err := zw.Write(chunk); err != nil {
			return err
		}
	}

	return zw.Close()
}

// Decode parses a gzip-compressed table. The returned Table keeps the
// decompressed buffer alive; the input slice is not retained.
func Decode(data []byte) (*Table, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("codec: %w", err)
	}
	defer zr.Close()

	buf, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("codec: %w", err)
	}
	if len(buf) < headerSize {
		return nil, fmt.Errorf("%w: truncated header", ErrFormat)
	}
	if string(buf[:4]) != magic {
		return nil, fmt.Errorf("%w: bad magic", ErrFormat)
	}
	if buf[4] != formatVersion {
		return nil, fmt.Errorf("%w: unsupported version %d", ErrFormat, buf[4])
	}

	level := Level(buf[5])
	wantFlags, err := flagsFor(level)
	if err != nil {
		return nil, err
	}
	flags := buf[6]
	if flags != wantFlags {
		return nil, fmt.Errorf("%w: flags %d do not match level %s", ErrFormat, flags, level)
	}
	width := int(buf[7])
	if width != widthFor(flags) {
		return nil, fmt.Errorf("%w: record width %d does not match flags", ErrFormat, width)
	}

	count := int(binary.LittleEndian.Uint32(buf[8:]))
	arenaLen := int(binary.LittleEndian.Uint32(buf[12:]))

	recStart := headerSize + arenaLen
	if recStart > len(buf) {
		return nil, fmt.Errorf("%w: arena out of bounds", ErrFormat)
	}
	if count > (len(buf)-recStart)/width {
		return nil, fmt.Errorf("%w: record block out of bounds", ErrFormat)
	}
	if trailing := len(buf) - recStart - count*width; trailing != 0 {
		return nil, fmt.Errorf("%w: %d trailing bytes", ErrFormat, trailing)
	}

	t := &Table{
		level: level,
		flags: flags,
		width: width,
		count: count,
		arena: buf[headerSize:recStart],
		recs:  buf[recStart:],
	}

	// Validate every offset once, here, so a corrupt blob fails loudly at load
	// time rather than panicking in the middle of a query later on. The field
	// layout is the same for every record, so it is computed outside the loop.
	offsets := t.stringFieldOffsets()

	for i := range count {
		rec := t.recs[i*width:]
		for _, at := range offsets {
			off := int(binary.LittleEndian.Uint32(rec[at:]))
			length := int(rec[at+4])
			if off+length > arenaLen {
				return nil, fmt.Errorf("%w: record %d points past the arena", ErrFormat, i)
			}
		}
	}

	return t, nil
}

// stringFieldOffsets lists where the (offset,length) pairs sit inside a record.
func (t *Table) stringFieldOffsets() []int {
	offsets := []int{0, 5}
	if t.flags&flagPostal != 0 {
		postalAt := 10
		if t.flags&flagType != 0 {
			postalAt++
		}
		offsets = append(offsets, postalAt)
	}
	return offsets
}

// Level reports which administrative level the table holds.
func (t *Table) Level() Level { return t.level }

// Len reports the number of records.
func (t *Table) Len() int { return t.count }

// At returns record i. It panics if i is out of range, like a slice index does.
// The returned strings share the arena of the table and must not be modified.
func (t *Table) At(i int) Record {
	if i < 0 || i >= t.count {
		panic(fmt.Sprintf("codec: index %d out of range [0:%d]", i, t.count))
	}

	rec := t.recs[i*t.width:]
	out := Record{
		Code: t.str(rec, 0),
		Name: t.str(rec, 5),
	}

	next := 10
	if t.flags&flagType != 0 {
		out.Type = rec[next]
		next++
	}
	if t.flags&flagPostal != 0 {
		out.PostalCode = t.str(rec, next)
	}

	return out
}

// str resolves the (offset,length) pair at position at into an arena string.
func (t *Table) str(rec []byte, at int) string {
	length := int(rec[at+4])
	if length == 0 {
		return ""
	}
	off := int(binary.LittleEndian.Uint32(rec[at:]))

	// The arena is never mutated after Decode, so handing out strings that
	// alias it is safe, and it keeps lookups free of allocation.
	return unsafeString(t.arena[off : off+length])
}
