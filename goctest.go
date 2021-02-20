package main // import "chipaca.com/goctest"

// © 2021 John Lenton
// MIT licensed.
// from https://github.com/chipaca/goctest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	// something you'd struggle to set it to, but won't freak you
	// out if it leaks
	unsetPrefix = " -(unset)- "
	// an unikely name for a test
	errorPlaceholder = " -(error)- "

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

func (ev *TestEvent) isTest() bool {
	return ev.Test != "" && ev.Test != errorPlaceholder
}

type sums struct {
	total   int
	failed  int
	errored int
	skipped int
	passed  int
}

func (s *sums) addFail() {
	s.failed++
	s.total++
}

func (s *sums) addError() {
	s.errored++
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

func (s *sums) isZero() bool {
	return s.total-s.skipped <= 0
}

type summary struct {
	tests    sums
	packages sums
}

func (ss *summary) add(ev *TestEvent) {
	var s *sums
	if ev.Test == "" || ev.Test == errorPlaceholder {
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
	case "error":
		s.addError()
	}
}

func (ss *summary) isZero() bool {
	return ss.tests.isZero() && ss.packages.isZero()
}

// big builds a big message, the takeaway from this test run for the user.
// It takes a font and returns as many lines of words as the font
// entries have. That is, a font with characters that are [N]string
// font produces a [N]string
func (ss *summary) big(esc *escape, fnt *font) []string {
	var lines []string
	p := 0
	if !ss.tests.isZero() {
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
		if ss.tests.isZero() {
			line = []string{esc.zero + fnt.numerals[0][i], fnt.tests[i], fnt.run[i] + esc.endc}
		} else {
			line = []string{esc.rgb(colourForRatio(ss.tests.passed, ss.tests.total-ss.tests.skipped))}
			if p == 100 {
				line[0] += fnt.numerals[1][i] + fnt.numerals[0][i] + fnt.numerals[0][i] + fnt.percent[i]
			} else {
				if r > 0 {
					line[0] += fnt.numerals[r][i]
				}
				line[0] += fnt.numerals[d][i] + fnt.percent[i]
			}
			line = append(line, fnt.tests[i], fnt.passed[i]+esc.endc)
		}
		lines = append(lines, strings.Join(line, fnt.space))
	}
	return lines
}

// returns a colour suitable for highlighting a ratio of passed to
// total tests.
// only bit tht uses 24-bit colour support
// (should just not work if not supported)
func colourForRatio(p, q int) [3]uint8 {
	r := (9 * p) / q
	if r == 9 {
		r = 8
	}
	if r < 0 {
		r = 0
	}
	// this tiny table is a 9-step HCL blend done using github.com/lucasb-eyer/go-colorful:
	//     const blocks = 9
	//     c1, _ := colorful.Hex("#af0000")
	//     c2, _ := colorful.Hex("#00af00")
	//     for i := float64(0) ; i < blocks ; i++ {
	//     	    r, g, b := c1.BlendHcl(c2, i/(blocks-1)).RGB255()
	//     	    fmt.Printf("{%d, %d, %d},", r, g, b)
	//     }
	return [][3]uint8{
		{175, 0, 0},
		{174, 47, 0},
		{169, 72, 0},
		{160, 94, 0},
		{147, 113, 0},
		{130, 130, 0},
		{109, 146, 0},
		{78, 161, 0},
		{0, 175, 0},
	}[r]
}

// a progressReporter takes a TestEvent and tells your mum about it
type progressReporter interface {
	report(*TestEvent)
	summarize(*summary)
	setEscape(string) *escape
}

type defaultProgress struct{ escape }

func (p *defaultProgress) report(ev *TestEvent) {
	if ev.isTest() {
		return
	}
	switch ev.Action {
	case "pass":
		fmt.Println(p.pass+"✓"+p.endc, ev.pkg())
	case "skip":
		fmt.Printf("%s- %s%s\n", p.skip, ev.pkg(), p.endc)
	case "fail":
		fmt.Println(p.fail+"×"+p.endc, ev.pkg())
	case "error":
		fmt.Printf("%sℯ %s%s\n", p.fail, ev.pkg(), p.endc)
	}
}

func gn(s, p string) func(int) string {
	return func(n int) string {
		x := p
		if n == 1 {
			x = s
		}
		return fmt.Sprintf("%d %s", n, x)
	}
}

func (p *defaultProgress) summarize(ss *summary) {
	pkg := gn("package", "packages")
	tst := gn("test", "tests")
	fmt.Printf("Found %s in %s", tst(ss.tests.total), pkg(ss.packages.total))
	if ss.packages.skipped > 0 {
		fmt.Printf(" (%s had %sNO tests%s)", pkg(ss.packages.skipped), p.skip, p.endc)
	}
	if ss.packages.errored > 0 {
		fmt.Printf(", and %d packages did not even build", ss.packages.errored)
	}
	if ss.tests.total > 0 {
		fmt.Printf(".\n%d tests %spassed%s", ss.tests.passed, p.pass, p.endc)
		if ss.tests.failed > 0 {
			fmt.Printf(", and %d tests %sfailed%s", ss.tests.failed, p.fail, p.endc)
		}
		if ss.tests.skipped > 0 {
			fmt.Printf(" (%d tests were %sskipped%s)", ss.tests.skipped, p.skip, p.endc)
		}
	}
	fmt.Println(".")

	for _, line := range ss.big(&p.escape, &fonts.braille) {
		fmt.Println(line)
	}
}

type verboseProgress struct {
	escape
	seenFails map[string]bool
}

func (p *verboseProgress) report(ev *TestEvent) {
	switch ev.Action {
	case "pass":
		if ev.Test != "" {
			fmt.Println(p.pass+"✓"+p.endc, ev.name())
		}
	case "skip":
		if ev.Test != "" {
			fmt.Printf("%s- %s%s\n", p.skip, ev.name(), p.endc)
		} else {
			fmt.Printf("%s- %s%s\n", p.skip, ev.pkg(), p.endc)
		}
	case "fail":
		if ev.Test != "" {
			if ev.Package != "" {
				p.seenFails[ev.Package] = true
			}
			fmt.Println(p.fail+"×"+p.endc, ev.name())
		} else if !p.seenFails[ev.Package] {
			fmt.Println(p.fail+"×"+p.endc, ev.pkg())
		}
	case "error":
		fmt.Println(p.fail+"ℯ"+p.endc, ev.pkg())
	}
}

func (p *verboseProgress) summarize(ss *summary) {
	if ss.isZero() {
		for _, line := range ss.big(&p.escape, &fonts.future) {
			fmt.Println(line)
		}
		return
	}
	big := ss.big(&p.escape, &fonts.future) // here we (ab)use that future is 3 rows tall
	var w = tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, p.nope+"\t\tTests\tPackages\t"+p.endc+"\t")
	fmt.Fprintf(w, "%s\tTotal\t%d \t%d \t%s\t\n", p.nope, ss.tests.total, ss.packages.total, p.endc)
	fmt.Fprintf(w, "%s\tPassed\t%d \t%d \t%s\t  %s\n", p.pass, ss.tests.passed, ss.packages.passed, p.endc, big[0])
	fmt.Fprintf(w, "%s\tSkipped\t%d \t%d \t%s\t  %s\n", p.skip, ss.tests.skipped, ss.packages.skipped, p.endc, big[1])
	fmt.Fprintf(w, "%s\tFailed\t%d \t%d \t%s\t  %s\n", p.fail, ss.tests.failed, ss.packages.failed, p.endc, big[2])
	fmt.Fprintf(w, "%s\tError'ed\t - \t%d \t%s\t\n", p.fail, ss.packages.errored, p.endc)
	w.Flush()
}

type quietProgress struct {
	escape
	needsNL bool
}

func (p *quietProgress) report(ev *TestEvent) {
	if ev.isTest() {
		return
	}
	switch ev.Action {
	case "pass":
		fmt.Print(p.pass, "•", p.endc)
		p.needsNL = true
	case "skip":
		fmt.Print(p.skip, "•", p.endc)
		p.needsNL = true
	case "fail":
		fmt.Printf("%s%s%s", p.fail, p.uri(ev.pkg(), "×"), p.endc)
		p.needsNL = true
	case "error":
		fmt.Printf("%s%s%s", p.fail, p.uri(ev.pkg(), "e"), p.endc)
	}
}

func (p *quietProgress) summarize(ss *summary) {
	if p.needsNL {
		fmt.Println()
	}
	var s []string
	if ss.tests.skipped > 0 {
		s = append(s, fmt.Sprintf("%d %sskipped%s", ss.tests.skipped, p.skip, p.endc))
	}
	if ss.tests.failed > 0 {
		s = append(s, fmt.Sprintf("%d %sfailed%s", ss.tests.failed, p.fail, p.endc))
	}
	if ss.tests.passed > 0 {
		s = append(s, fmt.Sprintf("%d %spassed%s", ss.tests.passed, p.pass, p.endc))
	}
	if len(s) > 0 {
		fmt.Print(strings.Join(s, ", "), ". ")
	}
	fmt.Println(" ", ss.big(&p.escape, &fonts.double)[0])
}

// disparage is long for 'diss'.
func disparage(esc *escape) {
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
		"No, no, I'm laughing " + esc.em("with") + " you.",
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

var failRx = regexp.MustCompile(`^FAIL\s+(\S+)\s*.*`)

func main() {
	log.SetFlags(0)
	ctx := mkContext()

	var stream io.Reader
	var progress progressReporter
	var sums summary
	escOverride := os.Getenv("GOCTEST_ESC")
	prefix := unsetPrefix
	compiled := ""

	args := make([]string, 2, len(os.Args)+1)
	args[0] = "test"
	args[1] = "-json"
loop:
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if idx := strings.IndexByte(arg, '='); idx > -1 {
			v := arg[idx+1:]
			switch arg[:idx] {
			case "-esc":
				escOverride = v
			case "-trim":
				prefix = v
			case "-c":
				compiled = v
			default:
				args = append(args, arg)
			}
		} else {
			switch arg {
			case "--":
				args = append(args, os.Args[i+1:]...)
				break loop
			case "-esc":
				i++
				escOverride = os.Args[i]
			case "-trim":
				i++
				prefix = os.Args[i]
			case "-":
				stream = os.Stdin
			case "-q":
				progress = &quietProgress{}
			case "-v":
				progress = &verboseProgress{seenFails: map[string]bool{}}
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
	}
	if progress == nil {
		progress = &defaultProgress{}
	}
	esc := progress.setEscape(escOverride)

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
		if compiled == "" {
			cmd.Stderr = cmd.Stdout
		}
		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		stream = pipe
	}

	var fails []string
	inProgress := map[string][]string{}
	// if it weren't for those pesky non-JSON lines, we could just
	//     dec := json.NewDecoder(stream)
	//     for dec.More() { ...
	// TODO: file a bug with Go about the non-JSON lines in JSON output
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		var ev TestEvent
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if line[0] == '{' {
			err := json.Unmarshal(line, &ev)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			if m := failRx.FindSubmatch(line); m != nil {
				// fake it
				ev = TestEvent{
					Action:  "error",
					Package: string(m[1]),
					Output:  string(line) + "\n",
					Test:    errorPlaceholder,
				}
			} else {
				ev = TestEvent{
					Action: "output",
					Output: string(line) + "\n",
					Test:   errorPlaceholder,
				}
			}
			fmt.Fprintln(os.Stderr, string(line))
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
				if ev.Output != "" {
					inProgress[name] = append(inProgress[name], ev.Output)
				}
			case "error":
				name = errorPlaceholder
				if ev.Output != "" {
					inProgress[name] = append(inProgress[name], ev.Output)
				}
				fallthrough
			case "fail":
				fails = append(fails, inProgress[name]...)
				fallthrough
			case "pass", "skip":
				delete(inProgress, name)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	progress.summarize(&sums)
	if len(fails) > 0 {
		disparage(esc)
		for _, ev := range fails {
			fmt.Print(ev)
		}
	}
}
