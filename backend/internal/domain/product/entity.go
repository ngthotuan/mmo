package product

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID                uuid.UUID  `db:"id"                  json:"id"`
	UserID            uuid.UUID  `db:"user_id"             json:"user_id"`
	ChannelID         *uuid.UUID `db:"channel_id"          json:"channel_id,omitempty"`
	Platform          string     `db:"platform"            json:"platform"`
	PlatformProductID string     `db:"platform_product_id" json:"platform_product_id"`
	Name              string     `db:"name"                json:"name"`
	Description       string     `db:"description"         json:"description"`
	Price             float64    `db:"price"               json:"price"`
	Currency          string     `db:"currency"            json:"currency"`
	CoverImageURL     string     `db:"cover_image_url"     json:"cover_image_url"`
	ProductURL        string     `db:"product_url"         json:"product_url"`
	Status            string     `db:"status"              json:"status"`
	RawData           []byte     `db:"raw_data"            json:"-"`
	SyncedAt          time.Time  `db:"synced_at"           json:"synced_at"`
	CreatedAt         time.Time  `db:"created_at"          json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"          json:"updated_at"`
}
