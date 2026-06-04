package server

import "testing"

func TestCanonicalRequestHostStripsTrailingDNSRootDot(t *testing.T) {
	tests := map[string]string{
		"codewords.pinky.lilf.ir.":      "codewords.pinky.lilf.ir",
		"codewords.pinky.lilf.ir.:7878": "codewords.pinky.lilf.ir:7878",
		"127.0.0.1:7878":                "127.0.0.1:7878",
	}

	for input, want := range tests {
		if got := canonicalRequestHost(input); got != want {
			t.Fatalf("canonicalRequestHost(%q) = %q, want %q", input, got, want)
		}
	}
}
