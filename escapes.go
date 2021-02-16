package main

import (
	"fmt"
	"os"
)

type escape struct {
	fail, pass, skip, zero, nope, endc string
	rgb                                func(rgb [3]uint8) string
	uri                                func(url, text string) string
	em                                 func(text string) string
}

const (
	fullEsc = iota
	monoEsc
	bareEsc
	testEsc
)

var escapes = []*escape{
	{
		fail: "\033[38;5;124m",
		pass: "\033[38;5;034m",
		skip: "\033[38;5;244m",
		zero: "\033[38;5;172m",
		nope: "\033[00000000m",
		endc: "\033[0m",
		rgb: func(rgb [3]uint8) string {
			return fmt.Sprintf("\033[38;2;%d;%d;%dm", rgb[0], rgb[1], rgb[2])
		},
		uri: func(url string, text string) string {
			return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
		},
		em: func(text string) string {
			return fmt.Sprintf("\033[3m%s\033[23m", text)
		},
	}, {
		fail: "\033[7m", // reversed
		pass: "\033[1m", // bold
		skip: "\033[2m", // dim
		zero: "\033[1m", // bold
		nope: "\033[0m",
		endc: "\033[0m",
		rgb:  func(rgb [3]uint8) string { return "" },
		uri: func(url string, text string) string {
			return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
		},
		em: func(text string) string {
			return fmt.Sprintf("\033[3m%s\033[23m", text)
		},
	}, {
		fail: "", pass: "", skip: "", zero: "", nope: "", endc: "",
		rgb: func(rgb [3]uint8) string { return "" },
		uri: func(url string, text string) string { return text },
		em:  func(text string) string { return "*" + text + "*" },
	}, {
		fail: "FAIL",
		pass: "PASS",
		skip: "SKIP",
		zero: "ZERO",
		nope: "NOPE",
		endc: "ENDC",
		rgb: func(rgb [3]uint8) string {
			return fmt.Sprintf("#%2x%2x%2x", rgb[0], rgb[1], rgb[2])
		},
		uri: func(url string, text string) string {
			return fmt.Sprintf("[%s](%s)", text, url)
		},
		em: func(text string) string {
			return "*" + text + "*"
		},
	},
}

func (esc *escape) setEscape(override string) *escape {
	*esc = *guessEscape(override)
	return esc
}

func guessEscape(override string) *escape {
	switch override {
	case "":
		// meh
	case "full":
		return escapes[fullEsc]
	case "mono":
		return escapes[monoEsc]
	case "bare":
		return escapes[bareEsc]
	case "test":
		return escapes[testEsc]
	default:
		// meh
	}
	// HMMMMMMM.
	// the NO_COLOR ‘informal standard’ simply says
	//
	//   All command-line software which outputs text with ANSI
	//   color added should check for the presence of a NO_COLOR
	//   environment variable that, when present (regardless of
	//   its value), prevents the addition of ANSI color.
	//
	// but some people take that to mean "just no colour", and
	// some people take it to mean "no ANSI escape codes".  So, if
	// the env var is there, we go no colour. If it is set to
	// "strict", we go no escape codes at all.
	if noTerm, set := os.LookupEnv("NO_COLOR"); set {
		if noTerm == "strict" {
			return escapes[bareEsc]
		}
		return escapes[monoEsc]
	}
	// ok, so no NO_COLOR. Let's look at TERM next.
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return escapes[bareEsc]
	}
	// TODO: use terminfo, like a baws :-)
	if os.Getenv("COLORTERM") == "truecolor" {
		return escapes[fullEsc]
	}
	return escapes[monoEsc]
}
