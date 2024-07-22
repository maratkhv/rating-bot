package main

import (
	"log"
	"os"

	tgb "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	token := os.Getenv("TOKEN")
	bot, err := tgb.NewBotAPI(token)
	if err != nil {
		log.Panic(err, token)
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgb.NewUpdate(0)
	u.Timeout = 600

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			msg := tgb.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}
	}
}
