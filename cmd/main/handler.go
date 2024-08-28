package main

import (
	"fmt"
	"ratinger/internal/models/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func worker(bot *tgbotapi.BotAPI, reqCH chan request) {
	for r := range reqCH {
		msg := tgbotapi.NewMessage(r.user.Id, "Собираю информацию...")
		var del tgbotapi.DeleteMessageConfig

		if message, err := bot.Send(msg); err == nil {
			del = tgbotapi.NewDeleteMessage(r.user.Id, message.MessageID)
		}

		for _, v := range r.job(repo, r.user) {
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
		msg := tgbotapi.NewMessage(id, fmt.Sprintf("Случилась ошибка: %v", err))
		bot.Send(msg)
		return
	}
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
		bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(id, "Готово")
	bot.Send(msg)
}
