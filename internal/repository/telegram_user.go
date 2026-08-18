package repository

import "github.com/jackc/pgx/v5/pgxpool"

type TelegramUserRepository struct {
	db *pgxpool.Pool
}

func NewTelegramUserRepository(db *pgxpool.Pool) *TelegramUserRepository {
	return &TelegramUserRepository{
		db: db,
	}
}
