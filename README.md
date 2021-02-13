# goctest

## What does it do?

Colorise “go test” output. Can be used to drive `go test` directly, e.g.
if you normally do

    go test -v ./...

instead do

    goctest -v ./...

for a slightly colorized output.

If what you have is a test *binary*, i.e. something built with `go test
-c`, never fear! Try

    goctest -c ./the.test

You can also pipe the output of `go test -json` through `goctest`. This is
particularly useful when `go test` runs on a Jenkins somewhere, and you
can't figure out what failed or why. Download the output and pipe it
through `goctest` and suddenly finding the failures is a lot easier.

    goctest - < /tmp/output.json

If the output you have is from a `go test` run *without* the `-json` flag,
you can try

    goctest -c - < /tmp/output.out

## Anything else?

A couple of things.

Here's the output of `goctest -h`:

    goctest [-q|-v|-h] [-trim prefix] [-|go help arguments]

    The -q and -v flags control the amount of progress reporting:
     -q  quieter: one character per non-failing package, one line per test fail.
     -v  verbose: one line per test.
    Without -q nor -v, progress is reported at one line per package.

    The -trim flag allows you to specify a prefix to remove from package names.
    If not given, we adjust it on the fly to be the longest common prefix of
    package names reported by the test runner. This means the very first test
    will probably get it wrong.

    The '-' flag tells goctest to read the JSON output of a test result from stdin.
    If the output you have is not JSON (i.e. it's from a plain 'go test' run,
    without '-json'), read 'go help tool test2json'.

    Lastly, the '--' flag tells goctest to stop looking at its arguments and get
    on with it.

    go help arguments flags are as per usual (or you can 'goctest -- -h'):
    [build/test flags] [packages] [build/test flags & test binary flags]
    Run 'go help test' and 'go help testflag' for details.
    ~/src/goctest (main-go)$  printf "%q\n" "$(tput nop)"


## What happened to the Python version?

It's still in git, in its own branch. It might still be useful for you.
I like this one better.
