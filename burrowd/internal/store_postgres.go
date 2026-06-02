package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresClient struct {
	pool *pgxpool.Pool
}

func NewPostgresClient(ctx context.Context, connString string) (*PostgresClient, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresClient{pool: pool}, nil
}

func (pg *PostgresClient) Close() error {
	pg.pool.Close()
	return nil
}

func (pg *PostgresClient) Migrate(ctx context.Context) error {
	src, err := MigrationsSource()
	if err != nil {
		return fmt.Errorf("create migrations source: %w", err)
	}
	connString := pg.pool.Config().ConnString()
	m, err := migrate.NewWithSourceInstance("iofs", src, connString)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (pg *PostgresClient) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := pg.pool.QueryRow(ctx,
		`SELECT id, uuid, email, password_hash, role, plan, stripe_customer_id, created_at, updated_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`, email,
	).Scan(&u.ID, &u.UUID, &u.Email, &u.PasswordHash, &u.Role, &u.Plan, &u.StripeCustomerID, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (pg *PostgresClient) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := pg.pool.QueryRow(ctx,
		`SELECT id, uuid, email, password_hash, role, plan, stripe_customer_id, created_at, updated_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&u.ID, &u.UUID, &u.Email, &u.PasswordHash, &u.Role, &u.Plan, &u.StripeCustomerID, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (pg *PostgresClient) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	var u User
	err := pg.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, uuid, email, role, plan, stripe_customer_id, created_at, updated_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.UUID, &u.Email, &u.Role, &u.Plan, &u.StripeCustomerID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (pg *PostgresClient) UpdateUserPlan(ctx context.Context, id int64, plan string) error {
	_, err := pg.pool.Exec(ctx, `UPDATE users SET plan = $1, updated_at = NOW() WHERE id = $2`, plan, id)
	return err
}

func (pg *PostgresClient) UpdateUserStripeCustomerID(ctx context.Context, id int64, customerID string) error {
	_, err := pg.pool.Exec(ctx, `UPDATE users SET stripe_customer_id = $1, updated_at = NOW() WHERE id = $2`, customerID, id)
	return err
}

func (pg *PostgresClient) CreateTunnel(ctx context.Context, tunnelID string, port int, clientIP string, userID *int64) (*Tunnel, error) {
	var t Tunnel
	err := pg.pool.QueryRow(ctx,
		`INSERT INTO tunnels (tunnel_id, port, client_ip, user_id) VALUES ($1, $2, $3, $4)
		 RETURNING id, tunnel_id, port, client_ip, user_id, status, started_at`,
		tunnelID, port, clientIP, userID,
	).Scan(&t.ID, &t.TunnelID, &t.Port, &t.ClientIP, &t.UserID, &t.Status, &t.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("create tunnel: %w", err)
	}
	return &t, nil
}

func (pg *PostgresClient) UpdateTunnelStatus(ctx context.Context, tunnelID, status string) error {
	_, err := pg.pool.Exec(ctx,
		`UPDATE tunnels SET status = $1, stopped_at = NOW() WHERE tunnel_id = $2 AND status = 'active'`,
		status, tunnelID,
	)
	return err
}

func (pg *PostgresClient) DeleteTunnel(ctx context.Context, tunnelID string) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM tunnels WHERE tunnel_id = $1`, tunnelID)
	return err
}

func (pg *PostgresClient) GetTunnelByTunnelID(ctx context.Context, tunnelID string) (*Tunnel, error) {
	var t Tunnel
	err := pg.pool.QueryRow(ctx,
		`SELECT id, tunnel_id, port, client_ip, user_id, status, started_at, stopped_at
		 FROM tunnels WHERE tunnel_id = $1`, tunnelID,
	).Scan(&t.ID, &t.TunnelID, &t.Port, &t.ClientIP, &t.UserID, &t.Status, &t.StartedAt, &t.StoppedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (pg *PostgresClient) ListTunnelsByUser(ctx context.Context, userID int64) ([]*Tunnel, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, tunnel_id, port, client_ip, user_id, status, started_at, stopped_at
		 FROM tunnels WHERE user_id = $1 ORDER BY started_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tunnels []*Tunnel
	for rows.Next() {
		var t Tunnel
		if err := rows.Scan(&t.ID, &t.TunnelID, &t.Port, &t.ClientIP, &t.UserID, &t.Status, &t.StartedAt, &t.StoppedAt); err != nil {
			return nil, err
		}
		tunnels = append(tunnels, &t)
	}
	return tunnels, nil
}

func (pg *PostgresClient) ListActiveTunnels(ctx context.Context) ([]*Tunnel, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, tunnel_id, port, client_ip, user_id, status, started_at, stopped_at
		 FROM tunnels WHERE status = 'active' ORDER BY started_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tunnels []*Tunnel
	for rows.Next() {
		var t Tunnel
		if err := rows.Scan(&t.ID, &t.TunnelID, &t.Port, &t.ClientIP, &t.UserID, &t.Status, &t.StartedAt, &t.StoppedAt); err != nil {
			return nil, err
		}
		tunnels = append(tunnels, &t)
	}
	return tunnels, nil
}

func (pg *PostgresClient) GetActiveTunnelCountByIP(ctx context.Context, clientIP string) (int, error) {
	var count int
	err := pg.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tunnels WHERE client_ip = $1 AND status = 'active'`, clientIP,
	).Scan(&count)
	return count, err
}

func (pg *PostgresClient) GetActiveTunnelCountByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := pg.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tunnels WHERE user_id = $1 AND status = 'active'`, userID,
	).Scan(&count)
	return count, err
}

func (pg *PostgresClient) UpsertTunnelStat(ctx context.Context, tunnelID string, bucketStart time.Time, bytesIn, bytesOut, requestCount int64) error {
	_, err := pg.pool.Exec(ctx,
		`INSERT INTO tunnel_stats (tunnel_id, bucket_start, bytes_in, bytes_out, request_count)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tunnel_id, bucket_start) DO UPDATE SET
		   bytes_in = tunnel_stats.bytes_in + EXCLUDED.bytes_in,
		   bytes_out = tunnel_stats.bytes_out + EXCLUDED.bytes_out,
		   request_count = tunnel_stats.request_count + EXCLUDED.request_count`,
		tunnelID, bucketStart, bytesIn, bytesOut, requestCount,
	)
	return err
}

func (pg *PostgresClient) GetTunnelStats(ctx context.Context, tunnelID string, from, to time.Time) ([]*TunnelStat, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, tunnel_id, bucket_start, bytes_in, bytes_out, request_count
		 FROM tunnel_stats WHERE tunnel_id = $1 AND bucket_start >= $2 AND bucket_start < $3
		 ORDER BY bucket_start ASC`, tunnelID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []*TunnelStat
	for rows.Next() {
		var s TunnelStat
		if err := rows.Scan(&s.ID, &s.TunnelID, &s.BucketStart, &s.BytesIn, &s.BytesOut, &s.RequestCount); err != nil {
			return nil, err
		}
		stats = append(stats, &s)
	}
	return stats, nil
}

func (pg *PostgresClient) InsertTunnelLog(ctx context.Context, tunnelID string, entry *TunnelLog) error {
	_, err := pg.pool.Exec(ctx,
		`INSERT INTO tunnel_logs (tunnel_id, timestamp, method, path, status_code, size, duration, ip)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tunnelID, entry.Timestamp, entry.Method, entry.Path, entry.StatusCode, entry.Size, entry.Duration, entry.IP,
	)
	return err
}

func (pg *PostgresClient) GetTunnelLogs(ctx context.Context, tunnelID string, from, to time.Time, limit, offset int) ([]*TunnelLog, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, tunnel_id, timestamp, method, path, status_code, size, duration, ip
		 FROM tunnel_logs WHERE tunnel_id = $1 AND timestamp >= $2 AND timestamp < $3
		 ORDER BY timestamp DESC LIMIT $4 OFFSET $5`, tunnelID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []*TunnelLog
	for rows.Next() {
		var l TunnelLog
		if err := rows.Scan(&l.ID, &l.TunnelID, &l.Timestamp, &l.Method, &l.Path, &l.StatusCode, &l.Size, &l.Duration, &l.IP); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

func (pg *PostgresClient) GetUniqueIPCount(ctx context.Context, tunnelID string, from, to time.Time) (int, error) {
	var count int
	err := pg.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT ip) FROM tunnel_logs WHERE tunnel_id = $1 AND timestamp >= $2 AND timestamp < $3`,
		tunnelID, from, to,
	).Scan(&count)
	return count, err
}

func (pg *PostgresClient) CreateSession(ctx context.Context, userID int64, refreshToken, userAgent, ipAddress string, expiresAt time.Time) (*Session, error) {
	var s Session
	err := pg.pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, refresh_token, user_agent, ip_address, expires_at)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at`,
		userID, refreshToken, userAgent, ipAddress, expiresAt,
	).Scan(&s.ID, &s.UserID, &s.RefreshToken, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &s, nil
}

func (pg *PostgresClient) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	var s Session
	err := pg.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at
		 FROM sessions WHERE refresh_token = $1 AND expires_at > NOW()`, refreshToken,
	).Scan(&s.ID, &s.UserID, &s.RefreshToken, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (pg *PostgresClient) DeleteSession(ctx context.Context, refreshToken string) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM sessions WHERE refresh_token = $1`, refreshToken)
	return err
}

func (pg *PostgresClient) CreateSubscription(ctx context.Context, userID int64, stripeSubscriptionID, stripePriceID, status string, currentPeriodStart, currentPeriodEnd time.Time) (*Subscription, error) {
	var sub Subscription
	err := pg.pool.QueryRow(ctx,
		`INSERT INTO subscriptions (user_id, stripe_subscription_id, stripe_price_id, status, current_period_start, current_period_end)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, stripe_subscription_id, stripe_price_id, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at`,
		userID, stripeSubscriptionID, stripePriceID, status, currentPeriodStart, currentPeriodEnd,
	).Scan(&sub.ID, &sub.UserID, &sub.StripeSubscriptionID, &sub.StripePriceID, &sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CancelAtPeriodEnd, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return &sub, nil
}

func (pg *PostgresClient) GetSubscriptionByUserID(ctx context.Context, userID int64) (*Subscription, error) {
	var sub Subscription
	err := pg.pool.QueryRow(ctx,
		`SELECT id, user_id, stripe_subscription_id, stripe_price_id, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at
		 FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID,
	).Scan(&sub.ID, &sub.UserID, &sub.StripeSubscriptionID, &sub.StripePriceID, &sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CancelAtPeriodEnd, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (pg *PostgresClient) UpdateSubscriptionStatus(ctx context.Context, stripeSubscriptionID, status string) error {
	_, err := pg.pool.Exec(ctx,
		`UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE stripe_subscription_id = $2`,
		status, stripeSubscriptionID,
	)
	return err
}