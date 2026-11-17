package models

import "time"

// UserRole constrains the set of roles a user may hold.
type UserRole string

const (
	UserRoleStudent UserRole = "student"
	UserRoleAdmin   UserRole = "admin"
	UserRoleEditor  UserRole = "editor" // can author curriculum content
)

// SubscriptionTier marks whether the user has paid for premium content.
// Premium sections (Section.IsPremium) are inaccessible to free users.
type SubscriptionTier string

const (
	SubscriptionFree    SubscriptionTier = "free"
	SubscriptionPremium SubscriptionTier = "premium"
)

// User holds authentication and profile data for a learner.
//
// Indexes:
//   - email:    unique, used for login lookups.
//   - username: unique, used for public display.
type User struct {
	Base

	// Auth fields
	Username     string `gorm:"type:varchar(50);not null;uniqueIndex"  json:"username"`
	Email        string `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"type:varchar(255);not null"             json:"-"` // bcrypt hash, never serialised

	// Role controls access to admin/editor routes.
	Role UserRole `gorm:"type:varchar(20);not null;default:'student'" json:"role"`

	// PreferredGloss controls which meaning is shown first on study cards and
	// review prompts. "en" = English first, "id" = Indonesian first, "both"
	// = show both (English above Indonesian). Defaults to "both" so new
	// learners see every available translation.
	PreferredGloss string `gorm:"type:varchar(8);not null;default:'both'" json:"preferred_gloss"`

	// SubscriptionTier gates premium sections. "free" users can access
	// everything up to and including N5; "premium" unlocks N4/N3 and the
	// Daily Vocab section. Admin/editor bypass regardless of tier.
	SubscriptionTier SubscriptionTier `gorm:"type:varchar(16);not null;default:'free'" json:"subscription_tier"`
	// PremiumUntil is reserved for future subscription plans. Null for the
	// current one-time lifetime purchase (premium never expires).
	PremiumUntil *time.Time `json:"premium_until,omitempty"`

	// ProgressionBypass grants a personal learning exception, never an administrative role.
	// It is effective only while the account has premium access.
	ProgressionBypass bool `gorm:"not null;default:false" json:"progression_bypass"`

	// Profile fields
	DisplayName string `gorm:"type:varchar(100)"     json:"display_name"`
	AvatarURL   string `gorm:"type:varchar(2048)"    json:"avatar_url,omitempty"`
	IsActive    bool   `gorm:"not null;default:true" json:"is_active"`

	// Associations (not loaded by default — use Preload explicitly)
	ModuleProgresses  []ModuleProgress  `gorm:"foreignKey:UserID" json:"-"`
	SectionProgresses []SectionProgress `gorm:"foreignKey:UserID" json:"-"`
	SRSReviews        []SRSReview       `gorm:"foreignKey:UserID" json:"-"`
	StudySessions     []StudySession    `gorm:"foreignKey:UserID" json:"-"`
}
