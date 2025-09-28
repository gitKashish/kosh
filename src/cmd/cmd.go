package cmd

import "fmt"

// Each command is just a function taking in some args
type Command func(args ...string)

// Central registry for all the commands
var Commands = map[string]Command{
	"help": Help,
}

func Help(args ...string) {
	fmt.Println("Usage:")
	fmt.Println("  kosh init    - Initialize Kosh (vault)")
	fmt.Println("  kosh help    - Show this help")
}
