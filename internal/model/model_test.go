package model

import "testing"

func TestNames(t *testing.T) {
	for _, s := range []string{".", "..", "C:", "CON", "con.txt", "NUL.x", "LPT9", "COM².doc", "a\\b", "a/b", "a\x00", "end.", "end ", "AUX", "CONOUT$"} {
		if ValidateName(s) == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, s := range []string{"Dockerfile", "LICENSE", "عربي", "a#b", "COM10", ".git", " leading", "a,b{}"} {
		if e := ValidateName(s); e != nil {
			t.Error(e)
		}
	}
}
