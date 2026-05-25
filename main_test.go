package main

import (
	"encoding/binary"
	"strings"
	"testing"
)

const (
	currentFixture = "fixtures/Dedicated.one_map_to_rule_them_all.explored"
	oldFixture     = "fixtures/Dedicated.one_map_to_rule_them_all.explored.old"
)

func TestDedicatedExploredFixture(t *testing.T) {
	decoded, err := readExploredFile(currentFixture)
	if err != nil {
		t.Fatal(err)
	}
	assertFixture(t, decoded, 533596, 120379, 325, 9300)
}

func TestDedicatedOldExploredFixture(t *testing.T) {
	decoded, err := readExploredFile(oldFixture)
	if err != nil {
		t.Fatal(err)
	}
	assertFixture(t, decoded, 1645822, 119706, 39255, 1121526)
}

func TestSummarizeByDecodedName(t *testing.T) {
	count2 := 2
	count3 := 3
	count5 := 5
	pins := []Pin{
		{Name: "Tu", DecodedName: "Turnip", DecodedAbbr: "Tu", Checked: true},
		{Name: "Tu 2", DecodedName: "Turnip", DecodedAbbr: "Tu", DecodedCnt: &count2},
		{Name: "Tu 3", DecodedName: "Turnip", DecodedAbbr: "Tu", DecodedCnt: &count3},
		{Name: "Cu 5", DecodedName: "Copper", DecodedAbbr: "Cu", DecodedCnt: &count5, Checked: true},
		{Name: "home", Checked: true},
		{Name: ""},
	}

	got := summaryByKey(summarizePins(pins))
	want := map[string]PinSummary{
		"decoded:Turnip": {
			Key:                "decoded:Turnip",
			DisplayName:        "Turnip",
			Source:             "decoded",
			PinCount:           3,
			ImpliedObjectCount: 5,
			BatchPins:          2,
			Checked:            1,
			Unchecked:          2,
		},
		"decoded:Copper": {
			Key:                "decoded:Copper",
			DisplayName:        "Copper",
			Source:             "decoded",
			PinCount:           1,
			ImpliedObjectCount: 5,
			BatchPins:          1,
			Checked:            1,
			Unchecked:          0,
		},
		"raw:home": {
			Key:                "raw:home",
			DisplayName:        "home",
			Source:             "raw",
			PinCount:           1,
			ImpliedObjectCount: 0,
			BatchPins:          0,
			Checked:            1,
			Unchecked:          0,
		},
		"unknown:": {
			Key:                "unknown:",
			DisplayName:        "<unknown>",
			Source:             "unknown",
			PinCount:           1,
			ImpliedObjectCount: 0,
			BatchPins:          0,
			Checked:            0,
			Unchecked:          1,
		},
	}

	if len(got) != len(want) {
		t.Fatalf("summary entry count = %d, want %d", len(got), len(want))
	}
	for key, wantEntry := range want {
		if gotEntry, ok := got[key]; !ok || gotEntry != wantEntry {
			t.Fatalf("summary[%q] = %#v, want %#v", key, gotEntry, wantEntry)
		}
	}
}

func TestCurrentFixtureSummary(t *testing.T) {
	decoded, err := readExploredFile(currentFixture)
	if err != nil {
		t.Fatal(err)
	}

	summary := summaryByKey(summarizePins(decoded.fullPins))
	assertSummaryTotal(t, summary, int(decoded.PinCount))
	assertSummaryEntry(t, summary, "decoded:Turnip", PinSummary{
		Key:                "decoded:Turnip",
		DisplayName:        "Turnip",
		Source:             "decoded",
		PinCount:           302,
		ImpliedObjectCount: 41412,
		BatchPins:          302,
		Checked:            115,
		Unchecked:          187,
	})
	assertSummaryEntry(t, summary, "decoded:Cloudberry", PinSummary{
		Key:                "decoded:Cloudberry",
		DisplayName:        "Cloudberry",
		Source:             "decoded",
		PinCount:           5,
		ImpliedObjectCount: 120,
		BatchPins:          5,
		Checked:            2,
		Unchecked:          3,
	})
	assertSummaryEntry(t, summary, "raw:test", PinSummary{
		Key:                "raw:test",
		DisplayName:        "test",
		Source:             "raw",
		PinCount:           1,
		ImpliedObjectCount: 0,
		BatchPins:          0,
		Checked:            0,
		Unchecked:          1,
	})
}

func TestOldFixtureSummary(t *testing.T) {
	decoded, err := readExploredFile(oldFixture)
	if err != nil {
		t.Fatal(err)
	}

	summary := summaryByKey(summarizePins(decoded.fullPins))
	assertSummaryTotal(t, summary, int(decoded.PinCount))
	assertSummaryEntry(t, summary, "decoded:Turnip", PinSummary{
		Key:                "decoded:Turnip",
		DisplayName:        "Turnip",
		Source:             "decoded",
		PinCount:           22393,
		ImpliedObjectCount: 6068318,
		BatchPins:          22341,
		Checked:            4495,
		Unchecked:          17898,
	})
	assertSummaryEntry(t, summary, "decoded:Onion", PinSummary{
		Key:                "decoded:Onion",
		DisplayName:        "Onion",
		Source:             "decoded",
		PinCount:           6697,
		ImpliedObjectCount: 1772124,
		BatchPins:          6693,
		Checked:            2126,
		Unchecked:          4571,
	})
	assertSummaryEntry(t, summary, "unknown:", PinSummary{
		Key:                "unknown:",
		DisplayName:        "<unknown>",
		Source:             "unknown",
		PinCount:           47,
		ImpliedObjectCount: 0,
		BatchPins:          0,
		Checked:            1,
		Unchecked:          46,
	})
}

func TestMarshalStableJSONDoesNotEscapeHTML(t *testing.T) {
	data, err := marshalStableJSON(PinSummary{
		Key:         "unknown:",
		DisplayName: "<unknown>",
		Source:      "unknown",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Contains(got, `\u003c`) || strings.Contains(got, `\u003e`) {
		t.Fatalf("JSON escaped angle brackets: %s", got)
	}
	if !strings.Contains(got, `"<unknown>"`) {
		t.Fatalf("JSON does not contain readable unknown marker: %s", got)
	}
}

func TestLooksLikePayloadAtValidatesReadableMapAndPinCount(t *testing.T) {
	short := make([]byte, headerBytes)
	binary.LittleEndian.PutUint32(short[0:], uint32(2))
	binary.LittleEndian.PutUint32(short[4:], uint32(mapSize))
	if looksLikePayloadAt(short, 0) {
		t.Fatal("short header-only payload looked valid")
	}

	data := make([]byte, headerBytes+packedBytes+4)
	binary.LittleEndian.PutUint32(data[0:], uint32(2))
	binary.LittleEndian.PutUint32(data[4:], uint32(mapSize))
	if !looksLikePayloadAt(data, 0) {
		t.Fatal("valid zero-pin payload did not look valid")
	}

	binary.LittleEndian.PutUint32(data[headerBytes+packedBytes:], uint32(1))
	if looksLikePayloadAt(data, 0) {
		t.Fatal("payload with unreadable pin count looked valid")
	}
}

func assertFixture(t *testing.T, decoded *DecodedFile, fileSize, exploredCount int, pinCount int32, estimatedPayloadBytes int) {
	t.Helper()
	if decoded.HeaderOffset != 0 {
		t.Fatalf("header offset = %d, want 0", decoded.HeaderOffset)
	}
	if decoded.Version != 2 {
		t.Fatalf("version = %d, want 2", decoded.Version)
	}
	if decoded.MapSize != mapSize {
		t.Fatalf("map size = %d, want %d", decoded.MapSize, mapSize)
	}
	if decoded.FileSize != fileSize {
		t.Fatalf("file size = %d, want %d", decoded.FileSize, fileSize)
	}
	if decoded.PackedMapBytes == nil || *decoded.PackedMapBytes != packedBytes {
		t.Fatalf("packed map bytes = %v, want %d", decoded.PackedMapBytes, packedBytes)
	}
	if decoded.FixedMapBytes != packedBytes {
		t.Fatalf("fixed map bytes = %d, want %d", decoded.FixedMapBytes, packedBytes)
	}
	if decoded.EstimatedPayloadBytes != estimatedPayloadBytes {
		t.Fatalf("estimated payload bytes = %d, want %d", decoded.EstimatedPayloadBytes, estimatedPayloadBytes)
	}
	if decoded.EstimatedPayloadBytes == decoded.FileSize {
		t.Fatalf("estimated payload bytes should exclude fixed packed map bytes")
	}
	if decoded.ExploredCount != exploredCount {
		t.Fatalf("explored count = %d, want %d", decoded.ExploredCount, exploredCount)
	}
	if decoded.PinCount != pinCount {
		t.Fatalf("pin count = %d, want %d", decoded.PinCount, pinCount)
	}
	if decoded.BytesConsumed != decoded.FileSize {
		t.Fatalf("bytes consumed = %d, want %d", decoded.BytesConsumed, decoded.FileSize)
	}
	if decoded.TrailingBytes != 0 {
		t.Fatalf("trailing bytes = %d, want 0", decoded.TrailingBytes)
	}
}

func assertSummaryEntry(t *testing.T, summary map[string]PinSummary, key string, want PinSummary) {
	t.Helper()
	got, ok := summary[key]
	if !ok {
		t.Fatalf("summary missing entry %q", key)
	}
	if got != want {
		t.Fatalf("summary[%q] = %#v, want %#v", key, got, want)
	}
	if got.PinCount != got.Checked+got.Unchecked {
		t.Fatalf("summary[%q] pin count = %d, checked + unchecked = %d", key, got.PinCount, got.Checked+got.Unchecked)
	}
}

func assertSummaryTotal(t *testing.T, summary map[string]PinSummary, want int) {
	t.Helper()
	var got int
	for _, entry := range summary {
		got += entry.PinCount
	}
	if got != want {
		t.Fatalf("summary pin count total = %d, want %d", got, want)
	}
}

func summaryByKey(entries []PinSummary) map[string]PinSummary {
	out := make(map[string]PinSummary, len(entries))
	for _, entry := range entries {
		out[entry.Key] = entry
	}
	return out
}
