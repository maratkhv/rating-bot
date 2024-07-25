package main

import (
	"log"
	"os"
	"ratinger/internal/leti"
	"ratinger/internal/poly"
	"ratinger/internal/spbu"
	"ratinger/pkg/auth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var usersData = auth.InitUserData()

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("ЛЭТИ"),
		tgbotapi.NewKeyboardButton("СПБПУ"),
		tgbotapi.NewKeyboardButton("СПБГУ"),
	),
)

func main() {
	defer usersData.Db.Close()
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

			id := update.Message.Chat.ID

			snils := usersData.GetSnils(id)

			if update.Message.IsCommand() {
				handleCommand(update.Message, bot)
				continue
			}

			if snils == "" {
				if auth.IsValidSnils(update.Message.Text) {
					usersData.AddUser(id, update.Message.Text)
					msg := tgbotapi.NewMessage(id, "СНИЛС успешно установлен\nТеперь выбери интересующий тебя вуз\n\nЧтобы сменить снилс введите команду /reset")
					msg.ReplyMarkup = numericKeyboard
					bot.Send(msg)
				} else {
					msg := tgbotapi.NewMessage(id, "Сначала введите ваш СНИЛС")
					bot.Send(msg)
				}
				continue
			}

			var handl func(string) []string
			switch update.Message.Text {
			case "ЛЭТИ":
				handl = leti.Check
			case "СПБПУ":
				handl = poly.Check
			case "СПБГУ":
				handl = spbu.Check
			default:
				unknownCommand(update.Message, bot)
				continue
			}

			msg := tgbotapi.NewMessage(id, "Собираю информацию...")
			bot.Send(msg)
			for _, v := range handl(snils) {
				msg := tgbotapi.NewMessage(id, v)
				bot.Send(msg)
			}

		}
	}
}

func sendHello(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Привет!\nЭто бот поможет Тебе быстро найти себя в списках, а также покажет полезную информацию")
	bot.Send(msg)
	msg.Text = "Для работы бота необходим Твой СНИЛС. Введи его в формате 555-666-777 89"
	bot.Send(msg)
}

func reset(chatID int64, bot *tgbotapi.BotAPI) {
	usersData.DeleteUser(chatID)
	msg := tgbotapi.NewMessage(chatID, "Введите новый СНИЛС")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	bot.Send(msg)
}

func handleCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	switch msg.Command() {
	case "start":
		sendHello(msg.Chat.ID, bot)
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
