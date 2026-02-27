package schedule

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"ispo-schedule/internal/auth"
)

func (r *Repository) CreateRefreshToken(userID int64, tokenHash string, expiresAt time.Time) (*auth.RefreshToken, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user_id required")
	}
	if tokenHash == "" {
		return nil, fmt.Errorf("token_hash required")
	}
	row := auth.RefreshToken{UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	if err := r.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) GetRefreshTokenByHash(tokenHash string) (*auth.RefreshToken, error) {
	if tokenHash == "" {
		return nil, fmt.Errorf("token_hash required")
	}
	var row auth.RefreshToken
	if err := r.db.Where("token_hash = ?", tokenHash).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) RevokeRefreshToken(id int64, replacedByID *int64) error {
	now := time.Now().UTC()
	q := r.db.Model(&auth.RefreshToken{}).Where("id = ? AND revoked_at IS NULL", id)
	updates := map[string]any{"revoked_at": now}
	if replacedByID != nil {
		updates["replaced_by_token_id"] = *replacedByID
	}
	return q.Updates(updates).Error
}

func (r *Repository) RevokeAllRefreshTokensForUser(userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("user_id required")
	}
	now := time.Now().UTC()
	return r.db.Model(&auth.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{"revoked_at": now}).Error
}

func (r *Repository) IsRefreshTokenActive(tx *gorm.DB, id int64) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("id required")
	}
	var row auth.RefreshToken
	if err := tx.First(&row, id).Error; err != nil {
		return false, err
	}
	if row.RevokedAt != nil {
		return false, nil
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return false, nil
	}
	return true, nil
}
