package main

import (
	"flag"

	"github.com/jnschaeffer/scheme/lang"
)

func main() {
	flag.Parse()
	lang.REPL()
}
