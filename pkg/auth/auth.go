package auth

import (
	"context"
	"errors"
	"log"
	"ratinger/model"
	"ratinger/pkg/db"
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

func DeleteUser(id int64) error {
	conn := db.Connect()
	defer conn.Close(context.Background())
	_, err := conn.Exec(context.Background(), "delete from users where id=$1", id)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) AddInfo(msg string) (model.AuthResponse, error) {
	r := model.AuthResponse{}
	db := db.Connect()
	defer db.Close(context.Background())
	var query string
	var err error

	switch u.AuthStatus {

	case NOT_AUTHED:
		if !isValidSnils(msg) {
			return r, ErrExpectedSnils
		}
		u.Snils, u.AuthStatus = msg, AUTHED_WITH_SNILS
		query = "insert into users(id, snils, auth_status) values($1, $2, $3)"
		_, err = db.Exec(context.Background(), query, u.Id, u.Snils, u.AuthStatus)
		r.Message = "Снилс успешно введен!\nТеперь выбери вид оплаты обучения, на которые ты подавал и введи /done когда закончишь"
		r.Markup = "payment"

	case AUTHED_WITH_SNILS:
		if !isValidPayment(msg) {
			if msg == "/done" && u.Payments != nil {
				query = "update users set auth_status=$1 where id=$2"
				u.AuthStatus = AUTHED_WITH_PAYMENTS
				_, err = db.Exec(context.Background(), query, u.AuthStatus, u.Id)
				r.Message = "Сейчас выбери формы поступления. Тут также - напиши /done как закончишь"
				r.Markup = "form"
				return r, err
			}
			return r, ErrExpectedPaymentFrom
		}
		query = "update users set payments = $1 where id=$2"
		u.Payments = append(u.Payments, msg)
		_, err = db.Exec(context.Background(), query, u.Payments, u.Id)

	case AUTHED_WITH_PAYMENTS:
		if !isValidForm(msg) {
			if msg == "/done" && u.Forms != nil {
				query = "update users set auth_status=$1 where id=$2"
				u.AuthStatus = AUTHED_WITH_FORMS
				_, err = db.Exec(context.Background(), query, u.AuthStatus, u.Id)
				r.Message = "И последнее - выбери уровень образования"
				r.Markup = "eduLevel"
				return r, err
			}
			return r, ErrExpectedForm
		}
		query = "update users set forms = $1 where id=$2"
		u.Forms = append(u.Forms, msg)
		_, err = db.Exec(context.Background(), query, u.Forms, u.Id)

	case AUTHED_WITH_FORMS:
		if !isValidEduLevel(msg) {
			return r, ErrExpectedEduLevel
		}
		u.EduLevel, u.AuthStatus = []string{msg}, AUTHED
		if u.EduLevel[0] == "Бакалавриат" {
			u.EduLevel = append(u.EduLevel, "Специалитет")
		}
		query = "update users set edu_level = $1, auth_status = $2 where id=$3"
		_, err = db.Exec(context.Background(), query, u.EduLevel, u.AuthStatus, u.Id)

		r.Message = "Отлично! Теперь ты можешь пользоваться ботом!\nПросто кликай на нужный вуз и наблюдай за своими позициями\nЕсли ты поменял что-то из введенных данных - введи команду /hardreset\nА если бот не находит тебя в списках попробуй ввести команду /reset\nПервый поиск может занять некоторое время, но дальше все будет быстрее"
		r.Markup = "vuzes"
	}

	return r, err
}

func GetUserData(id int64) *User {
	conn := db.Connect()
	defer conn.Close(context.Background())
	row, err := conn.Query(context.Background(), "select snils, payments, forms, vuzes, spbstu, spbu, auth_status, edu_level from users where id=$1", id)
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
		return &u
	}
	return &u
}
