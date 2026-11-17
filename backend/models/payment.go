package models

import (
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Payment (Midtrans one-time premium purchase)
// ─────────────────────────────────────────────────────────────────────────────

// PaymentStatus mirrors the lifecycle of a Midtrans transaction as tracked
// locally. The webhook handler maps Midtrans transaction_status values onto
// these states.
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"    // created, awaiting payment
	PaymentStatusSettlement PaymentStatus = "settlement" // paid — premium granted
	PaymentStatusExpired    PaymentStatus = "expired"    // unpaid past expiry
	PaymentStatusFailed     PaymentStatus = "failed"     // cancel / deny
)

// Payment records one premium purchase attempt. One-time lifetime plan only —
// a settled payment flips User.SubscriptionTier to "premium" permanently.
//
// Indexes:
//   - order_id: unique, the idempotency key shared with Midtrans (also used
//     by the webhook handler to look up the payment).
//   - user_id:  for listing a user's payment history.
type Payment struct {
	Base

	// OrderID is the merchant-side order reference sent to Midtrans. It is
	// generated at creation time (e.g. "LTH-<uuid>") and must be unique.
	OrderID string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"order_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index"        json:"user_id"`

	// Plan is the purchased SKU. Only "lifetime" exists today; the column
	// future-proofs for subscription plans.
	Plan string `gorm:"type:varchar(32);not null;default:'lifetime'" json:"plan"`
	// GrossAmount is the charged amount in IDR (no decimals).
	GrossAmount int64         `gorm:"not null"                          json:"gross_amount"`
	Status      PaymentStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`

	// SnapToken is the Midtrans Snap popup token. Server-side only — the
	// frontend receives it once at creation time and never needs it again.
	SnapToken string `gorm:"type:varchar(255)"  json:"-"`
	// SnapURL is the hosted-page redirect URL (fallback when the popup
	// cannot be shown, e.g. mobile webviews).
	SnapURL string `gorm:"type:varchar(2048)" json:"snap_url"`

	// PaidAt is set when the payment reaches settlement.
	PaidAt *time.Time `json:"paid_at,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
