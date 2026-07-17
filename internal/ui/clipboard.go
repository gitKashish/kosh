package ui

import (
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"golang.design/x/clipboard"
)

func CopyToClipboard(content []byte) {
	err := clipboard.Init()
	if err != nil {
		logger.Error("unable to initialize the clipboard")
		logger.Debug("error initializing clipboard: %v", err)
		return
	}

	clipboard.Write(clipboard.FmtText, content)
}
