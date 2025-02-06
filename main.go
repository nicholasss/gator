package main

import (
	"fmt"
	"github.com/nicholasss/gator/internal/config"
)

const username = "Nick"

// name is the command, and arguments are the 1-many arguments being supplied
type command struct {
	name      string
	arguments []string
}

type commands struct {
	names map[string]func(*state, command) error
}

// state... holds the state of the program
type state struct {
	cfg *config.Config
}

func handlerLogin(s *state, c command) error {
	if len(c.arguments) == 0 {
		return fmt.Errorf("Expected name in c.arguments")
	}

	username := c.arguments[0]
	err := s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("Set username to '%v' successfully.\n", username)
	return nil
}

// registers a new command handler function
func (c *commands) register(name string, f func(*state, command) error) {

	return
}

// runs a given command with the provided state (if it exists)
func (c *commands) run(s *state, cmd command) error {

	return nil
}

func main() {
	// read cfg from disk
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error occured: %v\n", err)
		return
	}

	// set username, which will write to disk
	cfg.SetUser(username)

	// read again from disk
	cfg, err = config.Read()
	if err != nil {
		fmt.Printf("Error occured: %v\n", err)
		return
	}

	fmt.Printf("Config file: %+v\n", cfg)
}
