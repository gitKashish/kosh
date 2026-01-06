package interaction

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"golang.org/x/term"
)

// ReadStringField prompts the user and reads input from the standard input
func ReadStringField(prompt string) (string, error) {
	logger.Prompt("%s", prompt)
	reader := bufio.NewReader(os.Stdin)
	data, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(data), nil
}

// ReadStringFieldWithRetry prompts with automatic retry on error
func ReadStringFieldWithRetry(prompt string) string {
	for {
		data, err := ReadStringField(prompt)
		if err != nil {
			logger.Error("failed to read input: %v", err)
			continue
		}
		return data
	}
}

// ReadSecretField prompts the user and reads input without displaying entered characters
func ReadSecretField(prompt string) (string, error) {
	logger.Prompt("%s", prompt)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after password input
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(data), nil
}

// ReadSecretFieldWithRetry prompts with automatic retry on error
func ReadSecretFieldWithRetry(prompt string) string {
	for {
		data, err := ReadSecretField(prompt)
		if err != nil {
			logger.Error("failed to read password: %v", err)
			continue
		}
		return data
	}
}

// GetOptionField prompts user with provided options
func GetOptionField(prompt string, options []string, defaultOption int) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}
	if defaultOption < 0 || defaultOption >= len(options) {
		defaultOption = 0
	}

	// Display prompt
	logger.Prompt("%s\n", prompt)

	// Display options
	for i, option := range options {
		if i == defaultOption {
			fmt.Printf("  [%d] %s %s(default)%s\n", i+1, option, logger.ColorCyan, logger.ColorReset)
		} else {
			fmt.Printf("  [%d] %s\n", i+1, option)
		}
	}

	// Get user input
	fmt.Printf("\n%s[?]%s enter choice [1-%d] (default: %d): ",
		logger.ColorCyan, logger.ColorReset, len(options), defaultOption+1)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return -1, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// If empty, return default
	if input == "" {
		return defaultOption, nil
	}

	// Parse input
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid input, please enter a number")
	}

	// Validate choice
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

// ConfirmWithText prompts user to type exact confirmation text
func ConfirmWithText(prompt, confirmationText string) (bool, error) {
	logger.Warn("%s", prompt)
	input, err := ReadStringField(fmt.Sprintf("enter '%s' to confirm or anything else to cancel: ", confirmationText))
	if err != nil {
		return false, err
	}
	return input == confirmationText, nil
}

// ConfirmWithTextRetry prompts with automatic retry on error (not wrong text)
func ConfirmWithTextRetry(prompt, confirmationText string) bool {
	for {
		confirmed, err := ConfirmWithText(prompt, confirmationText)
		if err != nil {
			logger.Error("failed to read input: %v", err)
			continue
		}
		return confirmed
	}
}

// ConfirmYesNo prompts user for a simple yes/no confirmation
func ConfirmYesNo(prompt string, defaultYes bool) (bool, error) {
	var suffix string
	if defaultYes {
		suffix = "[Y/n]"
	} else {
		suffix = "[y/N]"
	}

	logger.Prompt("%s %s: ", prompt, suffix)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.ToLower(strings.TrimSpace(input))

	// Empty input uses default
	if input == "" {
		return defaultYes, nil
	}

	return input == "y" || input == "yes", nil
}

// ConfirmYesNoRetry prompts with automatic retry on error
func ConfirmYesNoRetry(prompt string, defaultYes bool) bool {
	for {
		confirmed, err := ConfirmYesNo(prompt, defaultYes)
		if err != nil {
			logger.Error("failed to read input: %v", err)
			continue
		}
		return confirmed
	}
}
