package main

/*
TODO:
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
	"log/slog"
	"os"
	"ratinger/internal/bot"
	"ratinger/internal/repository"
)

func main() {
	logger := initLogger()

	repo, err := repository.New(context.Background())
	if err != nil {
		logger.Error("failed to connect to storage", slog.Any("error", err))
	}

	logger.Info("starting bot")

	bot.Start(bot.New(logger), repo, logger)
}

func initLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	))
}
