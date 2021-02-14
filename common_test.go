package main

// © 2021 John Lenton
// MIT licensed.
// from https://github.com/chipaca/goctest

import (
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// test for the ‘common()’ func
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

// check tht the output of ‘goctest -h’ mentioned in the README
// matches the usage string
func TestREADME(t *testing.T) {
	rx := regexp.MustCompile("(?m)^(.)")
	indented := rx.ReplaceAllString(usage, "    $1")
	readme, err := ioutil.ReadFile("README.md")
	if err != nil {
		t.Fatalf("can't open README: %v", err)
	}
	if !strings.Contains(string(readme), indented) {
		t.Error("README.md does not contain usage")
		// XXX: eww
		f, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatalf("i just wanted a tempfile: %v", err)
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		f.WriteString(indented)
		cmd := exec.Command("diff", "-u", "--color=always", f.Name(), "README.md")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
