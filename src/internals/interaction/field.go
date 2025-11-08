package interaction

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gitKashish/kosh/src/internals/logger"
	"golang.org/x/term"
)

// ReadStringField prompts the user and reads input from the standard input
func ReadStringField(prompt string) string {
	var data string
	logger.Prompt("%s", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		data = scanner.Text()
	}
	return data
}

// ReadSecretField prompts the user and reads input from the standard input but without displaying entered characters
func ReadSecretField(prompt string) (string, error) {
	logger.Prompt("%s", prompt)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(data), nil
}

// GetOptionField prompts user with provided options with error on invalid input
func GetOptionField(prompt string, options []string, defaultOption int) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	if defaultOption < 0 || defaultOption >= len(options) {
		defaultOption = 0
	}

	// display prompt
	logger.Prompt("%s\n", prompt)

	// display options
	for i, option := range options {
		if i == defaultOption {
			fmt.Printf("  [%d] %s %s(default)%s\n", i+1, option, logger.ColorCyan, logger.ColorReset)
		} else {
			fmt.Printf("  [%d] %s\n", i+1, option)
		}
	}

	// get user input
	fmt.Printf("\n%s[?]%s enter choice [1-%d] (default: %d): ",
		logger.ColorCyan, logger.ColorReset, len(options), defaultOption+1)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return -1, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// if empty, return default
	if input == "" {
		return defaultOption, nil
	}

	// parse input
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid input, please enter a number")
	}

	// validate choice
	if choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("invalid choice, please enter a number between 1 and %d", len(options))
	}

	return choice - 1, nil
}

// GetOptionFieldWithRetry prompts with automatic retry on invalid input
func GetOptionFieldWithRetry(prompt string, options []string, defaultOption int) int {
	for {
		index, err := GetOptionField(prompt, options, defaultOption)
		if err != nil {
			logger.Error("%v", err)
			fmt.Println()
			continue
		}
		return index
	}
}
