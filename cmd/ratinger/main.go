package main

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
