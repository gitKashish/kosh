package interaction

import (
	"fmt"

	"golang.design/x/clipboard"
)

func CopyToClipboard(content []byte) {
	err := clipboard.Init()
	if err != nil {
		fmt.Println("[Error] unable to initialize the clipboard")
	}
	clipboard.Write(clipboard.FmtText, content)
}
