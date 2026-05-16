package app

import "testing"

func TestDesktopHumanAuthDisabled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false},
		{name: "disabled", value: "0", want: false},
		{name: "enabled numeric", value: "1", want: true},
		{name: "enabled true", value: "true", want: true},
		{name: "enabled mixed case", value: " YeS ", want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := desktopHumanAuthDisabled(func(key string) string {
				if key != desktopDisableAuthEnv {
					t.Fatalf("unexpected env key %q", key)
				}
				return tc.value
			})

			if got != tc.want {
				t.Fatalf("desktopHumanAuthDisabled(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}
