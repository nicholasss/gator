package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nicholasss/gator/internal/config"
	"github.com/nicholasss/gator/internal/database"

	// imported postgres driver for side effects
	"github.com/lib/pq"
)

// header agent-identifier
const agent = "gator"

// PostgreSQL Error Codes
const (
	UniqueViolationErr = pq.ErrorCode("23505")
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
		Items       []RSSItem `xml:"item"`
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

// fetches an rss feed from given URL and returns a reference to it
func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &RSSFeed{}, err
	}

	req.Header.Set("User-Agent", agent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return &RSSFeed{}, err
	}
	defer res.Body.Close()

	rawFeed, err := io.ReadAll(res.Body)
	if err != nil {
		return &RSSFeed{}, err
	}

	var feed RSSFeed
	err = xml.Unmarshal(rawFeed, &feed)
	if err != nil {
		return &RSSFeed{}, err
	}

	// removing html artifacts
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i := range feed.Channel.Items {
		// has to be the items index to reference
		// if its the item (e.g. i, item :=) then
		// it is only modifying a copy of the item
		feed.Channel.Items[i].Title = html.UnescapeString(feed.Channel.Items[i].Title)
		feed.Channel.Items[i].Description = html.UnescapeString(feed.Channel.Items[i].Description)
	}

	return &feed, nil
}

// =============
// UTILITY FUNCS
// =============

// checks for number of arguments
func checkNumArgs(args []string, targetArgNum int) error {
	numArgs := len(args)
	if numArgs == targetArgNum {
		return nil
	}

	if numArgs > targetArgNum {
		return fmt.Errorf("too many arguments were provided. Needs %d\n", targetArgNum)
	} else if numArgs < targetArgNum {
		return fmt.Errorf("not enough arguments were provided. Need %d\n", targetArgNum)
	}

	return fmt.Errorf("error processing arguments in main.go:checkNumArgs()")
}

func scrapeFeeds(s *state) error {
	// TODO: need to take another look at the query to ensure that it does not show feeds that user does not follow
	feedRecord, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("scraping feeds error fetching feed list from db: %w", err)
	}

	// mark it as fetched
	now := sql.NullTime{}
	now.Scan(time.Now())

	err = s.db.MarkFeedFetched(context.Background(), database.MarkFeedFetchedParams{
		ID:            feedRecord.ID,
		LastFetchedAt: now,
	})
	if err != nil {
		return fmt.Errorf("scraping feeds error updating feed as fetched: %w", err)
	}

	log.Printf("Fetching '%s' feed at '%s'.\n", feedRecord.Name, feedRecord.Url)
	RSSItems, err := fetchFeed(context.Background(), feedRecord.Url)
	if err != nil {
		return fmt.Errorf("scraping feeds error fetching feeds: %w", err)
	}

	for _, item := range RSSItems.Channel.Items {
		title := item.Title
		if title == "" {
			title = "[NO TITLE]"
		}

		description := sql.NullString{}
		err := description.Scan(item.Description)
		if err != nil {
			log.Printf("post description to NullString error: %s\n", err)
		}

		publishedTime, err := timeDecode(item.PubDate)
		if err != nil {
			log.Fatalf("Error decoding time format provided: %s", err)
		}
		publishedAt := sql.NullTime{}
		err = publishedAt.Scan(publishedTime)
		if err != nil {
			log.Printf("post publishedAt to NullTime error: %s\n", err)
		}

		log.Printf("saving post '%s' to database\n", title)

		// save the item to the database
		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       title,
			Url:         item.Link,
			Description: description,
			PublishedAt: publishedAt,
			FeedID:      feedRecord.ID,
		})
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == UniqueViolationErr {
				log.Printf("Post was already added based on its URL.\n")
				continue
			}

		} else if err != nil {
			log.Printf("error inserting to posts table: %s\n", err)
		}

	}

	return nil
}

// time decoding function to try multiple different formats.
func timeDecode(str string) (time.Time, error) {
	// RFC1123Z format
	val, err := time.Parse(time.RFC1123Z, str)
	if err == nil {
		return val, nil
	}
	// UnixDate format
	val, err = time.Parse(time.UnixDate, str)
	if err == nil {
		return val, nil
	}
	// ANSIC format
	val, err = time.Parse(time.ANSIC, str)
	if err == nil {
		return val, nil
	}
	// DateTime format
	val, err = time.Parse(time.DateTime, str)
	if err == nil {
		return val, nil
	}
	// RFC822Z
	val, err = time.Parse(time.RFC822Z, str)
	if err == nil {
		return val, nil
	}

	return time.Time{}, fmt.Errorf("unknown time format: %s", str)
}

// ==========
// MIDDLEWARE
// ==========

// Allows for all handlers that require a logged in user to to accept them as an argument.
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	// first the outter function runs when registering a command
	// then it will run the inner code when the command is executed
	// and the original function will be called finally with the enriched info
	return func(s *state, c command) error {

		username := s.cfg.CurrentUsername
		userRecord, err := s.db.GetUserByName(context.Background(), username)

		if err == sql.ErrNoRows {
			fmt.Println("There does not appear to be any registered users.")
			fmt.Println("Please ensure that you are registered and logged in.")
			os.Exit(1)
		} else if err != nil {
			return fmt.Errorf("middlewareLoggedIn error fetching user by name: %w", err)
		}

		return handler(s, c, userRecord)
	}
}

// ================
// COMMAND HANDLERS
// ================

// list of valid command handlers
var validCommands map[string]string = map[string]string{
	"addfeed":   "Adds a new feed and follows it. Requires a Name & URL.",
	"agg":       "Begins aggregation of feeds.\n   Provide an time interval to wait between each feed.\n   e.g. 30m, 1h, etc.",
	"browse":    "Browse the downloaded posts from the feeds you follow.\n   Provide an int as a limit of posts.\n   e.g. 1, 5, 20, etc.",
	"feeds":     "Shows a list of all feeds.",
	"follow":    "Follow a feed by its URL.",
	"following": "Shows a list of all feeds the current user is following.",
	"help":      "Shows available commands.",
	"login":     "Logs into a user. Requires a Name.",
	"register":  "Registers a new user. Requires a Name.",
	"reset":     "Reset the 'users' and the 'feeds' table",
	"unfollow":  "Unfollow a feed by its URL.",
	"users":     "Shows a list of all registered users.",
}

// add feed command
func handlerAddFeed(s *state, c command, user database.User) error {
	if err := checkNumArgs(c.arguments, 2); err != nil {
		return fmt.Errorf("handlerAddFeed was passed wrong number of arguments, expected 2: %w", err)
	}

	userID := user.ID
	name := c.arguments[0]
	URL := c.arguments[1]

	newFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       URL,
		UserID:    userID,
	})
	if pqErr, ok := err.(*pq.Error); ok {
		if pqErr.Code == UniqueViolationErr {
			log.Printf("Feed has already been added.\n")
			return nil
		}
	} else if err != nil {
		return fmt.Errorf("handlerAddFeed error inserting new feed: %w", err)
	}

	feedFollowRecord, err := s.db.CreateFeedFollow(context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    userID,
			FeedID:    newFeed.ID,
		})

	fmt.Printf("%s is following: %s\n", user.Name, feedFollowRecord.FeedName)
	return nil
}

// Command to run in another terminal, will fetch the feeds in the background.
// This function needs to be explicitly terminated.
func handlerAgg(s *state, c command) error {
	if err := checkNumArgs(c.arguments, 1); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// takes a duration string as an argument
	// e.g. 1h, 1m, 30m, etc.
	durationString := c.arguments[0]
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return fmt.Errorf("handler agg unable to parse duration string: %w", err)
	}
	log.Printf("Collecting feeds every %s\n", duration.String())

	// sets up a ticker to execute the scraping
	ticker := time.NewTicker(duration)
	for ; ; <-ticker.C {

		err := scrapeFeeds(s)
		if err != nil {
			return err
		}

		log.Printf("Waiting %s to fetch next feed.\n", duration.String())
	}
}

// browse the downloaded posts.
func handlerBrowse(s *state, c command, user database.User) error {
	var limit int
	if err := checkNumArgs(c.arguments, 1); err != nil {
		limit = 2
	} else {
		limit, err = strconv.Atoi(c.arguments[0])
		if err != nil {
			return err
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int64(limit),
	})
	if err != nil {
		log.Fatalf("Unable to fetch posts from database: %s", err)
	}

	fmt.Printf("Showing %d posts:\n", limit)
	for _, post := range posts {
		fmt.Printf(" * %s * \n", post.Title)
		fmt.Printf(" * Published at: %s\n", post.PublishedAt.Time.String())
		fmt.Printf("%s\n\n", post.Description.String)
	}

	return nil
}

// prints out a list of feeds in the database
func handlerFeeds(s *state, c command) error {
	if err := checkNumArgs(c.arguments, 0); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	feeds, err := s.db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("handlerFeeds error fetching all feeds: %w", err)
	}

	fmt.Printf("List of all feeds in the database:\n")

	for i, feed := range feeds {
		user, err := s.db.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("unable to get user by id: %w", err)
		}

		fmt.Printf("Feed #%d:\n", i+1)
		fmt.Printf(" - User: %s\n", user.Name)
		fmt.Printf(" - Name: %s\n", feed.Name)
		fmt.Printf(" - URL:  %s\n", feed.Url)
		fmt.Printf("\n")
	}

	return nil
}

// as the current user, follows a feed
// prints the name of the feed and the current user
func handlerFollow(s *state, c command, user database.User) error {
	if err := checkNumArgs(c.arguments, 1); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	URL := c.arguments[0]
	feedRecord, err := s.db.GetFeedByURL(context.Background(), URL)
	if err == sql.ErrNoRows {
		fmt.Printf("Unable to find the feed by URL.\n")
		fmt.Printf("You may need to add the feed first, with 'addfeed'.\n")
		os.Exit(1)
	} else if err != nil {
		return fmt.Errorf("handlerfollow error fetching feed by url: %w", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    feedRecord.ID,
		})
	if err != nil {
		return fmt.Errorf("handlerFollow error creating feed follow record: %w", err)
	}

	fmt.Printf("User %s is following\n", user.Name)
	fmt.Printf("feed %s\n", feedRecord.Url)

	return nil
}

// prints out a list of all feeds the current user is following
func handlerFollowing(s *state, c command, user database.User) error {
	if err := checkNumArgs(c.arguments, 0); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	feedFollowRecords, err := s.db.GetFeedFollowForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("handlerFollowing error fetching users feeds by user id: %w", err)
	}

	fmt.Printf("User %s is following:\n", user.Name)
	for _, feedFollowRecord := range feedFollowRecords {
		feedRecord, err := s.db.GetFeedByID(context.Background(), feedFollowRecord.FeedID)
		if err != nil {
			return fmt.Errorf("unable to retrieve feed record: %w", err)
		}

		fmt.Printf(" - %s\n", feedRecord.Name)
	}
	return nil
}

// prints out valid commands
func handlerHelp(_ *state, c command) error {
	if err := checkNumArgs(c.arguments, 0); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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
	if err := checkNumArgs(c.arguments, 1); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	username := c.arguments[0]
	username = strings.ToLower(username)

	// check database for user
	dbFoundUser, _ := s.db.GetUserByName(context.Background(), username)
	if dbFoundUser.Name != username { // user not in database
		fmt.Printf("User '%s' does not exists.\n", username)
		os.Exit(1)
	}

	err := s.cfg.SetUser(username)
	if err != nil {
		return fmt.Errorf("handlerLogin error setting username in config: %w", err)
	}

	fmt.Printf("Logged into username:'%v' successfully.\n", username)
	return nil
}

// registers a new user
func handlerRegister(s *state, c command) error {
	if err := checkNumArgs(c.arguments, 1); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// name processing
	username := c.arguments[0]
	username = strings.ToLower(username)

	// check in the DB for existing user
	dbFoundUser, err := s.db.GetUserByName(context.Background(), username)
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
	fmt.Printf("New user was created: '%s'\nUser: %+v\n", username, dbUser)
	return nil
}

// resets database by deleting all records on user table
// this will delete the records in the feeds table as well.
func handlerReset(s *state, c command) error {
	if err := checkNumArgs(c.arguments, 0); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("handlerReset was unable to reset the users table: %w", err)
	}

	fmt.Println("Reset 'user' table successfully.")
	return nil
}

// unfollows a particular feed
// TODO: does not remove the feed from being aggregated?
func handlerUnfollow(s *state, c command, user database.User) error {
	if err := checkNumArgs(c.arguments, 1); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// URL assumed to be first item in list
	URL := c.arguments[0]
	feedRecord, err := s.db.GetFeedByURL(context.Background(), URL)
	if err == sql.ErrNoRows {
		fmt.Printf("Unable to find the feed by URL.\n")
		os.Exit(1)
	} else if err != nil {
		return fmt.Errorf("handlerUnfollow error fetching feed record: %w", err)
	}

	_, err = s.db.DeleteFeedFollowForUserURL(
		context.Background(),
		database.DeleteFeedFollowForUserURLParams{
			UserID: user.ID,
			FeedID: feedRecord.ID,
		})
	if err != nil {
		return fmt.Errorf("handlerUnfollow error deleting feed_follow record: %w", err)
	}

	fmt.Printf("Unfollowed '%s' successfully.\n", feedRecord.Name)

	return nil
}

// shows a list of all users from database,
// as well as the current logged in user
func handlerUsers(s *state, c command) error {
	if err := checkNumArgs(c.arguments, 0); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	currentName := s.cfg.CurrentUsername

	dbUsers, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("unable to query list of users in database: %w", err)
	}

	if len(dbUsers) == 0 {
		fmt.Println("There are currently no registered users.")
		fmt.Println("You may need to register first with 'register'.")
		os.Exit(1)
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
		// the command is either not registered or not valid.
		return fmt.Errorf("%v is not a valid command.\n", cmd.name)
	}

	err := handlerFunc(s, cmd)
	if err != nil {
		// pass up the error
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
	cmds.registerCommand("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.registerCommand("agg", handlerAgg)
	cmds.registerCommand("browse", middlewareLoggedIn(handlerBrowse))
	cmds.registerCommand("feeds", handlerFeeds)
	cmds.registerCommand("follow", middlewareLoggedIn(handlerFollow))
	cmds.registerCommand("following", middlewareLoggedIn(handlerFollowing))
	cmds.registerCommand("help", handlerHelp)
	cmds.registerCommand("login", handlerLogin)
	cmds.registerCommand("register", handlerRegister)
	cmds.registerCommand("reset", handlerReset)
	cmds.registerCommand("unfollow", middlewareLoggedIn(handlerUnfollow))
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
