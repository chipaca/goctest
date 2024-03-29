# goctest

    go install chipaca.com/goctest@latest

See what it looks like with no further arguments:

![a screencast of goctest 0.2.1 running a test suite, in default mode](img/default.gif)

<details><summary>or see the same run with <code>-v</code> (verbose)</summary>
<p>

![a screencast of goctest 0.2.1 running a test suite, in verbose mode](img/verbose.gif)

</p>
</details>

<details><summary>...or <code>-q</code> (quiet)</summary>
<p>

![a screencast of goctest 0.2.1 running a test suite, in quiet mode](img/quiet.gif)

</p>
</details>


## What does it do?

Colourise and becalm the `go test` output. Can be used to drive `go test`
directly, e.g.  if you normally do

    go test -v ./...

instead do

    goctest -v ./...

for a slightly colourised, and significantly less noisy output (with options
to make it even less so).

If what you have is a test *binary*, i.e. something built with `go test -c`,
never fear! Try

    goctest -c ./the.test

You can also pipe the output of `go test -json` through `goctest`. This is
particularly useful when `go test` runs on a Jenkins somewhere, and you can't
figure out what failed or why. Download the output and pipe it through
`goctest` and suddenly finding the failures is a lot easier.

    goctest - < /tmp/output.json

If the output you have is from a `go test` run *without* the `-json` flag, you
can try

    goctest -c - < /tmp/output.out

## Anything else?

A couple of things.

Here's the output of `goctest -h`:

    goctest [-q|-v] [-c (a.test|-)] [more goctest flags...] [-|go help arguments]

    The ‘-q’ and ‘-v’ flags control the amount of progress reporting:
     -q  quieter: one character per package.
     -v  verbose: one line per test (or skipped package).
    Without -q nor -v, progress is reported at one line per package.

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

    The above flags should do most of the work already. The remaining flags all
    start with a double dash, with the hopes that this will minimise collisions
    with ‘go test’ itself:

    ‘--esc’: this, or the environment variable GOCTEST_ESC, can be used to override
    goctest's decision about which escape sequence mode to use. As this decision
    depends on multiple envrionment variables, it can sometimes be wrong. The
    existing modes are:
      - ‘full’: uses 24-bit colour, italics, and URLs;
      - ‘mono’: no colour; bold, dim, reverse video, and italics, and URLs; and
      - ‘bare’: no escapes at all; lastly,
      - ‘test’: for testing.

    ‘--trim’: allows you to specify a prefix to remove from package names.
    If not given it defaults to the output of ‘go list -m’. If that fails (e.g.
    because you're not running in a module) it's adjusted on the fly to be the
    longest common prefix of package names reported by the test runner. This
    means the very first test will get it wrong. In a pinch you can ‘--trim ""’.

    Lastly, the ‘--’ flag tells goctest to stop looking at its arguments and get
    on with it.

    go help arguments and flags are as per usual (or you can ‘goctest -- -h’):
    [build/test flags] [packages] [build/test flags & test binary flags]
    Run ‘go help test’ and ‘go help testflag’ for details.


## What happened to the Python version?

It's still in git, in its own [branch]. It might still be useful for you.
I like this one better.

[branch]: https://github.com/chipaca/goctest/tree/python
