package main

// TODO: this whole package needs to be rewritten but after spbu i guess

import (
	"log"
	"os"

	// "ratinger/internal/leti"

	"ratinger/internal/poly"
	// "ratinger/internal/spbu"
	"ratinger/pkg/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("ЛЭТИ"),
		tgbotapi.NewKeyboardButton("СПБПУ"),
		tgbotapi.NewKeyboardButton("СПБГУ"),
	),
)

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

			if user.AuthStatus != auth.AUTHED {
				e := user.AddInfo(message)
				if e != nil {
					msg := tgbotapi.NewMessage(user.Id, e.Error())
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
			case "ЛЭТИ":
				// handl = leti.Check
			case "СПБПУ":
				handl = poly.Check
			case "СПБГУ":
				// handl = spbu.Check
			default:
				unknownCommand(update.Message, bot)
				continue
			}

			msg := tgbotapi.NewMessage(user.Id, "Собираю информацию...")
			bot.Send(msg)
			for _, v := range handl(user) {
				msg := tgbotapi.NewMessage(user.Id, v)
				bot.Send(msg)
			}

		}
	}
}

// TODO: these should go to diff file i guess?

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
