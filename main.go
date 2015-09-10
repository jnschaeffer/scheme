package main

import (
	"bufio"
	"io"
	"log"
	"os"

	"github.com/jnschaeffer/scheme/lang"
)

func main() {
	input := bufio.NewReader(os.Stdin)
	for {
		if _, err := os.Stdout.WriteString("> "); err != nil {
			log.Fatalf("WriteString: %s", err)
		}
		line, err := input.ReadBytes('\n')
		if err == io.EOF {
			return
		}

		if err != nil {
			log.Fatalf("ReadBytes: %s", err)
		}

		lang.Run(string(line))
	}
}
