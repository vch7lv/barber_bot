package domain

// ShopAddress — адрес салона для показа клиентам. PhotoFileID — file_id фото в Telegram (опционально).
type ShopAddress struct {
	AddressText     string
	AddressPhotoFileID string
}
