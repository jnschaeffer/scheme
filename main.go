package main

import (
	"flag"
	"os"

	"github.com/jnschaeffer/scheme/lang"
)

func main() {
	flag.Parse()
	lang.Run(os.Stdin)
}
