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

// =========
// RSS TYPES
// =========

// RSS feed is one feed with information and child items
type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

// One item from a larger RSS feed
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// =============
// RSS FUNCTIONS
// =============

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	return &RSSFeed{}, nil
}

// =============
// UTILITY FUNCS
// =============

// checks for number of arguments
func checkNumArgs(args []string, targetArgNum int) error {
	// early return if the target of arguments is zero
	if targetArgNum == 0 && len(args) == 0 {
		return nil
	}

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
	"help":     "Shows available commands",
	"login":    "Logs into a user",
	"register": "Registers a new user",
	"reset":    "Reset the 'users' table",
	"users":    "Shows a list of all users",
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

// resets database by deleting all records on user table
func handlerReset(s *state, c command) error {
	targetArgs := 0
	if err := checkNumArgs(c.arguments, targetArgs); err != nil {
		return err
	}

	err := s.db.ResetUsers(context.Background())
	if err != nil {
		fmt.Println("Unable to reset 'user' table.")
		return err
	}

	fmt.Println("Reset 'user' table successfully.")
	return nil
}

// shows a list of all users from database,
// as well as the current logged in user
func handlerUsers(s *state, c command) error {
	targetArgs := 0
	if err := checkNumArgs(c.arguments, targetArgs); err != nil {
		return err
	}

	currentName := s.cfg.CurrentUsername

	dbUsers, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("Unable to query list of users in database.")
		return err
	}

	if len(dbUsers) == 0 {
		fmt.Println("There are currently no registered users.")
		return nil
	}

	for _, user := range dbUsers {
		name := user.Name
		if name == currentName {
			name += " (current)"
		}

		fmt.Printf("* %s\n", name)
	}

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
	cmds.registerCommand("reset", handlerReset)
	cmds.registerCommand("users", handlerUsers)

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
