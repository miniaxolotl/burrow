package capture

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"time"
)

var ErrJobConflict = errors.New("replay job is active, terminal or fenced; inspect status and explicitly approve a linked retry")

type Job struct {
	ID             string     `json:"id"`
	CaptureID      string     `json:"capture_id"`
	State          string     `json:"state"`
	Generation     int64      `json:"generation"`
	HTTPStatus     *int       `json:"http_status"`
	Code           string     `json:"code"`
	ParentID       string     `json:"parent_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeadlineAt     time.Time  `json:"deadline_at"`
	DispatchIntent bool       `json:"dispatch_intent"`
	Attempts       int        `json:"preparation_attempts"`
	RetryAt        *time.Time `json:"retry_at,omitempty"`
	lease          int64
	tunnel, digest string
}

func getJob(tx *sql.Tx, id string) (Job, error) {
	var j Job
	var status sql.NullInt64
	var created, updated, deadline, next int64
	var intent int
	err := tx.QueryRow("SELECT id,capture_id,tunnel,state,generation,lease_ms,deadline_ms,created_ms,updated_ms,intent,http_status,code,parent_id,digest,attempts,next_ms FROM jobs WHERE id=?", id).Scan(&j.ID, &j.CaptureID, &j.tunnel, &j.State, &j.Generation, &j.lease, &deadline, &created, &updated, &intent, &status, &j.Code, &j.ParentID, &j.digest, &j.Attempts, &next)
	if err != nil {
		return j, ErrUnavailable
	}
	if status.Valid {
		n := int(status.Int64)
		j.HTTPStatus = &n
	}
	if next > 0 {
		at := time.UnixMilli(next).UTC()
		j.RetryAt = &at
	}
	j.DispatchIntent = intent == 1
	j.CreatedAt = time.UnixMilli(created).UTC()
	j.UpdatedAt = time.UnixMilli(updated).UTC()
	j.DeadlineAt = time.UnixMilli(deadline).UTC()
	return j, nil
}
func invalidateJobs(tx *sql.Tx, n int64) error {
	_, err := tx.Exec(`UPDATE jobs SET state=CASE WHEN intent=1 THEN 'uncertain' ELSE 'cancelled' END,code='capture_unavailable',updated_ms=?
WHERE state IN ('queued','processing','retry_wait','recovering') AND NOT EXISTS(SELECT 1 FROM captures WHERE captures.id=jobs.capture_id AND captures.tunnel=jobs.tunnel AND deleted=0 AND payload IS NOT NULL AND expires_ms>?)`, n, n)
	return err
}

func expireJobs(tx *sql.Tx, n int64) error {
	if _, err := tx.Exec("UPDATE jobs SET state=CASE WHEN intent=1 THEN 'uncertain' ELSE 'recovering' END,code='lease_expired',updated_ms=? WHERE state IN ('processing','retry_wait') AND lease_ms<=?", n, n); err != nil {
		return err
	}
	_, err := tx.Exec("UPDATE jobs SET state='failed',code='deadline_expired',updated_ms=? WHERE state IN ('queued','recovering') AND deadline_ms<=?", n, n)
	return err
}
func (s *Store) Status(id string) (Job, error) {
	var j Job
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		if err := expireJobs(tx, n); err != nil {
			return err
		}
		var err error
		j, err = getJob(tx, id)
		return err
	})
	return j, err
}

func (s *Store) preparation(j Job, waiting bool, attempt int) (Job, error) {
	var result Job
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		current, err := getJob(tx, j.ID)
		if err != nil {
			return err
		}
		if (current.State != "processing" && current.State != "retry_wait") || current.Generation != j.Generation || current.DispatchIntent || current.lease <= n || current.DeadlineAt.UnixMilli() <= n || attempt > 3 {
			return ErrJobConflict
		}
		state, code, next := "processing", "", int64(0)
		if waiting {
			state = "retry_wait"
			code = "preparation_unavailable"
			next = n + int64(attempt*100)
		}
		if _, err := tx.Exec("UPDATE jobs SET state=?,code=?,next_ms=?,attempts=?,updated_ms=? WHERE id=?", state, code, next, attempt, n, j.ID); err != nil {
			return err
		}
		result, err = getJob(tx, j.ID)
		return err
	})
	return result, err
}

func (s *Store) RecordForJob(id string) (Record, error) {
	var j Job
	var payload []byte
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		var err error
		j, err = getJob(tx, id)
		if err != nil {
			return err
		}
		payload, err = readJobPayload(tx, n, j)
		return err
	})
	if err != nil {
		return Record{}, err
	}
	r, err := s.unseal(j.CaptureID, payload)
	if err == nil && Hash(r.TunnelID) != j.tunnel {
		return Record{}, ErrUnavailable
	}
	return r, err
}

func (s *Store) Queue(captureID, tunnel, digest, parent string, approved bool) (Job, error) {
	var job Job
	if !approved {
		return job, errors.New("explicit replay approval required")
	}
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		if err := expireJobs(tx, n); err != nil {
			return err
		}
		payload, err := read(tx, n, captureID, tunnel)
		if err != nil {
			return err
		}
		record, err := s.unseal(captureID, payload)
		if err != nil {
			return err
		}
		if record.TunnelID != tunnel {
			return ErrUnavailable
		}
		if digest == "" || record.ConfigDigest != digest {
			return errors.New("replay configuration drift")
		}
		if parent != "" {
			old, err := getJob(tx, parent)
			if err != nil || old.CaptureID != captureID || old.tunnel != Hash(tunnel) {
				return ErrUnavailable
			}
			if (old.State == "processing" || old.State == "retry_wait") && old.lease > n {
				return ErrJobConflict
			}
			if old.State != "completed" {
				state := "cancelled"
				if old.DispatchIntent {
					state = "uncertain"
				}
				if _, err := tx.Exec("UPDATE jobs SET state=?,updated_ms=?,code='superseded_by_explicit_retry' WHERE id=?", state, n, parent); err != nil {
					return err
				}
			}
		}
		var count, total int
		if err := tx.QueryRow("SELECT count(*),coalesce(sum(CASE WHEN state IN ('queued','processing','retry_wait','recovering') THEN 1 ELSE 0 END),0) FROM jobs").Scan(&total, &count); err != nil {
			return err
		}
		if count >= 20 || total >= 2048 {
			return ErrQuota
		}
		id := rand.Text()
		_, err = tx.Exec("INSERT INTO jobs(id,capture_id,tunnel,state,deadline_ms,created_ms,updated_ms,parent_id,digest) VALUES(?,?,?,'queued',?,?,?,?,?)", id, captureID, Hash(tunnel), n+(30*time.Second).Milliseconds(), n, n, parent, digest)
		if err != nil {
			return err
		}
		job, err = getJob(tx, id)
		return err
	})
	return job, err
}

func (s *Store) Claim(id string) (Job, error) {
	var job Job
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		j, err := getJob(tx, id)
		if err != nil {
			return err
		}
		if j.State != "queued" || j.DispatchIntent || j.DeadlineAt.UnixMilli() <= n {
			return ErrJobConflict
		}
		var active int
		if err := tx.QueryRow("SELECT count(*) FROM jobs WHERE state IN ('processing','retry_wait') AND lease_ms>?", n).Scan(&active); err != nil {
			return err
		}
		if active > 0 {
			return ErrJobConflict
		}
		if _, err := tx.Exec("UPDATE jobs SET state='processing',generation=generation+1,lease_ms=deadline_ms+5000,attempts=attempts+1,updated_ms=? WHERE id=?", n, id); err != nil {
			return err
		}
		job, err = getJob(tx, id)
		return err
	})
	return job, err
}
func (s *Store) Intent(id string, generation int64) error {
	return s.transaction(func(tx *sql.Tx, n int64) error {
		j, err := getJob(tx, id)
		if err != nil {
			return err
		}
		if j.State != "processing" || j.Generation != generation || j.lease <= n || j.DeadlineAt.UnixMilli() <= n || j.DispatchIntent {
			return ErrJobConflict
		}
		var payload []byte
		if err := tx.QueryRow("SELECT payload FROM captures WHERE id=? AND tunnel=? AND deleted=0 AND expires_ms>?", j.CaptureID, j.tunnel, n).Scan(&payload); err != nil || payload == nil {
			return ErrUnavailable
		}
		r, err := s.unseal(j.CaptureID, payload)
		if err != nil || r.ConfigDigest != j.digest || Hash(r.TunnelID) != j.tunnel {
			return ErrUnavailable
		}
		_, err = tx.Exec("UPDATE jobs SET intent=1,updated_ms=? WHERE id=?", n, id)
		return err
	})
}
func (s *Store) finish(j Job, status *int, code string) (Job, error) {
	var result Job
	err := s.transaction(func(tx *sql.Tx, n int64) error {
		if err := invalidateJobs(tx, n); err != nil {
			return err
		}
		current, err := getJob(tx, j.ID)
		if err != nil {
			return err
		}
		if (current.State != "processing" && current.State != "retry_wait") || current.Generation != j.Generation || current.lease <= n {
			return ErrJobConflict
		}
		state := "failed"
		if current.DispatchIntent {
			state = "uncertain"
		}
		if status != nil {
			state = "completed"
		}
		if _, err := tx.Exec("UPDATE jobs SET state=?,http_status=?,code=?,updated_ms=? WHERE id=?", state, status, code, n, j.ID); err != nil {
			return err
		}
		result, err = getJob(tx, j.ID)
		return err
	})
	if err != nil {
		result, _ = s.Status(j.ID)
	}
	return result, err
}
