package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"mmo/internal/domain/product"
	apperr "mmo/pkg/errors"
	"mmo/pkg/util"
)

type ProductRepo struct{ db *sqlx.DB }

func NewProductRepo(db *sqlx.DB) *ProductRepo { return &ProductRepo{db: db} }

func (r *ProductRepo) Create(ctx context.Context, p *product.Product) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO products
			(id, user_id, channel_id, platform, platform_product_id, name, description,
			 price, currency, cover_image_url, product_url, status, raw_data, synced_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		p.ID, p.UserID, p.ChannelID, p.Platform, p.PlatformProductID, p.Name, p.Description,
		p.Price, p.Currency, p.CoverImageURL, p.ProductURL, p.Status, p.RawData, p.SyncedAt,
	)
	return err
}

func (r *ProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	var p product.Product
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM products WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) List(ctx context.Context, userID uuid.UUID, platform string, pg util.Pagination) ([]*product.Product, int, error) {
	args := []any{userID}
	where := "user_id = $1"

	if platform != "" {
		args = append(args, platform)
		where += fmt.Sprintf(" AND platform = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM products WHERE "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, pg.Limit(), pg.Offset())
	products := []*product.Product{}
	if err := r.db.SelectContext(ctx, &products,
		fmt.Sprintf("SELECT * FROM products WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
			where, len(args)-1, len(args)), args...,
	); err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *ProductRepo) Update(ctx context.Context, p *product.Product) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE products SET
			name=$1, description=$2, price=$3, currency=$4, cover_image_url=$5,
			product_url=$6, status=$7, raw_data=$8, synced_at=$9, updated_at=NOW()
		WHERE id=$10`,
		p.Name, p.Description, p.Price, p.Currency, p.CoverImageURL,
		p.ProductURL, p.Status, p.RawData, p.SyncedAt, p.ID,
	)
	return err
}

func (r *ProductRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

// Upsert inserts or updates a product by (user_id, platform, platform_product_id).
func (r *ProductRepo) Upsert(ctx context.Context, p *product.Product) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO products
			(id, user_id, channel_id, platform, platform_product_id, name, description,
			 price, currency, cover_image_url, product_url, status, raw_data, synced_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (user_id, platform, platform_product_id) DO UPDATE SET
			name            = EXCLUDED.name,
			description     = EXCLUDED.description,
			price           = EXCLUDED.price,
			currency        = EXCLUDED.currency,
			cover_image_url = EXCLUDED.cover_image_url,
			product_url     = EXCLUDED.product_url,
			status          = EXCLUDED.status,
			raw_data        = EXCLUDED.raw_data,
			synced_at       = EXCLUDED.synced_at,
			updated_at      = NOW()`,
		p.ID, p.UserID, p.ChannelID, p.Platform, p.PlatformProductID, p.Name, p.Description,
		p.Price, p.Currency, p.CoverImageURL, p.ProductURL, p.Status, p.RawData, p.SyncedAt,
	)
	return err
}

// ListByPublishJob returns products tagged on a publish job.
func (r *ProductRepo) ListByPublishJob(ctx context.Context, publishJobID uuid.UUID) ([]*product.Product, error) {
	products := []*product.Product{}
	err := r.db.SelectContext(ctx, &products, `
		SELECT p.* FROM products p
		JOIN publish_job_products pjp ON pjp.product_id = p.id
		WHERE pjp.publish_job_id = $1
		ORDER BY p.name`, publishJobID)
	return products, err
}

// SetPublishJobProducts replaces the product tags for a publish job.
func (r *ProductRepo) SetPublishJobProducts(ctx context.Context, publishJobID uuid.UUID, productIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM publish_job_products WHERE publish_job_id = $1`, publishJobID); err != nil {
		return err
	}
	for _, pid := range productIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO publish_job_products (publish_job_id, product_id) VALUES ($1, $2)`,
			publishJobID, pid,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
