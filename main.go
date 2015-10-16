package main

import (
	"flag"
	"log"
	"os"

	"github.com/jnschaeffer/scheme/lang"
)

var useREPL bool

func init() {
	flag.BoolVar(&useREPL, "repl", false, "use repl")
}

func main() {
	flag.Parse()
	rt := lang.NewRuntime()

	if useREPL {
		rt.REPL()
	} else {
		_, err := rt.Eval(os.Stdin)
		if err != nil {
			log.Print(err)
		}
	}
}
