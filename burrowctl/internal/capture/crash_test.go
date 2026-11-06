package capture

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// This helper runs only in the test binary, never in the packaged CLI. Its fixed
// synthetic key is deliberately not a production backend or environment override.
func TestReplayCrashFixture(t *testing.T) {
	phase := os.Getenv("BURROW_REPLAY_CRASH_PHASE")
	if phase == "" {
		t.Skip("subprocess fixture")
	}
	s, err := Open(os.Getenv("BURROW_REPLAY_CRASH_DIR"), "synthetic-scope", memoryKeys{key: bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	id := os.Getenv("BURROW_REPLAY_CRASH_JOB")
	if phase == "claimed" || phase == "intent" {
		j, err := s.Claim(id)
		if err != nil {
			t.Fatal(err)
		}
		if phase == "intent" {
			if err := s.Intent(j.ID, j.Generation); err != nil {
				t.Fatal(err)
			}
		}
		os.Exit(91)
	}
	if _, err := s.Run(context.Background(), id, nil); err != nil {
		t.Fatal(err)
	}
	os.Exit(91)
}

func TestReplayCrashRecoveryUsesDurableEvidence(t *testing.T) {
	for _, phase := range []string{"claimed", "intent", "received", "result"} {
		t.Run(phase, func(t *testing.T) {
			var calls atomic.Int32
			received := make(chan struct{}, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				io.Copy(io.Discard, r.Body)
				received <- struct{}{}
				if phase == "received" {
					<-r.Context().Done()
					return
				}
				w.WriteHeader(204)
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
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReplayCrashFixture$")
			command.Env = append(os.Environ(), "BURROW_REPLAY_CRASH_PHASE="+phase, "BURROW_REPLAY_CRASH_DIR="+s.dir, "BURROW_REPLAY_CRASH_JOB="+j.ID)
			var output bytes.Buffer
			command.Stdout, command.Stderr = &output, &output
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			if phase == "received" {
				select {
				case <-received:
					command.Process.Kill()
				case <-ctx.Done():
					t.Fatal("worker did not reach receipt")
				}
			}
			err = command.Wait()
			if err == nil {
				t.Fatal("fixture did not crash")
			}
			if phase != "received" {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 91 {
					t.Fatalf("fixture failed unexpectedly: %v %s", err, &output)
				}
			}
			if _, err := s.db.Exec("UPDATE jobs SET lease_ms=0 WHERE id=?", j.ID); err != nil {
				t.Fatal(err)
			}
			status, err := s.Status(j.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := "uncertain"
			if phase == "claimed" {
				want = "recovering"
			}
			if phase == "result" {
				want = "completed"
			}
			if status.State != want {
				t.Fatalf("crash %s recovered as %s", phase, status.State)
			}
			before := calls.Load()
			if _, err := s.Run(t.Context(), j.ID, nil); err == nil || calls.Load() != before {
				t.Fatal("recovery automatically redispatched old work")
			}
		})
	}
}
