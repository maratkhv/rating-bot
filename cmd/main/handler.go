package main

import (
	"ratinger/internal/poly"
	"ratinger/internal/spbu"
	"ratinger/pkg/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func worker(bot *tgbotapi.BotAPI, workCh chan tgbotapi.Update) {
	for update := range workCh {
		if update.Message == nil {
			return
		}

		message := update.Message.Text
		user := auth.GetUserData(update.Message.Chat.ID)

		if message == "/start" {
			sendHello(user.Id, bot)
			return
		}

		if user.AuthStatus != auth.AUTHED {
			resp, e := user.AddInfo(message)
			if e != nil {
				msg := tgbotapi.NewMessage(user.Id, e.Error())
				bot.Send(msg)
			}
			if resp.Message != "" {
				msg := tgbotapi.NewMessage(user.Id, resp.Message)
				if resp.Markup != "" {
					msg.ReplyMarkup = markup[resp.Markup]
				}
				bot.Send(msg)
			}
			return
		}

		var handl func(*auth.User) []string
		switch update.Message.Text {
		case "СПБПУ":
			handl = poly.Check
		case "СПБГУ":
			handl = spbu.Check
		default:
			unknownCommand(update.Message, bot)
			return
		}

		msg := tgbotapi.NewMessage(user.Id, "Собираю информацию...")
		var del tgbotapi.DeleteMessageConfig
		if message, err := bot.Send(msg); err == nil {
			del = tgbotapi.NewDeleteMessage(user.Id, message.MessageID)
		}
		for _, v := range handl(user) {
			msg := tgbotapi.NewMessage(user.Id, v)
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

// func reset(chatID int64, bot *tgbotapi.BotAPI) {
// 	auth.DeleteUser(chatID)
// 	msg := tgbotapi.NewMessage(chatID, "Введите новый СНИЛС")
// 	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
// 	bot.Send(msg)
// }

// func handleCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
// 	switch msg.Command() {
// 	case "reset":
// 		reset(msg.Chat.ID, bot)
// 	default:
// 		unknownCommand(msg, bot)
// 	}
// }

func unknownCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	response := tgbotapi.NewMessage(msg.Chat.ID, "?")
	response.ReplyToMessageID = msg.MessageID
	bot.Send(response)
}
