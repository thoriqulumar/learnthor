package services

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/learnthor/backend/models"
)

// fakeGateway implements MidtransGateway for tests.
type fakeGateway struct {
	token, url string
	createErr  error
	status     *TransactionStatus
	statusErr  error
	checkedIDs []string
}

func (f *fakeGateway) CreateSnapTransaction(orderID string, grossAmount int64) (string, string, error) {
	if f.createErr != nil {
		return "", "", f.createErr
	}
	return f.token, f.url, nil
}

func (f *fakeGateway) CheckTransaction(orderID string) (*TransactionStatus, error) {
	f.checkedIDs = append(f.checkedIDs, orderID)
	return f.status, f.statusErr
}

func newPaymentService(t *testing.T, gw MidtransGateway) (*PaymentService, *gorm.DB) {
	t.Helper()
	t.Setenv("MIDTRANS_SERVER_KEY", "test-server-key")
	t.Setenv("MIDTRANS_CLIENT_KEY", "test-client-key")
	t.Setenv("PREMIUM_PRICE_IDR", "99000")
	db := newTestDB(t)
	return NewPaymentService(db, gw), db
}

// sign computes the expected signature_key for a notification payload.
func sign(serverKey, orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	return hex.EncodeToString(sum[:])
}

func TestCreateTransaction_CreatesPendingPayment(t *testing.T) {
	gw := &fakeGateway{token: "snap-tok", url: "https://snap.test/pay"}
	svc, db := newPaymentService(t, gw)
	user := createTestUser(t, db, models.UserRoleStudent)

	resp, err := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if resp.SnapToken != "snap-tok" || resp.SnapURL != "https://snap.test/pay" {
		t.Fatalf("unexpected snap data: %+v", resp)
	}
	if resp.GrossAmount != 99000 {
		t.Fatalf("amount = %d, want 99000", resp.GrossAmount)
	}
	if resp.ClientKey != "test-client-key" {
		t.Fatalf("client key = %q", resp.ClientKey)
	}
	if resp.Reused {
		t.Fatal("first transaction should not be flagged as reused")
	}
}

func TestCreateTransaction_ReusesPendingPayment(t *testing.T) {
	gw := &fakeGateway{token: "tok1", url: "https://snap.test/1"}
	svc, db := newPaymentService(t, gw)
	user := createTestUser(t, db, models.UserRoleStudent)

	first, err := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first.OrderID != second.OrderID {
		t.Fatalf("order ids differ: %s vs %s — should reuse pending", first.OrderID, second.OrderID)
	}
	if !second.Reused {
		t.Fatal("second call should be flagged as reused")
	}

	var count int64
	db.Model(&models.Payment{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Fatalf("payments = %d, want 1 (no orphan orders)", count)
	}
}

func TestCreateTransaction_RejectsAlreadyPremium(t *testing.T) {
	gw := &fakeGateway{}
	svc, db := newPaymentService(t, gw)
	user := createTestUser(t, db, models.UserRoleStudent)
	db.Model(&user).Update("subscription_tier", models.SubscriptionPremium)

	_, err := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

func TestHandleNotification_SettlementGrantsPremium(t *testing.T) {
	gw := &fakeGateway{
		token: "tok", url: "https://snap.test/pay",
		status: &TransactionStatus{TransactionStatus: "settlement", StatusCode: "200", GrossAmount: "99000"},
	}
	svc, db := newPaymentService(t, gw)
	user := createTestUser(t, db, models.UserRoleStudent)

	resp, err := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	notif := NotificationPayload{
		OrderID:     resp.OrderID,
		StatusCode:  "200",
		GrossAmount: "99000",
	}
	notif.SignatureKey = sign("test-server-key", notif.OrderID, notif.StatusCode, notif.GrossAmount)

	if err := svc.HandleNotification(notif); err != nil {
		t.Fatalf("HandleNotification: %v", err)
	}

	// Payment settled.
	var payment models.Payment
	db.First(&payment, "order_id = ?", resp.OrderID)
	if payment.Status != models.PaymentStatusSettlement {
		t.Fatalf("status = %s, want settlement", payment.Status)
	}
	if payment.PaidAt == nil {
		t.Fatal("paid_at should be set")
	}

	// User upgraded.
	var updated models.User
	db.First(&updated, "id = ?", user.ID)
	if updated.SubscriptionTier != models.SubscriptionPremium {
		t.Fatalf("tier = %s, want premium", updated.SubscriptionTier)
	}

	// Ground-truth check hit Midtrans.
	if len(gw.checkedIDs) != 1 || gw.checkedIDs[0] != resp.OrderID {
		t.Fatalf("CheckTransaction calls = %v", gw.checkedIDs)
	}
}

func TestHandleNotification_RejectsBadSignature(t *testing.T) {
	gw := &fakeGateway{}
	svc, db := newPaymentService(t, gw)
	user := createTestUser(t, db, models.UserRoleStudent)
	resp, _ := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})

	err := svc.HandleNotification(NotificationPayload{
		OrderID:      resp.OrderID,
		StatusCode:   "200",
		GrossAmount:  "99000",
		SignatureKey: "deadbeef",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
	if len(gw.checkedIDs) != 0 {
		t.Fatal("CheckTransaction must not be called for bad signatures")
	}
}

func TestHandleNotification_IdempotentOnSettled(t *testing.T) {
	gw := &fakeGateway{
		token: "tok", url: "https://snap.test/pay",
		status: &TransactionStatus{TransactionStatus: "settlement", StatusCode: "200", GrossAmount: "99000"},
	}
	svc, db := newPaymentService(t, gw)
	user := createTestUser(t, db, models.UserRoleStudent)
	resp, _ := svc.CreateTransaction(CreateTransactionRequest{UserID: user.ID})

	notif := NotificationPayload{
		OrderID:     resp.OrderID,
		StatusCode:  "200",
		GrossAmount: "99000",
	}
	notif.SignatureKey = sign("test-server-key", notif.OrderID, notif.StatusCode, notif.GrossAmount)

	if err := svc.HandleNotification(notif); err != nil {
		t.Fatalf("first: %v", err)
	}
	calls := len(gw.checkedIDs)
	if err := svc.HandleNotification(notif); err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(gw.checkedIDs) != calls {
		t.Fatal("settled payment must skip the CheckTransaction round-trip")
	}
}

func TestHandleNotification_UnknownOrderNotFound(t *testing.T) {
	gw := &fakeGateway{}
	svc, _ := newPaymentService(t, gw)

	notif := NotificationPayload{
		OrderID:     "LTH-nonexistent",
		StatusCode:  "200",
		GrossAmount: "99000",
	}
	notif.SignatureKey = sign("test-server-key", notif.OrderID, notif.StatusCode, notif.GrossAmount)

	err := svc.HandleNotification(notif)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMapMidtransStatus(t *testing.T) {
	cases := []struct {
		in   TransactionStatus
		want models.PaymentStatus
	}{
		{TransactionStatus{TransactionStatus: "capture", FraudStatus: "accept"}, models.PaymentStatusSettlement},
		{TransactionStatus{TransactionStatus: "capture", FraudStatus: "challenge"}, models.PaymentStatusPending},
		{TransactionStatus{TransactionStatus: "settlement"}, models.PaymentStatusSettlement},
		{TransactionStatus{TransactionStatus: "expire"}, models.PaymentStatusExpired},
		{TransactionStatus{TransactionStatus: "cancel"}, models.PaymentStatusFailed},
		{TransactionStatus{TransactionStatus: "deny"}, models.PaymentStatusFailed},
		{TransactionStatus{TransactionStatus: "pending"}, models.PaymentStatusPending},
	}
	for _, tc := range cases {
		if got := mapMidtransStatus(&tc.in); got != tc.want {
			t.Fatalf("%+v: got %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestGetPaymentStatus_OnlyOwnPayments(t *testing.T) {
	gw := &fakeGateway{token: "tok", url: "https://snap.test/pay"}
	svc, db := newPaymentService(t, gw)
	owner := createTestUser(t, db, models.UserRoleStudent)
	other := createTestUser(t, db, models.UserRoleStudent)

	resp, _ := svc.CreateTransaction(CreateTransactionRequest{UserID: owner.ID})

	if _, err := svc.GetPaymentStatus(owner.ID, resp.OrderID); err != nil {
		t.Fatalf("owner lookup: %v", err)
	}
	if _, err := svc.GetPaymentStatus(other.ID, resp.OrderID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user lookup: err = %v, want ErrNotFound", err)
	}
}
