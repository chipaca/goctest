# goctest

## What does it do?

Colorise “go test” output. Can be used to drive `go test` directly, e.g.
if you normally do

    go test -v ./...

instead do

    goctest -v ./...

for a slightly colorized output.

Alternatively you can pipe the output of `go test` through
`goctest`. This is particularly useful when `go test` runs on a
Jenkins somewhere, and you can't figure out what failed or
why. Download the output and pipe it through `goctest` and suddenly
finding the failures is a lot easier. You can even pipe it to `less
-R` for easier pagination and searching:

    goctest < /tmp/consoleText | less -R

## Anything else?

The aim of this is to make test output readable and especially scannable.
As such, by default it will not copy lines that look like logs.  If you
need to see these lines, you can pass `--show-logs`.

If, on the contrary, you only want to see the bits that failed, then
try `--only-fails`. It will print a _bit_ of context around the fails,
but should do what you want.

Lastly you can pass `--debug` if you need to see what is matching
where, but you should be able to tell that from the output. This one
is an aid in debugging the tool itself.

## Why is this Python?

Well... if I wrote it in Go, I'd be tempted to make it an actual test
runner and not a lazy pattern-matching hack. Leaving it in Python
makes it clear that it is a hack and will continue to be so.

Useful, tho.
