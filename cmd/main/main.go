package main

// TODO: implement /reset and /hardreset cmds

import (
	"log"
	"os"

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

	workCh := make(chan tgbotapi.Update)

	for range 3 {
		go worker(bot, workCh)
	}

	updates := bot.GetUpdatesChan(u)

	for {
		workCh <- <-updates
	}
}
