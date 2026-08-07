package main

import (
	"git.plutolab.org/plutolab/kosh/cmd"
	"git.plutolab.org/plutolab/kosh/internal/logger"
)

func main() {
	logger.Setup()
	cmd.Execute()
}
