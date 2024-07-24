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

var usersData = make(map[int64]string)

// add autoremove from waiters if not responding for too long
var authWaiters = auth.NewWaiter()

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

			id := update.Message.Chat.ID

			if _, ok := authWaiters.List[id]; ok && auth.IsValidSnils(update.Message.Text) {
				delete(authWaiters.List, id)
				usersData[id] = update.Message.Text
				msg := tgbotapi.NewMessage(id, "СНИЛС успешно установлен\nЧтобы сменить снилс используйте комманду /restart")
				bot.Send(msg)
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
			case "/start":
				sendHello(id, bot)
				continue
			default:
				msg := tgbotapi.NewMessage(id, "?")
				bot.Send(msg)
				continue
			}

			snils, ok := usersData[id]

			if !ok {
				msg := tgbotapi.NewMessage(id, "Сначала введите ваш СНИЛС")
				bot.Send(msg)
				continue
			}

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
	authWaiters.List[chatID] = struct{}{}
}

func restart(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "В разработке")
	bot.Send(msg)
}
