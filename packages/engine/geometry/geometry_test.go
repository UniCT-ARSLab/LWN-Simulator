package geometry

import "testing"

func TestDistanceMetersUsesGreatCircleDistance(t *testing.T) {
	if got := DistanceMeters(37.5, 15.1, 37.5, 15.1); got != 0 {
		t.Fatalf("expected identical coordinates to have zero distance, got %f", got)
	}

	got := DistanceMeters(37.5, 15.1, 37.51, 15.1)
	if got < 1_000 || got > 1_200 {
		t.Fatalf("expected approximately 1.1km distance, got %f", got)
	}
}
