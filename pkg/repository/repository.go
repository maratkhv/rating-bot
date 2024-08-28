package repository

import "context"

type Repo struct {
	Db *database
	// Cache *cache
}

func New(ctx context.Context) (*Repo, error) {
	db, err := newDb(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: add cache

	return &Repo{
		Db: db,
	}, nil
}
