package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// IsPrivilegedUser reports whether the given user holds a role that bypasses
// module gating and section-quiz mastery checks (admin / editor). Students are
// never privileged. A missing user row returns false so a deleted account does
// not silently gain elevated access. Exported so HTTP handlers can reuse the
// same rule when building lock-state payloads for the homepage grid.
func IsPrivilegedUser(db *gorm.DB, userID uuid.UUID) bool {
	var user models.User
	if err := db.Select("role").
		First(&user, "id = ?", userID).Error; err != nil {
		return false
	}
	return user.Role == models.UserRoleAdmin || user.Role == models.UserRoleEditor
}

// isPrivilegedUser keeps the unexported name used by other services in this
// package; it simply delegates to the exported helper.
func isPrivilegedUser(db *gorm.DB, userID uuid.UUID) bool {
	return IsPrivilegedUser(db, userID)
}

// CanBypassProgression checks learning access without granting administrative privileges.
func CanBypassProgression(db *gorm.DB, userID uuid.UUID) bool {
	var user models.User
	if err := db.Select("role", "subscription_tier", "progression_bypass").First(&user, "id = ?", userID).Error; err != nil {
		return false
	}
	return user.Role == models.UserRoleAdmin || user.Role == models.UserRoleEditor ||
		(user.SubscriptionTier == models.SubscriptionPremium && user.ProgressionBypass)
}

// CheckSectionAccess applies premium and predecessor-section requirements on the server.
func (g *GatingService) CheckSectionAccess(userID, sectionID uuid.UUID) (GateResult, error) {
	var section models.Section
	if err := g.db.Where("is_hidden = ?", false).First(&section, "id = ?", sectionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GateResult{}, ErrNotFound
		}
		return GateResult{}, err
	}
	if CanBypassProgression(g.db, userID) {
		return GateResult{Allowed: true}, nil
	}
	var user models.User
	if err := g.db.First(&user, "id = ?", userID).Error; err != nil {
		return GateResult{}, err
	}
	if section.IsPremium && user.SubscriptionTier != models.SubscriptionPremium {
		return GateResult{RequiresPremium: true, Reason: "This section requires premium access."}, nil
	}
	var previous models.Section
	err := g.db.Where("course_id = ? AND sort_order < ? AND is_hidden = ?", section.CourseID, section.SortOrder, false).Order("sort_order DESC").First(&previous).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return GateResult{Allowed: true}, nil
	}
	if err != nil {
		return GateResult{}, err
	}
	var remaining int64
	err = g.db.Table("modules AS m").Joins("LEFT JOIN module_progresses AS p ON p.module_id = m.id AND p.user_id = ?", userID).
		Where("m.section_id = ? AND m.is_hidden = ? AND COALESCE(p.high_score, 0) < m.pass_threshold", previous.ID, false).Count(&remaining).Error
	if err != nil {
		return GateResult{}, err
	}
	return GateResult{Allowed: remaining == 0, Reason: "Master the previous section first."}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GatingService
// ─────────────────────────────────────────────────────────────────────────────

// GatingService contains all database query logic for curriculum access control.
// It has no knowledge of HTTP — the middleware translates its results into
// HTTP responses.
type GatingService struct {
	db *gorm.DB
}

// NewGatingService constructs a GatingService.
func NewGatingService(db *gorm.DB) *GatingService {
	return &GatingService{db: db}
}

// GateResult is the outcome returned by CheckModuleAccess.
type GateResult struct {
	// Allowed is true when the user may access the requested module.
	Allowed bool
	// Reason is a human-readable explanation when Allowed is false.
	Reason string
	// RequiredModuleID is the ID of the predecessor module the user must pass.
	// Zero value when the requested module is the first in its section.
	RequiredModuleID uuid.UUID
	// RequiredModuleTitle is the title of that predecessor module (for UX messages).
	RequiredModuleTitle string
	// UserHighScore is the user's best score on the predecessor module (0 if none).
	UserHighScore int
	// RequiresPremium is true when the denial is due to the module living in
	// a premium section and the user lacking a premium subscription (as
	// opposed to normal progression gating). Handlers use this to return an
	// upgrade prompt instead of a "pass the previous module" message.
	RequiresPremium bool
}

// CheckModuleAccess determines whether userID is permitted to access moduleID.
//
// Algorithm:
//  1. Load the target module to obtain its section_id and sort_order.
//  2. Find the immediately preceding module in the same section
//     (highest sort_order that is still < target's sort_order).
//  3. If no predecessor exists the module is the first in its section → allow.
//  4. Load the user's ModuleProgress for the predecessor.
//  5. Allow if high_score >= predecessor.PassThreshold; deny otherwise.
//
// All reads are performed with SELECT on the minimal columns needed to keep
// the query set small and avoid unnecessary data transfer.
func (g *GatingService) CheckModuleAccess(userID, moduleID uuid.UUID) (GateResult, error) {

	// Privileged users (admin / editor) bypass gating so they can preview and
	// test all curriculum content without first passing each module.
	if CanBypassProgression(g.db, userID) {
		// Still validate the module exists so callers get a 404 on bad IDs.
		var target models.Module
		if err := g.db.Select("id").Where("is_hidden = ?", false).First(&target, "id = ?", moduleID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return GateResult{}, fmt.Errorf("module %s not found: %w", moduleID, ErrNotFound)
			}
			return GateResult{}, fmt.Errorf("could not load target module: %w", err)
		}
		return GateResult{Allowed: true}, nil
	}

	// ── 1. Load target module ─────────────────────────────────────────────────
	var target models.Module
	if err := g.db.
		Select("id", "section_id", "sort_order", "pass_threshold", "title").
		Where("is_hidden = ?", false).
		First(&target, "id = ?", moduleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GateResult{}, fmt.Errorf("module %s not found: %w", moduleID, ErrNotFound)
		}
		return GateResult{}, fmt.Errorf("could not load target module: %w", err)
	}

	// ── 1b. Premium-section check ────────────────────────────────────────────
	sectionAccess, err := g.CheckSectionAccess(userID, target.SectionID)
	if err != nil || !sectionAccess.Allowed {
		return sectionAccess, err
	}
	// Free users are denied with RequiresPremium so the frontend shows an
	// upgrade prompt instead of a progression-lock message. This runs before
	// the predecessor check: premium is the outermost gate.
	var section models.Section
	if err := g.db.Select("id", "is_premium").
		First(&section, "id = ?", target.SectionID).Error; err != nil {
		return GateResult{}, fmt.Errorf("could not load section: %w", err)
	}
	if section.IsPremium {
		var user models.User
		if err := g.db.Select("subscription_tier").
			First(&user, "id = ?", userID).Error; err != nil {
			return GateResult{}, fmt.Errorf("could not load user: %w", err)
		}
		if user.SubscriptionTier != models.SubscriptionPremium {
			return GateResult{
				Allowed:         false,
				Reason:          "This section is part of premium content.",
				RequiresPremium: true,
			}, nil
		}
	}

	// ── 2. Find the predecessor module ───────────────────────────────────────
	// "Predecessor" = module in the same section with the highest sort_order
	// that is strictly less than the target's sort_order.
	var prev models.Module
	err = g.db.
		Select("id", "title", "pass_threshold").
		Where("section_id = ? AND sort_order < ? AND is_hidden = ?", target.SectionID, target.SortOrder, false).
		Order("sort_order DESC").
		Limit(1).
		First(&prev).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// ── 3. No predecessor → first module in section → always accessible ──
		return GateResult{Allowed: true}, nil
	}
	if err != nil {
		return GateResult{}, fmt.Errorf("could not query predecessor module: %w", err)
	}

	// ── 4. Load the user's progress on the predecessor ───────────────────────
	var progress models.ModuleProgress
	err = g.db.
		Select("high_score", "is_unlocked").
		Where("user_id = ? AND module_id = ?", userID, prev.ID).
		First(&progress).Error

	// No progress row means the user has never attempted the predecessor.
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return GateResult{
			Allowed:             false,
			Reason:              "You must master the previous module first.",
			RequiredModuleID:    prev.ID,
			RequiredModuleTitle: prev.Title,
			UserHighScore:       0,
		}, nil
	}
	if err != nil {
		return GateResult{}, fmt.Errorf("could not load module progress: %w", err)
	}

	// ── 5. Evaluate the threshold ─────────────────────────────────────────────
	if progress.HighScore >= prev.PassThreshold {
		return GateResult{
			Allowed:             true,
			RequiredModuleID:    prev.ID,
			RequiredModuleTitle: prev.Title,
			UserHighScore:       progress.HighScore,
		}, nil
	}

	return GateResult{
		Allowed:             false,
		Reason:              "You must master the previous module first.",
		RequiredModuleID:    prev.ID,
		RequiredModuleTitle: prev.Title,
		UserHighScore:       progress.HighScore,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// VocabService
// ─────────────────────────────────────────────────────────────────────────────

// VocabService handles vocabulary data retrieval.
type VocabService struct {
	db *gorm.DB
}

// NewVocabService constructs a VocabService.
func NewVocabService(db *gorm.DB) *VocabService {
	return &VocabService{db: db}
}

// GetVocabByModule returns all VocabItems belonging to moduleID.
// It returns ErrNotFound if no items exist for that module.
func (v *VocabService) GetVocabByModule(moduleID uuid.UUID) ([]models.VocabItem, error) {
	var items []models.VocabItem

	if err := v.db.
		Where("module_id = ?", moduleID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("could not fetch vocab items: %w", err)
	}

	return items, nil
}
