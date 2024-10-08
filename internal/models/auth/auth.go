package auth

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"ratinger/internal/repository"
)

const (
	NOT_AUTHED           int8 = 0
	AUTHED_WITH_SNILS    int8 = 1
	AUTHED_WITH_PAYMENTS int8 = 2
	AUTHED_WITH_FORMS    int8 = 3
	AUTHED               int8 = 4
)

var (
	ErrExpectedSnils       error = errors.New("expected snils")
	ErrExpectedForm        error = errors.New("expected form or /done")
	ErrExpectedPaymentFrom error = errors.New("expected payment form or /done")
	ErrExpectedEduLevel    error = errors.New("expected edu level")
)

type User struct {
	Id         int64
	AuthStatus int8
	Snils      string
	Forms      []string
	Payments   []string
	EduLevel   []string

	// reserved
	Vuzes []string

	// these are lists of naprav's IDs where user was found
	Spbstu []int
	Spbu   []int
}

type response struct {
	Message string
	Markup  string
	// TODO: use this instead of returning error in AddInfo
	Error error
}

func DeleteUser(repo *repository.Repo, id int64) error {
	err := repo.Db.DeleteUser(context.Background(), id)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) AddInfo(repo *repository.Repo, logger *slog.Logger, msg string) (response, error) {
	var r response
	var err error

	switch u.AuthStatus {

	case NOT_AUTHED:
		if !isValidSnils(msg) {
			return r, ErrExpectedSnils
		}
		u.Snils, u.AuthStatus = msg, AUTHED_WITH_SNILS
		err = repo.Db.InsertUser(context.Background(), u.Id, u.Snils, u.AuthStatus)
		r.Message = "Снилс успешно введен!\nТеперь выбери вид оплаты обучения, на которые ты подавал и введи /done когда закончишь"
		r.Markup = "payment"
		logger.Debug("user authed with snils", slog.String("snils", u.Snils), slog.Int64("user_id", u.Id))

	case AUTHED_WITH_SNILS:
		if !isValidPayment(msg) {
			if msg == "/done" && u.Payments != nil {
				u.AuthStatus = AUTHED_WITH_PAYMENTS
				args := repository.Args{
					"auth_status": u.AuthStatus,
				}
				err = repo.Db.UpdateUser(context.Background(), u.Id, args)
				r.Message = "Сейчас выбери формы поступления. Тут также - напиши /done как закончишь"
				r.Markup = "form"

				logger.Debug("user authed with payments", slog.Any("payments", u.Payments), slog.Int64("user_id", u.Id))

				return r, err
			}
			return r, ErrExpectedPaymentFrom
		}
		u.Payments = append(u.Payments, msg)
		args := repository.Args{
			"payments": u.Payments,
		}
		err = repo.Db.UpdateUser(context.Background(), u.Id, args)

	case AUTHED_WITH_PAYMENTS:
		if !isValidForm(msg) {
			if msg == "/done" && u.Forms != nil {
				u.AuthStatus = AUTHED_WITH_FORMS
				args := repository.Args{
					"auth_status": u.AuthStatus,
				}
				err = repo.Db.UpdateUser(context.Background(), u.Id, args)
				r.Message = "И последнее - выбери уровень образования"
				r.Markup = "eduLevel"

				logger.Debug("user authed with forms", slog.Any("forms", u.Forms), slog.Int64("user_id", u.Id))

				return r, err
			}
			return r, ErrExpectedForm
		}
		u.Forms = append(u.Forms, msg)
		args := repository.Args{
			"forms": u.Forms,
		}
		err = repo.Db.UpdateUser(context.Background(), u.Id, args)

	case AUTHED_WITH_FORMS:
		if !isValidEduLevel(msg) {
			return r, ErrExpectedEduLevel
		}
		u.EduLevel, u.AuthStatus = []string{msg}, AUTHED
		if u.EduLevel[0] == "Бакалавриат" {
			u.EduLevel = append(u.EduLevel, "Специалитет")
		}
		args := repository.Args{
			"edu_level":   u.EduLevel,
			"auth_status": u.AuthStatus,
		}
		err = repo.Db.UpdateUser(context.Background(), u.Id, args)

		logger.Debug("user authed with edu_level", slog.Any("edu_level", u.EduLevel), slog.Int64("user_id", u.Id))
		logger.Info("user authed", slog.Int64("user_id", u.Id), slog.Any("user data", u))

		r.Message = "Отлично! Теперь ты можешь пользоваться ботом!\nПросто кликай на нужный вуз и наблюдай за своими позициями\nЕсли ты поменял что-то из введенных данных - введи команду /reset\nА если бот не находит тебя в списках попробуй ввести команду /refresh\nПервый поиск может занять некоторое время, но дальше все будет быстрее"
		r.Markup = "vuzes"
	}

	return r, err
}

func GetUserData(repo *repository.Repo, id int64) *User {
	row, err := repo.Db.SelectQuery(context.Background(), "select snils, payments, forms, vuzes, spbstu, spbu, auth_status, edu_level from users where id=$1", id)
	if err != nil {
		log.Fatalf("query err: %v", err)
	}
	defer row.Close()

	var u User
	u.Id = id

	if row.Next() {
		err = row.Scan(&u.Snils, &u.Payments, &u.Forms, &u.Vuzes, &u.Spbstu, &u.Spbu, &u.AuthStatus, &u.EduLevel)
		if err != nil {
			log.Fatalf("scan err: %v", err)
		}
	}
	return &u
}

func RefreshVuzes(repo *repository.Repo, id int64) error {
	err := repo.Db.RefreshVuzes(context.Background(), id)
	return err
}
