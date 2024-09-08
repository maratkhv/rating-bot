package main

/*
TODO:
 - rename dir to /bot
 - add logger
 - add error handling (pretty much the same as logger)
prolly use logger as an argument to .Check() /mb as config struct??/
and think of what to do with auth /config sounds good tho/
 - use methods for repository  instead of repo.Db methods which use them unexported so you chache and insert to db at once
 - write tests
 - use github actions for deploy and gh secrets for bot token
*/

import (
	"context"
	"log"
	"ratinger/internal/bot"
	"ratinger/internal/repository"
)

func main() {
	repo, err := repository.New(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	bot.Start(bot.New(), repo)
}
