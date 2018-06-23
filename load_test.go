package dauntless

import "testing"

func TestOverStrike(t *testing.T) {
	for i, test := range []struct {
		in, out string
	}{
		{"", ""},
		{"abc", "abc"},
		{"abc\n", "abc\n"},
		{"ab\bbc", "abc"},
		{"a_\bbc", "abc"},
		{"a_\bb_b\bbc", "ab_bc"},
	} {
		got := string(eliminateOverStrike([]byte(test.in)))
		if got != test.out {
			t.Errorf("%d: in=%q out=%q got=%q", i, test.in, test.out, got)
		}
	}
}
