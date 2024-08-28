package main

// TODO: implement /reset and /hardreset cmds

import (
	"context"
	"log"
	"os"
	"ratinger/internal/models/auth"
	"ratinger/internal/repository"
	"ratinger/vuzes/poly"
	"ratinger/vuzes/spbu"

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

type request struct {
	user *auth.User
	job  func(*repository.Repo, *auth.User) []string
}

var repo *repository.Repo

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

	requestsCh := make(chan request)

	for range 3 {
		go worker(bot, requestsCh)
	}

	repo, err = repository.New(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		message := update.Message.Text
		user := auth.GetUserData(repo, update.Message.Chat.ID)

		if message == "/start" {
			sendHello(user.Id, bot)
			continue
		}

		if user.AuthStatus != auth.AUTHED {
			resp, e := user.AddInfo(repo, message)
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

		var job func(*repository.Repo, *auth.User) []string
		switch update.Message.Text {
		case "СПБПУ":
			job = poly.Check
		case "СПБГУ":
			job = spbu.Check
		default:
			unknownCommand(update.Message, bot)
			continue
		}

		requestsCh <- request{
			user: user,
			job:  job,
		}

	}
}
