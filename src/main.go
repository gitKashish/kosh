package main

import (
	"fmt"
	"os"

	"github.com/gitKashish/kosh/src/cmd"
	"github.com/gitKashish/kosh/src/internals/dao"
)

func main() {
	// initialize connection with the database
	if err := dao.Initialize(); err != nil {
		fmt.Println("[Error] error connecting to database")
		fmt.Printf("[Debug] %s", err.Error())
		os.Exit(1)
	}
	defer dao.Close()

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
