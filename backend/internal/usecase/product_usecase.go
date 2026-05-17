package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/product"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/tiktok"
	"mmo/pkg/crypto"
	"mmo/pkg/util"
)

type ProductUsecase struct {
	productRepo *repository.ProductRepo
	channelRepo *repository.ChannelRepoWithAll
	tiktok      *tiktok.Client
	facebook    *facebook.Client
	encKey      string
}

func NewProductUsecase(
	productRepo *repository.ProductRepo,
	channelRepo *repository.ChannelRepoWithAll,
	tiktokClient *tiktok.Client,
	fbClient *facebook.Client,
	encKey string,
) *ProductUsecase {
	return &ProductUsecase{
		productRepo: productRepo,
		channelRepo: channelRepo,
		tiktok:      tiktokClient,
		facebook:    fbClient,
		encKey:      encKey,
	}
}

type CreateProductRequest struct {
	Platform          string  `json:"platform"`
	PlatformProductID string  `json:"platform_product_id"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Price             float64 `json:"price"`
	Currency          string  `json:"currency"`
	CoverImageURL     string  `json:"cover_image_url"`
	ProductURL        string  `json:"product_url"`
}

func (u *ProductUsecase) Create(ctx context.Context, userID uuid.UUID, req CreateProductRequest) (*product.Product, error) {
	p := &product.Product{
		ID:                uuid.New(),
		UserID:            userID,
		Platform:          req.Platform,
		PlatformProductID: req.PlatformProductID,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		Currency:          req.Currency,
		CoverImageURL:     req.CoverImageURL,
		ProductURL:        req.ProductURL,
		Status:            "active",
		SyncedAt:          time.Now(),
	}
	if err := u.productRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (u *ProductUsecase) List(ctx context.Context, userID uuid.UUID, platform string, page, perPage int) ([]*product.Product, int, error) {
	pg := util.Pagination{Page: page, PerPage: perPage}
	return u.productRepo.List(ctx, userID, platform, pg)
}

func (u *ProductUsecase) GetByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	return u.productRepo.GetByID(ctx, id)
}

func (u *ProductUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	return u.productRepo.Delete(ctx, id)
}

// SyncFromTikTok fetches products from TikTok Shop and upserts them locally.
// channelID identifies which channel's access token to use as the shop token.
func (u *ProductUsecase) SyncFromTikTok(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	channel, err := u.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return 0, err
	}

	shopToken, err := crypto.Decrypt([]byte(u.encKey), channel.AccessToken)
	if err != nil {
		return 0, err
	}

	shopProducts, _, err := u.tiktok.ListShopProducts(ctx, shopToken, 1, 100)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, sp := range shopProducts {
		raw, _ := json.Marshal(sp)
		p := &product.Product{
			ID:                uuid.New(),
			UserID:            userID,
			ChannelID:         &channelID,
			Platform:          "tiktok",
			PlatformProductID: sp.ProductID,
			Name:              sp.Name,
			Description:       sp.Description,
			CoverImageURL:     sp.CoverImageURL,
			ProductURL:        sp.ProductURL,
			Status:            "active",
			RawData:           raw,
			SyncedAt:          time.Now(),
		}
		if err := u.productRepo.Upsert(ctx, p); err == nil {
			count++
		}
	}
	return count, nil
}

// SyncFromFacebook fetches products from a Facebook Product Catalog and upserts them.
// catalogID is stored in channel.Metadata["catalog_id"].
func (u *ProductUsecase) SyncFromFacebook(ctx context.Context, userID, channelID uuid.UUID, catalogID string) (int, error) {
	channel, err := u.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return 0, err
	}

	pageToken, err := crypto.Decrypt([]byte(u.encKey), channel.AccessToken)
	if err != nil {
		return 0, err
	}

	catalogProducts, err := u.facebook.ListCatalogProducts(ctx, pageToken, catalogID, 100)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, cp := range catalogProducts {
		raw, _ := json.Marshal(cp)
		p := &product.Product{
			ID:                uuid.New(),
			UserID:            userID,
			ChannelID:         &channelID,
			Platform:          "facebook",
			PlatformProductID: cp.ID,
			Name:              cp.Name,
			Description:       cp.Description,
			CoverImageURL:     cp.ImageURL,
			ProductURL:        cp.URL,
			Status:            "active",
			RawData:           raw,
			SyncedAt:          time.Now(),
		}
		if err := u.productRepo.Upsert(ctx, p); err == nil {
			count++
		}
	}
	return count, nil
}

func (u *ProductUsecase) TagPublishJob(ctx context.Context, publishJobID uuid.UUID, productIDs []uuid.UUID) error {
	return u.productRepo.SetPublishJobProducts(ctx, publishJobID, productIDs)
}

func (u *ProductUsecase) ListByPublishJob(ctx context.Context, publishJobID uuid.UUID) ([]*product.Product, error) {
	return u.productRepo.ListByPublishJob(ctx, publishJobID)
}
