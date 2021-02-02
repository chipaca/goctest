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
finding the failures is a lot easier.

## Why is this Python?

Well... if I wrote it in Go, I'd be tempted to make it an actual test
runner and not a lazy pattern-matching hack. Leaving it in Python
makes it clear that it is a hack and will continue to be so.

Useful, tho.
