package main // import "chipaca.com/goctest"

// © 2021 John Lenton
// MIT licensed.
// from https://github.com/chipaca/goctest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	// something you'd struggle to set it to, but won't freak you
	// out if it leaks
	unsetPrefix = " -(unset)- "

	usage = `
goctest [-q|-v] [-c (a.test|-)] [-trim prefix] [-|go help arguments]

The ‘-q’ and ‘-v’ flags control the amount of progress reporting:
 -q  quieter: one character per package.
 -v  verbose: one line per test (or skipped package).
Without -q nor -v, progress is reported at one line per package.

The ‘-trim’ flag allows you to specify a prefix to remove from package names.
If not given it defaults to the output of ‘go list -m’. If that fails (e.g.
because you're not running in a module) it's adjusted on the fly to be the
longest common prefix of package names reported by the test runner. This
means the very first test will get it wrong. In a pinch you can ‘-trim ""’.

The ‘-’ flag tells goctest to read the JSON output of a test result from stdin.
For example, you could do

    go test -json ./... > tests.json
    goctest - < tests.json

The ‘-c’ flag tells goctest to run the precompiled test binary. That is,
for example,

    go test -c ./foo/
    goctest -c foo.test

If the argument to ‘-c’ is ‘-’, then read plain (non-JSON) input from stdin:

    go test -v ./... > tests.out
    goctest -c - < tests.out

but note that unless the tests were run with ‘-v’, the output is going to be
slightly off from wht you'd expect (and even with it, it's not great).

Lastly, the ‘--’ flag tells goctest to stop looking at its arguments and get
on with it.

go help arguments and flags are as per usual (or you can ‘goctest -- -h’):
[build/test flags] [packages] [build/test flags & test binary flags]
Run ‘go help test’ and ‘go help testflag’ for details.
`

	// color escapes padded to be the same length, for tabwriter
	fail = "\033[38;5;124m" // #af0000 (Red3)
	pass = "\033[38;5;034m" // #00af00 (Green3)
	skip = "\033[38;5;244m" // #808080 (Grey50)
	zero = "\033[38;5;172m" // #d78700 (Orange3)
	nope = "\033[00000000m" // filler
	endc = "\033[0m"
)

type TestEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
	// these fields are there in the JSON (some of the time!) but we don't use them so why bother
	//   Time    time.Time // encodes as an RFC3339-format string
	//   Elapsed float64 // seconds
	// private stuff sneakily piggybacking
	prefix string
}

func (ev *TestEvent) pkg() string {
	if ev.prefix == "" || ev.prefix == unsetPrefix {
		return ev.Package
	}
	pkg := strings.TrimPrefix(ev.Package, ev.prefix)
	if pkg == "" {
		pkg = "/"
	}
	return "…" + pkg
}

func (ev *TestEvent) name() string {
	pkg := ev.pkg()
	if pkg == "" {
		return ev.Test
	}
	return pkg + ":" + ev.Test
}

type sums struct {
	total   int
	failed  int
	skipped int
	passed  int
}

func (s *sums) addFail() {
	s.failed++
	s.total++
}

func (s *sums) addSkip() {
	s.skipped++
	s.total++
}

func (s *sums) addPass() {
	s.total++
	s.passed++
}

type summary struct {
	tests    sums
	packages sums
}

func (ss *summary) add(ev *TestEvent) {
	var s *sums
	if ev.Test == "" {
		s = &ss.packages
	} else {
		s = &ss.tests
	}
	switch ev.Action {
	case "pass":
		s.addPass()
	case "skip":
		s.addSkip()
	case "fail":
		s.addFail()
	}
}

func (ss *summary) isZero() bool {
	return ss.tests.total-ss.tests.skipped <= 0
}

// big builds a big message, the takeaway from this test run for the user.
// It takes a font and returns as many lines of words as the font
// entries have. That is, a font with characters that are [N]string
// font produces a [N]string
func (ss *summary) big(fnt *font) []string {
	var lines []string
	p := 0
	if !ss.isZero() {
		p = (100 * ss.tests.passed) / (ss.tests.total - ss.tests.skipped)
		if p > 100 {
			p = 100
		} else if p < 0 {
			p = 0
		}
	}
	d := p % 10
	r := p / 10
	for i := range fnt.numerals[0] {
		var line []string
		if ss.isZero() {
			line = []string{zero + fnt.numerals[0][i], fnt.tests[i], fnt.run[i] + endc}
		} else {
			line = []string{colourForRatio(ss.tests.passed, ss.tests.total-ss.tests.skipped)}
			if p == 100 {
				line[0] += fnt.numerals[1][i] + fnt.numerals[0][i] + fnt.numerals[0][i] + fnt.percent[i]
			} else {
				if r > 0 {
					line[0] += fnt.numerals[r][i]
				}
				line[0] += fnt.numerals[d][i] + fnt.percent[i]
			}
			line = append(line, fnt.tests[i], fnt.passed[i]+endc)
		}
		lines = append(lines, strings.Join(line, fnt.space))
	}
	return lines
}

// returns a colour suitable for highlighting a ratio of passed to
// total tests.
// only bit tht uses 24-bit colour support
// (should just not work if not supported)
func colourForRatio(p, q int) string {
	r := (9 * p) / q
	if r == 9 {
		r = 8
	}
	if r < 0 {
		r = 0
	}
	// this tiny table is a 9-step HCL blend done using github.com/lucasb-eyer/go-colorful:
	//     blocks := 9
	//     c1, _ := colorful.Hex("#af0000")
	//     c2, _ := colorful.Hex("#00af00")
	//     for i := 0 ; i < blocks ; i++ {
	//     	    r, g, _ := c1.BlendHcl(c2, float64(i)/float64(blocks-1)).RGB255()
	//     	    fmt.Printf(`"%d;%d", `, r, g)
	//     }
	c := []string{"175;0", "174;47", "169;72", "160;94", "147;113", "130;130", "109;146", "78;161", "0;175"}[r]
	return fmt.Sprintf("\033[38;2;%s;0m", c)
}

// a progressReporter takes a TestEvent and tells your mum about it
type progressReporter interface {
	report(*TestEvent)
	summarize(*summary)
}

type defaultProgress struct{}

func (*defaultProgress) report(ev *TestEvent) {
	if ev.Test != "" {
		return
	}
	switch ev.Action {
	case "pass":
		fmt.Println(pass+"✓"+endc, ev.pkg())
	case "skip":
		fmt.Printf("%s- %s%s\n", skip, ev.pkg(), endc)
		fmt.Println(skip+"-", ev.pkg(), endc)
	case "fail":
		fmt.Println(fail+"×"+endc, ev.pkg())
	}
}

func (*defaultProgress) summarize(ss *summary) {
	fmt.Printf("Found %d tests in %d packages", ss.tests.total, ss.packages.total)
	if ss.packages.skipped > 0 {
		fmt.Printf(" (%d packages had %sNO tests%s)", ss.packages.skipped, skip, endc)
	}
	if ss.tests.total > 0 {
		fmt.Printf(".\n%d tests %spassed%s", ss.tests.passed, pass, endc)
		if ss.tests.failed > 0 {
			fmt.Printf(", and %d tests %sfailed%s", ss.tests.failed, fail, endc)
		}
		if ss.tests.skipped > 0 {
			fmt.Printf(" (%d tests were %sskipped%s)", ss.tests.skipped, skip, endc)
		}
	}
	fmt.Println(".")

	for _, line := range ss.big(&fonts.braille) {
		fmt.Println(line)
	}
}

type verboseProgress struct{}

func (*verboseProgress) report(ev *TestEvent) {
	switch ev.Action {
	case "pass":
		if ev.Test != "" {
			fmt.Println(pass+"✓"+endc, ev.name())
		}
	case "skip":
		if ev.Test != "" {
			fmt.Printf("%s- %s%s\n", skip, ev.name(), endc)
		} else {
			fmt.Printf("%s- %s%s\n", skip, ev.pkg(), endc)
		}
	case "fail":
		if ev.Test != "" {
			fmt.Println(fail+"×"+endc, ev.name())
		}
	}
}

func (*verboseProgress) summarize(ss *summary) {
	if ss.isZero() {
		for _, line := range ss.big(&fonts.future) {
			fmt.Println(line)
		}
		return
	}
	big := ss.big(&fonts.future) // here we (ab)use that future is 3 rows tall
	var w = tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, nope+endc+"\tTests\tPackages\t")
	fmt.Fprintf(w, "%sTotal%s\t%d \t%d \t\n", nope, endc, ss.tests.total, ss.packages.total)
	fmt.Fprintf(w, "%sPassed%s\t%d \t%d \t  %s\n", pass, endc, ss.tests.passed, ss.packages.passed, big[0])
	fmt.Fprintf(w, "%sSkipped%s\t%d \t%d \t  %s\n", skip, endc, ss.tests.skipped, ss.packages.skipped, big[1])
	fmt.Fprintf(w, "%sFailed%s\t%d \t%d \t  %s\n", fail, endc, ss.tests.failed, ss.packages.failed, big[2])
	w.Flush()
}

type quietProgress struct {
	needsNL bool
}

func (r *quietProgress) uri(ev *TestEvent, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", ev.pkg(), text)
}

func (r *quietProgress) report(ev *TestEvent) {
	if ev.Test != "" {
		return
	}
	switch ev.Action {
	case "pass":
		fmt.Print(pass, "▪", endc)
		r.needsNL = true
	case "skip":
		fmt.Print(skip, "▪", endc)
		r.needsNL = true
	case "fail":
		fmt.Printf("%s%s%s", fail, r.uri(ev, "×"), endc)
		r.needsNL = true
	}
}

func (r *quietProgress) summarize(ss *summary) {
	if r.needsNL {
		fmt.Println()
	}
	var s []string
	if ss.tests.skipped > 0 {
		s = append(s, fmt.Sprintf("%d %sskipped%s", ss.tests.skipped, skip, endc))
	}
	if ss.tests.failed > 0 {
		s = append(s, fmt.Sprintf("%d %sfailed%s", ss.tests.failed, fail, endc))
	}
	if ss.tests.passed > 0 {
		s = append(s, fmt.Sprintf("%d %spassed%s", ss.tests.passed, pass, endc))
	}
	if len(s) > 0 {
		fmt.Print(strings.Join(s, ", "), ". ")
	}
	fmt.Println(" ", ss.big(&fonts.double)[0])
}

// disparage is long for 'diss'.
func disparage() {
	rand.Seed(time.Now().UnixNano())
	disses := [...]string{
		"Below is a catalogue of your failures.",
		"I'm not mad. I'm disappointed.",
		"Crushing failure and despair.",
		"Are you even trying?",
		"Maybe you should take a break.",
		"One should not fear failure. But oh, dear.",
		"Once more unto the breach, dear friends, once more.",
		"Aw, bless.",
		"No, no, I'm laughing \033[3mwith\033[23m you.",
	}

	fmt.Print("\n", disses[rand.Intn(len(disses))], "\n\n")
}

func common(a, b string) string {
	if a == b {
		return a
	}
	last := -1
	if len(a) > len(b) {
		a, b = b, a
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			break
		}
		if a[i] == '/' {
			last = i
		}
	}
	if last == -1 {
		return ""
	}
	return a[:last]
}

func mkContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		select {
		case <-ctx.Done():
			signal.Stop(ch)
			close(ch)
		case sig := <-ch:
			if sig == nil {
				// channel was closed
				return
			}
			cancel()
		}
	}()
	return ctx
}

func main() {
	log.SetFlags(0)
	ctx := mkContext()

	var stream io.Reader
	var progress progressReporter = (*defaultProgress)(nil)
	var sums summary
	prefix := unsetPrefix
	compiled := ""

	args := make([]string, 2, len(os.Args)+1)
	args[0] = "test"
	args[1] = "-json"
loop:
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--":
			args = append(args, os.Args[i+1:]...)
			break loop
		case "-trim":
			i++
			prefix = os.Args[i]
		case "-":
			stream = os.Stdin
		case "-q":
			progress = &quietProgress{}
		case "-v":
			progress = (*verboseProgress)(nil)
		case "-c":
			i++
			compiled = os.Args[i]
		case "-json":
		case "-h", "-help", "--help":
			fmt.Print(usage[1:])
			return
		default:
			args = append(args, arg)
		}
	}
	if prefix == unsetPrefix {
		// don't give up hope
		out, err := exec.CommandContext(ctx, "go", "list", "-m").Output()
		if err == nil {
			prefix = strings.TrimSpace(string(out))
		}
	}

	if compiled != "" && compiled != "-" {
		if stream != nil {
			log.Fatal("The flags ‘-c’ and ‘-’ are mutualy exclusive (did you mean ‘-c -’?)")
		}
		compiled, err := exec.LookPath(compiled)
		if err != nil {
			log.Fatal(err)
		}
		x := make([]string, len(args)+2)
		copy(x, []string{"tool", "test2json", compiled, "-test.v"})
		copy(x[4:], args[2:])
		args = x
	}
	if stream == nil {
		var cmd *exec.Cmd
		if compiled == "-" {
			cmd = exec.CommandContext(ctx, "go", "tool", "test2json")
			cmd.Stdin = os.Stdin
		} else {
			cmd = exec.CommandContext(ctx, "go", args...)
		}
		cmd.Stderr = os.Stderr
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		stream = pipe
	}
	dec := json.NewDecoder(stream)
	var fails []*TestEvent

	inProgress := map[string][]*TestEvent{}
	for dec.More() {
		ev := TestEvent{}
		err := dec.Decode(&ev)
		if err != nil {
			log.Fatal(err)
		}
		if prefix == unsetPrefix {
			// take a wild guess
			prefix = ev.Package
		} else if !strings.HasPrefix(ev.Package, prefix) {
			// adjust that guess
			prefix = common(prefix, ev.Package)
		}
		ev.prefix = prefix

		progress.report(&ev)
		sums.add(&ev)

		if ev.Test != "" {
			name := ev.name()

			switch ev.Action {
			default:
				inProgress[name] = append(inProgress[name], &ev)
			case "fail":
				fails = append(fails, inProgress[name]...)
				fallthrough
			case "pass", "skip":
				delete(inProgress, name)
			}
		}
	}
	fmt.Println()
	progress.summarize(&sums)
	if len(fails) > 0 {
		disparage()
		for _, ev := range fails {
			fmt.Print(ev.Output)
		}
	}
}
