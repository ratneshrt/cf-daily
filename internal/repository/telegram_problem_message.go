package repository

import "github.com/jackc/pgx/v5/pgxpool"

type TelegramProblemMessageRepository struct {
	db *pgxpool.Pool
}

func NewTelegramProblemMessageRepository(db *pgxpool.Pool) *TelegramProblemMessageRepository {
	return &TelegramProblemMessageRepository{db: db}
}
