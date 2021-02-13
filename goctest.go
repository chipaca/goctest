package main // import "chipaca.com/goctest"

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
 -q  quieter: one character per non-failing package, one line per test fail.
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

go help arguments and flags are as per usual (or you can 'goctest -- -h'):
[build/test flags] [packages] [build/test flags & test binary flags]
Run 'go help test' and 'go help testflag' for details.
`

	// color escapes padded to be the same length, for tabwriter
	fail = "\033[38;5;196m"
	pass = "\033[38;5;028m"
	skip = "\033[38;5;244m"
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

// a progressReporter takes a TestEvent and tells your mum about it
type progressReporter interface {
	report(*TestEvent)
	summarize(*summary)
}

type defaultProgress struct{}

func (*defaultProgress) report(ev *TestEvent) {
	switch ev.Action {
	case "pass":
		if ev.Test == "" {
			fmt.Println(pass+"✓"+endc, ev.pkg())
		}
	case "skip":
		if ev.Test == "" {
			fmt.Printf("%s- %s%s\n", skip, ev.pkg(), endc)
			fmt.Println(skip+"-", ev.pkg(), endc)
		}
	case "fail":
		if ev.Test == "" {
			fmt.Println(fail+"×"+endc, ev.pkg())
		}
	}
}

func (*defaultProgress) summarize(ss *summary) {
	fmt.Printf("Found %d tests in %d packages", ss.tests.total, ss.packages.total)
	if ss.packages.skipped > 0 {
		fmt.Printf(" (%d packages had %sNO tests%s)", ss.packages.skipped, skip, endc)
	}
	fmt.Printf(".\n%d tests %spassed%s", ss.tests.passed, pass, endc)
	if ss.tests.failed > 0 {
		fmt.Printf(", and %d tests %sfailed%s", ss.tests.failed, fail, endc)
	}
	if ss.tests.skipped > 0 {
		fmt.Printf(" (%d tests were %sskipped%s)", ss.tests.skipped, skip, endc)
	}
	// TODO: percentage -> green/red slider
	fmt.Printf(". That's %d%%.\n", ss.tests.passed*100/ss.tests.total)
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
	var w = tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, nope+endc+"\tTests\tPackages\t")
	fmt.Fprintf(w, "%sTotal%s\t%d \t%d \t\n", nope, endc, ss.tests.total, ss.packages.total)
	fmt.Fprintf(w, "%sPassed%s\t%d \t%d \t\n", pass, endc, ss.tests.passed, ss.packages.passed)
	fmt.Fprintf(w, "%sSkipped%s\t%d \t%d \t\n", skip, endc, ss.tests.skipped, ss.packages.skipped)
	fmt.Fprintf(w, "%sFailed%s\t%d \t%d \t\n", fail, endc, ss.tests.failed, ss.packages.failed)
	w.Flush()
}

type quietProgress struct {
	needsNL bool
}

func (r *quietProgress) report(ev *TestEvent) {
	switch ev.Action {
	case "pass":
		if ev.Test == "" {
			fmt.Print(pass, "▪", endc)
			r.needsNL = true
		}
	case "skip":
		if ev.Test == "" {
			fmt.Print(skip, "▪", endc)
			r.needsNL = true
		}
	case "fail":
		if ev.Test != "" {
			if r.needsNL {
				fmt.Println()
			}
			fmt.Printf("%s%s%s\n", fail, ev.name(), endc)
			r.needsNL = false
		}
	}
}

func (r *quietProgress) summarize(ss *summary) {
	if r.needsNL {
		fmt.Println()
	}
	fmt.Printf("%d %sfailed%s, %d %spassed%s.\n", ss.tests.failed, fail, endc, ss.tests.passed, pass, endc)
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
	}

	fmt.Print("\n", disses[rand.Intn(len(disses))], "\n\n")
}

func common(a, b string) string {
	aa := strings.Split(a, "/")
	bb := strings.Split(b, "/")
	if len(aa) > len(bb) {
		aa, bb = bb, aa
	}
	var cc []string
	for i, a := range aa {
		if a != bb[i] {
			break
		}
		cc = append(cc, a)
	}
	return strings.Join(cc, "/")
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
	progress.summarize(&sums)
	if len(fails) > 0 {
		disparage()
		for _, ev := range fails {
			fmt.Print(ev.Output)
		}
	}
}
