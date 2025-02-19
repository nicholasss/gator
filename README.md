# Gator

Gator is an RSS feed aggregator. Users can follow specific feeds.
Gator was a guided project with directions and instructions from Boot.dev.

## Installation

Required:

- Go 1.23.1 (or above)
- PostgreSQL 17.2 (or above)

Use the following command to install Gator (as written by me):
```go installl github.com/nicholasss/gator```

The help command can be called with `gator help`. This will list all commands.

## Configuration

There is a mandatory configuration file required in order for gator to work.
Here is location where it will search:
`~/.gatorconfig.json`

Here is the expected format of the file:

```json
{
    "db_url": "postgres://database_url",
    "current_user_name": ""
}
```

The `db_url` should be specified, this should point to a valid postgres instance.

The username will get filled in when a user registers with the program.

## Commands

- register: Registers a user with the program, required for new users.
    `<Username>`
- login: Logs into a previously registered user, not required when registering.
    `<Username>`
- addfeed: Adds an RSS feed to begin following.
    `<Name> <URL>`
- agg: Begins aggregating an RSS feed for browsing later.
    `<Duration>` Must specify a time duration between requests, e.g. 30m, 1h, etc.
- browse: Lists out the latest RSS posts that have been aggregated.
    `<Number>` Specify a number of posts to view at once.
