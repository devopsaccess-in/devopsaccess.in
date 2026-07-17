package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type razorpayHandler struct {
	store  *store
	mailer *mailer
	secret string
	log    zerolog.Logger
}

// razorpayEvent is the subset of the webhook payload we care about.
type razorpayEvent struct {
	Event   string `json:"event"`
	Payload struct {
		Payment struct {
			Entity struct {
				ID       string `json:"id"`
				OrderID  string `json:"order_id"`
				Email    string `json:"email"`
				Contact  string `json:"contact"`
				Amount   int64  `json:"amount"` // paise
				Currency string `json:"currency"`
				Status   string `json:"status"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

// verifySignature checks the HMAC-SHA256 of the raw body against the header.
func verifySignature(body []byte, sig, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func (h *razorpayHandler) handle(w http.ResponseWriter, r *http.Request) {
	if h.secret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payments not configured"})
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 256*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	if !verifySignature(body, r.Header.Get("X-Razorpay-Signature"), h.secret) {
		h.log.Warn().Str("ip", clientIP(r)).Msg("razorpay signature mismatch")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signature"})
		return
	}

	var ev razorpayEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Acknowledge non-payment events fast (200) so Razorpay stops retrying.
	if ev.Event != "payment.captured" {
		w.WriteHeader(http.StatusOK)
		return
	}

	e := ev.Payload.Payment.Entity
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.store.recordPayment(ctx, paymentRecord{
		PaymentID: e.ID, OrderID: e.OrderID, Email: e.Email,
		Amount: e.Amount, Currency: e.Currency, Status: e.Status, Raw: body,
	}); err != nil {
		h.log.Error().Err(err).Str("payment_id", e.ID).Msg("record payment failed")
		// 500 => Razorpay retries later, which is what we want.
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not record payment"})
		return
	}

	amount := fmt.Sprintf("%.2f %s", float64(e.Amount)/100, e.Currency)
	body2 := fmt.Sprintf("Payment captured\nAmount: %s\nEmail:  %s\nContact: %s\nPayment ID: %s\nOrder ID: %s",
		amount, e.Email, e.Contact, e.ID, e.OrderID)
	if err := h.mailer.notify(ctx, e.Email, "Payment received: "+amount, body2); err != nil {
		h.log.Error().Err(err).Msg("payment notify email failed")
	}
	h.log.Info().Str("payment_id", e.ID).Str("amount", amount).Msg("payment captured")

	w.WriteHeader(http.StatusOK)
}
