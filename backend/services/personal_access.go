package services

import (
	"fmt"

	"github.com/learnthor/backend/models"
	"gorm.io/gorm"
)

// GrantPersonalPremium makes one existing account a premium student with a learning bypass.
// It is used only by the operator CLI, never exposed through an HTTP route.
func GrantPersonalPremium(database *gorm.DB, username string) (*models.User, error) {
	var user models.User
	err := database.Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(73492161)").Error; err != nil {
				return err
			}
		}
		if err := tx.Where("username = ?", username).First(&user).Error; err != nil {
			return fmt.Errorf("find existing account %q: %w", username, err)
		}
		if err := tx.Model(&models.User{}).Where("progression_bypass = ? AND id <> ?", true, user.ID).Update("progression_bypass", false).Error; err != nil {
			return err
		}
		if err := tx.Model(&user).Updates(map[string]any{"role": models.UserRoleStudent, "subscription_tier": models.SubscriptionPremium, "premium_until": nil, "progression_bypass": true}).Error; err != nil {
			return err
		}
		return tx.First(&user, "id = ?", user.ID).Error
	})
	return &user, err
}
