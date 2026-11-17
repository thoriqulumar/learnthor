package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/learnthor/backend/services"
)

// PaymentHandler wires premium purchases and the Midtrans webhook to HTTP.
type PaymentHandler struct {
	svc *services.PaymentService
}

// NewPaymentHandler constructs a PaymentHandler.
func NewPaymentHandler(svc *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// CreateTransaction godoc
//
//	@Summary     Start a premium purchase
//	@Description Creates (or reuses) a pending Midtrans Snap transaction for
//	             the authenticated user and returns the token + redirect URL
//	             the frontend needs to open the checkout.
//	@Tags        payments
//	@Produce     json
//	@Success     200 {object} services.CreateTransactionResponse
//	@Failure     400 {object} errorResponse
//	@Failure     500 {object} errorResponse
//	@Router      /payments/create [post]
func (h *PaymentHandler) CreateTransaction(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	resp, err := h.svc.CreateTransaction(services.CreateTransactionRequest{UserID: userID})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: "user not found"})
		case errors.Is(err, services.ErrBadRequest):
			c.JSON(http.StatusBadRequest, errorResponse{Code: "ALREADY_PREMIUM", Message: "you already have premium access"})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleNotification godoc
//
//	@Summary     Midtrans payment webhook
//	@Description Receives Midtrans transaction-status notifications. The
//	             signature is verified, then the transaction is re-checked
//	             server-to-server before any state change. Always answers
//	             200 quickly for valid-but-unactionable notifications so
//	             Midtrans does not retry-storm.
//	@Tags        payments
//	@Accept      json
//	@Produce     json
//	@Param       body body services.NotificationPayload true "Midtrans notification"
//	@Success     200 {object} map[string]string
//	@Failure     403 {object} errorResponse
//	@Router      /payments/notification [post]
func (h *PaymentHandler) HandleNotification(c *gin.Context) {
	var payload services.NotificationPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_PAYLOAD", Message: err.Error()})
		return
	}

	if err := h.svc.HandleNotification(payload); err != nil {
		switch {
		case errors.Is(err, services.ErrForbidden):
			c.JSON(http.StatusForbidden, errorResponse{Code: "INVALID_SIGNATURE", Message: "signature verification failed"})
		case errors.Is(err, services.ErrNotFound):
			// Acknowledge anyway — Midtrans retries on non-200, and an unknown
			// order can never become valid.
			c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetPayment godoc
//
//	@Summary     Get own payment status
//	@Description Returns the authenticated user's payment for an order ID.
//	             The frontend polls this after the Snap popup closes to learn
//	             whether the payment settled.
//	@Tags        payments
//	@Produce     json
//	@Param       order_id path string true "Order ID (LTH-...)"
//	@Success     200 {object} models.Payment
//	@Failure     404 {object} errorResponse
//	@Router      /payments/{order_id} [get]
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	orderID := c.Param("order_id")

	payment, err := h.svc.GetPaymentStatus(userID, orderID)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "an unexpected error occurred"})
		return
	}

	c.JSON(http.StatusOK, payment)
}
