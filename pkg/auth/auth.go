package auth

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const connString = "postgres://postgres:42136620@localhost:5432/ratingerdb"

type UserData struct {
	Db *pgxpool.Pool
}

func (u *UserData) DeleteUser(id int64) error {
	_, err := u.Db.Exec(context.Background(), "delete from users where id=$1", id)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserData) AddUser(id int64, snils string) {
	_, err := u.Db.Query(context.Background(), "insert into users(id, snils) values($1,$2)", id, snils)
	if err != nil {
		log.Fatalln(err)
	}

}

func (u *UserData) GetSnils(id int64) string {
	row, err := u.Db.Query(context.Background(), "select snils from users where id=$1", id)
	if err != nil {
		log.Fatalf("query err: %v", err)
	}
	defer row.Close()

	var (
		snils string
	)

	if row.Next() {
		err = row.Scan(&snils)
		if err != nil {
			log.Fatalf("scan err: %v", err)
		}
		return snils
	}
	return snils
}

func IsValidSnils(s string) bool {
	if len(s) == 14 {
		for i := range s {
			switch i {
			case 3, 7:
				if s[i] != '-' {
					return false
				}
			case 11:
				if s[i] != ' ' {
					return false
				}
			default:
				if !strings.Contains("0123456789", string(s[i])) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func InitUserData() *UserData {
	conn, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalln(err)
	}
	return &UserData{Db: conn}
}
