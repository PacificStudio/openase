package catalog

import "testing"

func TestIsTerminalTicketStatusName(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{name: "done", input: "Done", expect: true},
		{name: "cancelled with whitespace", input: " cancelled ", expect: true},
		{name: "american canceled", input: "CANCELED", expect: true},
		{name: "archived", input: "archived", expect: true},
		{name: "active status", input: "in_progress", expect: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := IsTerminalTicketStatusName(testCase.input); got != testCase.expect {
				t.Fatalf("IsTerminalTicketStatusName(%q) = %t, want %t", testCase.input, got, testCase.expect)
			}
		})
	}
}

func TestIsActiveProjectStatus(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{name: "active", input: "active", expect: true},
		{name: "blank defaults active", input: " ", expect: true},
		{name: "archived", input: "archived", expect: false},
		{name: "cancelled", input: " Cancelled ", expect: false},
		{name: "american canceled", input: "canceled", expect: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := IsActiveProjectStatus(testCase.input); got != testCase.expect {
				t.Fatalf("IsActiveProjectStatus(%q) = %t, want %t", testCase.input, got, testCase.expect)
			}
		})
	}
}
