package services

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// ─────────────────────────────────────────────────────────────────────────────
// PaymentService — Midtrans one-time premium purchase
//
// Flow:
//  1. POST /payments/create       → Snap token + redirect URL, pending Payment row
//  2. Midtrans hosted checkout    → user pays (sandbox: simulator)
//  3. POST /payments/notification → webhook: verify signature, re-check the
//     transaction server-to-server (never trust the webhook body), grant premium
//  4. GET  /payments/:order_id    → frontend polls while the Snap popup is open
// ─────────────────────────────────────────────────────────────────────────────

// TransactionStatus is the subset of the Midtrans check-transaction response
// the service needs. Defined locally so tests can fake the gateway without
// importing SDK types.
type TransactionStatus struct {
	TransactionStatus string // capture | settlement | deny | cancel | expire | pending
	FraudStatus       string // accept | challenge | deny (capture only)
	GrossAmount       string
	StatusCode        string
}

// MidtransGateway abstracts the Midtrans SDK so PaymentService is unit-testable.
type MidtransGateway interface {
	// CreateSnapTransaction returns (snapToken, redirectURL, error).
	CreateSnapTransaction(orderID string, grossAmount int64) (string, string, error)
	// CheckTransaction is the server-to-server ground truth for a payment.
	CheckTransaction(orderID string) (*TransactionStatus, error)
}

// midtransSDKGateway is the production gateway backed by the official SDK.
type midtransSDKGateway struct {
	snap    snap.Client
	coreapi coreapi.Client
}

// NewMidtransGateway builds the real gateway from environment variables.
// MIDTRANS_ENV=sandbox uses the sandbox endpoints; anything else is production.
func NewMidtransGateway() MidtransGateway {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_ENV") == "production" {
		env = midtrans.Production
	}
	g := &midtransSDKGateway{}
	g.snap.New(serverKey, env)
	g.coreapi.New(serverKey, env)
	return g
}

func (g *midtransSDKGateway) CreateSnapTransaction(orderID string, grossAmount int64) (string, string, error) {
	resp, err := g.snap.CreateTransaction(&snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: grossAmount,
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("midtrans create transaction: %s", err.GetMessage())
	}
	return resp.Token, resp.RedirectURL, nil
}

func (g *midtransSDKGateway) CheckTransaction(orderID string) (*TransactionStatus, error) {
	resp, err := g.coreapi.CheckTransaction(orderID)
	if err != nil {
		return nil, fmt.Errorf("midtrans check transaction: %s", err.GetMessage())
	}
	return &TransactionStatus{
		TransactionStatus: resp.TransactionStatus,
		FraudStatus:       resp.FraudStatus,
		GrossAmount:       resp.GrossAmount,
		StatusCode:        resp.StatusCode,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────

// PaymentService handles premium purchase creation and webhook processing.
type PaymentService struct {
	db        *gorm.DB
	gateway   MidtransGateway
	serverKey string
	priceIDR  int64
}

// NewPaymentService constructs a PaymentService. priceIDR is the one-time
// premium price in IDR (from PREMIUM_PRICE_IDR, default 99000).
func NewPaymentService(db *gorm.DB, gateway MidtransGateway) *PaymentService {
	price := int64(99000)
	if v := os.Getenv("PREMIUM_PRICE_IDR"); v != "" {
		var parsed int64
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil && parsed > 0 {
			price = parsed
		}
	}
	return &PaymentService{
		db:        db,
		gateway:   gateway,
		serverKey: os.Getenv("MIDTRANS_SERVER_KEY"),
		priceIDR:  price,
	}
}

// CreateTransactionRequest starts a premium purchase.
type CreateTransactionRequest struct {
	UserID uuid.UUID `json:"-"`
}

// CreateTransactionResponse carries everything the frontend needs to open
// the Midtrans Snap popup (or fall back to the hosted redirect URL).
type CreateTransactionResponse struct {
	OrderID     string `json:"order_id"`
	SnapToken   string `json:"snap_token"`
	SnapURL     string `json:"snap_url"`
	GrossAmount int64  `json:"gross_amount"`
	// ClientKey is the publishable Midtrans key the snap.js script needs.
	// Safe to expose — it can only create transactions, never verify them.
	ClientKey string `json:"client_key"`
	// Env is "sandbox" or "production" — the frontend loads snap.js from the
	// matching Midtrans host (app.sandbox.midtrans.com vs app.midtrans.com).
	Env string `json:"env"`
	// Reused is true when an existing pending transaction was returned
	// instead of creating a new one (prevents order-id spam on retries).
	Reused bool `json:"reused"`
}

// midtransEnv returns "sandbox" or "production" for the response payload.
func midtransEnv() string {
	if os.Getenv("MIDTRANS_ENV") == "production" {
		return "production"
	}
	return "sandbox"
}

// CreateTransaction starts (or reuses) a premium purchase for the user.
// Already-premium users get ErrBadRequest — the UI should hide the button.
func (s *PaymentService) CreateTransaction(req CreateTransactionRequest) (*CreateTransactionResponse, error) {
	var user models.User
	if err := s.db.Select("id", "subscription_tier").First(&user, "id = ?", req.UserID).Error; err != nil {
		return nil, ErrNotFound
	}
	if user.SubscriptionTier == models.SubscriptionPremium {
		return nil, fmt.Errorf("user is already premium: %w", ErrBadRequest)
	}

	// Reuse the most recent pending payment if one exists — Snap tokens stay
	// valid for a while and this avoids orphan orders on double-clicks.
	var pending models.Payment
	err := s.db.
		Where("user_id = ? AND status = ?", req.UserID, models.PaymentStatusPending).
		Order("created_at DESC").
		First(&pending).Error
	if err == nil {
		return &CreateTransactionResponse{
			OrderID:     pending.OrderID,
			SnapToken:   pending.SnapToken,
			SnapURL:     pending.SnapURL,
			GrossAmount: pending.GrossAmount,
			ClientKey:   os.Getenv("MIDTRANS_CLIENT_KEY"),
			Env:         midtransEnv(),
			Reused:      true,
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("could not query pending payments: %w", err)
	}

	orderID := "LTH-" + uuid.NewString()
	token, redirectURL, err := s.gateway.CreateSnapTransaction(orderID, s.priceIDR)
	if err != nil {
		return nil, err
	}

	payment := models.Payment{
		OrderID:     orderID,
		UserID:      req.UserID,
		Plan:        "lifetime",
		GrossAmount: s.priceIDR,
		Status:      models.PaymentStatusPending,
		SnapToken:   token,
		SnapURL:     redirectURL,
	}
	if err := s.db.Create(&payment).Error; err != nil {
		return nil, fmt.Errorf("could not persist payment: %w", err)
	}

	return &CreateTransactionResponse{
		OrderID:     orderID,
		SnapToken:   token,
		SnapURL:     redirectURL,
		GrossAmount: s.priceIDR,
		ClientKey:   os.Getenv("MIDTRANS_CLIENT_KEY"),
		Env:         midtransEnv(),
	}, nil
}

// NotificationPayload mirrors the Midtrans webhook body (subset).
type NotificationPayload struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
}

// VerifySignature checks the webhook's signature_key:
//
//	SHA512(order_id + status_code + gross_amount + server_key)
//
// Returns false when the signature is missing or does not match.
func (s *PaymentService) VerifySignature(p NotificationPayload) bool {
	if p.SignatureKey == "" || s.serverKey == "" {
		return false
	}
	sum := sha512.Sum512([]byte(p.OrderID + p.StatusCode + p.GrossAmount + s.serverKey))
	return hex.EncodeToString(sum[:]) == p.SignatureKey
}

// HandleNotification processes a verified Midtrans webhook. It re-checks the
// transaction server-to-server (the webhook body alone is never trusted),
// updates the local Payment, and grants premium on settlement. Idempotent:
// repeated notifications for an already-settled payment are no-ops.
func (s *PaymentService) HandleNotification(p NotificationPayload) error {
	if !s.VerifySignature(p) {
		return fmt.Errorf("invalid signature: %w", ErrForbidden)
	}

	var payment models.Payment
	if err := s.db.First(&payment, "order_id = ?", p.OrderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("payment %s not found: %w", p.OrderID, ErrNotFound)
		}
		return err
	}

	// Idempotency: nothing to do once settled.
	if payment.Status == models.PaymentStatusSettlement {
		return nil
	}

	// Ground truth from Midtrans — the webhook body only tells us to look.
	status, err := s.gateway.CheckTransaction(p.OrderID)
	if err != nil {
		return err
	}

	newStatus := mapMidtransStatus(status)
	if newStatus == payment.Status {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": newStatus}
		if newStatus == models.PaymentStatusSettlement {
			now := time.Now().UTC()
			updates["paid_at"] = &now
		}
		if err := tx.Model(&payment).Updates(updates).Error; err != nil {
			return fmt.Errorf("could not update payment: %w", err)
		}

		if newStatus == models.PaymentStatusSettlement {
			if err := tx.Model(&models.User{}).
				Where("id = ?", payment.UserID).
				Updates(map[string]any{
					"subscription_tier": models.SubscriptionPremium,
					"premium_until":     nil, // lifetime
				}).Error; err != nil {
				return fmt.Errorf("could not grant premium: %w", err)
			}
		}
		return nil
	})
}

// mapMidtransStatus collapses Midtrans transaction/fraud statuses onto the
// local PaymentStatus set.
func mapMidtransStatus(s *TransactionStatus) models.PaymentStatus {
	switch s.TransactionStatus {
	case "capture":
		if s.FraudStatus == "accept" {
			return models.PaymentStatusSettlement
		}
		return models.PaymentStatusPending // challenge → manual review
	case "settlement":
		return models.PaymentStatusSettlement
	case "expire":
		return models.PaymentStatusExpired
	case "cancel", "deny":
		return models.PaymentStatusFailed
	default: // pending and anything unknown
		return models.PaymentStatusPending
	}
}

// GetPaymentStatus returns a user's own payment by order ID.
func (s *PaymentService) GetPaymentStatus(userID uuid.UUID, orderID string) (*models.Payment, error) {
	var payment models.Payment
	if err := s.db.
		Where("order_id = ? AND user_id = ?", orderID, userID).
		First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &payment, nil
}
