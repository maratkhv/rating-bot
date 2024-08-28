package repository

import (
	"context"
	"os"
	"ratinger/pkg/models/db"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type database struct {
	pool *pgxpool.Pool
}

var (
	dbInstance *database
	dbOnce     sync.Once
)

func newDb(ctx context.Context) (*database, error) {
	var err error
	dbOnce.Do(func() {
		godotenv.Load()
		var connString = os.Getenv("DB_CONNECTION_STRING")

		var pool *pgxpool.Pool
		pool, err = pgxpool.New(ctx, connString)

		dbInstance = &database{
			pool: pool,
		}
	})

	return dbInstance, err
}

func (db *database) InsertUser(ctx context.Context, id int64, snils string, authStatus int8) error {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	query := "insert into users(id, snils, auth_status) values(@userId, @snils, @authStatus)"
	args := pgx.NamedArgs{
		"userId":     id,
		"snils":      snils,
		"authStatus": authStatus,
	}

	_, err = tx.Exec(ctx, query, args)
	if err != nil {
		return err
	}

	tx.Commit(ctx)
	return nil
}

func (db *database) UpdateUser(ctx context.Context, userId int64, args db.Args) error {
	query := "update users set "
	nArgs := pgx.NamedArgs{
		"id": userId,
	}

	for k, v := range args {
		query += k + "=@" + k + ","
		nArgs[k] = v
	}
	query = query[:len(query)-1] + " where id = @id"

	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, query, nArgs)
	if err != nil {
		return err
	}

	tx.Commit(ctx)
	return err
}

func (db *database) SelectQuery(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return db.pool.Query(ctx, query, args...)
}
