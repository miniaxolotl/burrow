package capture

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReplayIntentIsNeverAutomaticallyRedispatched(t *testing.T) {
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	if err := s.Put(record("one"), epoch); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue("one", "tunnel", "digest", "", false); err == nil {
		t.Fatal("unapproved work accepted")
	}
	job, err := s.Queue("one", "tunnel", "digest", "", true)
	if err != nil {
		t.Fatal(err)
	}
	job, err = s.Claim(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Intent(job.ID, job.Generation); err != nil {
		t.Fatal(err)
	}
	s.db.Exec("UPDATE jobs SET lease_ms=0 WHERE id=?", job.ID)
	status, err := s.Status(job.ID)
	if err != nil || status.State != "uncertain" {
		t.Fatalf("lost dispatch intent: %+v %v", status, err)
	}
	if _, err := s.Claim(job.ID); err == nil {
		t.Fatal("uncertain job dispatched again")
	}
	linked, err := s.Queue("one", "tunnel", "digest", job.ID, true)
	if err != nil || linked.ParentID != job.ID {
		t.Fatalf("explicit approved linked retry: %+v %v", linked, err)
	}
	if err := s.Delete("one", "tunnel"); err != nil {
		t.Fatal(err)
	}
	claimed, err := s.Claim(linked.ID)
	if err == nil {
		if err := s.Intent(claimed.ID, claimed.Generation); err == nil {
			t.Fatal("erased capture dispatched")
		}
	}
}

func TestReplayDisconnectAndEraseCannotCauseDuplicateEffects(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.Copy(io.Discard, r.Body)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	defer upstream.Close()
	_, p, _ := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	n, _ := strconv.Atoi(p)
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	r := record("one")
	r.Route.Port = uint16(n)
	if err := s.Put(r, epoch); err != nil {
		t.Fatal(err)
	}
	j, err := s.Queue("one", "tunnel", "digest", "", true)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Run(t.Context(), j.ID, nil)
	if err == nil || result.State != "uncertain" || calls.Load() != 1 {
		t.Fatalf("lost response not uncertain: %+v %v", result, err)
	}
	if _, err := s.Run(t.Context(), j.ID, nil); err == nil || calls.Load() != 1 {
		t.Fatal("uncertain request resent")
	}
	if err := s.Delete("one", "tunnel"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue("one", "tunnel", "digest", j.ID, true); err == nil {
		t.Fatal("retry revived erased evidence")
	}
}

func TestReplayDeletionCancelsActiveWorkerWithoutPublishingPayload(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer upstream.Close()
	_, p, _ := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	n, _ := strconv.Atoi(p)
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	r := record("one")
	r.Route.Port = uint16(n)
	if err := s.Put(r, epoch); err != nil {
		t.Fatal(err)
	}
	j, err := s.Queue("one", "tunnel", "digest", "", true)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan Job, 1)
	go func() { result, _ := s.Run(t.Context(), j.ID, nil); done <- result }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not dispatch")
	}
	if err := s.Clear("tunnel"); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		if result.State != "uncertain" {
			t.Fatal(result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker ignored durable cancellation")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("upstream cancellation missing")
	}
}

func TestReplayUsesFreshLoopbackConnectionAndSeparatesHTTPOutcome(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || r.URL.RequestURI() != "/hook" || string(body) != "synthetic-private-body" {
			t.Error("replay reconstruction changed")
		}
		w.WriteHeader(500)
	}))
	defer upstream.Close()
	_, p, _ := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	n, _ := strconv.Atoi(p)
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	r := record("one")
	r.Route.Port = uint16(n)
	if err := s.Put(r, epoch); err != nil {
		t.Fatal(err)
	}
	job, err := s.Queue("one", "tunnel", "digest", "", true)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Run(context.Background(), job.ID, nil)
	if err == nil || result.State != "completed" || result.HTTPStatus == nil || *result.HTTPStatus != 500 || calls.Load() != 1 {
		t.Fatalf("HTTP failure conflated/retried: %+v %v calls=%d", result, err, calls.Load())
	}
	if _, err := s.Run(context.Background(), job.ID, nil); err == nil || calls.Load() != 1 {
		t.Fatal("completed job resent")
	}
}

func TestExpiredPendingJobsReleaseQuotaWithoutExecution(t *testing.T) {
	s := testStore(t)
	epoch, _ := s.Epoch("tunnel")
	if err := s.Put(record("one"), epoch); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if _, err := s.Queue("one", "tunnel", "digest", "", true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Queue("one", "tunnel", "digest", "", true); err == nil {
		t.Fatal("pending quota not enforced")
	}
	if _, err := s.db.Exec("UPDATE jobs SET deadline_ms=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Queue("one", "tunnel", "digest", "", true); err != nil {
		t.Fatalf("expired unsent work blocked scope forever: %v", err)
	}
}
