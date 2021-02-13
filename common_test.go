package main

import (
	"testing"
)

func TestCommon(t *testing.T) {
	tests := []struct {
		a string
		b string
		c string
	}{
		{"foo", "", ""},
		{"foo", "bar", ""},
		{"fo", "foo", ""},
		{"foo/ba", "foo/bar", "foo"},
		{"foo/bar", "foo/baz", "foo"},
		{"foo/bar/baz/quux/moo/blah", "foo/zip", "foo"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			if aa := common(tt.a, tt.a); aa != tt.a {
				t.Errorf("common(%q, %q) == %q, expected %q", tt.a, tt.a, aa, tt.a)
			}
			if bb := common(tt.b, tt.b); bb != tt.b {
				t.Errorf("common(%q, %q) == %q, expected %q", tt.b, tt.b, bb, tt.b)
			}
			if ab := common(tt.a, tt.b); ab != tt.c {
				t.Errorf("common(%q, %q) == %q, expected %q", tt.a, tt.b, ab, tt.c)
			}
			if ba := common(tt.b, tt.a); ba != tt.c {
				t.Errorf("common(%q, %q) == %q, expected %q", tt.b, tt.a, ba, tt.c)
			}
		})
	}
}
