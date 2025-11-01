package interaction

import (
	"github.com/gitKashish/kosh/src/internals/logger"
	"golang.design/x/clipboard"
)

func CopyToClipboard(content []byte) {
	err := clipboard.Init()
	if err != nil {
		logger.Error("unable to initialize the clipboard")
	}
	clipboard.Write(clipboard.FmtText, content)
}
