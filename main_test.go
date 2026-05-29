package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

const (
	currentFixture = "fixtures/Dedicated.one_map_to_rule_them_all.explored"
	oldFixture     = "fixtures/Dedicated.one_map_to_rule_them_all.explored.old"
	serverFixture  = "fixtures/Dedicated.mod.serversidemap.explored"
	emptyFixture   = "fixtures/Dedicated.mod.serversidemap.explored.empty"
)

func TestDedicatedExploredFixture(t *testing.T) {
	decoded, err := readExploredFile(currentFixture)
	if err != nil {
		t.Fatal(err)
	}
	assertFixture(t, decoded, "one_map_to_rule_them_all", "packed_bits", 2, 533596, 120379, 325, 9300)
}

func TestDedicatedOldExploredFixture(t *testing.T) {
	decoded, err := readExploredFile(oldFixture)
	if err != nil {
		t.Fatal(err)
	}
	assertFixture(t, decoded, "one_map_to_rule_them_all", "packed_bits", 2, 1645822, 119706, 39255, 1121526)
}

func TestDedicatedServerSideMapExploredFixture(t *testing.T) {
	decoded, err := readExploredFile(serverFixture)
	if err != nil {
		t.Fatal(err)
	}
	assertFixture(t, decoded, "serversidemap", "bool_bytes", 3, 4194613, 145930, 11, 301)
	for _, pin := range decoded.fullPins {
		if pin.OwnerID != "" {
			t.Fatalf("ServerSideMap pin owner = %q, want empty", pin.OwnerID)
		}
	}
}

func TestDedicatedEmptyServerSideMapExploredFixture(t *testing.T) {
	decoded, err := readExploredFile(emptyFixture)
	if err != nil {
		t.Fatal(err)
	}
	assertFixture(t, decoded, "serversidemap", "bool_bytes", 3, 4194316, 162, 0, 4)
	if decoded.UnexploredCount != mapCells-162 {
		t.Fatalf("unexplored count = %d, want %d", decoded.UnexploredCount, mapCells-162)
	}
	if len(decoded.fullPins) != 0 {
		t.Fatalf("full pin count = %d, want 0", len(decoded.fullPins))
	}
	if got := summarizePins(decoded.fullPins); len(got) != 0 {
		t.Fatalf("summary entry count = %d, want 0", len(got))
	}
}

func TestEmptyFixtureBaseOutputOmitsPins(t *testing.T) {
	out := runJSON(t, emptyFixture)
	assertEmptyFixtureMetadata(t, out)
	assertJSONBool(t, out, "pins_omitted", true)
	assertJSONMissing(t, out, "pins")
	assertJSONMissing(t, out, "pin_summary")
}

func TestEmptyFixturePinsOutputHasNoOmittedMarker(t *testing.T) {
	out := runJSON(t, "--pins", emptyFixture)
	assertEmptyFixtureMetadata(t, out)
	assertJSONMissing(t, out, "pins_omitted")
	assertJSONMissing(t, out, "pins")
	assertJSONMissing(t, out, "pin_summary")
}

func TestEmptyFixtureSummaryOutputOmitsEmptySummary(t *testing.T) {
	out := runJSON(t, "--summary", emptyFixture)
	assertEmptyFixtureMetadata(t, out)
	assertJSONBool(t, out, "pins_omitted", true)
	assertJSONMissing(t, out, "pins")
	assertJSONMissing(t, out, "pin_summary")
}

func TestServerSideMapBaseOutputOmitsPins(t *testing.T) {
	out := runJSON(t, serverFixture)
	assertServerFixtureMetadata(t, out)
	assertJSONBool(t, out, "pins_omitted", true)
	assertJSONMissing(t, out, "pins")
	assertJSONMissing(t, out, "pin_summary")
}

func TestServerSideMapPinsOutputIncludesRepresentativePins(t *testing.T) {
	out := runJSON(t, "--pins", serverFixture)
	assertServerFixtureMetadata(t, out)
	assertJSONMissing(t, out, "pins_omitted")
	assertJSONMissing(t, out, "pin_summary")

	pins := assertJSONArray(t, out, "pins", 11)
	anything := findJSONObject(t, pins, "name", "ANYTHING")
	assertJSONMissing(t, anything, "decoded_name")
	assertJSONMissing(t, anything, "decoded_abbr")
	assertJSONMissing(t, anything, "decoded_count")
	assertJSONNumber(t, anything, "type", 0)
	assertJSONBool(t, anything, "checked", false)
	assertJSONMissing(t, anything, "owner_id")
	assertJSONVector(t, anything, "pos", 801.8745, 0, -784.032)

	mapDay := findJSONObject(t, pins, "name", "$hud_mapday 496")
	assertJSONMissing(t, mapDay, "decoded_name")
	assertJSONMissing(t, mapDay, "decoded_abbr")
	assertJSONMissing(t, mapDay, "decoded_count")
	assertJSONNumber(t, mapDay, "type", 4)
	assertJSONBool(t, mapDay, "checked", false)
	assertJSONMissing(t, mapDay, "owner_id")
	assertJSONVector(t, mapDay, "pos", 1268.3069, 39.010666, 4389.2583)

	village := findJSONObject(t, pins, "name", "fuling village")
	assertJSONMissing(t, village, "decoded_name")
	assertJSONMissing(t, village, "decoded_abbr")
	assertJSONMissing(t, village, "decoded_count")
	assertJSONNumber(t, village, "type", 0)
	assertJSONMissing(t, village, "owner_id")
}

func TestServerSideMapSummaryOutputIncludesRawGroups(t *testing.T) {
	out := runJSON(t, "--summary", serverFixture)
	assertServerFixtureMetadata(t, out)
	assertJSONBool(t, out, "pins_omitted", true)
	assertJSONMissing(t, out, "pins")

	summary := assertJSONArray(t, out, "pin_summary", 11)
	assertJSONSummary(t, findJSONObject(t, summary, "display_name", "$enemy_dragon"), "$enemy_dragon", "raw", 1, 0, 0, 0, 1)
	assertJSONSummary(t, findJSONObject(t, summary, "display_name", "$hud_mapday 496"), "$hud_mapday 496", "raw", 1, 0, 0, 0, 1)
	assertJSONSummary(t, findJSONObject(t, summary, "display_name", "$hud_mapday 497"), "$hud_mapday 497", "raw", 1, 0, 0, 0, 1)
	assertJSONSummary(t, findJSONObject(t, summary, "display_name", "fuling village"), "fuling village", "raw", 1, 0, 0, 0, 1)
}

func TestServerSideMapFixtureSummary(t *testing.T) {
	decoded, err := readExploredFile(serverFixture)
	if err != nil {
		t.Fatal(err)
	}

	summary := summaryByKey(summarizePins(decoded.fullPins))
	assertSummaryTotal(t, summary, int(decoded.PinCount))
	if len(summary) != 11 {
		t.Fatalf("summary entry count = %d, want 11", len(summary))
	}
	assertSummaryEntry(t, summary, "raw:$enemy_dragon", PinSummary{
		Key:         "raw:$enemy_dragon",
		DisplayName: "$enemy_dragon",
		Source:      "raw",
		PinCount:    1,
		Unchecked:   1,
	})
	assertSummaryEntry(t, summary, "raw:$hud_mapday 496", PinSummary{
		Key:         "raw:$hud_mapday 496",
		DisplayName: "$hud_mapday 496",
		Source:      "raw",
		PinCount:    1,
		Unchecked:   1,
	})
	assertSummaryEntry(t, summary, "raw:$hud_mapday 497", PinSummary{
		Key:         "raw:$hud_mapday 497",
		DisplayName: "$hud_mapday 497",
		Source:      "raw",
		PinCount:    1,
		Unchecked:   1,
	})
	assertSummaryEntry(t, summary, "raw:fuling village", PinSummary{
		Key:         "raw:fuling village",
		DisplayName: "fuling village",
		Source:      "raw",
		PinCount:    1,
		Unchecked:   1,
	})
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

	serverData := make([]byte, headerBytes+mapCells+4)
	binary.LittleEndian.PutUint32(serverData[0:], uint32(3))
	binary.LittleEndian.PutUint32(serverData[4:], uint32(mapSize))
	if !looksLikePayloadAt(serverData, 0) {
		t.Fatal("valid zero-pin ServerSideMap payload did not look valid")
	}
}

func assertFixture(t *testing.T, decoded *DecodedFile, format, mapEncoding string, version int32, fileSize, exploredCount int, pinCount int32, estimatedPayloadBytes int) {
	t.Helper()
	if decoded.Format != format {
		t.Fatalf("format = %q, want %q", decoded.Format, format)
	}
	if decoded.MapEncoding != mapEncoding {
		t.Fatalf("map encoding = %q, want %q", decoded.MapEncoding, mapEncoding)
	}
	if decoded.Version != version {
		t.Fatalf("version = %d, want %d", decoded.Version, version)
	}
	if decoded.MapSize != mapSize {
		t.Fatalf("map size = %d, want %d", decoded.MapSize, mapSize)
	}
	if decoded.FileSize != fileSize {
		t.Fatalf("file size = %d, want %d", decoded.FileSize, fileSize)
	}
	if mapEncoding == "packed_bits" {
		if decoded.PackedMapBytes == nil || *decoded.PackedMapBytes != packedBytes {
			t.Fatalf("packed map bytes = %v, want %d", decoded.PackedMapBytes, packedBytes)
		}
	} else if decoded.PackedMapBytes != nil {
		t.Fatalf("packed map bytes = %v, want nil", *decoded.PackedMapBytes)
	}
	wantFixedMapBytes := packedBytes
	if mapEncoding == "bool_bytes" {
		wantFixedMapBytes = mapCells
	}
	if decoded.FixedMapBytes != wantFixedMapBytes {
		t.Fatalf("fixed map bytes = %d, want %d", decoded.FixedMapBytes, wantFixedMapBytes)
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

func runJSON(t *testing.T, args ...string) map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("run(%v) exit code = %d, stderr = %q", args, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run(%v) stderr = %q, want empty", args, stderr.String())
	}
	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	return out
}

func assertEmptyFixtureMetadata(t *testing.T, out map[string]any) {
	t.Helper()
	assertJSONString(t, out, "file", emptyFixture)
	assertJSONNumber(t, out, "file_size", 4194316)
	assertJSONString(t, out, "format", "serversidemap")
	assertJSONString(t, out, "map_encoding", "bool_bytes")
	assertJSONNumber(t, out, "version", 3)
	assertJSONNumber(t, out, "map_size", mapSize)
	assertJSONNumber(t, out, "cells", mapCells)
	assertJSONNull(t, out, "packed_map_bytes")
	assertJSONNumber(t, out, "fixed_map_bytes", mapCells)
	assertJSONNumber(t, out, "estimated_payload_bytes", 4)
	assertJSONNumber(t, out, "explored_count", 162)
	assertJSONNumber(t, out, "unexplored_count", mapCells-162)
	assertJSONFloat(t, out, "explored_percent", float64(162)*100/mapCells)
	assertJSONNumber(t, out, "pin_count", 0)
}

func assertServerFixtureMetadata(t *testing.T, out map[string]any) {
	t.Helper()
	assertJSONString(t, out, "file", serverFixture)
	assertJSONNumber(t, out, "file_size", 4194613)
	assertJSONString(t, out, "format", "serversidemap")
	assertJSONString(t, out, "map_encoding", "bool_bytes")
	assertJSONNumber(t, out, "version", 3)
	assertJSONNumber(t, out, "map_size", mapSize)
	assertJSONNumber(t, out, "cells", mapCells)
	assertJSONNull(t, out, "packed_map_bytes")
	assertJSONNumber(t, out, "fixed_map_bytes", mapCells)
	assertJSONNumber(t, out, "estimated_payload_bytes", 301)
	assertJSONNumber(t, out, "explored_count", 145930)
	assertJSONNumber(t, out, "unexplored_count", mapCells-145930)
	assertJSONFloat(t, out, "explored_percent", float64(145930)*100/mapCells)
	assertJSONNumber(t, out, "pin_count", 11)
}

func assertJSONArray(t *testing.T, out map[string]any, key string, wantLen int) []any {
	t.Helper()
	got, ok := out[key].([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", key, out[key])
	}
	if len(got) != wantLen {
		t.Fatalf("%s length = %d, want %d", key, len(got), wantLen)
	}
	return got
}

func findJSONObject(t *testing.T, entries []any, key, want string) map[string]any {
	t.Helper()
	for _, entry := range entries {
		obj, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("entry = %#v, want object", entry)
		}
		if got, ok := obj[key].(string); ok && got == want {
			return obj
		}
	}
	t.Fatalf("array missing object with %s = %q", key, want)
	return nil
}

func assertJSONVector(t *testing.T, out map[string]any, key string, wantX, wantY, wantZ float64) {
	t.Helper()
	pos, ok := out[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", key, out[key])
	}
	assertJSONFloatClose(t, pos, "x", wantX)
	assertJSONFloatClose(t, pos, "y", wantY)
	assertJSONFloatClose(t, pos, "z", wantZ)
}

func assertJSONSummary(t *testing.T, out map[string]any, displayName, source string, pinCount, impliedObjectCount, batchPins, checked, unchecked int) {
	t.Helper()
	assertJSONString(t, out, "display_name", displayName)
	assertJSONString(t, out, "source", source)
	assertJSONNumber(t, out, "pin_count", pinCount)
	assertJSONNumber(t, out, "implied_object_count", impliedObjectCount)
	assertJSONNumber(t, out, "batch_pins", batchPins)
	assertJSONNumber(t, out, "checked", checked)
	assertJSONNumber(t, out, "unchecked", unchecked)
}

func assertJSONString(t *testing.T, out map[string]any, key, want string) {
	t.Helper()
	got, ok := out[key].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", key, out[key], want)
	}
}

func assertJSONNumber(t *testing.T, out map[string]any, key string, want int) {
	t.Helper()
	got, ok := out[key].(float64)
	if !ok || got != float64(want) {
		t.Fatalf("%s = %#v, want %d", key, out[key], want)
	}
}

func assertJSONFloat(t *testing.T, out map[string]any, key string, want float64) {
	t.Helper()
	got, ok := out[key].(float64)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %g", key, out[key], want)
	}
}

func assertJSONFloatClose(t *testing.T, out map[string]any, key string, want float64) {
	t.Helper()
	got, ok := out[key].(float64)
	if !ok || math.Abs(got-want) > 0.00001 {
		t.Fatalf("%s = %#v, want %g", key, out[key], want)
	}
}

func assertJSONBool(t *testing.T, out map[string]any, key string, want bool) {
	t.Helper()
	got, ok := out[key].(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %t", key, out[key], want)
	}
}

func assertJSONNull(t *testing.T, out map[string]any, key string) {
	t.Helper()
	got, ok := out[key]
	if !ok || got != nil {
		t.Fatalf("%s = %#v, want null", key, got)
	}
}

func assertJSONMissing(t *testing.T, out map[string]any, key string) {
	t.Helper()
	if _, ok := out[key]; ok {
		t.Fatalf("%s present, want omitted", key)
	}
}
