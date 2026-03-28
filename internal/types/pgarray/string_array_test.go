package pgarray

import "testing"

func TestStringArrayValueAndScan(t *testing.T) {
	var nilArray StringArray
	value, err := nilArray.Value()
	if err != nil {
		t.Fatalf("Value(nil) error = %v", err)
	}
	if value != nil {
		t.Fatalf("Value(nil) = %v, want nil", value)
	}

	value, err = StringArray{"alpha", "beta"}.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if value != `{"alpha","beta"}` {
		t.Fatalf("Value() = %v, want PostgreSQL array literal", value)
	}

	var parsed StringArray
	if err := parsed.Scan(`{"first","second"}`); err != nil {
		t.Fatalf("Scan(string) error = %v", err)
	}
	if len(parsed) != 2 || parsed[0] != "first" || parsed[1] != "second" {
		t.Fatalf("Scan(string) = %+v", parsed)
	}

	if err := parsed.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error = %v", err)
	}
	if parsed != nil {
		t.Fatalf("Scan(nil) = %+v, want nil", parsed)
	}

	if err := parsed.Scan(123); err == nil {
		t.Fatal("Scan() expected invalid source type error")
	}
}
