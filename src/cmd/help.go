package cmd

import "fmt"

func init() {
	Commands["help"] = CommandInfo{
		Exec:        HelpCmd,
		Description: "show help information",
		Usage:       "kosh help",
	}
}

func HelpCmd(args ...string) error {
	fmt.Println("Kosh - Secure Password Manager")
	fmt.Println()
	fmt.Println("Usage:")

	// generate help from registered commands
	for _, info := range Commands {
		fmt.Printf("%s - %s\n", info.Usage, info.Description)
	}

	fmt.Println("\nFor more information, visit: https://git.plutolab.org/plutolab/kosh")

	return nil
}
