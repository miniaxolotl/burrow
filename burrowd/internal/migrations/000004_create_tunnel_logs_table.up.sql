CREATE TABLE tunnel_logs (
    id BIGSERIAL PRIMARY KEY,
    tunnel_id TEXT NOT NULL REFERENCES tunnels(tunnel_id),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    method TEXT NOT NULL,
    path TEXT NOT NULL DEFAULT '/',
    status_code INTEGER NOT NULL DEFAULT 0,
    size BIGINT NOT NULL DEFAULT 0,
    duration TEXT NOT NULL DEFAULT '0ms',
    ip TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_tunnel_logs_tunnel_id ON tunnel_logs (tunnel_id, timestamp DESC);