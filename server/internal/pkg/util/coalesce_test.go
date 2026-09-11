package util

import "testing"

func TestCoalesce(t *testing.T) {
	var pf *float64
	if got := Coalesce(pf); got != 0 {
		t.Errorf("Coalesce(nil *float64) = %v, want 0", got)
	}
	v := 3.14
	if got := Coalesce(&v); got != 3.14 {
		t.Errorf("Coalesce(&3.14) = %v, want 3.14", got)
	}
	var ps *string
	if got := Coalesce(ps); got != "" {
		t.Errorf("Coalesce(nil *string) = %q, want empty", got)
	}
	s := "x"
	if got := Coalesce(&s); got != "x" {
		t.Errorf("Coalesce(&x) = %q, want x", got)
	}
}
