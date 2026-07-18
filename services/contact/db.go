package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// store is the Postgres-backed persistence for waitlist signups and payments.
type store struct {
	pool *pgxpool.Pool
}

const schema = `
CREATE TABLE IF NOT EXISTS waitlist (
	id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	email      TEXT NOT NULL UNIQUE,
	name       TEXT NOT NULL DEFAULT '',
	source     TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS payments (
	id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	razorpay_payment_id TEXT NOT NULL UNIQUE,
	order_id            TEXT NOT NULL DEFAULT '',
	email               TEXT NOT NULL DEFAULT '',
	amount              BIGINT NOT NULL DEFAULT 0,
	currency            TEXT NOT NULL DEFAULT '',
	status              TEXT NOT NULL DEFAULT '',
	raw                 JSONB,
	created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS scans (
	id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	host           TEXT NOT NULL,
	security_grade TEXT NOT NULL DEFAULT '',
	tls_grade      TEXT NOT NULL DEFAULT '',
	seo_grade      TEXT NOT NULL DEFAULT '',
	ip             TEXT NOT NULL DEFAULT '',
	created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);`

// newStore connects, pings (with a short retry so we tolerate Postgres still
// starting after a reboot), and ensures the schema exists.
func newStore(ctx context.Context, url string) (*store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	var pingErr error
	for i := 0; i < 10; i++ {
		if pingErr = pool.Ping(ctx); pingErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("ping postgres: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
	if pingErr != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", pingErr)
	}

	s := &store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// addWaitlist inserts a signup, returning false if the email already exists.
func (s *store) addWaitlist(ctx context.Context, email, name, source string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO waitlist (email, name, source) VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO NOTHING`,
		email, name, source)
	if err != nil {
		return false, fmt.Errorf("insert waitlist: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

type paymentRecord struct {
	PaymentID string
	OrderID   string
	Email     string
	Amount    int64
	Currency  string
	Status    string
	Raw       []byte
}

// recordPayment upserts a payment; duplicate webhook deliveries are idempotent.
func (s *store) recordPayment(ctx context.Context, p paymentRecord) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO payments (razorpay_payment_id, order_id, email, amount, currency, status, raw)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (razorpay_payment_id) DO UPDATE SET status = EXCLUDED.status`,
		p.PaymentID, p.OrderID, p.Email, p.Amount, p.Currency, p.Status, p.Raw)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

// recordScan logs a site-check result — a demand signal (what people scan).
func (s *store) recordScan(ctx context.Context, host, sec, tlsg, seo, ip string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO scans (host, security_grade, tls_grade, seo_grade, ip)
		 VALUES ($1, $2, $3, $4, $5)`,
		host, sec, tlsg, seo, ip)
	if err != nil {
		return fmt.Errorf("insert scan: %w", err)
	}
	return nil
}

func (s *store) close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}
