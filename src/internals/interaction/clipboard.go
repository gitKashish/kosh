package interaction

import (
	"git.plutolab.org/plutolab/kosh-cli/src/internals/logger"
	"golang.design/x/clipboard"
)

func CopyToClipboard(content []byte) {
	err := clipboard.Init()
	if err != nil {
		logger.Error("unable to initialize the clipboard")
	}
	clipboard.Write(clipboard.FmtText, content)
}
