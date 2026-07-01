package postgres

import (
	"context"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *catalogRepository) ListImages(ctx context.Context, query interfaces.CatalogImageQuery) ([]model.PrebuiltMealImage, error) {
	db := r.db.WithContext(ctx).Preload("PrebuiltMeal").Order("id")
	if query.Status != "" {
		db = db.Where("match_status = ?", query.Status)
	}
	if query.Provider != "" {
		db = db.Where("provider = ?", query.Provider)
	}
	if query.MealID != nil {
		db = db.Where("prebuilt_meal_id = ?", *query.MealID)
	}
	if query.MinScore != nil {
		db = db.Where("match_score >= ?", *query.MinScore)
	}
	if query.MaxScore != nil {
		db = db.Where("match_score <= ?", *query.MaxScore)
	}
	if query.AfterID != nil {
		db = db.Where("id > ?", *query.AfterID)
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var images []model.PrebuiltMealImage
	return images, db.Limit(limit + 1).Find(&images).Error
}

func (r *catalogRepository) FindImage(ctx context.Context, id uuid.UUID) (*model.PrebuiltMealImage, error) {
	var image model.PrebuiltMealImage
	if err := r.db.WithContext(ctx).Preload("PrebuiltMeal").First(&image, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, interfaces.ErrCatalogNotFound
		}
		return nil, err
	}
	return &image, nil
}

func (r *catalogRepository) ApplyImageAction(ctx context.Context, id uuid.UUID, action interfaces.CatalogImageAction) (*model.PrebuiltMealImage, error) {
	var image model.PrebuiltMealImage
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&image, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return interfaces.ErrCatalogNotFound
			}
			return err
		}
		now := time.Now()
		if err := validateImageAction(image, action); err != nil {
			return err
		}
		switch action.Action {
		case "approve", "make_primary":
			if err := tx.Model(&model.PrebuiltMealImage{}).Where("prebuilt_meal_id = ? AND id <> ?", image.PrebuiltMealID, image.ID).Update("is_primary", false).Error; err != nil {
				return err
			}
			image.MatchStatus, image.IsPrimary, image.ReviewedAt = "approved", true, &now
		case "reject":
			image.MatchStatus, image.IsPrimary, image.ReviewReason, image.ReviewedAt = "rejected", false, &action.Reason, &now
		case "request_rematch":
			image.RematchRequestedAt = &now
		default:
			return interfaces.ErrInvalidImageTransition
		}
		return tx.Save(&image).Error
	})
	return &image, err
}

func validateImageAction(image model.PrebuiltMealImage, action interfaces.CatalogImageAction) error {
	switch action.Action {
	case "approve", "make_primary":
		if image.MinioObjectKey == nil || image.MIMEType == nil || image.Width == nil || image.Height == nil ||
			image.SHA256 == nil || image.LicenseName == nil || image.AttributionText == nil ||
			(image.MatchStatus != "auto_accepted" && image.MatchStatus != "needs_review" && image.MatchStatus != "approved") {
			return interfaces.ErrInvalidImageTransition
		}
	case "reject", "request_rematch":
		return nil
	default:
		return interfaces.ErrInvalidImageTransition
	}
	return nil
}
