package main

import "testing"

func TestCourseDeliveryNeedsEducatorFollowUp(t *testing.T) {
	tests := []struct {
		name      string
		delivery  CourseDelivery
		threshold float64
		attention bool
	}{
		{"late submissions", CourseDelivery{Due: 38, Submitted: 31}, 0.9, true},
		{"deadline met", CourseDelivery{Due: 38, Submitted: 36}, 0.9, false},
		{"no deadline", CourseDelivery{Due: 0, Submitted: 0}, 0.9, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.delivery.NeedsAttention(test.threshold); got != test.attention {
				t.Fatalf("NeedsAttention() = %v, want %v", got, test.attention)
			}
		})
	}
}
