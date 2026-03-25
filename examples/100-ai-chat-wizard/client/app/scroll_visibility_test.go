package app

import "testing"

func TestHasScrollSpaceBelow(t *testing.T) {
	tests := []struct {
		name         string
		scrollTop    float64
		scrollHeight float64
		clientHeight float64
		threshold    float64
		want         bool
	}{
		{
			name:         "shows when plenty of content remains below",
			scrollTop:    120,
			scrollHeight: 1200,
			clientHeight: 600,
			threshold:    80,
			want:         true,
		},
		{
			name:         "hides when within threshold of bottom",
			scrollTop:    525,
			scrollHeight: 1200,
			clientHeight: 600,
			threshold:    80,
			want:         false,
		},
		{
			name:         "hides when content fits without overflow",
			scrollTop:    0,
			scrollHeight: 480,
			clientHeight: 600,
			threshold:    80,
			want:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasScrollSpaceBelow(test.scrollTop, test.scrollHeight, test.clientHeight, test.threshold); got != test.want {
				t.Fatalf("hasScrollSpaceBelow(%v, %v, %v, %v) = %v, want %v", test.scrollTop, test.scrollHeight, test.clientHeight, test.threshold, got, test.want)
			}
		})
	}
}
