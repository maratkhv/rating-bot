package bot

import (
	"fmt"
	"log/slog"
	"ratinger/internal/models/auth"
	"ratinger/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type workerRequest struct {
	user *auth.User
	job  func(*repository.Repo, *slog.Logger, *auth.User) []string
}

func worker(bot *tgbotapi.BotAPI, reqCH chan workerRequest) {
	for r := range reqCH {
		msg := tgbotapi.NewMessage(r.user.Id, "Собираю информацию...")
		var del tgbotapi.DeleteMessageConfig

		if message, err := bot.Send(msg); err == nil {
			del = tgbotapi.NewDeleteMessage(r.user.Id, message.MessageID)
		}

		for _, v := range r.job(repo, logger, r.user) {
			msg := tgbotapi.NewMessage(r.user.Id, v)
			bot.Send(msg)
		}

		bot.Request(del)
	}
}

func sendHello(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Привет!\nЭто бот поможет Тебе быстро найти себя в списках, а также покажет полезную информацию")
	bot.Send(msg)
	msg.Text = "Для работы бота необходим Твой СНИЛС. Введи его в формате 555-666-777 89"
	bot.Send(msg)
}

func reset(id int64, bot *tgbotapi.BotAPI) {
	err := auth.DeleteUser(repo, id)
	if err != nil {

		logger.Error(
			"failed to reset",
			slog.Any("error", err),
			slog.Int64("user_id", id),
		)

		msg := tgbotapi.NewMessage(id, fmt.Sprintf("Случилась ошибка: %v", err))
		bot.Send(msg)
		return
	}

	logger.Debug("successfully reset", slog.Int64("user id", id))

	msg := tgbotapi.NewMessage(id, "Введите новый СНИЛС")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	bot.Send(msg)
}

func handleCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	switch msg.Command() {
	case "reset":
		reset(msg.Chat.ID, bot)
	case "refresh":
		refresh(msg.Chat.ID, bot)
	default:
		unknownCommand(msg, bot)
	}
}

func unknownCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	response := tgbotapi.NewMessage(msg.Chat.ID, "?")
	response.ReplyToMessageID = msg.MessageID
	bot.Send(response)
}

func refresh(id int64, bot *tgbotapi.BotAPI) {
	err := auth.RefreshVuzes(repo, id)
	if err != nil {
		msg := tgbotapi.NewMessage(id, fmt.Sprintf("Случилась ошибка: %v", err))

		logger.Error(
			"failed to refresh",
			slog.Any("error", err),
			slog.Int64("user_id", id),
		)

		bot.Send(msg)
		return
	}

	logger.Debug("successfully refresh", slog.Int64("user id", id))

	msg := tgbotapi.NewMessage(id, "Готово")
	bot.Send(msg)
}
