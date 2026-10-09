package pose

import (
	"reflect"
	"testing"
	"time"
)

// A projection reads the specs once; what it reads equals what each call read
// before, and no caller can change another's copy (spec
// pose-attention-within-a-second).
func TestAttentionSnapshotListsWhatTheDiskLists(t *testing.T) {
	_, store := reviewBundleFixture(t)
	direct, err := store.ListSpecs("", "")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := store.withSpecSnapshot()
	first, err := snapshot.ListSpecs("", "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(direct, first) {
		t.Fatalf("the snapshot lists differently:\n%+v\n%+v", direct, first)
	}
	if len(first) > 0 {
		first[0].Components = append(first[0].Components, "mutated")
		again, _ := snapshot.ListSpecs("", "")
		if !reflect.DeepEqual(direct, again) {
			t.Fatal("a caller's edit reached the snapshot")
		}
		one, err := snapshot.GetSpec(direct[0].Slug)
		if err != nil || one.Slug != direct[0].Slug || one.Body == "" {
			t.Fatalf("GetSpec through the snapshot = %+v, %v", one, err)
		}
	}
	filtered, _ := snapshot.ListSpecs("in-progress", "")
	want, _ := store.ListSpecs("in-progress", "")
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("a filtered snapshot read differs:\n%+v\n%+v", filtered, want)
	}
}

func TestAttentionSnapshotListsTheSameBundles(t *testing.T) {
	_, store := reviewBundleFixture(t)
	if _, err := store.SealReviewBundle("spec:backend", time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	direct, err := store.ListReviewBundles("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	through, err := store.withSpecSnapshot().ListReviewBundles("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) == 0 || !reflect.DeepEqual(direct, through) {
		t.Fatalf("the scoped bundle listing differs: %d vs %d", len(direct), len(through))
	}
}
