package bot

import (
	"log/slog"
	"os"
	"ratinger/internal/bot/logger"
	"ratinger/internal/models/auth"
	"ratinger/internal/repository"
	"ratinger/vuzes/poly"
	"ratinger/vuzes/spbu"
	"sync"

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

var (
	repo    *repository.Repo
	log     *slog.Logger
	botOnce sync.Once
	bot     *tgbotapi.BotAPI
)

func New(log *slog.Logger) *tgbotapi.BotAPI {
	botOnce.Do(
		func() {
			godotenv.Load()
			token := os.Getenv("TOKEN")
			var err error
			bot, err = tgbotapi.NewBotAPI(token)
			if err != nil {
				panic(err)
			}
			if err := tgbotapi.SetLogger(logger.New(log)); err != nil {
				panic(err)
			}
			bot.Debug = true
		},
	)
	return bot
}

func Start(bot *tgbotapi.BotAPI, r *repository.Repo, lg *slog.Logger) {
	log = lg
	repo = r

	requestsCh := make(chan workerRequest)

	for range 3 {
		go worker(bot, requestsCh)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 600

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
			resp, e := user.AddInfo(repo, log, message)
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

		var job func(*repository.Repo, *slog.Logger, *auth.User) []string
		switch update.Message.Text {
		case "СПБПУ":
			job = poly.Check
		case "СПБГУ":
			job = spbu.Check
		default:
			unknownCommand(update.Message, bot)
			continue
		}

		requestsCh <- workerRequest{
			user: user,
			job:  job,
		}

	}
}
