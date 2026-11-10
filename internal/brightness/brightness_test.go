package brightness

import "testing"

func TestClampPercent(t *testing.T) {
	if ClampPercent(-5) != 0 || ClampPercent(150) != 100 || ClampPercent(42) != 42 {
		t.Fatal("clamp")
	}
}

func TestUnitRoundTrip(t *testing.T) {
	for _, p := range []int{0, 1, 25, 50, 74, 99, 100} {
		if got := UnitToPercent(PercentToUnit(p)); got != p {
			t.Fatalf("round-trip %d → %d", p, got)
		}
	}
}
