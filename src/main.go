package main

import (
	"os"

	"github.com/gitKashish/kosh/src/cmd"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/logger"
)

func main() {
	// initialize connection with the database
	if err := dao.Initialize(); err != nil {
		logger.Error("error connecting to database")
		logger.Debug("%s", err.Error())
		os.Exit(1)
	}
	defer dao.Close()

	if len(os.Args) < 2 {
		cmd.HelpCmd()
		os.Exit(2)
	}

	command := os.Args[1]
	args := os.Args[2:]

	if c, ok := cmd.Commands[command]; ok {
		if err := c.Exec(args...); err != nil {
			logger.Debug("%s", err.Error())
		}
	} else {
		logger.Error("unknown command %s\n", command)
		cmd.HelpCmd()
	}
}
