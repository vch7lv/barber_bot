package repository

import (
	"context"
	"database/sql"

	"barber_bot/internal/domain"
)

// ShopAddressRepo реализует port.ShopAddressRepository для SQLite.
type ShopAddressRepo struct {
	db *sql.DB
}

// NewShopAddressRepo создаёт репозиторий адреса салона.
func NewShopAddressRepo(db *sql.DB) *ShopAddressRepo {
	return &ShopAddressRepo{db: db}
}

// Get возвращает адрес салона. Всегда возвращает не-nil (хотя бы пустой текст).
func (r *ShopAddressRepo) Get(ctx context.Context) (*domain.ShopAddress, error) {
	var a domain.ShopAddress
	var photo sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT address_text, address_photo_file_id FROM shop_address WHERE id = 1`,
	).Scan(&a.AddressText, &photo)
	if err == sql.ErrNoRows {
		return &domain.ShopAddress{}, nil
	}
	if err != nil {
		return nil, err
	}
	if photo.Valid {
		a.AddressPhotoFileID = photo.String
	}
	return &a, nil
}

// Set сохраняет адрес салона (одна запись id=1).
func (r *ShopAddressRepo) Set(ctx context.Context, a *domain.ShopAddress) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO shop_address (id, address_text, address_photo_file_id) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET address_text=excluded.address_text, address_photo_file_id=excluded.address_photo_file_id`,
		a.AddressText, nullIfEmpty(a.AddressPhotoFileID),
	)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
