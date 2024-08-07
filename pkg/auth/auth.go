package auth

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
)

var PSWD = os.Getenv("PSWD")

var connString = "postgresql://myneondb_owner:" + PSWD + "@ep-shiny-sun-a2swl5c2.eu-central-1.aws.neon.tech/myneondb?sslmode=require"

func DeleteUser(id int64) error {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	_, err = conn.Exec(context.Background(), "delete from users where id=$1", id)
	if err != nil {
		return err
	}
	return nil
}

func AddUser(id int64, snils string) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	_, err = conn.Query(context.Background(), "insert into users(id, snils) values($1,$2)", id, snils)
	if err != nil {
		log.Fatalln(err)
	}

}

func GetSnils(id int64) string {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	row, err := conn.Query(context.Background(), "select snils from users where id=$1", id)
	if err != nil {
		log.Fatalf("query err: %v", err)
	}
	defer row.Close()

	var snils string

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
