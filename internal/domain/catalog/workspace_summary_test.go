package catalog

import "testing"

func TestIsTerminalTicketStatusStage(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{name: "completed", input: "completed", expect: true},
		{name: "canceled with whitespace", input: " canceled ", expect: true},
		{name: "invalid stage", input: "done", expect: false},
		{name: "active stage", input: "started", expect: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := IsTerminalTicketStatusStage(testCase.input); got != testCase.expect {
				t.Fatalf("IsTerminalTicketStatusStage(%q) = %t, want %t", testCase.input, got, testCase.expect)
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
