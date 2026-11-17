package models

import "github.com/google/uuid"

// SectionProgress is a denormalised gate record that caches whether a user
// has completed an entire Section (i.e. all Modules within it are passed at
// >= their PassThreshold score).
//
// Why denormalise: checking "is previous section 100% complete?" at request
// time would require a multi-table aggregate; storing the result here reduces
// that to a single indexed row lookup — critical for the <100 ms gate-check
// latency requirement.
//
// Service layer responsibility: whenever a ModuleProgress.IsUnlocked is set
// to true, the service must recheck whether ALL Modules in that Section are
// now unlocked and flip SectionProgress.IsComplete accordingly.
//
// Indexes:
//   - (user_id, section_id): composite unique — one row per user/section pair.
//   - user_id:               for loading the full section map on a dashboard.
type SectionProgress struct {
	Base

	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_section_progress_user_section,unique" json:"user_id"`
	SectionID uuid.UUID `gorm:"type:uuid;not null;index:idx_section_progress_user_section,unique" json:"section_id"`

	// IsComplete is true when every Module in the Section has IsUnlocked = true
	// for this user.
	IsComplete bool `gorm:"not null;default:false" json:"is_complete"`

	// ModulesTotal is snapshotted at session start to detect curriculum edits.
	ModulesTotal int `gorm:"not null;default:0" json:"modules_total"`
	// ModulesUnlocked mirrors the count of passed modules for progress-bar UIs.
	ModulesUnlocked int `gorm:"not null;default:0" json:"modules_unlocked"`

	User    *User    `gorm:"foreignKey:UserID"    json:"user,omitempty"`
	Section *Section `gorm:"foreignKey:SectionID" json:"section,omitempty"`
}
