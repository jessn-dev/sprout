package main

import (
	"log"

	"github.com/jessn-dev/sprout/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
