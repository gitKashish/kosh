package main

import (
	"fmt"
	"os"

	"github.com/gitKashish/kosh/src/cmd"
)

func main() {
	if len(os.Args) < 2 {
		cmd.Help()
		os.Exit(2)
	}

	command := os.Args[1]
	args := os.Args[2:]

	if c, ok := cmd.Commands[command]; ok {
		c(args...)
	} else {
		fmt.Printf("[Error] unknown command %s\n", command)
		cmd.Help()
		os.Exit(2)
	}
}
