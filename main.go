package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	mapSize              = 2048
	mapCells             = mapSize * mapSize
	packedBytes          = mapCells / 8
	headerBytes          = 8
	minPinBytesNoOwner   = 18
	minPinBytesWithOwner = 19
	maxVersion           = 3
)

var abbrToName = map[string]string{
	"Rb": "Raspberry", "Bb": "Blueberry", "Mu": "Mushroom", "Th": "Thistle",
	"Cb": "Cloudberry", "Br": "Barley", "Fl": "Flax", "On": "Onion",
	"Ca": "Carrot", "Tu": "Turnip", "Ch": "Chitin", "Cr": "Crystal",
	"Su": "SurtlingCore", "Bc": "BlackCore", "JP": "JotunPuff", "Mc": "Magecap",
	"Dan": "Dandelion", "Fli": "Flint", "BI": "BogIron", "DEgg": "DragonEgg",
	"Tar": "Tar", "Fern": "Fiddlehead", "RJly": "RoyalJelly", "Sulf": "Sulfur",
	"VEgg": "VoltureEgg", "Ash": "Ashstone", "MCor": "MoltenCore", "CSku": "Charredskull",
	"Cu": "Copper", "Sn": "Tin", "Ag": "Silver", "Fe": "Iron",
	"Ob": "Obsidian", "Fm": "Flametal", "Met": "Meteorite", "BkM": "BlackMarble",
	"Stn": "Stone", "SCrypt": "SunkenCrypt", "Crypt": "Crypt", "MCave": "MountainCave",
	"TCave": "TrollCave", "Mud": "MudPile", "Shop": "Vendor", "DNest": "DrakeNest",
	"Henge": "Henge", "Eik": "Eikthyr", "Elder": "GDKing", "Bone": "Bonemass",
	"Moder": "DragonQueen", "Yag": "GoblinKing", "Queen": "SeekerQueen", "Fader": "Fader",
	"FCave": "FrostCave", "IMine": "InfestedMine", "Hild": "Hildir", "Witch": "BogWitch",
	"GCamp": "GoblinCamp", "WFarm": "WoodFarm", "Dtown": "DvergrTown", "Ship": "ShipWreck",
	"Rune": "Runestone", "GSku": "GiantSkull", "BHive": "Beehive", "Grl": "Greyling",
	"Boar": "Boar", "Deer": "Deer", "Neck": "Neck", "Hen": "Chicken",
	"Hare": "Hare", "GdwE": "GreydwarfElite", "GdwS": "GreydwarfShaman", "Gdw": "Greydwarf",
	"Trl": "Troll", "Skel": "Skeleton", "DrE": "DraugrElite", "Dr": "Draugr",
	"Lch": "Leech", "Blob": "Blob", "Wra": "Wraith", "Srt": "Surtling",
	"Abom": "Abomination", "Gst": "Ghost", "Wolf": "Wolf", "Drk": "Drake",
	"Glm": "Stone Golem", "Fen": "Fenring", "Cult": "Cultist", "Bat": "Bat",
	"FulB": "FulingBerserker", "FulS": "FulingShaman", "FulA": "FulingArcher", "Ful": "Fuling",
	"Dsq": "Deathsquito", "Lox": "Lox", "SkrB": "SeekerBrood", "Skr": "Seeker",
	"Gjl": "Gjall", "Tick": "Tick", "DvrM": "DvergrMage", "Dvr": "Dvergr",
	"Ulv": "Ulv", "Chr": "Charred", "Mrg": "Morgen", "Ask": "Asksvin",
	"Vlt": "Volture", "Ubj": "Unbjorn", "Kvas": "Kvastur", "Spt": "Serpent",
	"Lev": "Leviathan", "Fish": "Fish",
}

type reader struct {
	data []byte
	pos  int
}

func (r *reader) remaining() int { return len(r.data) - r.pos }

func (r *reader) need(n int) error {
	if r.pos+n > len(r.data) {
		return fmt.Errorf("wanted %d bytes at offset %d, only %d bytes remain", n, r.pos, r.remaining())
	}
	return nil
}

func (r *reader) readU8() (byte, error) {
	if err := r.need(1); err != nil {
		return 0, err
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

func (r *reader) readBool() (bool, error) {
	v, err := r.readU8()
	return v != 0, err
}

func (r *reader) readI32() (int32, error) {
	if err := r.need(4); err != nil {
		return 0, err
	}
	v := int32(binary.LittleEndian.Uint32(r.data[r.pos:]))
	r.pos += 4
	return v, nil
}

func (r *reader) readF32() (float32, error) {
	if err := r.need(4); err != nil {
		return 0, err
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(r.data[r.pos:]))
	r.pos += 4
	return v, nil
}

func (r *reader) readVector3() (Vector3, error) {
	x, err := r.readF32()
	if err != nil {
		return Vector3{}, err
	}
	y, err := r.readF32()
	if err != nil {
		return Vector3{}, err
	}
	z, err := r.readF32()
	if err != nil {
		return Vector3{}, err
	}
	return Vector3{X: x, Y: y, Z: z}, nil
}

func (r *reader) read7BitEncodedInt() (int, error) {
	value := 0
	shift := 0
	for i := 0; i < 5; i++ {
		b, err := r.readU8()
		if err != nil {
			return 0, err
		}
		value |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			return value, nil
		}
		shift += 7
	}
	return 0, fmt.Errorf("invalid 7-bit encoded int at offset %d", r.pos)
}

func (r *reader) readString() (string, error) {
	n, err := r.read7BitEncodedInt()
	if err != nil {
		return "", err
	}
	if n > r.remaining() {
		return "", fmt.Errorf("invalid string length %d at offset %d; only %d bytes remain", n, r.pos, r.remaining())
	}
	raw := r.data[r.pos : r.pos+n]
	r.pos += n
	return string(raw), nil
}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type DecodedName struct {
	Abbr    string `json:"abbr,omitempty"`
	Decoded string `json:"decoded,omitempty"`
	Count   *int   `json:"count,omitempty"`
}

type Pin struct {
	Name        string  `json:"name"`
	DecodedName string  `json:"decoded_name,omitempty"`
	DecodedAbbr string  `json:"decoded_abbr,omitempty"`
	DecodedCnt  *int    `json:"decoded_count,omitempty"`
	Pos         Vector3 `json:"pos"`
	Type        int32   `json:"type"`
	Checked     bool    `json:"checked"`
	OwnerID     string  `json:"owner_id,omitempty"`
}

type PinSummary struct {
	Key                string `json:"-"`
	DisplayName        string `json:"display_name"`
	Source             string `json:"source"`
	PinCount           int    `json:"pin_count"`
	ImpliedObjectCount int    `json:"implied_object_count"`
	BatchPins          int    `json:"batch_pins"`
	Checked            int    `json:"checked"`
	Unchecked          int    `json:"unchecked"`
}

type DecodedFile struct {
	File                  string       `json:"file"`
	FileSize              int          `json:"file_size"`
	Format                string       `json:"format"`
	MapEncoding           string       `json:"map_encoding"`
	Version               int32        `json:"version"`
	MapSize               int32        `json:"map_size"`
	Cells                 int          `json:"cells"`
	PackedMapBytes        *int         `json:"packed_map_bytes"`
	FixedMapBytes         int          `json:"fixed_map_bytes"`
	EstimatedPayloadBytes int          `json:"estimated_payload_bytes"`
	ExploredCount         int          `json:"explored_count"`
	UnexploredCount       int          `json:"unexplored_count"`
	ExploredPercent       float64      `json:"explored_percent"`
	PinCount              int32        `json:"pin_count"`
	PinsOmitted           bool         `json:"pins_omitted,omitempty"`
	Pins                  []Pin        `json:"pins,omitempty"`
	PinSummary            []PinSummary `json:"pin_summary,omitempty"`
	fullPins              []Pin
}

func findHeader(data []byte) (int, error) {
	if len(data) < headerBytes {
		return 0, io.ErrUnexpectedEOF
	}
	searchLimit := len(data) - headerBytes
	if searchLimit > 2_000_000 {
		searchLimit = 2_000_000
	}
	for off := 0; off <= searchLimit; off++ {
		if looksLikePayloadAt(data, off) {
			return off, nil
		}
	}
	return 0, errors.New("could not find plausible header: int32 version {1,2}, int32 size 2048, readable map and pin count")
}

func looksLikePayloadAt(data []byte, off int) bool {
	if off+headerBytes > len(data) {
		return false
	}
	version := int32(binary.LittleEndian.Uint32(data[off:]))
	size := int32(binary.LittleEndian.Uint32(data[off+4:]))
	if version < 1 || version > maxVersion || size != mapSize {
		return false
	}

	mapBytes, ok := mapBytesForVersion(version)
	if !ok {
		return false
	}

	pinCountOffset := off + headerBytes + mapBytes
	if pinCountOffset+4 > len(data) {
		return false
	}
	pinCount := int32(binary.LittleEndian.Uint32(data[pinCountOffset:]))
	if pinCount < 0 {
		return false
	}
	remaining := len(data) - pinCountOffset - 4
	return int(pinCount) <= remaining/minPinBytesForVersion(version)
}

func mapBytesForVersion(version int32) (int, bool) {
	switch version {
	case 1, 3:
		return mapCells, true
	case 2:
		return packedBytes, true
	default:
		return 0, false
	}
}

func minPinBytesForVersion(version int32) int {
	if pinHasOwner(version) {
		return minPinBytesWithOwner
	}
	return minPinBytesNoOwner
}

func pinHasOwner(version int32) bool {
	return version == 1 || version == 2
}

func formatForVersion(version int32) string {
	switch version {
	case 1, 2:
		return "one_map_to_rule_them_all"
	case 3:
		return "serversidemap"
	default:
		return "unknown"
	}
}

func mapEncodingForVersion(version int32) string {
	switch version {
	case 2:
		return "packed_bits"
	case 1, 3:
		return "bool_bytes"
	default:
		return "unknown"
	}
}

func countV2Bits(blob []byte) (int, error) {
	if len(blob) < packedBytes {
		return 0, fmt.Errorf("need %d packed map bytes, got %d", packedBytes, len(blob))
	}
	count := 0
	for _, b := range blob[:packedBytes] {
		for bit := 0; bit < 8; bit++ {
			if b&(1<<bit) != 0 {
				count++
			}
		}
	}
	return count, nil
}

func decodePinName(name string) DecodedName {
	if name == "" {
		return DecodedName{}
	}
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return DecodedName{}
	}
	out := DecodedName{Abbr: parts[0], Decoded: abbrToName[parts[0]]}
	if len(parts) >= 2 {
		if n, err := strconv.Atoi(parts[1]); err == nil {
			out.Count = &n
		}
	}
	return out
}

func readExploredFile(path string) (*DecodedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	start, err := findHeader(data)
	if err != nil {
		return nil, err
	}
	r := &reader{data: data, pos: start}
	version, err := r.readI32()
	if err != nil {
		return nil, err
	}
	size, err := r.readI32()
	if err != nil {
		return nil, err
	}
	if size != mapSize {
		return nil, fmt.Errorf("unexpected map size %d; expected %d", size, mapSize)
	}

	var exploredCount int
	fixedMapBytes := mapCells
	var packed *int
	switch version {
	case 2:
		packedStart := r.pos
		p := packedBytes
		packed = &p
		fixedMapBytes = packedBytes
		exploredCount, err = countV2Bits(data[packedStart:])
		if err != nil {
			return nil, err
		}
		r.pos += packedBytes
	case 1, 3:
		for i := 0; i < mapCells; i++ {
			v, err := r.readBool()
			if err != nil {
				return nil, err
			}
			if v {
				exploredCount++
			}
		}
	default:
		return nil, fmt.Errorf("unsupported version %d", version)
	}

	pinCount, err := r.readI32()
	if err != nil {
		return nil, err
	}
	if pinCount < 0 {
		return nil, fmt.Errorf("invalid negative pin count %d", pinCount)
	}
	if int(pinCount) > r.remaining()/minPinBytesForVersion(version) {
		return nil, fmt.Errorf("invalid pin count %d at offset %d; only %d bytes remain", pinCount, r.pos-4, r.remaining())
	}
	pins := make([]Pin, 0, pinCount)
	for i := int32(0); i < pinCount; i++ {
		name, err := r.readString()
		if err != nil {
			return nil, err
		}
		var decoded DecodedName
		if formatForVersion(version) == "one_map_to_rule_them_all" {
			decoded = decodePinName(name)
		}
		pos, err := r.readVector3()
		if err != nil {
			return nil, err
		}
		pinType, err := r.readI32()
		if err != nil {
			return nil, err
		}
		checked, err := r.readBool()
		if err != nil {
			return nil, err
		}
		var owner string
		if pinHasOwner(version) {
			owner, err = r.readString()
			if err != nil {
				return nil, err
			}
		}
		pins = append(pins, Pin{
			Name: name, DecodedName: decoded.Decoded, DecodedAbbr: decoded.Abbr,
			DecodedCnt: decoded.Count, Pos: pos, Type: pinType, Checked: checked, OwnerID: owner,
		})
	}

	estimatedPayloadBytes := r.pos - start - headerBytes - fixedMapBytes
	if estimatedPayloadBytes < 0 {
		estimatedPayloadBytes = 0
	}
	return &DecodedFile{
		File: filepath.Clean(path), FileSize: len(data),
		Format: formatForVersion(version), MapEncoding: mapEncodingForVersion(version),
		Version: version, MapSize: size, Cells: mapCells, PackedMapBytes: packed,
		FixedMapBytes: fixedMapBytes, EstimatedPayloadBytes: estimatedPayloadBytes,
		ExploredCount: exploredCount, UnexploredCount: mapCells - exploredCount,
		ExploredPercent: float64(exploredCount) * 100 / mapCells,
		PinCount:        pinCount, Pins: pins, fullPins: pins,
	}, nil
}

func summarizePins(pins []Pin) []PinSummary {
	byKey := make(map[string]PinSummary)
	for _, pin := range pins {
		key, displayName, source := summaryKey(pin)
		s := byKey[key]
		s.Key = key
		s.DisplayName = displayName
		s.Source = source
		s.PinCount++
		if pin.DecodedCnt != nil {
			s.ImpliedObjectCount += *pin.DecodedCnt
			s.BatchPins++
		}
		if pin.Checked {
			s.Checked++
		} else {
			s.Unchecked++
		}
		byKey[key] = s
	}

	out := make([]PinSummary, 0, len(byKey))
	for _, summary := range byKey {
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].DisplayName != out[j].DisplayName {
			return out[i].DisplayName < out[j].DisplayName
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func summaryKey(pin Pin) (key, displayName, source string) {
	if pin.DecodedName != "" {
		return "decoded:" + pin.DecodedName, pin.DecodedName, "decoded"
	}
	if pin.Name != "" {
		return "raw:" + pin.Name, pin.Name, "raw"
	}
	return "unknown:", "<unknown>", "unknown"
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var jsonPath string
	var showPins, showSummary bool
	flags := flag.NewFlagSet("onemap", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&jsonPath, "json", "", "write decoded metadata/pins JSON")
	flags.BoolVar(&showPins, "pins", false, "print full pin list to stdout")
	flags.BoolVar(&showSummary, "summary", false, "include pin summary grouped by source and display name")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: onemap [--json path] [--pins] [--summary] file")
		return 2
	}

	decoded, err := readExploredFile(flags.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if showSummary {
		decoded.PinSummary = summarizePins(decoded.fullPins)
	}
	if !showPins {
		decoded.Pins = nil
		decoded.PinsOmitted = true
	}

	if jsonPath != "" {
		clone := *decoded
		clone.PinsOmitted = false
		clone.Pins = decoded.fullPins
		data, err := marshalStableJSON(&clone)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	data, err := marshalStableJSON(decoded)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	stdout.Write(data)
	stdout.Write([]byte("\n"))
	return 0
}

func marshalStableJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
