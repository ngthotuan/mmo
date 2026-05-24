package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/channel"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/tiktok"
	"mmo/pkg/crypto"
	apperr "mmo/pkg/errors"
)

type ChannelUsecase struct {
	repo                *repository.ChannelRepoWithAll
	tiktok              *tiktok.Client
	facebook            *facebook.Client
	cryptoKey           []byte
	facebookTokenExpiry time.Duration
	redis               *redis.Client
}

func NewChannelUsecase(
	repo *repository.ChannelRepoWithAll,
	tiktokClient *tiktok.Client,
	facebookClient *facebook.Client,
	encryptionKey string,
	facebookTokenExpiry time.Duration,
	redisClient *redis.Client,
) *ChannelUsecase {
	return &ChannelUsecase{
		repo:                repo,
		tiktok:              tiktokClient,
		facebook:            facebookClient,
		cryptoKey:           []byte(encryptionKey),
		facebookTokenExpiry: facebookTokenExpiry,
		redis:               redisClient,
	}
}

const pkceKeyPrefix = "pkce:"
const pkceTTL = 10 * time.Minute

// GetAuthURL returns the OAuth authorization URL for the given platform.
// For TikTok it generates a PKCE verifier, stores it in Redis keyed by state, and
// embeds the code_challenge in the redirect URL.
func (uc *ChannelUsecase) GetAuthURL(platform channel.Platform, state string) (string, error) {
	switch platform {
	case channel.PlatformTikTok:
		verifier, challenge, err := tiktok.GeneratePKCE()
		if err != nil {
			return "", fmt.Errorf("generate pkce: %w", err)
		}
		if err := uc.redis.Set(context.Background(), pkceKeyPrefix+state, verifier, pkceTTL).Err(); err != nil {
			return "", fmt.Errorf("store pkce verifier: %w", err)
		}
		return uc.tiktok.AuthURL(state, challenge), nil
	case channel.PlatformFacebook:
		return uc.facebook.AuthURL(state), nil
	default:
		return "", apperr.Newf(400, "unsupported platform: %s", platform)
	}
}

// ConnectTikTok exchanges the OAuth code for tokens and upserts the channel.
// state is required to look up the PKCE verifier stored during GetAuthURL.
func (uc *ChannelUsecase) ConnectTikTok(ctx context.Context, userID uuid.UUID, code, state string) (*channel.Channel, error) {
	verifier, err := uc.redis.GetDel(ctx, pkceKeyPrefix+state).Result()
	if err != nil {
		return nil, apperr.New(400, "invalid or expired oauth state — please retry the TikTok login")
	}
	tokens, err := uc.tiktok.ExchangeCode(ctx, code, verifier)
	if err != nil {
		return nil, fmt.Errorf("tiktok exchange code: %w", err)
	}

	profile, err := uc.tiktok.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("tiktok get user info: %w", err)
	}

	encAccess, err := crypto.Encrypt(uc.cryptoKey, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	encRefresh, err := crypto.Encrypt(uc.cryptoKey, tokens.RefreshToken)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Upsert: check if channel already connected
	existing, err := uc.repo.GetByPlatformUserID(ctx, channel.PlatformTikTok, profile.OpenID)
	if err == nil {
		// Update tokens
		existing.AccessToken = encAccess
		existing.RefreshToken = encRefresh
		existing.TokenExpiresAt = &expiresAt
		existing.DisplayName = profile.DisplayName
		existing.AvatarURL = profile.AvatarURL
		existing.IsActive = true
		if err := uc.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	ch := &channel.Channel{
		ID:             uuid.New(),
		UserID:         userID,
		Platform:       channel.PlatformTikTok,
		PlatformUserID: profile.OpenID,
		Username:       profile.Username,
		DisplayName:    profile.DisplayName,
		AvatarURL:      profile.AvatarURL,
		AccessToken:    encAccess,
		RefreshToken:   encRefresh,
		TokenExpiresAt: &expiresAt,
		IsActive:       true,
	}
	if err := uc.repo.Create(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// ConnectFacebook connects a Facebook Page using the long-lived user token obtained from GetFacebookPages.
func (uc *ChannelUsecase) ConnectFacebook(ctx context.Context, userID uuid.UUID, userToken, pageID string) (*channel.Channel, error) {
	// Get page-specific token using the already-exchanged long-lived user token
	page, err := uc.facebook.GetPageToken(ctx, userToken, pageID)
	if err != nil {
		return nil, fmt.Errorf("facebook page token: %w", err)
	}

	encAccess, err := crypto.Encrypt(uc.cryptoKey, page.AccessToken)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(uc.facebookTokenExpiry)

	existing, err := uc.repo.GetByPlatformUserID(ctx, channel.PlatformFacebook, page.ID)
	if err == nil {
		existing.AccessToken = encAccess
		existing.TokenExpiresAt = &expiresAt
		existing.DisplayName = page.Name
		existing.AvatarURL = page.Picture
		existing.IsActive = true
		if err := uc.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	ch := &channel.Channel{
		ID:             uuid.New(),
		UserID:         userID,
		Platform:       channel.PlatformFacebook,
		PlatformUserID: page.ID,
		Username:       page.ID,
		DisplayName:    page.Name,
		AvatarURL:      page.Picture,
		AccessToken:    encAccess,
		PageID:         page.ID,
		TokenExpiresAt: &expiresAt,
		IsActive:       true,
	}
	if err := uc.repo.Create(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// GetFacebookPages exchanges the OAuth code once, fetches available pages, and returns both.
// The caller must pass userToken back to ConnectFacebook — the code must not be exchanged again.
func (uc *ChannelUsecase) GetFacebookPages(ctx context.Context, code string) ([]facebook.Page, string, error) {
	tokens, err := uc.facebook.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", err
	}
	longLived, err := uc.facebook.GetLongLivedToken(ctx, tokens.AccessToken)
	if err != nil {
		return nil, "", err
	}
	pages, err := uc.facebook.ListPages(ctx, longLived)
	if err != nil {
		return nil, "", err
	}
	return pages, longLived, nil
}

func (uc *ChannelUsecase) List(ctx context.Context, userID uuid.UUID) ([]*channel.Channel, error) {
	return uc.repo.ListByUserID(ctx, userID)
}

func (uc *ChannelUsecase) Delete(ctx context.Context, userID, channelID uuid.UUID) error {
	ch, err := uc.repo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.UserID != userID {
		return apperr.ErrForbidden
	}
	return uc.repo.Delete(ctx, channelID)
}

func (uc *ChannelUsecase) SetActive(ctx context.Context, userID, channelID uuid.UUID, active bool) error {
	ch, err := uc.repo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.UserID != userID {
		return apperr.ErrForbidden
	}
	return uc.repo.SetActive(ctx, channelID, active)
}

// DecryptToken returns the plaintext access token for a channel.
func (uc *ChannelUsecase) DecryptToken(ch *channel.Channel) (string, error) {
	return crypto.Decrypt(uc.cryptoKey, ch.AccessToken)
}
