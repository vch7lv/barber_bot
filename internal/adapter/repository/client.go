package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// ClientRepo реализует port.ClientRepository для SQLite.
type ClientRepo struct {
	db *sql.DB
}

// NewClientRepo создаёт репозиторий клиентов.
func NewClientRepo(db *sql.DB) *ClientRepo {
	return &ClientRepo{db: db}
}

// GetByID возвращает клиента по ID или nil.
func (r *ClientRepo) GetByID(ctx context.Context, id int64) (*domain.Client, error) {
	var c domain.Client
	err := r.db.QueryRowContext(ctx,
		`SELECT id, telegram_id, name, username, contact, created_at FROM clients WHERE id = ?`,
		id,
	).Scan(&c.ID, &c.TelegramID, &c.Name, &c.Username, &c.Contact, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListAll возвращает всех клиентов.
func (r *ClientRepo) ListAll(ctx context.Context) ([]*domain.Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, telegram_id, name, username, contact, created_at FROM clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.Client
	for rows.Next() {
		c := &domain.Client{}
		if err := rows.Scan(&c.ID, &c.TelegramID, &c.Name, &c.Username, &c.Contact, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetByTelegramID возвращает клиента по Telegram ID или nil, если не найден.
func (r *ClientRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Client, error) {
	var c domain.Client
	err := r.db.QueryRowContext(ctx,
		`SELECT id, telegram_id, name, username, contact, created_at FROM clients WHERE telegram_id = ?`,
		telegramID,
	).Scan(&c.ID, &c.TelegramID, &c.Name, &c.Username, &c.Contact, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Save создаёт или обновляет клиента. Если ID == 0 — INSERT, иначе UPDATE.
func (r *ClientRepo) Save(ctx context.Context, c *domain.Client) error {
	if c.ID == 0 {
		res, err := r.db.ExecContext(ctx,
			`INSERT INTO clients (telegram_id, name, username, contact, created_at) VALUES (?, ?, ?, ?, ?)`,
			c.TelegramID, c.Name, c.Username, c.Contact, c.CreatedAt,
		)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		c.ID = id
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET name = ?, username = ?, contact = ? WHERE id = ?`,
		c.Name, c.Username, c.Contact, c.ID,
	)
	return err
}
