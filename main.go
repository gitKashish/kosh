package main

import (
	"plutolab.org/kosh/cmd"
	"plutolab.org/kosh/internal/logger"
)

func main() {
	logger.Setup()
	cmd.Execute()
}
