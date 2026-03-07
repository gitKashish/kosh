package main

import (
	"os"
	"runtime/secret"

	"git.plutolab.org/plutolab/kosh/src/cmd"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
)

const (
	DEFAULT_COMMAND = "search"
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

	c, ok := cmd.Commands[command]
	if !ok {
		// Fetch default command
		c, ok = cmd.Commands[DEFAULT_COMMAND]
		args = os.Args[1:]
	}

	if !ok {
		logger.Error("unknown command %s\n", command)
		cmd.HelpCmd()
		os.Exit(1)
	} else {
		secret.Do(func() {
			if err := c.Exec(args...); err != nil {
				logger.Debug("%s", err.Error())
			}
		})
	}
}
