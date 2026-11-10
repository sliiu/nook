package bands

import "testing"

func TestOf(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, 0}, {24, 0}, {25, 1}, {49, 1}, {50, 2}, {74, 2}, {75, 3}, {100, 3},
		{-3, 0}, {140, 3},
	}
	for _, c := range cases {
		if got := Of(c.in); got != c.want {
			t.Fatalf("Of(%d)=%d want %d", c.in, got, c.want)
		}
	}
}

func TestID(t *testing.T) {
	if got := ID(10, 80); got != "V0B3" {
		t.Fatalf("got %s", got)
	}
	if got := ID(60, 30); got != "V2B1" {
		t.Fatalf("got %s", got)
	}
}
