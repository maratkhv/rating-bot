package main

import (
	"ratinger/pkg/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendHello(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Привет!\nЭто бот поможет Тебе быстро найти себя в списках, а также покажет полезную информацию")
	bot.Send(msg)
	msg.Text = "Для работы бота необходим Твой СНИЛС. Введи его в формате 555-666-777 89"
	bot.Send(msg)
}

func reset(chatID int64, bot *tgbotapi.BotAPI) {
	auth.DeleteUser(chatID)
	msg := tgbotapi.NewMessage(chatID, "Введите новый СНИЛС")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	bot.Send(msg)
}

func handleCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	switch msg.Command() {
	case "reset":
		reset(msg.Chat.ID, bot)
	default:
		unknownCommand(msg, bot)
	}
}

func unknownCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	response := tgbotapi.NewMessage(msg.Chat.ID, "?")
	response.ReplyToMessageID = msg.MessageID
	bot.Send(response)
}
