package interaction

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ReadStringField prompts the user and reads input from the standard input
func ReadStringField(prompt string) string {
	var data string
	fmt.Print(prompt)
	fmt.Scanln(&data)
	return data
}

// ReadSecretField prompts the user and reads input from the standard input but without displaying entered characters
func ReadSecretField(prompt string) (string, error) {
	fmt.Print(prompt)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("[Error] error reading text")
		return "", err
	}
	fmt.Println()
	return string(data), nil
}
