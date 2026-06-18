package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"golang.org/x/term"
)

const (
	ansiiEnter = 13
	ansiiEscape = 27
	ansiiControl = 3
	ansiiBacksapce = 127
	ansiiMoveUp = "\033[%dA"
    ansiiClearBelow = "\033[J"
)

type Searchable interface {
	Display() string
}

// InteractiveSearch presents an interactive, type-to-filter list selector in
// the terminal and returns the item the user picks.
//
// As the user types, searchFn is called to produce the matching results, which
// are listed below the prompt. The user moves through them with the arrow keys
// and confirms with Enter. Each result is shown using its Display method.
//
// Controls:
//   - Up / Down: move the selection
//   - Enter: select the highlighted item
//   - Esc / Ctrl-C: cancel
//
// Type parameter:
//   - T: the result type, which must implement Searchable so each result can
//     render itself as a line of text.
//
// Parameters:
//   - prompt: label shown before the input.
//   - searchFn: returns the results matching a query; called on each keystroke.
//     Return an empty slice to display no results (e.g. for an empty query).
//
// It returns the chosen item on Enter. If the user cancels, it returns the zero
// value of T with constants.ErrSearchCancelled; callers should treat this as a
// normal, quiet exit rather than a failure. Any error entering raw mode or
// reading input is returned with the zero value of T. Requires an interactive
// terminal (stdin must be a TTY, not piped or redirected).
func InteractiveSearch[T Searchable](
	prompt string,
	searchFn func(query string) []T,
) (T, error) {
	var zero T
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return zero, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	
	defer logger.Pause()()

	reader := bufio.NewReader(os.Stdin)
	selectedIndex := 0
	query := ""
	filtered := searchFn(query)
	prevLines := 0

	var buf bytes.Buffer
	render := func() {
		buf.Reset()
		// Move cursor to start of our region (don't clear yet).
		if prevLines > 0 {
			fmt.Fprintf(&buf, ansiiMoveUp, prevLines)
		}

		// Prompt + query line. \033[K clears to end of line as we overwrite.
		buf.WriteString(prompt)
		fmt.Fprintf(&buf, "%s\033[K\r\n", query)
		curLines := 1

		// Navigation hint (only when there's a list to move through).
		if len(filtered) > 0 {
			buf.WriteString("\033[90m↑/↓ navigate · enter select · esc cancel\033[0m\033[K\r\n")
			curLines++
		}
		
		for i, item := range filtered {
			if i == selectedIndex {
				fmt.Fprintf(&buf, "> \033[32m%s\033[0m\033[K\r\n", item.Display())
			} else {
				fmt.Fprintf(&buf, "  %s\033[K\r\n", item.Display())
			}
			curLines++
		}

		// Only when the list shrank do we need to wipe leftover lines.
		if curLines < prevLines {
			buf.WriteString("\033[J")
		}

		// One single write per frame = no flicker.
		os.Stdout.Write(buf.Bytes())
		prevLines = curLines
	}

	for {
		render()

		c1, c2, c3, err := readKey(reader)
		if err != nil {
			return zero, err
		}

		switch {
		case c1 == ansiiEnter:
			if len(filtered) > 0 {
				fmt.Printf(ansiiMoveUp, prevLines)
				fmt.Print(ansiiClearBelow)
				return filtered[selectedIndex], nil
			}
			continue

		case c1 == ansiiControl || (c1 == ansiiEscape && c2 == 0):
			fmt.Printf(ansiiMoveUp, prevLines)
			fmt.Print(ansiiClearBelow)
			return zero, constants.ErrSearchCancelled

		case c1 == ansiiBacksapce:
			if len(query) > 0 {
				query = query[:len(query)-1]
			}

		case c1 == ansiiEscape && c2 == 91:
			switch c3 {
			case 65: // up
				if selectedIndex > 0 {
					selectedIndex--
				}
				continue // selection moved, no re-filter needed
			case 66: // down
				if selectedIndex < len(filtered)-1 {
					selectedIndex++
				}
				continue
			}

		case c1 >= 32 && c1 <= 126:
			query += string(c1)

		default:
			continue // unknown key, no state change, skip re-filter
		}

		filtered = searchFn(query)
		selectedIndex = 0
	}
}

// readKey safely reads ASCII characters from the reader (stdin in this case)
func readKey(reader *bufio.Reader) (byte, byte, byte, error) {
	char1, err := reader.ReadByte()
	if err != nil {
		return 0, 0, 0, err
	}

	var char2, char3 byte
	if char1 == ansiiEscape && reader.Buffered() >= 2 { // If character is an escape sequence read next two bytes
		char2, _ = reader.ReadByte()
		char3, _ = reader.ReadByte()
	}

	return char1, char2, char3, nil
}
