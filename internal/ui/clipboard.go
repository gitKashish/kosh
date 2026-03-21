package ui

import (
	"os"
	"os/exec"

	"git.plutolab.org/plutolab/kosh/internal/logger"
	"golang.design/x/clipboard"
)

const (
	waylandCopy = "wl-copy"
)

func CopyToClipboard(content []byte) {
	// check wayland environment
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if err := copyOnWayland(content); err != nil {
			logger.Error("error copying via wayland, trying alternate methods")
		} else {
			return
		}
	}

	err := clipboard.Init()
	if err != nil {
		logger.Error("unable to initialize the clipboard")
		logger.Debug("error initializing clipboard: %v", err)
		return
	}

	clipboard.Write(clipboard.FmtText, content)
}

func copyOnWayland(content []byte) error {
	command := exec.Command(waylandCopy, "--no-newline")
	in, err := command.StdinPipe()
	if err != nil {
		logger.Error("error copying using wl-copy")
		return err
	}

	if err := command.Start(); err != nil {
		logger.Error("error starting wl-copy")
		return err
	}

	if _, err := in.Write(content); err != nil {
		logger.Error("error writing to command")
		return err
	}

	if err := in.Close(); err != nil {
		logger.Error("error writing to command")
		return err
	}

	return command.Wait()
}
