package audio

import "testing"

func TestClampPercent(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{-1, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{101, 100},
	}
	for _, c := range cases {
		if got := ClampPercent(c.in); got != c.want {
			t.Fatalf("ClampPercent(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestScalarRoundTrip(t *testing.T) {
	for _, p := range []int{0, 1, 25, 50, 74, 99, 100} {
		if got := ScalarToPercent(PercentToScalar(p)); got != p {
			t.Fatalf("round-trip %d → %d", p, got)
		}
	}
}

func TestScalarToPercentEdges(t *testing.T) {
	if ScalarToPercent(-0.5) != 0 {
		t.Fatal("negative scalar")
	}
	if ScalarToPercent(1.5) != 100 {
		t.Fatal("over-one scalar")
	}
	if ScalarToPercent(0.005) != 1 {
		t.Fatal("round half up from 0.5%")
	}
}
