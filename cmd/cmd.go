package cmd

// Each command is just a function taking in some args
type Command func(args ...string) error

type CommandInfo struct {
	Exec        Command
	Description string
	Usage       string
}

// Central registry for all the commands
var Commands = map[string]CommandInfo{}
