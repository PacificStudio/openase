package workflow

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseStatusBindingSet(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	set, err := ParseStatusBindingSet("pickup_status_ids", []uuid.UUID{id1, id2, id1})
	if err != nil {
		t.Fatalf("ParseStatusBindingSet() error = %v", err)
	}
	if set.Len() != 2 {
		t.Fatalf("Len() = %d", set.Len())
	}
	if !set.Contains(id1) || !set.Contains(id2) {
		t.Fatalf("Contains() missing expected ids: %#v", set.IDs())
	}
	if set.Contains(uuid.New()) {
		t.Fatal("Contains() unexpectedly reported unknown id")
	}

	ids := set.IDs()
	ids[0] = uuid.Nil
	if set.IDs()[0] == uuid.Nil {
		t.Fatal("IDs() should return a copy")
	}

	if _, ok := set.Single(); ok {
		t.Fatal("Single() should be false for multi-item sets")
	}
}

func TestParseStatusBindingSetErrors(t *testing.T) {
	id := uuid.New()

	if _, err := ParseStatusBindingSet("finish_status_ids", nil); err == nil {
		t.Fatal("expected empty status binding set to fail")
	}
	if _, err := ParseStatusBindingSet("finish_status_ids", []uuid.UUID{id, uuid.Nil}); err == nil {
		t.Fatal("expected zero UUID to fail")
	}
}

func TestMustStatusBindingSet(t *testing.T) {
	id := uuid.New()
	set := MustStatusBindingSet(id)

	got, ok := set.Single()
	if !ok {
		t.Fatal("Single() should be true for single-item sets")
	}
	if got != id {
		t.Fatalf("Single() = %s, want %s", got, id)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected MustStatusBindingSet() to panic for invalid input")
		}
	}()
	_ = MustStatusBindingSet(uuid.Nil)
}

func TestStatusBindingSetOverlaps(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	left := MustStatusBindingSet(id1, id2)
	right := MustStatusBindingSet(id2, id3)
	disjoint := MustStatusBindingSet(id3)

	if !left.Overlaps(right) {
		t.Fatal("Overlaps() should report shared status bindings")
	}
	if left.Overlaps(disjoint) {
		t.Fatal("Overlaps() should be false for disjoint status bindings")
	}
}
