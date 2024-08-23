package main

import (
	"log"
	"os"

	"ratinger/internal/poly"
	"ratinger/internal/spbu"
	"ratinger/pkg/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var vuzesKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("СПБПУ"),
		tgbotapi.NewKeyboardButton("СПБГУ"),
	),
)

var formKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Очная"),
		tgbotapi.NewKeyboardButton("Очно-заочная"),
		tgbotapi.NewKeyboardButton("Заочная"),
	),
)

var eduLevelKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Бакалавриат"),
		tgbotapi.NewKeyboardButton("Магистратура"),
		tgbotapi.NewKeyboardButton("Аспирантура"),
	),
)

var paymentKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Контракт"),
		tgbotapi.NewKeyboardButton("Бюджет"),
		tgbotapi.NewKeyboardButton("Целевое"),
	),
)

var markup = map[string]*tgbotapi.ReplyKeyboardMarkup{
	"payment":  &paymentKeyboard,
	"form":     &formKeyboard,
	"eduLevel": &eduLevelKeyboard,
	"vuzes":    &vuzesKeyboard,
}

func main() {
	godotenv.Load()
	token := os.Getenv("TOKEN")
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 600

	updates := bot.GetUpdatesChan(u)
	for update := range updates {

		if update.Message != nil {
			message := update.Message.Text
			user := auth.GetUserData(update.Message.Chat.ID)

			if message == "/start" {
				sendHello(user.Id, bot)
				continue
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
				continue
			}

			if update.Message.IsCommand() {
				handleCommand(update.Message, bot)
				continue
			}

			var handl func(*auth.User) []string
			switch update.Message.Text {
			case "СПБПУ":
				handl = poly.Check
			case "СПБГУ":
				handl = spbu.Check
			default:
				unknownCommand(update.Message, bot)
				continue
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
}
