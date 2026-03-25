package app

import "testing"

func TestHasScrollSpaceBelow(parseT *testing.T) {
	parseTests := []struct {
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

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := hasScrollSpaceBelow(parseTest.scrollTop, parseTest.scrollHeight, parseTest.clientHeight, parseTest.threshold); parseGot != parseTest.want {
				parseT2.Fatalf("hasScrollSpaceBelow(%v, %v, %v, %v) = %v, want %v", parseTest.scrollTop, parseTest.scrollHeight, parseTest.clientHeight, parseTest.threshold, parseGot, parseTest.want)
			}
		})
	}
}
