/*
	TODO:
DATABASE TABLES TEMPLATES:


*/

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
	var PSWD = os.Getenv("PSWD")
	var connString = "postgresql://myneondb_owner:" + PSWD + "@ep-shiny-sun-a2swl5c2.eu-central-1.aws.neon.tech/myneondb?sslmode=require"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}
