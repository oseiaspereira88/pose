package pose

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func cursorFixture(n int) ([]string, []string) {
	items, keys := make([]string, n), make([]string, n)
	for i := range items {
		items[i] = fmt.Sprintf("item-%02d", i)
		keys[i] = items[i]
	}
	return items, keys
}

func mustKey(t *testing.T) []byte {
	t.Helper()
	key, err := NewCursorKey()
	if err != nil {
		t.Fatal(err)
	}
	return key
}

var cursorQuery = GenerationQuery{Tool: "pose_list_specs", Project: "proj|abc", Filters: "status=draft"}

func TestPageByGenerationWalksTheCollectionExactlyOnce(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(23)
	var seen []string
	cursor := ""
	for pages := 0; pages < 20; pages++ {
		page, next, meta, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, cursor, 5, now)
		if err != nil {
			t.Fatal(err)
		}
		if meta.Total != 23 || meta.Generation != "gen-1" || meta.Collection == "" || meta.Disclosure == "" {
			t.Fatalf("a page must disclose what it was read from: %+v", meta)
		}
		seen = append(seen, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	if strings.Join(seen, ",") != strings.Join(items, ",") {
		t.Fatalf("pages skipped or duplicated items: %v", seen)
	}
}

func TestPageByGenerationWithoutCursorOrLimitIsTheLegacyCall(t *testing.T) {
	items, keys := cursorFixture(7)
	page, next, _, err := PageByGeneration(mustKey(t), cursorQuery, "gen-1", items, keys, "", 0, time.Now())
	if err != nil || len(page) != 7 || next != "" {
		t.Fatalf("legacy call changed: %d items next=%q err=%v", len(page), next, err)
	}
}

func TestPageByGenerationClampsAndDisclosesTheLimit(t *testing.T) {
	items, keys := cursorFixture(GenerationPageMax + 40)
	page, next, meta, err := PageByGeneration(mustKey(t), cursorQuery, "gen-1", items, keys, "", GenerationPageMax+100, time.Now())
	if err != nil || len(page) != GenerationPageMax || next == "" || meta.LimitApplied != GenerationPageMax {
		t.Fatalf("limit was not bounded and disclosed: %d next=%q meta=%+v err=%v", len(page), next, meta, err)
	}
}

func firstCursor(t *testing.T, key []byte, items, keys []string, now time.Time) string {
	t.Helper()
	_, next, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, "", 5, now)
	if err != nil || next == "" {
		t.Fatalf("no first cursor: %v", err)
	}
	return next
}

func TestACursorIsRefusedWhenTheCollectionChanged(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(12)
	cursor := firstCursor(t, key, items, keys, now)

	mutations := map[string]func() ([]string, []string, string){
		"insertion": func() ([]string, []string, string) {
			i2 := append([]string{"item-aa"}, items...)
			return i2, i2, "gen-1"
		},
		"removal": func() ([]string, []string, string) { return items[1:], keys[1:], "gen-1" },
		"rename": func() ([]string, []string, string) {
			i2 := append([]string(nil), items...)
			i2[3] = "item-renamed"
			return i2, i2, "gen-1"
		},
		"reordering": func() ([]string, []string, string) {
			i2 := append([]string(nil), items...)
			i2[0], i2[1] = i2[1], i2[0]
			return i2, i2, "gen-1"
		},
		"new generation": func() ([]string, []string, string) { return items, keys, "gen-2" },
	}
	for name, mutate := range mutations {
		i2, k2, gen := mutate()
		_, _, _, err := PageByGeneration(key, cursorQuery, gen, i2, k2, cursor, 5, now)
		if CursorErrorCode(err) != CursorStale {
			t.Errorf("%s: want %s, got %v", name, CursorStale, err)
		}
	}
	if _, _, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, cursor, 5, now); err != nil {
		t.Fatalf("an unchanged collection must continue: %v", err)
	}
}

func TestACursorStaysWithItsQuery(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(12)
	cursor := firstCursor(t, key, items, keys, now)
	for name, q := range map[string]GenerationQuery{
		"tool":    {Tool: "pose_list_roadmaps", Project: cursorQuery.Project, Filters: cursorQuery.Filters},
		"project": {Tool: cursorQuery.Tool, Project: "proj|other", Filters: cursorQuery.Filters},
		"filters": {Tool: cursorQuery.Tool, Project: cursorQuery.Project, Filters: "status=done"},
	} {
		_, _, _, err := PageByGeneration(key, q, "gen-1", items, keys, cursor, 5, now)
		if CursorErrorCode(err) != CursorBinding {
			t.Errorf("%s: want %s, got %v", name, CursorBinding, err)
		}
	}
}

func TestHostileCursorsAreRefusedNotCoerced(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(12)
	cursor := firstCursor(t, key, items, keys, now)
	body, mac, _ := strings.Cut(cursor, ".")

	cases := map[string]struct{ token, code string }{
		"garbage":           {"not-a-cursor!!", CursorMalformed},
		"truncated":         {cursor[:len(cursor)/2], CursorMalformed},
		"oversized":         {strings.Repeat("a", GenerationCursorMaxBytes+1), CursorMalformed},
		"altered payload":   {body[:len(body)-2] + "AA." + mac, CursorSignature},
		"altered signature": {body + "." + mac[:len(mac)-2] + "AA", CursorSignature},
		"legacy ordinal":    {"eyJ2IjoxLCJhZnRlciI6NX0", CursorUnsupportedVersion},
	}
	for name, c := range cases {
		_, _, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, c.token, 5, now)
		if CursorErrorCode(err) == "" {
			t.Errorf("%s: accepted", name)
		} else if got := CursorErrorCode(err); got != c.code && !(name == "altered payload" && got == CursorMalformed) {
			t.Errorf("%s: want %s, got %s", name, c.code, got)
		}
	}
	// A cursor from another process (another key) is refused as a signature failure.
	if _, _, _, err := PageByGeneration(mustKey(t), cursorQuery, "gen-1", items, keys, cursor, 5, now); CursorErrorCode(err) != CursorSignature {
		t.Fatalf("another process's cursor: %v", err)
	}
}

func TestAnExpiredCursorIsRefused(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(12)
	cursor := firstCursor(t, key, items, keys, now)
	_, _, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, cursor, 5, now.Add(GenerationCursorTTL+time.Minute))
	if CursorErrorCode(err) != CursorExpired {
		t.Fatalf("want %s, got %v", CursorExpired, err)
	}
}

func TestAPositionOutsideTheCollectionIsRefused(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(12)
	digest := CollectionDigest("gen-1", keys)
	for name, p := range map[string]cursorPayload{
		"beyond the end": {V: GenerationCursorVersion, Tool: cursorQuery.Tool, Bind: queryBinding(cursorQuery), Collection: digest, After: 99, AfterKey: "item-98", Expires: now.Add(time.Minute).Unix()},
		"wrong key":      {V: GenerationCursorVersion, Tool: cursorQuery.Tool, Bind: queryBinding(cursorQuery), Collection: digest, After: 5, AfterKey: "item-77", Expires: now.Add(time.Minute).Unix()},
		"zero":           {V: GenerationCursorVersion, Tool: cursorQuery.Tool, Bind: queryBinding(cursorQuery), Collection: digest, After: 0, AfterKey: "", Expires: now.Add(time.Minute).Unix()},
	} {
		_, _, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, sealCursor(key, p), 5, now)
		if CursorErrorCode(err) != CursorPosition {
			t.Errorf("%s: want %s, got %v", name, CursorPosition, err)
		}
	}
	bad := cursorPayload{V: 9, Tool: cursorQuery.Tool, Bind: queryBinding(cursorQuery), Collection: digest, After: 5, AfterKey: "item-04", Expires: now.Add(time.Minute).Unix()}
	if _, _, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, sealCursor(key, bad), 5, now); CursorErrorCode(err) != CursorUnsupportedVersion {
		t.Fatalf("an unsupported version was accepted: %v", err)
	}
}

func TestAContinuationWithoutALimitIsStillABoundedPage(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(GenerationPageMax*2 + 200)
	_, next, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, "", 1, now)
	if err != nil || next == "" {
		t.Fatal(err)
	}
	page, _, _, err := PageByGeneration(key, cursorQuery, "gen-1", items, keys, next, 0, now)
	if err != nil || len(page) != GenerationPageMax {
		t.Fatalf("a continuation without limit returned %d items, want %d: %v", len(page), GenerationPageMax, err)
	}
}

func TestAnIssuedCursorStaysWithinTheSizeBoundForHugeFilters(t *testing.T) {
	key, now := mustKey(t), time.Now()
	items, keys := cursorFixture(12)
	q := GenerationQuery{Tool: "pose_list_specs", Project: "proj|abc", Filters: strings.Repeat("status=draft;", 400)}
	_, next, _, err := PageByGeneration(key, q, "gen-1", items, keys, "", 5, now)
	if err != nil || len(next) == 0 || len(next) > GenerationCursorMaxBytes {
		t.Fatalf("a cursor for a huge filter set is %d bytes: %v", len(next), err)
	}
	if _, _, _, err := PageByGeneration(key, q, "gen-1", items, keys, next, 5, now); err != nil {
		t.Fatalf("the issued cursor is refused on receipt: %v", err)
	}
}

func TestEveryPageStatesItsCost(t *testing.T) {
	items, keys := cursorFixture(3)
	_, _, meta, err := PageByGeneration(mustKey(t), cursorQuery, "gen-1", items, keys, "", 2, time.Now())
	if err != nil || !strings.Contains(meta.Cost, "full-scan") {
		t.Fatalf("a page must state its cost: %+v %v", meta, err)
	}
}
