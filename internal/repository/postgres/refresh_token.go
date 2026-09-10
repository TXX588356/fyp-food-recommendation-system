package postgres

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenPostgresRepository(db *gorm.DB) interfaces.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, refreshToken *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(refreshToken).Error
}

// FindByHash finds the refresh token row when /auth/refresh is called
func (r *refreshTokenRepository) FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var refreshToken model.RefreshToken

	if err := r.db.WithContext(ctx).
		Where("token_hash = ?", hash).
		First(&refreshToken).Error; err != nil {
		return nil, err
	}

	return &refreshToken, nil
}

// Rotate creates a new refresh token and revokes the old one
func (r *refreshTokenRepository) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *model.RefreshToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newToken).Error; err != nil {
			return err
		}

		now := time.Now()

		return tx.Model(&model.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", oldTokenID).
			Updates(map[string]any{
				"revoked_at":  now,
				"replaced_by": newToken.ID,
			}).Error
	})
}

func (r *refreshTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	now := time.Now()

	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", now).Error
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()

	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}
