package main

import (
	"fmt"
	"os"

	"github.com/nicholasss/gator/internal/config"
)

const username = "Nick"

// name is the command, and arguments are the 1-many arguments being supplied
type command struct {
	name      string
	arguments []string
}

type commands struct {
	commands map[string]func(*state, command) error
}

// state... holds the state of the program
type state struct {
	cfg *config.Config
}

func newCommands() *commands {
	var cmds commands
	cmds.commands = make(map[string]func(*state, command) error)
	return &cmds
}

func handlerLogin(s *state, c command) error {
	if len(c.arguments) == 0 {
		return fmt.Errorf("Expected username in c.arguments\n")
	}

	username := c.arguments[0]
	err := s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("Set username to '%v' successfully.\n", username)
	return nil
}

// registers a new command handler function.
// added an error return value for uninitialized map.
func (c *commands) register(name string, f func(*state, command) error) error {
	if c.commands == nil { // uninitialized map
		return fmt.Errorf("Uninitialized map was passed in commands struct.\n")
	}

	c.commands[name] = f
	return nil
}

// runs a given command with the provided state (if it exists)
func (c *commands) run(s *state, cmd command) error {
	handlerFunc, ok := c.commands[cmd.name]
	if !ok {
		return fmt.Errorf("%v is not a registered handler.\n", cmd.name)
	}

	err := handlerFunc(s, cmd)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// read cfg from disk
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error occured: %v\n", err)
		os.Exit(1)
		return
	}

	state := state{
		cfg: &cfg,
	}

	cmds := newCommands()
	cmds.register("login", handlerLogin)

	args := os.Args
	numArgs := len(args)
	if numArgs < 2 {
		fmt.Printf("Not enough arguments provided: %d\nArgs: %v\n", numArgs-1, args[1:])
		os.Exit(1)
	}

	cmd := command{
		name:      args[1],  // command name
		arguments: args[2:], // inclusive of the arguments after command name
	}

	err = cmds.run(&state, cmd)
	if err != nil {
		fmt.Printf("command error: %v", err)
		os.Exit(1)
	}
}
