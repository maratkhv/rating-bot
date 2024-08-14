package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func NeonConnect() *pgx.Conn {
	godotenv.Load()
	var connString = os.Getenv("DB_CONNECTION_STRING")
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}
