package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nicholasss/gator/internal/config"
	"github.com/nicholasss/gator/internal/database"

	// imported postgres driver for side effects
	_ "github.com/lib/pq"
)

// ================
// TYPE DEFINITIONS
// ================

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
	db  *database.Queries
	cfg *config.Config
}

// =============
// UTILITY FUNCS
// =============

// checks for number of arguments
func checkNumArgs(args []string, targetArgNum int) error {
	num := len(args)
	if num == 0 {
		return fmt.Errorf("no arguments were passed in.\n")
	} else if num > targetArgNum {
		return fmt.Errorf("too many arguments were provided. needs %d\n", targetArgNum)
	}
	return nil
}

// ================
// COMMAND HANDLERS
// ================

// list of valid command handlers
var validCommands map[string]string = map[string]string{
	"login":    "Logs into a user",
	"register": "Registers a new user",
	"help":     "Shows available commands",
}

// prints out valid commands
func handlerHelp(_ *state, _ command) error {
	fmt.Println("Available commands:")
	for cName, cDesc := range validCommands {
		fmt.Printf(" - %s: %s\n", cName, cDesc)
	}
	fmt.Println("")

	return nil
}

// logs in a given user
// sets the given user within the configuration json
func handlerLogin(s *state, c command) error {
	targetArgNum := 1
	if err := checkNumArgs(c.arguments, targetArgNum); err != nil {
		return err
	}

	username := c.arguments[0]
	username = strings.ToLower(username)

	// check database for user
	dbFoundUser, _ := s.db.GetUser(context.Background(), username)
	if dbFoundUser.Name != username { // user not in database
		fmt.Printf("User '%s' does not exists.\n", username)
		os.Exit(1)
	}

	err := s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("Logged into username:'%v' successfully.\n", username)
	return nil
}

// registers a new user
func handlerRegister(s *state, c command) error {
	targetArgNum := 1
	if err := checkNumArgs(c.arguments, targetArgNum); err != nil {
		return err
	}

	// name processing
	username := c.arguments[0]
	username = strings.ToLower(username)

	// check in the DB for existing user
	dbFoundUser, err := s.db.GetUser(context.Background(), username)
	if dbFoundUser.Name == username {
		fmt.Printf("User '%s' already exists.\n", username)
		os.Exit(1)
	}

	// if username does not exist create a new user in the database
	dbUser, err := s.db.CreateUser(context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      username,
		})
	if err != nil {
		fmt.Printf("Error inserting new user: %s\n", err)
		os.Exit(1)
	}

	// changes to this new user in the config
	s.cfg.SetUser(username)
	fmt.Printf("New user was created: '%s\nUser: %+v\n", username, dbUser)
	return nil
}

// ==================
// COMMAND MANAGEMENT
// ==================

// new commands struct that holds the command map
func newCommands() *commands {
	var cmds commands
	cmds.commands = make(map[string]func(*state, command) error)
	return &cmds
}

// registers a new command handler function.
// added an error return value for uninitialized map.
func (c *commands) registerCommand(name string, f func(*state, command) error) error {
	if c.commands == nil { // uninitialized map
		return fmt.Errorf("Uninitialized map was passed in commands struct.\n")
	}

	c.commands[name] = f
	return nil
}

// runs a given command with the provided state
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

// =========
// MAIN FUNC
// =========

func main() {
	// read saved config from home directory
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error occured: %v\n", err)
		os.Exit(1)
	}

	// opening database connection
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Printf("Error occured: %v", err)
		os.Exit(1)
	}

	// setting up program state
	dbQueries := database.New(db)
	state := state{
		db:  dbQueries,
		cfg: &cfg,
	}

	// registering commands
	cmds := newCommands()
	cmds.registerCommand("help", handlerHelp)
	cmds.registerCommand("login", handlerLogin)
	cmds.registerCommand("register", handlerRegister)

	// processing arguments
	// set to require 2 arguments, command and string
	// e.g. "register <name>", "login <name>"
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

	// runs the command
	err = cmds.run(&state, cmd)
	if err != nil {
		fmt.Printf("command error: %v\n", err)
		os.Exit(1)
	}
}
