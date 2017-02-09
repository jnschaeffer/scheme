# scheme - An implementation of scheme

From [Wikipedia][1]:
  "Scheme is a functional programming language and one of the two main dialects
  of the programming language Lisp."

This is an incomplete implementation of [R5RS][2] Scheme. Some things may be
broken or poorly documented.

## Installation

`go generate ./lang && go install ./cmd/scheme`

## Usage

Running `scheme` will launch a REPL. File and stdin program execution are in progress.

[1]: https://en.wikipedia.org/wiki/Scheme_%28programming_language%29 "Scheme"
[2]: http://www.schemers.org/Documents/Standards/R5RS/ "R5RS"
