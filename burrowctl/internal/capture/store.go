// Package capture owns optional encrypted local capture and durable replay work.
// Ordinary tunneling never opens this store or accesses the OS credential backend.
package capture

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"burrow/burrowctl/internal/localstate"
	"burrow/burrowctl/internal/routing"
	_ "modernc.org/sqlite"
)

const MaxBody = 1024 * 1024
const MaxRecords = 200

// SQLite and a worst-case rollback journal together fit below the 50 MiB
// physical envelope; payload accounting leaves room for bounded metadata.
const MaxPayloadBytes = 20 * 1024 * 1024
const TTL = 24 * time.Hour

var ErrUnavailable = errors.New("capture unavailable, expired or erased")
var ErrQuota = errors.New("capture quota unavailable")

type Store struct {
	db         *sql.DB
	dir, scope string
	aead       cipher.AEAD
}
type Record struct {
	ID           string        `json:"id"`
	TunnelID     string        `json:"tunnel_id"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	RawQuery     string        `json:"raw_query"`
	Headers      http.Header   `json:"headers"`
	Body         []byte        `json:"body"`
	Host         string        `json:"host"`
	Route        routing.Route `json:"route"`
	ConfigDigest string        `json:"config_digest"`
	ConfigPath   string        `json:"config_path"`
	Redacted     bool          `json:"redacted"`
	CreatedAt    time.Time     `json:"created_at"`
}
type Summary struct {
	ID         string    `json:"id"`
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	SizeBytes  int       `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
	Replayable bool      `json:"replayable"`
	Redacted   bool      `json:"redacted"`
	LastJob    *Job      `json:"last_job,omitempty"`
}

func Hash(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }

func Open(dir, scope string, keys KeyStore) (*Store, error) {
	if scope == "" || keys == nil {
		return nil, errors.New("capture scope and secure key backend required")
	}
	if err := localstate.Directory(dir); err != nil {
		return nil, err
	}
	lock, err := localstate.Lock(filepath.Join(dir, "init.lock"), true)
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	key, err := keys.Key(scope)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("invalid secure capture key")
	}
	block, err := aes.NewCipher(key)
	for i := range key {
		key[i] = 0
	}
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "capture.sqlite")
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		if err := localstate.Write(path, nil); err != nil {
			return nil, err
		}
	}
	for _, suffix := range []string{"", "-journal", "-wal", "-shm"} {
		if err := localstate.Check(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	u := url.URL{Scheme: "file", Path: path}
	q := u.Query()
	q.Set("_txlock", "immediate")
	q.Set("_pragma", "busy_timeout(1000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, dir: dir, scope: scope, aead: aead}
	if err := s.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) initialize() error {
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version != 0 && version != 1 {
		return errors.New("unsupported capture schema; no files rewritten")
	}
	_, err := s.db.Exec(`PRAGMA page_size=4096; PRAGMA journal_mode=DELETE; PRAGMA synchronous=FULL; PRAGMA temp_store=MEMORY; PRAGMA cache_size=-2048; PRAGMA max_page_count=6144; PRAGMA secure_delete=ON;
CREATE TABLE IF NOT EXISTS captures(id TEXT PRIMARY KEY, tunnel TEXT NOT NULL, created_ms INTEGER NOT NULL, expires_ms INTEGER NOT NULL, payload BLOB, deleted INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS epochs(tunnel TEXT PRIMARY KEY, generation INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS erased(id TEXT PRIMARY KEY, tunnel TEXT NOT NULL, expires_ms INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS maintenance(id INTEGER PRIMARY KEY CHECK(id=1), state TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS jobs(id TEXT PRIMARY KEY,capture_id TEXT NOT NULL,tunnel TEXT NOT NULL,state TEXT NOT NULL,generation INTEGER NOT NULL DEFAULT 0,lease_ms INTEGER NOT NULL DEFAULT 0,deadline_ms INTEGER NOT NULL,created_ms INTEGER NOT NULL,updated_ms INTEGER NOT NULL,intent INTEGER NOT NULL DEFAULT 0,http_status INTEGER,code TEXT NOT NULL DEFAULT '',parent_id TEXT NOT NULL DEFAULT '',digest TEXT NOT NULL,attempts INTEGER NOT NULL DEFAULT 0,next_ms INTEGER NOT NULL DEFAULT 0);
PRAGMA user_version=1;`)
	return err
}
func (s *Store) Close() error { return s.db.Close() }
func now(tx *sql.Tx) (int64, error) {
	var n int64
	err := tx.QueryRow("SELECT CAST(unixepoch('subsec')*1000 AS INTEGER)").Scan(&n)
	return n, err
}
func (s *Store) transaction(fn func(*sql.Tx, int64) error) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	n, err := now(tx)
	if err != nil {
		return err
	}
	if err := fn(tx, n); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Epoch(tunnel string) (int64, error) {
	var epoch int64
	err := s.transaction(func(tx *sql.Tx, _ int64) error {
		var count, exists int
		if err := tx.QueryRow("SELECT count(*) FROM epochs").Scan(&count); err != nil {
			return err
		}
		if err := tx.QueryRow("SELECT count(*) FROM epochs WHERE tunnel=?", Hash(tunnel)).Scan(&exists); err != nil {
			return err
		}
		if count >= 2048 && exists == 0 {
			return ErrQuota
		}
		_, err := tx.Exec("INSERT OR IGNORE INTO epochs VALUES(?,0)", Hash(tunnel))
		if err != nil {
			return err
		}
		return tx.QueryRow("SELECT generation FROM epochs WHERE tunnel=?", Hash(tunnel)).Scan(&epoch)
	})
	return epoch, err
}
func (s *Store) seal(id string, plain []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, plain, []byte("burrow-capture-v1\x00"+s.scope+"\x00"+id)), nil
}
func (s *Store) unseal(id string, payload []byte) (Record, error) {
	var r Record
	n := s.aead.NonceSize()
	if len(payload) < n {
		return r, ErrUnavailable
	}
	plain, err := s.aead.Open(nil, payload[:n], payload[n:], []byte("burrow-capture-v1\x00"+s.scope+"\x00"+id))
	if err != nil {
		return r, ErrUnavailable
	}
	if json.Unmarshal(plain, &r) != nil || r.ID != id {
		return r, ErrUnavailable
	}
	return r, nil
}

func (s *Store) Put(r Record, epoch int64) error {
	if r.ID == "" || r.TunnelID == "" || len(r.Body) > MaxBody || r.Route.Port == 0 {
		return ErrUnavailable
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if time.Since(r.CreatedAt) > TTL {
		return ErrUnavailable
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(data) > 2*MaxBody {
		return ErrQuota
	}
	payload, err := s.seal(r.ID, data)
	if err != nil {
		return err
	}
	return s.transaction(func(tx *sql.Tx, n int64) error {
		if _, err := tx.Exec("UPDATE captures SET payload=NULL,deleted=1 WHERE expires_ms<=?", n); err != nil {
			return err
		}
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		if err := expireJobs(tx, n); err != nil {
			return err
		}
		var current int64
		if err := tx.QueryRow("SELECT generation FROM epochs WHERE tunnel=?", Hash(r.TunnelID)).Scan(&current); err != nil || current != epoch {
			return ErrUnavailable
		}
		var erased int
		if err := tx.QueryRow("SELECT count(*) FROM erased WHERE id=? AND tunnel=?", r.ID, Hash(r.TunnelID)).Scan(&erased); err != nil {
			return err
		}
		if erased > 0 {
			return ErrUnavailable
		}
		var retained int
		if err := tx.QueryRow("SELECT count(*) FROM captures").Scan(&retained); err != nil {
			return err
		}
		if retained >= 2048 {
			return ErrQuota
		}
		for {
			var count, total int
			if err := tx.QueryRow("SELECT count(*),coalesce(sum(length(payload)),0) FROM captures WHERE payload IS NOT NULL").Scan(&count, &total); err != nil {
				return err
			}
			if count < MaxRecords && total+len(payload) <= MaxPayloadBytes {
				break
			}
			var oldest string
			if err := tx.QueryRow("SELECT id FROM captures WHERE payload IS NOT NULL AND NOT EXISTS(SELECT 1 FROM jobs WHERE jobs.capture_id=captures.id AND state IN ('queued','processing','retry_wait','recovering')) ORDER BY created_ms LIMIT 1").Scan(&oldest); err != nil {
				return ErrQuota
			}
			if _, err := tx.Exec("UPDATE captures SET payload=NULL,deleted=1 WHERE id=?", oldest); err != nil {
				return err
			}
		}
		_, err := tx.Exec("INSERT INTO captures(id,tunnel,created_ms,expires_ms,payload) VALUES(?,?,?,?,?)", r.ID, Hash(r.TunnelID), r.CreatedAt.UnixMilli(), r.CreatedAt.Add(TTL).UnixMilli(), payload)
		return err
	})
}
func read(tx *sql.Tx, n int64, id, tunnel string) ([]byte, error) {
	var data []byte
	err := tx.QueryRow("SELECT payload FROM captures WHERE id=? AND tunnel=? AND deleted=0 AND expires_ms>?", id, Hash(tunnel), n).Scan(&data)
	if err != nil || data == nil {
		return nil, ErrUnavailable
	}
	return data, nil
}
func (s *Store) Read(id, tunnel string) (Record, error) {
	var payload []byte
	err := s.transaction(func(tx *sql.Tx, n int64) error { var err error; payload, err = read(tx, n, id, tunnel); return err })
	if err != nil {
		return Record{}, err
	}
	r, err := s.unseal(id, payload)
	if err == nil && r.TunnelID != tunnel {
		return Record{}, ErrUnavailable
	}
	return r, err
}
func (s *Store) List(tunnel string) ([]Summary, error) {
	items := []Summary{}
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		if err := expireJobs(tx, n); err != nil {
			return err
		}
		rows, err := tx.Query("SELECT id,payload FROM captures WHERE tunnel=? AND deleted=0 AND expires_ms>? AND payload IS NOT NULL ORDER BY created_ms DESC LIMIT 200", Hash(tunnel), n)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var payload []byte
			if err := rows.Scan(&id, &payload); err != nil {
				return err
			}
			r, err := s.unseal(id, payload)
			if err != nil {
				return err
			}
			if r.TunnelID != tunnel {
				return ErrUnavailable
			}
			job, err := latestJob(tx, r.ID, Hash(tunnel))
			if err != nil {
				return err
			}
			items = append(items, Summary{r.ID, r.Method, r.Route.Path, len(r.Body), r.CreatedAt, true, r.Redacted, job})
		}
		return rows.Err()
	})
	return items, err
}
func (s *Store) Delete(id, tunnel string) error {
	return s.transaction(func(tx *sql.Tx, n int64) error {
		var other int
		if err := tx.QueryRow("SELECT count(*) FROM captures WHERE id=? AND tunnel<>?", id, Hash(tunnel)).Scan(&other); err != nil {
			return err
		}
		if other > 0 {
			return ErrUnavailable
		}
		var count int
		if err := tx.QueryRow("SELECT count(*) FROM erased").Scan(&count); err != nil {
			return err
		}
		if count >= 2048 {
			return ErrQuota
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO erased VALUES(?,?,?)", id, Hash(tunnel), n+(7*24*time.Hour).Milliseconds()); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE captures SET payload=NULL,deleted=1 WHERE id=? AND tunnel=?", id, Hash(tunnel)); err != nil {
			return err
		}
		return invalidateJobs(tx, n)
	})
}
func (s *Store) Clear(tunnel string) error {
	return s.transaction(func(tx *sql.Tx, n int64) error {
		result, err := tx.Exec("UPDATE epochs SET generation=generation+1 WHERE tunnel=?", Hash(tunnel))
		if err != nil {
			return err
		}
		if count, err := result.RowsAffected(); err != nil || count != 1 {
			return ErrUnavailable
		}
		if _, err := tx.Exec("DELETE FROM captures WHERE tunnel=?", Hash(tunnel)); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM erased WHERE tunnel=?", Hash(tunnel)); err != nil {
			return err
		}
		return invalidateJobs(tx, n)
	})
}

// Cleanup acceptance is durable. No daemon is assumed: execute pending cleanup
// only while this foreground CLI worker runs, never promise wall-clock erasure.
func (s *Store) Cleanup() error {
	if _, err := s.db.Exec("INSERT INTO maintenance VALUES(1,'queued') ON CONFLICT(id) DO UPDATE SET state='queued'"); err != nil {
		return err
	}
	return s.transaction(func(tx *sql.Tx, n int64) error {
		if _, err := tx.Exec("UPDATE captures SET payload=NULL,deleted=1 WHERE expires_ms<=?", n); err != nil {
			return err
		}
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		if err := expireJobs(tx, n); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM jobs WHERE updated_ms<? AND state IN ('failed','uncertain','cancelled','completed')", n-(7*24*time.Hour).Milliseconds()); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM captures WHERE deleted=1 AND expires_ms<?", n-(7*24*time.Hour).Milliseconds()); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM erased WHERE expires_ms<=?", n); err != nil {
			return err
		}
		_, err := tx.Exec("UPDATE maintenance SET state='completed' WHERE id=1")
		return err
	})
}
